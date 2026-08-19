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
		DeepSeekAPIKey  string
		DeepSeekBaseURL string
		DeepSeekModel   string
		FallbackBaseURL string
		FallbackAPIKey  string
		MaxRetries      int
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

	cfg.LLM.DeepSeekAPIKey = v.GetString("DEEPSEEK_API_KEY")
	cfg.LLM.DeepSeekBaseURL = getString("DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1")
	cfg.LLM.DeepSeekModel = getString("DEEPSEEK_MODEL", "deepseek-v4-flash")
	cfg.LLM.FallbackBaseURL = v.GetString("LLM_FALLBACK_BASE_URL")
	cfg.LLM.FallbackAPIKey = v.GetString("LLM_FALLBACK_API_KEY")
	cfg.LLM.MaxRetries = getInt("LLM_MAX_RETRIES", 3)

	cfg.Memory.Driver = getString("MEMORY_DRIVER", "sqlite")
	if cfg.Memory.Driver != "sqlite" && cfg.Memory.Driver != "postgres" {
		return nil, fmt.Errorf("invalid MEMORY_DRIVER %q: must be sqlite or postgres", cfg.Memory.Driver)
	}
	cfg.Memory.DataDir = getString("MEMORY_DATA_DIR", "./data")
	cfg.Embeddings.Enabled = v.GetBool("EMBEDDINGS_ENABLED")
	cfg.Embeddings.ModelDir = getString("EMBED_MODEL_DIR", "./models")
	cfg.Sentry.DSN = v.GetString("SENTRY_DSN")

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
