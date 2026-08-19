package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
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
	evapi "github.com/intivai/backend/internal/evaluation/api"
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
	notifapp "github.com/intivai/backend/internal/notification/application"
	sbapi "github.com/intivai/backend/internal/sandbox/api"
	sbapp "github.com/intivai/backend/internal/sandbox/application"
	"github.com/intivai/backend/internal/sandbox/infrastructure/sidecarclient"
	scrapi "github.com/intivai/backend/internal/screening/api"
	scrapp "github.com/intivai/backend/internal/screening/application"
	scrrepo "github.com/intivai/backend/internal/screening/infrastructure/persistence"
	"github.com/intivai/backend/internal/shared/httpmw"
	"github.com/intivai/backend/pkg/config"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/logger"
	"github.com/intivai/backend/pkg/mailer"
	"github.com/intivai/backend/pkg/queue"
	"github.com/intivai/backend/pkg/storage"
	"github.com/rs/zerolog"
)

// interviewEnqueuer — async evaluation retry and candidate invitation emails via the shared asynq client.
type interviewEnqueuer struct {
	client    *queue.Client
	publicURL string
}

func (e interviewEnqueuer) EnqueueEvaluation(ctx context.Context, orgID, interviewID string) error {
	_, err := e.client.Enqueue(ctx, evalapp.TaskEvaluateInterview, evalapp.EvaluatePayload{
		OrgID: orgID, InterviewID: interviewID,
	}, asynq.MaxRetry(5))
	return err
}

