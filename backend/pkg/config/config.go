package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App struct {
		Port           string
		Env            string // dev | prod
		AllowedOrigins []string
		PublicURL      string // candidate-facing base URL (magic links, invites)
	}

	// Sandbox — the code-execution sidecar (ADR-0002). Empty SidecarAddr
	// disables sandbox execution (fail closed).
	Sandbox struct {
		SidecarAddr string // e.g. sandbox-sidecar:8443
		CACert      string
		ClientCert  string
		ClientKey   string
	}
	Database struct {
		URL        string
		MigrateURL string
	}
	Redis struct {
		Addr string
	}
	MinIO struct {
		Endpoint  string
		AccessKey string
		SecretKey string
		Bucket    string
		UseSSL    bool
	}
	Auth struct {
		JWTSecret    string
		JWTExpiryHrs int
		WSTicketMins int
		BcryptCost   int
	}
	LLM struct {
		APIKey          string // LLM_API_KEY
		BaseURL         string // LLM_BASE_URL
		Model           string // LLM_MODEL
		ProviderName    string // LLM_PROVIDER_NAME — human label for metrics/logs
		FallbackBaseURL string // LLM_FALLBACK_BASE_URL
		FallbackAPIKey  string // LLM_FALLBACK_API_KEY
		MaxRetries      int
		TimeoutSeconds  int // LLM_TIMEOUT_SECONDS — HTTP client timeout; reasoning models need more than the default
	}
	Memory struct {
		Driver  string // sqlite | postgres
		DataDir string
	}
	Embeddings struct {
		Enabled  bool
		ModelDir string
	}
	Sentry struct {
		DSN string
	}
	// Telemetry — OpenTelemetry tracing (docs/plans/active/otel-tracing-plan-2026-08-26.md).
	// Disabled by default; flipping OTEL_ENABLE=false at runtime is the kill switch.
	Telemetry struct {
		Enable       bool
		ServiceName  string
		OTLPEndpoint string
		SampleRatio  float64
	}
	Cv struct {
		MaxUploadMB int
	}
	RateLimit struct {
		TenantPerMin int
		UserPerMin   int
		AuthPerMin   int
	}
	SMTP struct {
		Host     string
		Port     int
		Username string
		Password string
		From     string
	}
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("INTIVAI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// getString returns def when the env var is unset/empty.
	getString := func(key, def string) string {
		if s := v.GetString(key); s != "" {
			return s
		}
		return def
	}
	// getInt returns def when the env var is unset/zero.
	getInt := func(key string, def int) int {
		if n := v.GetInt(key); n != 0 {
			return n
		}
		return def
	}

	cfg := &Config{}

	cfg.App.Port = getString("APP_PORT", "8080")
	cfg.App.Env = getString("ENV", "dev")
	cfg.App.AllowedOrigins = splitCSV(v.GetString("ALLOWED_ORIGINS"))
	cfg.App.PublicURL = strings.TrimSuffix(getString("APP_PUBLIC_URL", "http://localhost:5173"), "/")
	cfg.Sandbox.SidecarAddr = v.GetString("SANDBOX_SIDECAR_ADDR")
	cfg.Sandbox.CACert = v.GetString("SANDBOX_CA_CERT")
	cfg.Sandbox.ClientCert = v.GetString("SANDBOX_CLIENT_CERT")
	cfg.Sandbox.ClientKey = v.GetString("SANDBOX_CLIENT_KEY")

	cfg.Database.URL = v.GetString("DATABASE_URL")
	cfg.Database.MigrateURL = v.GetString("MIGRATE_URL")

	cfg.Redis.Addr = getString("REDIS_ADDR", "localhost:6379")

	cfg.MinIO.Endpoint = getString("MINIO_ENDPOINT", "localhost:9000")
	cfg.MinIO.AccessKey = v.GetString("MINIO_ACCESS_KEY")
	cfg.MinIO.SecretKey = v.GetString("MINIO_SECRET_KEY")
	cfg.MinIO.Bucket = getString("MINIO_BUCKET", "intivai")
	cfg.MinIO.UseSSL = v.GetBool("MINIO_USE_SSL")

	cfg.Auth.JWTSecret = v.GetString("JWT_SECRET")
	cfg.Auth.JWTExpiryHrs = getInt("JWT_EXPIRY_HRS", 12)
	cfg.Auth.WSTicketMins = getInt("WS_TICKET_MINS", 10)
	cfg.Auth.BcryptCost = getInt("BCRYPT_COST", 10)

	cfg.LLM.APIKey = v.GetString("LLM_API_KEY")
	cfg.LLM.BaseURL = getString("LLM_BASE_URL", "https://ai.sumopod.com/v1")
	cfg.LLM.Model = getString("LLM_MODEL", "MiniMax-M2.7-highspeed")
	cfg.LLM.ProviderName = getString("LLM_PROVIDER_NAME", "sumopod")
	cfg.LLM.FallbackBaseURL = v.GetString("LLM_FALLBACK_BASE_URL")
	cfg.LLM.FallbackAPIKey = v.GetString("LLM_FALLBACK_API_KEY")
	cfg.LLM.MaxRetries = getInt("LLM_MAX_RETRIES", 3)
	cfg.LLM.TimeoutSeconds = getInt("LLM_TIMEOUT_SECONDS", 60)
	if cfg.LLM.TimeoutSeconds <= 0 {
		return nil, fmt.Errorf("invalid LLM_TIMEOUT_SECONDS %d: must be positive", cfg.LLM.TimeoutSeconds)
	}

	cfg.Memory.Driver = getString("MEMORY_DRIVER", "sqlite")
	if cfg.Memory.Driver != "sqlite" && cfg.Memory.Driver != "postgres" {
		return nil, fmt.Errorf("invalid MEMORY_DRIVER %q: must be sqlite or postgres", cfg.Memory.Driver)
	}
	cfg.Memory.DataDir = getString("MEMORY_DATA_DIR", "./data")
	cfg.Embeddings.Enabled = v.GetBool("EMBEDDINGS_ENABLED")
	cfg.Embeddings.ModelDir = getString("EMBED_MODEL_DIR", "./models")
	cfg.Sentry.DSN = v.GetString("SENTRY_DSN")

	// Telemetry — ratio default 1.0 (dev-friendly); prod compose pins 0.1.
	cfg.Telemetry.Enable = v.GetBool("OTEL_ENABLE")
	cfg.Telemetry.ServiceName = getString("OTEL_SERVICE_NAME", "intivai-server")
	cfg.Telemetry.OTLPEndpoint = getString("OTEL_EXPORTER_OTLP_ENDPOINT", "http://jaeger:4318")
	if r := v.GetFloat64("OTEL_TRACES_SAMPLER_ARG"); r > 0 {
		cfg.Telemetry.SampleRatio = r
	} else if cfg.Telemetry.Enable {
		cfg.Telemetry.SampleRatio = 1
	}

	cfg.Cv.MaxUploadMB = getInt("CV_MAX_UPLOAD_MB", 10)
	if cfg.Cv.MaxUploadMB < 0 {
		return nil, fmt.Errorf("invalid CV_MAX_UPLOAD_MB %d: must be positive", cfg.Cv.MaxUploadMB)
	}

	cfg.RateLimit.TenantPerMin = getInt("RATE_LIMIT_TENANT_PER_MIN", 1000)
	cfg.RateLimit.UserPerMin = getInt("RATE_LIMIT_USER_PER_MIN", 100)
	cfg.RateLimit.AuthPerMin = getInt("RATE_LIMIT_AUTH_PER_MIN", 10)

	cfg.SMTP.Host = getString("SMTP_HOST", "localhost")
	cfg.SMTP.Port = getInt("SMTP_PORT", 1025)
	cfg.SMTP.Username = v.GetString("SMTP_USER")
	cfg.SMTP.Password = v.GetString("SMTP_PASS")
	cfg.SMTP.From = getString("SMTP_FROM", "Intivai Talent <no-reply@intivai.com>")

	return cfg, nil
}

func splitCSV(s string) []string {
	if s == "" {
		return []string{"http://localhost:5173"}
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
