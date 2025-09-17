package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	NATS     NATSConfig
	Auth     AuthConfig
	Stripe   StripeConfig
	Email    EmailConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	URL         string
	MaxConns    int
	MinConns    int
	MaxLifetime time.Duration
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type NATSConfig struct {
	URL       string
	ClusterID string
}

type AuthConfig struct {
	JWTSecret           string
	GuestSessionTTL     time.Duration
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	EmailVerificationTTL time.Duration
}

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	Environment   string // sandbox or live
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	SMTPFrom     string
	SMTPUseTLS   bool
	MailerSendKey string
	DevMode      bool // print emails to logs instead of sending
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getDuration("SERVER_READ_TIMEOUT", 5*time.Second),
			WriteTimeout: getDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			URL:         getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/luxsuv?sslmode=disable"),
			MaxConns:    getInt("DB_MAX_CONNS", 10),
			MinConns:    getInt("DB_MIN_CONNS", 1),
			MaxLifetime: getDuration("DB_MAX_LIFETIME", time.Hour),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
		},
		NATS: NATSConfig{
			URL:       getEnv("NATS_URL", "nats://localhost:4222"),
			ClusterID: getEnv("NATS_CLUSTER_ID", "luxsuv-cluster"),
		},
		Auth: AuthConfig{
			JWTSecret:           getEnv("JWT_SECRET", "dev-only-secret-change-in-prod"),
			GuestSessionTTL:     getDuration("GUEST_SESSION_TTL", 30*time.Minute),
			AccessTokenTTL:      getDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL:     getDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
			EmailVerificationTTL: getDuration("EMAIL_VERIFICATION_TTL", 2*time.Hour),
		},
		Stripe: StripeConfig{
			SecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
			Environment:   getEnv("STRIPE_ENV", "sandbox"),
		},
		Email: EmailConfig{
			SMTPHost:      getEnv("SMTP_HOST", "localhost"),
			SMTPPort:      getInt("SMTP_PORT", 1025),
			SMTPUser:      getEnv("SMTP_USER", ""),
			SMTPPass:      getEnv("SMTP_PASS", ""),
			SMTPFrom:      getEnv("SMTP_FROM", "noreply@luxsuv.local"),
			SMTPUseTLS:    getBool("SMTP_USE_TLS", false),
			MailerSendKey: getEnv("MAILERSEND_API_KEY", ""),
			DevMode:       getBool("EMAIL_DEV_MODE", true),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}