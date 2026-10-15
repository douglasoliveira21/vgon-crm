package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret        string
	JWTRefreshSecret string
	JWTExpiration    time.Duration
	JWTRefreshExpiry time.Duration

	// Evolution API
	EvolutionAPIURL     string
	EvolutionAPIKey     string
	EvolutionWebhookURL string
	// Shared secret echoed back by Evolution on every callback. Empty keeps
	// the endpoint open, for deployments that haven't configured it yet.
	EvolutionWebhookToken string

	// App
	AppPort     string
	AppEnv      string
	FrontendURL string

	// Storage
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageUseSSL    bool
	ClamAVAddr       string

	// Rate Limit
	RateLimitMax    int
	RateLimitWindow time.Duration

	// Background job workers. Long-running jobs (campaign sends, bot delay
	// nodes) hold a worker for their whole duration.
	JobWorkers int

	// Login CAPTCHA (Cloudflare Turnstile). Disabled when either key is empty.
	TurnstileSiteKey   string
	TurnstileSecretKey string

	// Whether anyone can create a tenant from the public signup form.
	AllowPublicRegistration bool

	// GLPI. The credentials are a single global account, not per tenant, so
	// GLPICompanyID names the one company allowed to reach it. Empty means
	// super admins only — never every tenant, which would expose one
	// helpdesk's tickets to anyone who can sign up.
	GLPIBaseURL   string
	GLPIAppToken  string
	GLPIUserToken string
	GLPICompanyID string

	// Email OAuth
	GoogleClientID        string
	GoogleClientSecret    string
	MicrosoftClientID     string
	MicrosoftClientSecret string
	OAuthRedirectBaseURL  string

	// SMTP
	SMTPHost           string
	SMTPPort           string
	SMTPUser           string
	SMTPPass           string
	SMTPFrom           string
	EmailWebhookSecret string
}

func Load() *Config {
	return &Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgresql://postgres:password@localhost:5432/evocrm"),
		RedisURL:              getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:             getEnv("JWT_SECRET", "default-secret-change-me"),
		JWTRefreshSecret:      getEnv("JWT_REFRESH_SECRET", "default-refresh-secret-change-me"),
		JWTExpiration:         parseDuration(getEnv("JWT_EXPIRATION", "24h")),
		JWTRefreshExpiry:      parseDuration(getEnv("JWT_REFRESH_EXPIRATION", "720h")),
		EvolutionAPIURL:       getEnv("EVOLUTION_API_URL", "http://localhost:8080"),
		EvolutionAPIKey:       getEnv("EVOLUTION_API_KEY", ""),
		EvolutionWebhookURL:   getEnv("EVOLUTION_WEBHOOK_URL", "http://localhost:3001/api/webhooks/evolution"),
		EvolutionWebhookToken: getEnv("EVOLUTION_WEBHOOK_TOKEN", ""),
		AppPort:               getEnv("APP_PORT", "3001"),
		AppEnv:                getEnv("APP_ENV", "development"),
		FrontendURL:           getEnv("FRONTEND_URL", "http://localhost:3000"),
		StorageEndpoint:       getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey:      getEnv("STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey:      getEnv("STORAGE_SECRET_KEY", "minioadmin"),
		StorageBucket:         getEnv("STORAGE_BUCKET", "evocrm"),
		StorageUseSSL:         getEnv("STORAGE_USE_SSL", "false") == "true",
		ClamAVAddr:            getEnv("CLAMAV_ADDR", ""),
		RateLimitMax:          parseInt(getEnv("RATE_LIMIT_MAX", "1200"), 1200),
		RateLimitWindow:       parseDuration(getEnv("RATE_LIMIT_WINDOW", "1m")),
		JobWorkers:            parseInt(getEnv("JOB_WORKERS", "8"), 8),
		TurnstileSiteKey:      getEnv("TURNSTILE_SITE_KEY", ""),
		TurnstileSecretKey:    getEnv("TURNSTILE_SECRET_KEY", ""),
		// Closed by default: there is no public signup page, and tenants are
		// created by a super admin through /api/admin/tenants, so the only
		// traffic this endpoint ever saw was somebody calling it directly.
		AllowPublicRegistration: getEnv("ALLOW_PUBLIC_REGISTRATION", "false") == "true",
		GLPIBaseURL:           getEnv("GLPI_BASE_URL", ""),
		GLPIAppToken:          getEnv("GLPI_APP_TOKEN", ""),
		GLPIUserToken:         getEnv("GLPI_USER_TOKEN", ""),
		GLPICompanyID:         getEnv("GLPI_COMPANY_ID", ""),
		GoogleClientID:        getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:    getEnv("GOOGLE_CLIENT_SECRET", ""),
		MicrosoftClientID:     getEnv("MICROSOFT_CLIENT_ID", ""),
		MicrosoftClientSecret: getEnv("MICROSOFT_CLIENT_SECRET", ""),
		OAuthRedirectBaseURL:  getEnv("OAUTH_REDIRECT_BASE_URL", ""),
		SMTPHost:              getEnv("SMTP_HOST", ""),
		SMTPPort:              getEnv("SMTP_PORT", "587"),
		SMTPUser:              getEnv("SMTP_USER", ""),
		SMTPPass:              getEnv("SMTP_PASS", ""),
		SMTPFrom:              getEnv("SMTP_FROM", getEnv("SMTP_USER", "")),
		EmailWebhookSecret:    getEnv("EMAIL_WEBHOOK_SECRET", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}

func parseInt(s string, defaultValue int) int {
	value, err := strconv.Atoi(s)
	if err != nil || value <= 0 {
		return defaultValue
	}
	return value
}
