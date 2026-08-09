package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	ctxapi "github.com/intivai/backend/internal/context/api"
	ctxapp "github.com/intivai/backend/internal/context/application"
	ctxrepo "github.com/intivai/backend/internal/context/infrastructure/persistence"
	cvapi "github.com/intivai/backend/internal/cv/api"
	cvapp "github.com/intivai/backend/internal/cv/application"
	cvrepo "github.com/intivai/backend/internal/cv/infrastructure/persistence"
	emb "github.com/intivai/backend/internal/embedding"
	evalapp "github.com/intivai/backend/internal/evaluation/application"
	evalllm "github.com/intivai/backend/internal/evaluation/infrastructure/llm"
	"github.com/intivai/backend/internal/iam/api"
	"github.com/intivai/backend/internal/iam/application"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	iamrepo "github.com/intivai/backend/internal/iam/infrastructure/persistence"
	ivapi "github.com/intivai/backend/internal/interview/api"
	ivapp "github.com/intivai/backend/internal/interview/application"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	ivrepo "github.com/intivai/backend/internal/interview/infrastructure/persistence"
	jobapi "github.com/intivai/backend/internal/job/api"
	jobapp "github.com/intivai/backend/internal/job/application"
	jobrepo "github.com/intivai/backend/internal/job/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	memapp "github.com/intivai/backend/internal/memory/application"
	memdomain "github.com/intivai/backend/internal/memory/domain"
	"github.com/intivai/backend/internal/memory/infrastructure/native"
	pgmem "github.com/intivai/backend/internal/memory/infrastructure/postgres"
	scrapi "github.com/intivai/backend/internal/screening/api"
	scrapp "github.com/intivai/backend/internal/screening/application"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/internal/shared/httpmw"
	"github.com/intivai/backend/pkg/config"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/logger"
	"github.com/intivai/backend/pkg/queue"
	"github.com/intivai/backend/pkg/storage"
	"github.com/rs/zerolog"
)

var _ = ivdomain.Clock(nil)

// evalEnqueuer — async evaluation retry via the shared asynq client.
type evalEnqueuer struct {
	client *queue.Client
}

