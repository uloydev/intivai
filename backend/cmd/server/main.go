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
	"github.com/intivai/backend/internal/iam/api"
	"github.com/intivai/backend/internal/iam/application"
	"github.com/intivai/backend/internal/iam/infrastructure/auth"
	iamrepo "github.com/intivai/backend/internal/iam/infrastructure/persistence"
	"github.com/intivai/backend/internal/llm"
	memapp "github.com/intivai/backend/internal/memory/application"
	memdomain "github.com/intivai/backend/internal/memory/domain"
	"github.com/intivai/backend/internal/memory/infrastructure/native"
	pgmem "github.com/intivai/backend/internal/memory/infrastructure/postgres"
	"github.com/intivai/backend/internal/shared/httpmw"
	"github.com/intivai/backend/pkg/config"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/logger"
	"github.com/intivai/backend/pkg/queue"
	"github.com/intivai/backend/pkg/storage"
	"github.com/rs/zerolog"
)

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
	defer sqlDB.Close()

	rdb := queue.NewRedis(cfg.Redis.Addr)
	defer rdb.Close()

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
	_ = llmClient

	var memoryFactory memdomain.BankFactory
	if cfg.Memory.Driver == "postgres" {
		memoryFactory = pgmem.NewPostgresFactory(pool)
	} else {
		memoryFactory = native.NewNativeFactory(cfg.Memory.DataDir)
	}
	syncWorker := memapp.NewSyncWorker(memoryFactory)

	// --- IAM ---
	iamRepo := iamrepo.NewPostgresIAMRepo(pool)
	txManager := iamrepo.NewPostgresTxManager(pool)
	hasher := auth.NewBcryptHasher(cfg.Auth.BcryptCost)
	tokens := auth.NewJWTProvider(cfg.Auth.JWTSecret)

	registerOrg := application.NewRegisterOrg(iamRepo, hasher, txManager)
	authenticate := application.NewAuthenticate(iamRepo, hasher, tokens, time.Duration(cfg.Auth.JWTExpiryHrs)*time.Hour)
	createUser := application.NewCreateUser(iamRepo, hasher)
	authHandler := api.NewAuthHandler(registerOrg, authenticate, createUser)

	// --- HTTP ---
	app := fiber.New(fiber.Config{
		AppName:      "intivai",
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

	authed := v1.Group("", authMW, tenantMW, userRateLimit)
	authed.Get("/me", authHandler.Me)
	authed.Post("/users", authHandler.CreateUser)

	// --- Worker (asynq) ---
	worker := queue.NewServer(cfg.Redis.Addr, 10, logger)
	go func() {
		if err := worker.Start(syncWorker.Mux()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("worker")
		}
	}()

	// --- Graceful shutdown ---
	go func() {
		<-ctx.Done()
		logger.Info().Msg("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
