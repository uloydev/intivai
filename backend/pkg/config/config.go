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

	cfg := &Config{}

	cfg.App.Port = v.GetString("APP_PORT")
	if cfg.App.Port == "" {
		cfg.App.Port = "8080"
	}
	cfg.App.Env = v.GetString("ENV")
	if cfg.App.Env == "" {
		cfg.App.Env = "dev"
	}
	cfg.App.AllowedOrigins = splitCSV(v.GetString("ALLOWED_ORIGINS"))
	cfg.App.PublicURL = strings.TrimSuffix(v.GetString("APP_PUBLIC_URL"), "/")
	if cfg.App.PublicURL == "" {
		cfg.App.PublicURL = "http://localhost:5173"
	}
	cfg.Sandbox.SidecarAddr = v.GetString("SANDBOX_SIDECAR_ADDR")
	cfg.Sandbox.CACert = v.GetString("SANDBOX_CA_CERT")
	cfg.Sandbox.ClientCert = v.GetString("SANDBOX_CLIENT_CERT")
	cfg.Sandbox.ClientKey = v.GetString("SANDBOX_CLIENT_KEY")

	cfg.Database.URL = v.GetString("DATABASE_URL")
	cfg.Database.MigrateURL = v.GetString("MIGRATE_URL")

	cfg.Redis.Addr = v.GetString("REDIS_ADDR")
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379"
	}

	cfg.MinIO.Endpoint = v.GetString("MINIO_ENDPOINT")
	if cfg.MinIO.Endpoint == "" {
		cfg.MinIO.Endpoint = "localhost:9000"
	}
	cfg.MinIO.AccessKey = v.GetString("MINIO_ACCESS_KEY")
	cfg.MinIO.SecretKey = v.GetString("MINIO_SECRET_KEY")
	cfg.MinIO.Bucket = v.GetString("MINIO_BUCKET")
	if cfg.MinIO.Bucket == "" {
		cfg.MinIO.Bucket = "intivai"
	}
	cfg.MinIO.UseSSL = v.GetBool("MINIO_USE_SSL")

	cfg.Auth.JWTSecret = v.GetString("JWT_SECRET")
	cfg.Auth.JWTExpiryHrs = v.GetInt("JWT_EXPIRY_HRS")
	if cfg.Auth.JWTExpiryHrs == 0 {
		cfg.Auth.JWTExpiryHrs = 12
	}
	cfg.Auth.WSTicketMins = v.GetInt("WS_TICKET_MINS")
	if cfg.Auth.WSTicketMins == 0 {
		cfg.Auth.WSTicketMins = 10
	}
	cfg.Auth.BcryptCost = v.GetInt("BCRYPT_COST")
	if cfg.Auth.BcryptCost == 0 {
		cfg.Auth.BcryptCost = 10
	}

	cfg.LLM.DeepSeekAPIKey = v.GetString("DEEPSEEK_API_KEY")
	cfg.LLM.DeepSeekBaseURL = v.GetString("DEEPSEEK_BASE_URL")
	if cfg.LLM.DeepSeekBaseURL == "" {
		cfg.LLM.DeepSeekBaseURL = "https://api.deepseek.com/v1"
	}
	cfg.LLM.DeepSeekModel = v.GetString("DEEPSEEK_MODEL")
	if cfg.LLM.DeepSeekModel == "" {
		cfg.LLM.DeepSeekModel = "deepseek-chat"
	}
	cfg.LLM.FallbackBaseURL = v.GetString("LLM_FALLBACK_BASE_URL")
	cfg.LLM.FallbackAPIKey = v.GetString("LLM_FALLBACK_API_KEY")
	cfg.LLM.MaxRetries = v.GetInt("LLM_MAX_RETRIES")
	if cfg.LLM.MaxRetries == 0 {
		cfg.LLM.MaxRetries = 3
	}

	cfg.Memory.Driver = v.GetString("MEMORY_DRIVER")
	if cfg.Memory.Driver == "" {
		cfg.Memory.Driver = "sqlite"
	}
	if cfg.Memory.Driver != "sqlite" && cfg.Memory.Driver != "postgres" {
		return nil, fmt.Errorf("invalid MEMORY_DRIVER %q: must be sqlite or postgres", cfg.Memory.Driver)
	}
	cfg.Memory.DataDir = v.GetString("MEMORY_DATA_DIR")
	if cfg.Memory.DataDir == "" {
		cfg.Memory.DataDir = "./data"
	}
	cfg.Embeddings.Enabled = v.GetBool("EMBEDDINGS_ENABLED")
	cfg.Embeddings.ModelDir = v.GetString("EMBED_MODEL_DIR")
	if cfg.Embeddings.ModelDir == "" {
		cfg.Embeddings.ModelDir = "./models"
	}
	cfg.Sentry.DSN = v.GetString("SENTRY_DSN")

	cfg.Cv.MaxUploadMB = v.GetInt("CV_MAX_UPLOAD_MB")
	if cfg.Cv.MaxUploadMB == 0 {
		cfg.Cv.MaxUploadMB = 10
	}
	if cfg.Cv.MaxUploadMB < 0 {
		return nil, fmt.Errorf("invalid CV_MAX_UPLOAD_MB %d: must be positive", cfg.Cv.MaxUploadMB)
	}

	cfg.RateLimit.TenantPerMin = v.GetInt("RATE_LIMIT_TENANT_PER_MIN")
	if cfg.RateLimit.TenantPerMin == 0 {
		cfg.RateLimit.TenantPerMin = 1000
	}
	cfg.RateLimit.UserPerMin = v.GetInt("RATE_LIMIT_USER_PER_MIN")
	if cfg.RateLimit.UserPerMin == 0 {
		cfg.RateLimit.UserPerMin = 100
	}
	cfg.RateLimit.AuthPerMin = v.GetInt("RATE_LIMIT_AUTH_PER_MIN")
	if cfg.RateLimit.AuthPerMin == 0 {
		cfg.RateLimit.AuthPerMin = 10
	}

	cfg.SMTP.Host = v.GetString("SMTP_HOST")
	if cfg.SMTP.Host == "" {
		cfg.SMTP.Host = "localhost"
	}
	cfg.SMTP.Port = v.GetInt("SMTP_PORT")
	if cfg.SMTP.Port == 0 {
		cfg.SMTP.Port = 1025
	}
	cfg.SMTP.Username = v.GetString("SMTP_USER")
	cfg.SMTP.Password = v.GetString("SMTP_PASS")
	cfg.SMTP.From = v.GetString("SMTP_FROM")
	if cfg.SMTP.From == "" {
		cfg.SMTP.From = "Intivai Talent <no-reply@intivai.com>"
	}

	return cfg, nil
}

func splitCSV(s string) []string {
	if s == "" {
		return []string{"http://localhost:3000"}
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
