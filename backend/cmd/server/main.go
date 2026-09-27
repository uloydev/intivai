package main

import (
	"context"
	"encoding/json"
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
	fibersentry "github.com/gofiber/contrib/fibersentry"
	"github.com/gofiber/fiber/v2"
	fiberRecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	billingapi "github.com/intivai/backend/internal/billing/api"
	billingapp "github.com/intivai/backend/internal/billing/application"
	billingdomain "github.com/intivai/backend/internal/billing/domain"
	billinggateway "github.com/intivai/backend/internal/billing/infrastructure/gateway"
	billingrepo "github.com/intivai/backend/internal/billing/infrastructure/persistence"
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
	intapi "github.com/intivai/backend/internal/integration/api"
	intapp "github.com/intivai/backend/internal/integration/application"
	intdomain "github.com/intivai/backend/internal/integration/domain"
	intrepo "github.com/intivai/backend/internal/integration/infrastructure/persistence"
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
	notifapi "github.com/intivai/backend/internal/notification/api"
	notifapp "github.com/intivai/backend/internal/notification/application"
	notifdomain "github.com/intivai/backend/internal/notification/domain"
	notifrepo "github.com/intivai/backend/internal/notification/infrastructure/persistence"
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
	"github.com/intivai/backend/pkg/observability"
	"github.com/intivai/backend/pkg/queue"
	"github.com/intivai/backend/pkg/storage"
	"github.com/intivai/backend/pkg/telemetry"
	"github.com/rs/zerolog"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

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

func (e interviewEnqueuer) EnqueueHumanRequest(ctx context.Context, to, candidateName, jobTitle, interviewID string) error {
	reportURL := fmt.Sprintf("%s/interviews/%s", strings.TrimSuffix(e.publicURL, "/"), interviewID)
	_, err := e.client.Enqueue(ctx, notifapp.TaskSendEmail, notifapp.SendEmailPayload{
		Type:          notifapp.EmailTypeHumanRequest,
		To:            to,
		CandidateName: candidateName,
		JobTitle:      jobTitle,
		ReportURL:     reportURL,
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

	if err := observability.Init(cfg, observability.WithRelease(version)); err != nil {
		logger.Warn().Err(err).Msg("sentry init")
	} else if cfg.Sentry.DSN != "" {
		defer observability.Flush(2 * time.Second)
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

	// A8: empty/malformed PublicURL silently generates dead invite/review
	// links. In prod, fail fast; in dev, keep the local default (5173 via
	// config fallback) so local testing is unaffected.
	if cfg.App.Env == "prod" {
		pub := cfg.App.PublicURL
		if pub == "" || !strings.HasPrefix(pub, "http://") && !strings.HasPrefix(pub, "https://") {
			logger.Fatal().Msg("INTIVAI_APP_PUBLIC_URL must be an absolute http(s) URL in prod")
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Telemetry (tracing) — init before any instrumented component boots;
	// disabled by default, OTEL_ENABLE=false keeps everything noop.
	shutdownTracing, err := telemetry.Init(ctx, telemetry.Config{
		Enable:       cfg.Telemetry.Enable,
		ServiceName:  cfg.Telemetry.ServiceName,
		Env:          cfg.App.Env,
		OTLPEndpoint: cfg.Telemetry.OTLPEndpoint,
		SampleRatio:  cfg.Telemetry.SampleRatio,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("telemetry")
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	// --- Infrastructure ---
	// otelgorm emits one db span per statement (plan batch C) — noop when
	// tracing is disabled.
	pool, err := db.NewPool(ctx, cfg.Database.URL, db.WithPlugin(otelgorm.NewPlugin()))
	if err != nil {
		logger.Fatal().Err(err).Msg("database")
	}
	sqlDB, err := pool.DB()
	if err != nil {
		logger.Fatal().Err(err).Msg("database")
	}
	defer func() { _ = sqlDB.Close() }()

	asynqOpt, redisOpt, err := queue.ParseRedisConnOpt(queue.RedisConfig{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		URL:      cfg.Redis.URL,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("redis config")
	}

	rdb := queue.NewRedisClient(redisOpt)
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
		fallback = llm.NewOpenAIProvider("fallback", cfg.LLM.FallbackAPIKey, cfg.LLM.FallbackBaseURL, cfg.LLM.Model, cfg.LLM.TimeoutSeconds)
	}
	tokenLedger := llm.NewRedisTokenLedger(rdb, 100_000) // Default 100k daily cap for now

	llmClient := llm.NewClient(
		llm.NewOpenAIProvider(cfg.LLM.ProviderName, cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model, cfg.LLM.TimeoutSeconds),
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

	queueClient := queue.NewClientWithOpt(asynqOpt)
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
	orgSettingsHandler := api.NewOrgSettingsHandler(iamRepo)

	// --- M2 contexts: job, cv, screening, company context ---
	jobRepo := jobrepo.NewPostgresJobRepo(pool)
	jobCandRepo := jobrepo.NewPostgresCandidateContextRepo(pool)
	jobService := jobapp.NewJobService(jobRepo, queueClient).WithLogger(logger)
	jobHandler := jobapi.NewJobHandler(jobService)
	candidateContextHandler := jobapi.NewCandidateContextHandler(
		jobapp.NewCandidateContextService(pool, jobCandRepo, jobRepo, llmClient))

	appRepo := scrrepo.NewPostgresApplicationRepo(pool)

	candidateRepo := cvrepo.NewPostgresCandidateRepo(pool)
	cvService := cvapp.NewCVService(candidateRepo, appRepo, store, queueClient, pool)
	cvHandler := cvapi.NewCVHandler(cvService, cfg.Cv.MaxUploadMB)

	screeningService := scrapp.NewScreeningService(pool, appRepo, candidateRepo, jobRepo, queueClient, cfg.App.PublicURL)
	screeningHandler := scrapi.NewScreeningHandler(screeningService)

	contextRepo := ctxrepo.NewPostgresContextRepo(pool)
	contextService := ctxapp.NewContextService(pool, contextRepo, store, queueClient, logger)
	contextHandler := ctxapi.NewContextHandler(contextService)

	// --- Notifications ---
	notifRepo := notifrepo.NewPostgresNotificationRepo(pool)
	notifService := notifapp.NewNotificationService(pool, notifRepo)
	notifHandler := notifapi.NewNotificationHandler(notifService)

	// --- M3: interviews ---
	ivRepo := ivrepo.NewPostgresInterviewRepo(pool)
	tokenRepo := ivrepo.NewPostgresTokenRepo(pool)
	questionBank := ivrepo.NewPostgresQuestionBank(pool)
	evalWorker := evalapp.NewEvaluationWorker(pool, ivRepo, evalllm.NewEvaluator(llmClient), queueClient, cfg.App.PublicURL, logger).WithWebhookDispatch(func(ctx context.Context, orgID uuid.UUID, event intdomain.WebhookEvent, payload []byte) {
		// Query active webhook configs, create deliveries, enqueue — all inside
		// a tenant tx so FORCED RLS (025) applies and delivery insert + enqueue
		// commit atomically (enqueue failure rolls the delivery back).
		eventJSON, err := json.Marshal([]string{string(event)})
		if err != nil {
			logger.Error().Err(err).Str("event", string(event)).Msg("webhook dispatch: marshal event failed")
			return
		}
		if err := db.RunInTx(ctx, pool, orgID.String(), func(tctx context.Context) error {
			if event == intdomain.EventInterviewCompleted {
				_ = notifService.Create(tctx, orgID, notifdomain.EventInterviewCompleted,
					"Interview Completed",
					"An interview assessment finished and is ready for review.",
					"/interviews")
			}
			var configs []struct {
				ID  uuid.UUID
				URL string
			}
			if err := pool.WithContext(tctx).Raw(
				`SELECT id, url FROM webhook_configs WHERE org_id = $1 AND active = true AND events @> $2::jsonb`, orgID, string(eventJSON)).Scan(&configs).Error; err != nil {
				return fmt.Errorf("query webhook configs: %w", err)
			}
			for _, whCfg := range configs {
				deliveryID := uuid.New()
				if err := pool.WithContext(tctx).Exec(
					`INSERT INTO webhook_deliveries (id, org_id, webhook_id, event, payload, final_status, attempts, created_at, updated_at) VALUES ($1, $2, $3, $4, $5::jsonb, 'pending', 0, NOW(), NOW())`,
					deliveryID, orgID, whCfg.ID, string(event), string(payload)).Error; err != nil {
					return fmt.Errorf("insert webhook delivery: %w", err)
				}
				pl, err := json.Marshal(intapp.DeliverWebhookPayload{DeliveryID: deliveryID.String(), OrgID: orgID.String()})
				if err != nil {
					return fmt.Errorf("marshal delivery payload: %w", err)
				}
				if _, err := queueClient.Enqueue(ctx, intapp.TaskDeliverWebhook, pl, asynq.MaxRetry(3)); err != nil {
					return fmt.Errorf("enqueue webhook delivery: %w", err)
				}
			}
			return nil
		}); err != nil {
			logger.Error().Err(err).Str("org_id", orgID.String()).Str("event", string(event)).Msg("webhook dispatch failed")
		}
	})
	interviewService := ivapp.NewInterviewService(pool, ivRepo, tokenRepo, questionBank, appRepo, candidateRepo, jobRepo, jobCandRepo, contextRepo, store, tokens, ivdomain.SystemClock(), interviewEnqueuer{client: queueClient, publicURL: cfg.App.PublicURL}, orgSettings{repo: iamRepo}, logger)
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
	portalRepo := scrrepo.NewPostgresCandidatePortalRepo(pool)
	publicJobHandler := jobapi.NewPublicJobHandler(pool, jobRepo, candidateRepo, appRepo, store, queueClient, portalRepo, cfg.App.PublicURL)
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

	// --- Webhooks ---
	webhookRepo := intrepo.NewPostgresWebhookRepo(pool)
	webhookService := intapp.NewWebhookService(webhookRepo)
	webhookHandler := intapi.NewWebhookHandler(webhookService, logger)
	webhookWorker := intapp.NewWebhookWorker(webhookRepo, pool, logger)

	// --- Billing & Payment Gateway ---
	var paymentGateway billingdomain.PaymentGateway = billinggateway.NewNoopPaymentGateway()
	billingRepo := billingrepo.NewPostgresBillingRepo(pool)
	billingService := billingapp.NewBillingService(pool, billingRepo, paymentGateway)
	billingHandler := billingapi.NewBillingHandler(billingService)

	// --- Workers ---
	parseWorker := cvapp.NewParseWorker(pool, candidateRepo, store, queueClient, logger)
	extractWorker := cvapp.NewExtractWorker(pool, candidateRepo, appRepo, jobRepo, llmClient, queueClient, cfg.App.PublicURL, logger)
	scoreWorker := scrapp.NewScoreWorker(pool, appRepo, candidateRepo, jobRepo, orgSettings{repo: iamRepo}, embedder, logger)
	indexWorker := ctxapp.NewIndexWorker(pool, contextRepo, store, memoryFactory, logger)
	rubricWorker := jobapp.NewRubricWorker(pool, jobRepo, llmClient, logger)
	// Finding I1: the question worker was constructed nowhere while
	// Create/Update kept enqueueing generate_question_set tasks — they piled
	// up unconsumed. Same dependency set as the rubric worker.
	questionWorker := jobapp.NewQuestionWorker(pool, jobRepo, llmClient, logger)
	workerMux := asynq.NewServeMux()
	// Tracing first: delivery spans wrap panic recovery + handlers, and link
	// back to the producing HTTP trace via task headers (plan batch D).
	workerMux.Use(queue.TracingMiddleware())
	workerMux.Use(func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) (err error) {
			defer func() {
				if r := recover(); r != nil {
					observability.CapturePanic(ctx, r)
					logger.Error().Interface("panic", r).Str("task", task.Type()).Msg("worker panic recovered")
					err = fmt.Errorf("worker panic in %s: %v", task.Type(), r)
				}
			}()
			return h.ProcessTask(ctx, task)
		})
	})
	syncWorker.Register(workerMux)
	parseWorker.Register(workerMux)
	extractWorker.Register(workerMux)
	scoreWorker.Register(workerMux)
	indexWorker.Register(workerMux)
	rubricWorker.Register(workerMux)
	questionWorker.Register(workerMux)
	evalWorker.Register(workerMux)
	emailWorker.Register(workerMux)
	webhookWorker.Register(workerMux)

	// --- HTTP ---
	app := fiber.New(fiber.Config{
		AppName:      "intivai",
		BodyLimit:    32 * 1024 * 1024, // uploads (cv pdf, context files)
		ErrorHandler: errorHandler,
	})

	prometheus := fiberprometheus.New("intivai")
	prometheus.RegisterAt(app, "/metrics")
	// Tracing first: every downstream span nests inside the server span and
	// RequestID picks up trace_id for log correlation (plan B batch).
	app.Use(httpmw.Tracing(httpmw.TracingConfig{}))
	app.Use(prometheus.Middleware)

	// Sentry wraps recovery: it captures panics and repanics so the standard
	// recover middleware still logs and converts them to 500s.
	app.Use(fibersentry.New(fibersentry.Config{Repanic: true}))
	app.Use(fiberRecover.New())
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
	// Public apply gets its own tighter per-IP bucket — it is the only
	// unauthenticated endpoint that writes (candidate + application rows).
	publicApplyRateLimit := httpmw.RateLimit(rdb, cfg.RateLimit.AuthPerMin, time.Minute, httpmw.IPKey("public-apply:"))
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
	publicRoutes.Post("/jobs/:id/apply", publicApplyRateLimit, publicJobHandler.Apply)
	publicRoutes.Post("/candidate/auth/otp", authRateLimit, candidatePortalHandler.RequestOTP)
	publicRoutes.Post("/candidate/auth/verify", authRateLimit, candidatePortalHandler.VerifyOTP)
	publicRoutes.Post("/candidate/auth/logout", authRateLimit, candidatePortalHandler.Logout)
	publicRoutes.Get("/candidate-review/:token", cvHandler.ReviewProfile)
	publicRoutes.Post("/candidate-review/:token/confirm", cvHandler.ConfirmProfile)
	publicRoutes.Get("/invite-preview", publicRateLimit, chatHandler.InvitePreview)

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
	v1.Post("/candidate/interviews/:id/request-human", authRateLimit, chatHandler.RequestHuman)
	v1.Post("/candidate/interviews/:id/telemetry", httpmw.RateLimit(rdb, cfg.RateLimit.AuthPerMin, time.Minute, httpmw.IPKey("telemetry:")), chatHandler.Telemetry)
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
	candidateContextHandler.RegisterRoutes(authed)

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
	authed.Delete("/orgs/:orgId/contexts/:contextID", contextHandler.Delete)
	authed.Put("/orgs/:orgId/prompt", contextHandler.SetPrompt)
	authed.Get("/orgs/:orgId/prompt", contextHandler.GetPrompt)
	authed.Put("/orgs/:orgId/settings/candidate-qa-limit", orgSettingsHandler.UpdateCandidateQALimit)

	authed.Post("/interviews", chatHandler.Create)
	authed.Get("/interviews", evalHandler.ListInterviews)
	authed.Get("/interviews/:id", evalHandler.GetInterview)
	authed.Get("/interviews/:id/report/pdf", evalHandler.GetInterviewPDF)
	authed.Put("/interviews/:id/decision", evalHandler.UpdateDecision)
	authed.Get("/candidates/:id/report", evalHandler.GetCandidateReport)

	authed.Post("/webhooks", webhookHandler.Create)
	authed.Get("/webhooks", webhookHandler.List)
	authed.Get("/webhooks/:id", webhookHandler.Get)
	authed.Patch("/webhooks/:id", webhookHandler.Update)
	authed.Delete("/webhooks/:id", webhookHandler.Delete)
	authed.Get("/webhooks/:id/deliveries", webhookHandler.ListDeliveries)

	// --- Billing & Notifications ---
	authed.Get("/billing/summary", billingHandler.GetSummary)
	authed.Post("/billing/checkout", billingHandler.CreateCheckout)
	authed.Post("/billing/portal", billingHandler.CreatePortal)

	authed.Get("/notifications", notifHandler.List)
	authed.Patch("/notifications/:id/read", notifHandler.MarkRead)
	authed.Post("/notifications/read-all", notifHandler.MarkAllRead)

	// --- Worker (asynq) ---
	worker := queue.NewServerWithOpt(asynqOpt, 10, logger)
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
	observability.CaptureError(observability.FiberContext(c), err)
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

// CandidateQALimit adapts the IAM org repo to interview.OrgQALimitReader (D3/B4).
func (a orgSettings) CandidateQALimit(ctx context.Context, orgID uuid.UUID) (int, error) {
	org, err := a.repo.GetOrg(ctx, orgID)
	if err != nil {
		return 0, err
	}
	if org.CandidateQALimit == nil {
		return 0, nil // ResolveQALimit falls back to DefaultQALimit
	}
	return *org.CandidateQALimit, nil
}
