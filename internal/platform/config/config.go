package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds validated runtime configuration for the TutorGrid API.
type Config struct {
	AppEnv               string
	HTTPAddr             string
	DatabaseURL          string
	DatabaseMigrationURL string
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetime    time.Duration
	ShutdownTimeout      time.Duration
}

// LoadFromEnv reads and validates configuration from environment variables.
func LoadFromEnv() (Config, error) {
	cfg := Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		HTTPAddr:             getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		DatabaseMigrationURL: os.Getenv("DATABASE_MIGRATION_URL"),
		DBMaxOpenConns:       10,
		DBMaxIdleConns:       5,
		DBConnMaxLifetime:    30 * time.Minute,
		ShutdownTimeout:      10 * time.Second,
	}

	if v := os.Getenv("DB_MAX_OPEN_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return Config{}, fmt.Errorf("config: invalid DB_MAX_OPEN_CONNS: %q", v)
		}
		cfg.DBMaxOpenConns = n
	}

	if v := os.Getenv("DB_MAX_IDLE_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > cfg.DBMaxOpenConns {
			return Config{}, fmt.Errorf("config: invalid DB_MAX_IDLE_CONNS: %q", v)
		}
		cfg.DBMaxIdleConns = n
	}

	if v := os.Getenv("DB_CONN_MAX_LIFETIME"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("config: invalid DB_CONN_MAX_LIFETIME: %q", v)
		}
		cfg.DBConnMaxLifetime = d
	}

	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("config: invalid SHUTDOWN_TIMEOUT: %q", v)
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
		return errors.New("config: DATABASE_URL is required")
	}
	if c.HTTPAddr == "" {
		return errors.New("config: HTTP_ADDR is required")
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
