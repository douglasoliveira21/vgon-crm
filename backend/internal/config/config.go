package config

import (
	"os"
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

	// Rate Limit
	RateLimitMax    int
	RateLimitWindow time.Duration

	// GLPI
	GLPIBaseURL   string
	GLPIAppToken  string
	GLPIUserToken string

	// Email OAuth
	GoogleClientID        string
	GoogleClientSecret    string
	MicrosoftClientID     string
	MicrosoftClientSecret string
	OAuthRedirectBaseURL  string
}

func Load() *Config {
	return &Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgresql://postgres:password@localhost:5432/evocrm"),
		RedisURL:              getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:             getEnv("JWT_SECRET", "default-secret-change-me"),
		JWTRefreshSecret:      getEnv("JWT_REFRESH_SECRET", "default-refresh-secret-change-me"),
		JWTExpiration:         parseDuration(getEnv("JWT_EXPIRATION", "15m")),
		JWTRefreshExpiry:      parseDuration(getEnv("JWT_REFRESH_EXPIRATION", "168h")),
		EvolutionAPIURL:       getEnv("EVOLUTION_API_URL", "http://localhost:8080"),
		EvolutionAPIKey:       getEnv("EVOLUTION_API_KEY", ""),
		EvolutionWebhookURL:   getEnv("EVOLUTION_WEBHOOK_URL", "http://localhost:3001/api/webhooks/evolution"),
		AppPort:               getEnv("APP_PORT", "3001"),
		AppEnv:                getEnv("APP_ENV", "development"),
		FrontendURL:           getEnv("FRONTEND_URL", "http://localhost:3000"),
		StorageEndpoint:       getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey:      getEnv("STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey:      getEnv("STORAGE_SECRET_KEY", "minioadmin"),
		StorageBucket:         getEnv("STORAGE_BUCKET", "evocrm"),
		StorageUseSSL:         getEnv("STORAGE_USE_SSL", "false") == "true",
		RateLimitMax:          100,
		RateLimitWindow:       time.Minute,
		GLPIBaseURL:           getEnv("GLPI_BASE_URL", ""),
		GLPIAppToken:          getEnv("GLPI_APP_TOKEN", ""),
		GLPIUserToken:         getEnv("GLPI_USER_TOKEN", ""),
		GoogleClientID:        getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:    getEnv("GOOGLE_CLIENT_SECRET", ""),
		MicrosoftClientID:     getEnv("MICROSOFT_CLIENT_ID", ""),
		MicrosoftClientSecret: getEnv("MICROSOFT_CLIENT_SECRET", ""),
		OAuthRedirectBaseURL:  getEnv("OAUTH_REDIRECT_BASE_URL", ""),
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
