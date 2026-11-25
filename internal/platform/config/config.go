package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds validated runtime configuration for the TutorGrid API.
// All environment variables use the TUTORGRID_ prefix.
type Config struct {
	AppEnv               string
	HTTPAddr             string
	AppBaseURL           string
	FrontendBaseURL      string
	DatabaseURL          string
	DatabaseMigrationURL string
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetime    time.Duration
	ShutdownTimeout      time.Duration

	// Limen & OAuth (Google and GitHub only; no email/password)
	LimenSecret        string
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
}

// LoadFromEnv reads and validates configuration from TUTORGRID_-prefixed environment variables.
func LoadFromEnv() (Config, error) {
	cfg := Config{
		AppEnv:               getEnv("TUTORGRID_APP_ENV", "development"),
		HTTPAddr:             getEnv("TUTORGRID_HTTP_ADDR", ":8080"),
		AppBaseURL:           getEnv("TUTORGRID_APP_BASE_URL", "http://localhost:8080"),
		FrontendBaseURL:      getEnv("TUTORGRID_FRONTEND_BASE_URL", "http://localhost:5173"),
		DatabaseURL:          os.Getenv("TUTORGRID_DATABASE_URL"),
		DatabaseMigrationURL: os.Getenv("TUTORGRID_DATABASE_MIGRATION_URL"),
		DBMaxOpenConns:       10,
		DBMaxIdleConns:       5,
		DBConnMaxLifetime:    30 * time.Minute,
		ShutdownTimeout:      10 * time.Second,
		LimenSecret:          os.Getenv("TUTORGRID_LIMEN_SECRET"),
		GoogleClientID:       os.Getenv("TUTORGRID_GOOGLE_CLIENT_ID"),
		GoogleClientSecret:   os.Getenv("TUTORGRID_GOOGLE_CLIENT_SECRET"),
		GitHubClientID:       os.Getenv("TUTORGRID_GITHUB_CLIENT_ID"),
		GitHubClientSecret:   os.Getenv("TUTORGRID_GITHUB_CLIENT_SECRET"),
	}

	if v := os.Getenv("TUTORGRID_DB_MAX_OPEN_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return Config{}, fmt.Errorf("config: invalid TUTORGRID_DB_MAX_OPEN_CONNS: %q", v)
		}
		cfg.DBMaxOpenConns = n
	}

	if v := os.Getenv("TUTORGRID_DB_MAX_IDLE_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > cfg.DBMaxOpenConns {
			return Config{}, fmt.Errorf("config: invalid TUTORGRID_DB_MAX_IDLE_CONNS: %q", v)
		}
		cfg.DBMaxIdleConns = n
	}

	if v := os.Getenv("TUTORGRID_DB_CONN_MAX_LIFETIME"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("config: invalid TUTORGRID_DB_CONN_MAX_LIFETIME: %q", v)
		}
		cfg.DBConnMaxLifetime = d
	}

	if v := os.Getenv("TUTORGRID_SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("config: invalid TUTORGRID_SHUTDOWN_TIMEOUT: %q", v)
		}
		cfg.ShutdownTimeout = d
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks required fields and safe operational bounds.
func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("config: TUTORGRID_DATABASE_URL is required")
	}
	if c.HTTPAddr == "" {
		return errors.New("config: TUTORGRID_HTTP_ADDR is required")
	}
	if c.AppBaseURL == "" {
		return errors.New("config: TUTORGRID_APP_BASE_URL is required")
	}
	if c.FrontendBaseURL == "" {
		return errors.New("config: TUTORGRID_FRONTEND_BASE_URL is required")
	}
	if len(c.LimenSecret) < 32 {
		return errors.New("config: TUTORGRID_LIMEN_SECRET must be at least 32 characters")
	}
	if c.GoogleClientID == "" || c.GoogleClientSecret == "" {
		return errors.New("config: TUTORGRID_GOOGLE_CLIENT_ID and TUTORGRID_GOOGLE_CLIENT_SECRET are required")
	}
	if c.GitHubClientID == "" || c.GitHubClientSecret == "" {
		return errors.New("config: TUTORGRID_GITHUB_CLIENT_ID and TUTORGRID_GITHUB_CLIENT_SECRET are required")
	}
	if c.DBMaxOpenConns < 1 {
		return errors.New("config: DBMaxOpenConns must be >= 1")
	}
	if c.DBMaxIdleConns < 0 || c.DBMaxIdleConns > c.DBMaxOpenConns {
		return errors.New("config: DBMaxIdleConns must be between 0 and DBMaxOpenConns")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
