package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds every runtime setting for the API, loaded from environment
// variables (.env file or process env). All settings are production-safe with
// sensible defaults for local development.
type Config struct {
	AppName     string
	Environment string
	Debug       bool
	Port        string
	APIBasePath string
	SiteURL     string

	// Database (PostgreSQL)
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	DBMaxOpenConns int
	DBMaxIdleConns int
	DBConnMaxLift  time.Duration

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	CacheTTL      time.Duration

	// MinIO / S3-compatible object storage
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
	MinIORegion    string
	PublicBaseURL  string

	// JWT
	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration

	// SMTP (email)
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string

	// MailAfrica (email over REST API). FromDomain is the sending domain the
	// user owns on MailAfrica (e.g. ziadapos.com); the numeric domain ID is
	// resolved automatically from the account on first use.
	MailAfricaAPIKey      string
	MailAfricaBaseURL     string
	MailAfricaFromDomain  string
	MailAfricaFromAddress string
	MailAfricaFromName    string

	// SendAfrica SMS integration
	SendAfricaBaseURL  string
	SendAfricaAPIKey   string
	SendAfricaSenderID string

	// Ngamia AI integration
	NgamiaBaseURL string
	NgamiaAPIKey  string
	NgamiaModel   string

	// Scheduler (Africa/Dar_es_Salaam)
	DailyReportHour   int
	DailyReportMinute int
	Timezone          string

	// CORS
	CORSAllowedOrigins []string
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func splitList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Load reads config from the environment. If envFile is non-empty and the
// file exists it is loaded first (godotenv never overrides already-set vars).
func Load(envFile string) (*Config, error) {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			log.Printf("config: no .env file at %s (%v)", envFile, err)
		}
	}

	environment := env("APP_ENV", "development")

	cfg := &Config{
		AppName:     env("APP_NAME", "ZiadaPOS API"),
		Environment: environment,
		Debug:       envBool("DEBUG", environment == "development"),
		Port:        env("PORT", "8080"),
		APIBasePath: env("API_BASE_PATH", "/api/v1"),
		SiteURL:     env("SITE_URL", "http://localhost:3000"),

		DBHost:         env("DB_HOST", "localhost"),
		DBPort:         env("DB_PORT", "5432"),
		DBUser:         env("DB_USER", "ziada"),
		DBPassword:     env("DB_PASSWORD", "ziada"),
		DBName:         env("DB_NAME", "ziada"),
		DBSSLMode:      env("DB_SSLMODE", "disable"),
		DBMaxOpenConns: envInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: envInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLift:  envDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),

		RedisAddr:     env("REDIS_ADDR", "localhost:6379"),
		RedisPassword: env("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),
		CacheTTL:      envDuration("CACHE_TTL", 5*time.Minute),

		MinIOEndpoint:  env("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: env("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: env("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    env("MINIO_BUCKET", "ziada"),
		MinIOUseSSL:    envBool("MINIO_USE_SSL", false),
		MinIORegion:    env("MINIO_REGION", "us-east-1"),
		PublicBaseURL:  env("PUBLIC_BASE_URL", "http://localhost:9000/ziada"),

		JWTAccessSecret:  env("JWT_ACCESS_SECRET", "change-me-access"),
		JWTRefreshSecret: env("JWT_REFRESH_SECRET", "change-me-refresh"),
		JWTAccessTTL:     envDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:    envDuration("JWT_REFRESH_TTL", 7*24*time.Hour),

		SMTPHost:     env("SMTP_HOST", "smtp.zoho.com"),
		SMTPPort:     envInt("SMTP_PORT", 587),
		SMTPUser:     env("SMTP_USER", ""),
		SMTPPassword: env("SMTP_PASSWORD", ""),
		SMTPFrom:     env("SMTP_FROM", "no-reply@ziadapos.com"),
		SMTPFromName: env("SMTP_FROM_NAME", "ZiadaPOS"),

		MailAfricaAPIKey:      env("MAILAFRICA_API_KEY", ""),
		MailAfricaBaseURL:     env("MAILAFRICA_BASE_URL", "https://api.mailafrica.online"),
		MailAfricaFromDomain:  env("MAILAFRICA_FROM_DOMAIN", ""),
		MailAfricaFromAddress: env("MAILAFRICA_FROM_ADDRESS", ""),
		MailAfricaFromName:    env("MAILAFRICA_FROM_NAME", "ZiadaPOS"),

		SendAfricaBaseURL:  env("SENDAFRICA_BASE_URL", "https://api.sendafrica.online"),
		SendAfricaAPIKey:   env("SENDAFRICA_API_KEY", ""),
		SendAfricaSenderID: env("SENDAFRICA_SENDER_ID", "SendAfrika"),

		NgamiaBaseURL: env("NGAMIA_BASE_URL", "https://api.ngamia.cc/v1"),
		NgamiaAPIKey:  env("NGAMIA_API_KEY", ""),
		NgamiaModel:   env("NGAMIA_MODEL", "openai/gpt-4o-mini"),

		DailyReportHour:   envInt("DAILY_REPORT_HOUR", 22),
		DailyReportMinute: envInt("DAILY_REPORT_MINUTE", 0),
		Timezone:          env("APP_TIMEZONE", "Africa/Dar_es_Salaam"),

		CORSAllowedOrigins: splitList(env("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:1420,http://localhost:5173,https://app.ziadapos.com")),
	}

	if cfg.JWTAccessSecret == "change-me-access" || cfg.JWTRefreshSecret == "change-me-refresh" {
		log.Println("config: WARNING using default JWT secrets - set JWT_ACCESS_SECRET and JWT_REFRESH_SECRET in production")
	}

	return cfg, nil
}

// DSN builds the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

// IsProduction reports whether the app runs in a production environment.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}