func (e evalEnqueuer) EnqueueEvaluation(ctx context.Context, orgID, interviewID string) error {
	_, err := e.client.Enqueue(ctx, evalapp.TaskEvaluateInterview, evalapp.EvaluatePayload{
		OrgID: orgID, InterviewID: interviewID,
	})
	return err
}

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply migrations with INTIVAI_MIGRATE_URL and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	logger := logger.New(cfg.App.Env)

	if *migrateOnly {
		if cfg.Database.MigrateURL == "" {
			logger.Fatal().Msg("INTIVAI_MIGRATE_URL is required with -migrate-only")
		}
		if err := db.Migrate(context.Background(), cfg.Database.MigrateURL); err != nil {
			logger.Fatal().Err(err).Msg("migrations")
		}
		logger.Info().Msg("migrations applied")
		return
	}

	if cfg.Auth.JWTSecret == "" {
		logger.Fatal().Msg("JWT_SECRET is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Infrastructure ---
	pool, err := db.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		logger.Fatal().Err(err).Msg("database")
	}
	sqlDB, err := pool.DB()
	if err != nil {
		logger.Fatal().Err(err).Msg("database")
	}
	defer func() { _ = sqlDB.Close() }()

	rdb := queue.NewRedis(cfg.Redis.Addr)
	defer func() { _ = rdb.Close() }()

	store, err := storage.New(cfg.MinIO.Endpoint, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, cfg.MinIO.Bucket, cfg.MinIO.UseSSL)
	if err != nil {
		logger.Fatal().Err(err).Msg("storage")
	}
	if err := store.EnsureBucket(ctx); err != nil {
		logger.Warn().Err(err).Msg("bucket ensure (will retry on first upload)")
	}

	// --- LLM + memory ---
	var fallback llm.Provider
	if cfg.LLM.FallbackBaseURL != "" {
		fallback = llm.NewDeepSeekProvider(cfg.LLM.FallbackAPIKey, cfg.LLM.FallbackBaseURL, cfg.LLM.DeepSeekModel)
	}
	llmClient := llm.NewClient(
		llm.NewDeepSeekProvider(cfg.LLM.DeepSeekAPIKey, cfg.LLM.DeepSeekBaseURL, cfg.LLM.DeepSeekModel),
		fallback,
		cfg.LLM.MaxRetries,
	)

	var memoryFactory memdomain.BankFactory
	if cfg.Memory.Driver == "postgres" {
		pgFactory := pgmem.NewPostgresFactory(pool)
		if cfg.Embeddings.Enabled {
			pgFactory = pgFactory.WithEmbedder(emb.NewBGESmall(cfg.Embeddings.ModelDir))
		}
		memoryFactory = pgFactory
	} else {
		memoryFactory = native.NewNativeFactory(cfg.Memory.DataDir)
	}
	syncWorker := memapp.NewSyncWorker(memoryFactory)

	queueClient := queue.NewClient(cfg.Redis.Addr)
	defer func() { _ = queueClient.Close() }()

	// --- IAM ---
	iamRepo := iamrepo.NewPostgresIAMRepo(pool)
	txManager := iamrepo.NewPostgresTxManager(pool)
	hasher := auth.NewBcryptHasher(cfg.Auth.BcryptCost)
	tokens := auth.NewJWTProvider(cfg.Auth.JWTSecret)

	registerOrg := application.NewRegisterOrg(iamRepo, hasher, txManager)
	authenticate := application.NewAuthenticate(iamRepo, hasher, tokens, time.Duration(cfg.Auth.JWTExpiryHrs)*time.Hour)
	createUser := application.NewCreateUser(iamRepo, hasher)
	authHandler := api.NewAuthHandler(registerOrg, authenticate, createUser)

	// --- M2 contexts: job, cv, screening, company context ---
	jobRepo := jobrepo.NewPostgresJobRepo(pool)
	jobService := jobapp.NewJobService(jobRepo)
	jobHandler := jobapi.NewJobHandler(jobService)

	candidateRepo := cvrepo.NewPostgresCandidateRepo(pool)
	cvService := cvapp.NewCVService(candidateRepo, store, queueClient)
	cvHandler := cvapi.NewCVHandler(cvService, cfg.Cv.MaxUploadMB)

	appRepo := scrrepo.NewPostgresApplicationRepo(pool)
	screeningService := scrapp.NewScreeningService(pool, appRepo, candidateRepo, jobRepo, queueClient)
	screeningHandler := scrapi.NewScreeningHandler(screeningService)

	contextRepo := ctxrepo.NewPostgresContextRepo(pool)
	contextService := ctxapp.NewContextService(pool, contextRepo, store, queueClient, logger)
	contextHandler := ctxapi.NewContextHandler(contextService)

	// --- M3: interviews ---
	ivRepo := ivrepo.NewPostgresInterviewRepo(pool)
	tokenRepo := ivrepo.NewPostgresTokenRepo(pool)
	questionBank := ivrepo.NewPostgresQuestionBank(pool)
	evalWorker := evalapp.NewEvaluationWorker(pool, ivRepo, evalllm.NewEvaluator(llmClient))
	interviewService := ivapp.NewInterviewService(pool, ivRepo, tokenRepo, questionBank, appRepo, candidateRepo, jobRepo, contextRepo, store, tokens, ivdomain.SystemClock(), evalEnqueuer{client: queueClient})
	chatHandler := ivapi.NewChatHandler(interviewService, llmClient, tokens, logger)

	// --- Workers ---
	parseWorker := cvapp.NewParseWorker(pool, candidateRepo, store, queueClient, logger)
	extractWorker := cvapp.NewExtractWorker(pool, candidateRepo, appRepo, jobRepo, llmClient, queueClient, logger)
	scoreWorker := scrapp.NewScoreWorker(pool, appRepo, candidateRepo, jobRepo, orgSettings{repo: iamRepo}, logger)
	indexWorker := ctxapp.NewIndexWorker(pool, contextRepo, store, memoryFactory, logger)
	workerMux := asynq.NewServeMux()
	syncWorker.Register(workerMux)
	parseWorker.Register(workerMux)
	extractWorker.Register(workerMux)
	scoreWorker.Register(workerMux)
	indexWorker.Register(workerMux)
	evalWorker.Register(workerMux)

	// --- HTTP ---
	app := fiber.New(fiber.Config{
		AppName:      "intivai",
		BodyLimit:    32 * 1024 * 1024, // uploads (cv pdf, context files)
		ErrorHandler: errorHandler,
	})
	app.Use(recover.New())
	app.Use(httpmw.RequestID(logger))
	app.Use(httpmw.Audit(logger))
	app.Use(httpmw.CORS(cfg.App.AllowedOrigins))

	app.Use(healthcheck.New(healthcheck.Config{
		LivenessProbe: func(c *fiber.Ctx) bool { return true },
		ReadinessProbe: func(c *fiber.Ctx) bool {
			sqlDB, err := pool.DB()
			if err != nil {
				return false
			}
			ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
			defer cancel()
			if err := sqlDB.PingContext(ctx); err != nil {
				return false
			}
			if err := rdb.Ping(ctx).Err(); err != nil {
				return false
			}
			return true
		},
	}))
	app.Get("/health", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	// Per-IP auth rate limit (10/min) + tenant/user limits via Redis sliding window
	authRateLimit := httpmw.RateLimit(rdb, cfg.RateLimit.AuthPerMin, time.Minute, func(c *fiber.Ctx) string {
		return "auth:" + c.IP()
	})
	tenantRateLimit := httpmw.RateLimit(rdb, cfg.RateLimit.TenantPerMin, time.Minute, func(c *fiber.Ctx) string {
		if actor, ok := api.Actor(c); ok {
			return "tenant:" + actor.OrgID.String()
		}
		return ""
	})
	userRateLimit := httpmw.RateLimit(rdb, cfg.RateLimit.UserPerMin, time.Minute, func(c *fiber.Ctx) string {
		if actor, ok := api.Actor(c); ok {
			return "user:" + actor.UserID.String()
		}
		return ""
	})

	authMW := api.AuthMiddleware(tokens)
	tenantMW := api.TenantTxMiddleware(pool)

	v1 := app.Group("/api/v1", tenantRateLimit)
	authRoutes := v1.Group("/auth")
	authRoutes.Post("/register", authRateLimit, authHandler.Register)
	authRoutes.Post("/login", authRateLimit, authHandler.Login)

	// Candidate (public) routes MUST be registered BEFORE the authed group:
	// fiber's Group("", handlers...) registers app.Use("/") — global for every
	// route registered AFTER it. Candidate endpoints would otherwise inherit
	// the tenant/auth middleware (regression-tested in route_groups_test.go).
	v1.Post("/candidate/interviews/:id/ticket", authRateLimit, chatHandler.Ticket)
	v1.Get("/candidate/interviews/:id/chat", chatHandler.RequireTicket, chatHandler.Chat(cfg.App.AllowedOrigins))

	authed := v1.Group("", authMW, tenantMW, userRateLimit)
	authed.Get("/me", authHandler.Me)
	authed.Post("/users", authHandler.CreateUser)

	authed.Post("/jobs", jobHandler.Create)
	authed.Get("/jobs", jobHandler.List)
	authed.Get("/jobs/:id", jobHandler.Get)
	authed.Patch("/jobs/:id", jobHandler.Update)

	authed.Post("/cvs", cvHandler.Upload)
	authed.Get("/cvs", cvHandler.List)
	authed.Get("/cvs/:id", cvHandler.Get)
	authed.Post("/cvs/:id/extract", cvHandler.ReExtract)

	authed.Post("/screenings", screeningHandler.Create)
	authed.Get("/applications", screeningHandler.List)

	authed.Post("/orgs/:orgId/contexts", contextHandler.UploadContext)
	authed.Get("/orgs/:orgId/contexts", contextHandler.ListContexts)
	authed.Put("/orgs/:orgId/prompt", contextHandler.SetPrompt)
	authed.Get("/orgs/:orgId/prompt", contextHandler.GetPrompt)

	authed.Post("/interviews", chatHandler.Create)

	// --- Worker (asynq) ---
	worker := queue.NewServer(cfg.Redis.Addr, 10, logger)
	go func() {
		if err := worker.Start(workerMux); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("worker")
		}
	}()

	// --- Graceful shutdown ---
	go func() {
		<-ctx.Done()
		logger.Info().Msg("shutting down...")
		// Must exceed the LLM client timeout (60s) so in-flight extraction
		// finishes instead of being redelivered after restart.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
		defer cancel()
		_ = app.ShutdownWithContext(shutdownCtx)
		_ = worker.Shutdown(shutdownCtx)
	}()

	logger.Info().Str("port", cfg.App.Port).Msg("listening")
	if err := app.Listen(":" + cfg.App.Port); err != nil {
		logger.Fatal().Err(err).Msg("listen")
	}
}

func errorHandler(c *fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}
	if logger, ok := c.Locals("logger").(zerolog.Logger); ok {
		logger.Error().Err(err).Msg("unhandled error")
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
}

// orgSettings adapts the IAM org repo to screening.OrgSettingsReader.
type orgSettings struct {
	repo *iamrepo.PostgresIAMRepo
}

func (a orgSettings) ReadOrgSettings(ctx context.Context, orgID uuid.UUID) (map[string]float64, float64, error) {
	org, err := a.repo.GetOrg(ctx, orgID)
	if err != nil {
		return nil, 0, err
	}
	weights := map[string]float64{}
	if org.ScoringWeights != nil {
		weights = org.ScoringWeights
	}
	minScore := 50.0
	if org.MinScoreToProceed != nil {
		minScore = *org.MinScoreToProceed
	}
	return weights, minScore, nil
}