func (e interviewEnqueuer) EnqueueInterviewInvitation(ctx context.Context, to, name, jobTitle, interviewID, inviteToken string) error {
	inviteURL := fmt.Sprintf("%s/invite/%s?t=%s", strings.TrimSuffix(e.publicURL, "/"), interviewID, inviteToken)
	_, err := e.client.Enqueue(ctx, notifapp.TaskSendEmail, notifapp.SendEmailPayload{
		Type:          notifapp.EmailTypeInvitation,
		To:            to,
		CandidateName: name,
		JobTitle:      jobTitle,
		InviteURL:     inviteURL,
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

	if cfg.Sentry.DSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.Sentry.DSN,
			Environment:      cfg.App.Env,
			TracesSampleRate: 0.1,
		}); err != nil {
			logger.Warn().Err(err).Msg("sentry init")
		} else {
			defer sentry.Flush(2 * time.Second)
		}
	}

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
	tokenLedger := llm.NewRedisTokenLedger(rdb, 100_000) // Default 100k daily cap for now

	llmClient := llm.NewClient(
		llm.NewDeepSeekProvider(cfg.LLM.DeepSeekAPIKey, cfg.LLM.DeepSeekBaseURL, cfg.LLM.DeepSeekModel),
		fallback,
		tokenLedger,
		cfg.LLM.MaxRetries,
	)

	var memoryFactory memdomain.BankFactory
	var embedder emb.Embedder
	if cfg.Memory.Driver == "postgres" {
		pgFactory := pgmem.NewPostgresFactory(pool)
		if cfg.Embeddings.Enabled {
			embedder = emb.NewBGESmall(cfg.Embeddings.ModelDir)
			pgFactory = pgFactory.WithEmbedder(embedder)
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
	jobService := jobapp.NewJobService(jobRepo, queueClient)
	jobHandler := jobapi.NewJobHandler(jobService)

	appRepo := scrrepo.NewPostgresApplicationRepo(pool)

	candidateRepo := cvrepo.NewPostgresCandidateRepo(pool)
	cvService := cvapp.NewCVService(candidateRepo, appRepo, store, queueClient)
	cvHandler := cvapi.NewCVHandler(cvService, cfg.Cv.MaxUploadMB)

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
	interviewService := ivapp.NewInterviewService(pool, ivRepo, tokenRepo, questionBank, appRepo, candidateRepo, jobRepo, contextRepo, store, tokens, ivdomain.SystemClock(), interviewEnqueuer{client: queueClient, publicURL: cfg.App.PublicURL})
	sessionRegistry := ivapi.NewRedisSessionRegistry(rdb, 35*time.Minute)
	chatHandler := ivapi.NewChatHandler(interviewService, llmClient, tokens, logger, sessionRegistry)
	evalService := evalapp.NewEvaluationService(pool, ivRepo, appRepo, candidateRepo, jobRepo, store)
	evalHandler := evapi.NewEvaluationHandler(evalService)

	mailClient := mailer.NewSMTPMailer(mailer.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	}, logger)
	emailWorker := notifapp.NewEmailWorker(mailClient, logger)
	publicJobHandler := jobapi.NewPublicJobHandler(pool, jobRepo, candidateRepo, appRepo, store, queueClient)
	portalRepo := scrrepo.NewPostgresCandidatePortalRepo(pool)
	candidatePortalHandler := scrapi.NewCandidatePortalHandler(portalRepo, tokens, queueClient, cfg.App.PublicURL)
	// --- Sandbox sidecar (ADR-0002): the app talks to the sandbox executor
	// over mTLS gRPC; it never executes code itself. Fail closed when the
	// sidecar is not configured/unreachable (code.run frames return an error)
	// — the app must still boot so the rest of the product stays up.
	var codeRunner sbapp.CodeRunner
	if cfg.Sandbox.SidecarAddr != "" && cfg.Sandbox.CACert != "" && cfg.Sandbox.ClientCert != "" && cfg.Sandbox.ClientKey != "" {
		sidecarClient, err := sidecarclient.NewClient(ctx, cfg.Sandbox.SidecarAddr, cfg.Sandbox.CACert, cfg.Sandbox.ClientCert, cfg.Sandbox.ClientKey)
		if err != nil {
			logger.Warn().Err(err).Msg("sandbox sidecar unavailable — code execution disabled (fail closed)")
		} else {
			defer func() { _ = sidecarClient.Close() }()
			codeRunner = sidecarClient
		}
	} else {
		logger.Warn().Msg("sandbox sidecar not configured — code execution disabled (fail closed)")
	}
	sandboxService := sbapp.NewSandboxService(pool, codeRunner, llmClient, ivRepo)
	sandboxHandler := sbapi.NewSandboxHandler(sandboxService)
	chatHandler.WithCodeRunner(codeRunner)

	// --- Workers ---
	parseWorker := cvapp.NewParseWorker(pool, candidateRepo, store, queueClient, logger)
	extractWorker := cvapp.NewExtractWorker(pool, candidateRepo, appRepo, jobRepo, llmClient, queueClient, cfg.App.PublicURL, logger)
	scoreWorker := scrapp.NewScoreWorker(pool, appRepo, candidateRepo, jobRepo, orgSettings{repo: iamRepo}, embedder, logger)
	indexWorker := ctxapp.NewIndexWorker(pool, contextRepo, store, memoryFactory, logger)
	rubricWorker := jobapp.NewRubricWorker(pool, jobRepo, llmClient, logger)
	workerMux := asynq.NewServeMux()
	syncWorker.Register(workerMux)
	parseWorker.Register(workerMux)
	extractWorker.Register(workerMux)
	scoreWorker.Register(workerMux)
	indexWorker.Register(workerMux)
	rubricWorker.Register(workerMux)
	evalWorker.Register(workerMux)
	emailWorker.Register(workerMux)

	// --- HTTP ---
	app := fiber.New(fiber.Config{
		AppName:      "intivai",
		BodyLimit:    32 * 1024 * 1024, // uploads (cv pdf, context files)
		ErrorHandler: errorHandler,
	})

	prometheus := fiberprometheus.New("intivai")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

	app.Use(recover.New())
	app.Use(httpmw.RequestID(logger))
	app.Use(httpmw.Audit(logger))
	app.Use(httpmw.CORS(cfg.App.AllowedOrigins))

	app.Get("/health", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	app.Get("/live", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	app.Get("/ready", func(c *fiber.Ctx) error {
		sqlDB, err := pool.DB()
		if err != nil {
			return c.SendStatus(http.StatusServiceUnavailable)
		}
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(ctx); err != nil {
			return c.SendStatus(http.StatusServiceUnavailable)
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			return c.SendStatus(http.StatusServiceUnavailable)
		}
		if err := store.Ping(ctx); err != nil {
			return c.SendStatus(http.StatusServiceUnavailable)
		}
		return c.SendStatus(http.StatusOK)
	})

	// Per-IP auth rate limit (10/min) + tenant/user limits via Redis sliding window
	authRateLimit := httpmw.RateLimit(rdb, cfg.RateLimit.AuthPerMin, time.Minute, func(c *fiber.Ctx) string {
		return "auth:" + c.IP()
	})
	publicRateLimit := httpmw.RateLimit(rdb, 100, time.Minute, func(c *fiber.Ctx) string {
		return "public:" + c.IP()
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

	v1 := app.Group("/api/v1")

	// Public Job Board & Application endpoints (unauthenticated, rate-limited)
	publicRoutes := v1.Group("/public", publicRateLimit)
	publicRoutes.Get("/jobs", publicJobHandler.ListPublicJobs)
	publicRoutes.Get("/jobs/:id", publicJobHandler.GetPublicJob)
	publicRoutes.Post("/jobs/:id/apply", authRateLimit, publicJobHandler.Apply)
	publicRoutes.Post("/candidate/auth/otp", authRateLimit, candidatePortalHandler.RequestOTP)
	publicRoutes.Post("/candidate/auth/verify", authRateLimit, candidatePortalHandler.VerifyOTP)
	publicRoutes.Get("/candidate-review/:token", cvHandler.ReviewProfile)
	publicRoutes.Post("/candidate-review/:token/confirm", cvHandler.ConfirmProfile)

	authRoutes := v1.Group("/auth")
	authRoutes.Post("/register", authRateLimit, authHandler.Register)
	authRoutes.Post("/login", authRateLimit, authHandler.Login)

	// Candidate (public) routes MUST be registered BEFORE the authed group:
	// fiber's Group("", handlers...) registers app.Use("/") — global for every
	// route registered AFTER it. Candidate endpoints would otherwise inherit
	// the tenant/auth middleware (regression-tested in route_groups_test.go).
	v1.Get("/candidate/portal/applications", authRateLimit, candidatePortalHandler.RequireCandidateAuth, candidatePortalHandler.ListApplications)
	v1.Get("/candidate/portal/export", authRateLimit, candidatePortalHandler.RequireCandidateAuth, candidatePortalHandler.Export)
	v1.Delete("/candidate/portal/me", authRateLimit, candidatePortalHandler.RequireCandidateAuth, candidatePortalHandler.DeleteMe)
	v1.Post("/candidate/interviews/:id/consent", authRateLimit, chatHandler.Consent)
	v1.Post("/candidate/interviews/:id/ticket", authRateLimit, chatHandler.Ticket)
	v1.Post("/candidate/interviews/:id/telemetry", userRateLimit, chatHandler.Telemetry)
	v1.Get("/candidate/interviews/:id/chat", chatHandler.RequireTicket, chatHandler.Chat(cfg.App.AllowedOrigins))
	chatHandler.RegisterVoiceRoutes(v1, cfg.App.AllowedOrigins)

	authed := v1.Group("", authMW, tenantRateLimit, userRateLimit, tenantMW)
	authed.Get("/me", authHandler.Me)
	authed.Post("/users", authHandler.CreateUser)
	authed.Post("/sandbox/execute", sandboxHandler.Execute)
	authed.Post("/sandbox/evaluate", sandboxHandler.Evaluate)

	authed.Post("/jobs", jobHandler.Create)
	authed.Get("/jobs", jobHandler.List)
	authed.Get("/jobs/:id", jobHandler.Get)
	authed.Patch("/jobs/:id", jobHandler.Update)

	authed.Post("/cvs", cvHandler.Upload)
	authed.Post("/cvs/bulk", cvHandler.BulkUpload)
	authed.Get("/cvs", cvHandler.List)
	authed.Get("/cvs/:id", cvHandler.Get)
	authed.Post("/cvs/:id/extract", cvHandler.ReExtract)
	authed.Delete("/cvs/:id", cvHandler.Delete)
	authed.Delete("/candidates/:id", cvHandler.Delete)

	authed.Post("/screenings", screeningHandler.Create)
	authed.Get("/applications", screeningHandler.List)
	authed.Patch("/applications/:id", screeningHandler.UpdateDecision)

	authed.Post("/orgs/:orgId/contexts", contextHandler.UploadContext)
	authed.Get("/orgs/:orgId/contexts", contextHandler.ListContexts)
	authed.Put("/orgs/:orgId/prompt", contextHandler.SetPrompt)
	authed.Get("/orgs/:orgId/prompt", contextHandler.GetPrompt)

	authed.Post("/interviews", chatHandler.Create)
	authed.Get("/interviews", evalHandler.ListInterviews)
	authed.Get("/interviews/:id", evalHandler.GetInterview)
	authed.Get("/interviews/:id/report/pdf", evalHandler.GetInterviewPDF)
	authed.Get("/candidates/:id/report", evalHandler.GetCandidateReport)

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
	if sentry.CurrentHub().Client() != nil {
		sentry.CaptureException(err)
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
