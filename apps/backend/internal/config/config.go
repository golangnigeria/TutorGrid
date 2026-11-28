package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
)

type Config struct {
	Primary       Primary              `koanf:"primary" validate:"required"`
	Server        ServerConfig         `koanf:"server" validate:"required"`
	Database      DatabaseConfig       `koanf:"database" validate:"required"`
	Auth          AuthConfig           `koanf:"auth" validate:"required"`
	Redis         RedisConfig          `koanf:"redis" validate:"required"`
	NATS          NATSConfig           `koanf:"nats" validate:"required"`
	Storage       StorageConfig        `koanf:"storage" validate:"required"`
	Integration   IntegrationConfig    `koanf:"integration" validate:"required"`
	Observability *ObservabilityConfig `koanf:"observability"`
}

type Primary struct {
	Env string `koanf:"env" validate:"required"`
}

type ServerConfig struct {
	Port               string   `koanf:"port" validate:"required"`
	ReadTimeout        int      `koanf:"read_timeout" validate:"required"`
	WriteTimeout       int      `koanf:"write_timeout" validate:"required"`
	IdleTimeout        int      `koanf:"idle_timeout" validate:"required"`
	CORSAllowedOrigins []string `koanf:"cors_allowed_origins" validate:"required"`
}

type DatabaseConfig struct {
	Host            string `koanf:"host" validate:"required"`
	Port            int    `koanf:"port" validate:"required"`
	User            string `koanf:"user" validate:"required"`
	Password        string `koanf:"password"`
	Name            string `koanf:"name" validate:"required"`
	SSLMode         string `koanf:"ssl_mode" validate:"required"`
	MaxOpenConns    int    `koanf:"max_open_conns" validate:"required"`
	MaxIdleConns    int    `koanf:"max_idle_conns" validate:"required"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime" validate:"required"`
	ConnMaxIdleTime int    `koanf:"conn_max_idle_time" validate:"required"`
}

type RedisConfig struct {
	Address string `koanf:"address" validate:"required"`
}

type NATSConfig struct {
	URL string `koanf:"url" validate:"required"`
}

// StorageConfig configures TutorGrid's Backblaze B2 S3-compatible object storage (ADR 0003).
// Local MinIO, localhost, and non-HTTPS endpoints are rejected in all environments.
type StorageConfig struct {
	Endpoint        string `koanf:"endpoint" validate:"required,url"`
	Region          string `koanf:"region" validate:"required"`
	Bucket          string `koanf:"bucket" validate:"required"`
	AccessKeyID     string `koanf:"access_key_id" validate:"required"`
	SecretAccessKey string `koanf:"secret_access_key" validate:"required"`
	UsePathStyle    bool   `koanf:"use_path_style"`
}

// Validate enforces ADR 0003 Backblaze B2-only security rules across all environments.
func (s StorageConfig) Validate() error {
	raw := strings.TrimSpace(s.Endpoint)
	if raw == "" {
		return fmt.Errorf("storage endpoint is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid storage endpoint URL: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("storage endpoint must use https:// (got %q); non-HTTPS/local endpoints are disallowed", parsed.Scheme)
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return fmt.Errorf("storage endpoint hostname is required")
	}
	if host == "localhost" ||
		host == "127.0.0.1" ||
		host == "::1" ||
		host == "0.0.0.0" ||
		host == "minio" ||
		strings.Contains(host, "minio") ||
		strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("storage endpoint host %q is disallowed; TutorGrid uses Backblaze B2 exclusively (ADR 0003)", host)
	}
	if strings.TrimSpace(s.Region) == "" {
		return fmt.Errorf("storage region is required")
	}
	if strings.TrimSpace(s.Bucket) == "" {
		return fmt.Errorf("storage bucket is required")
	}
	if strings.TrimSpace(s.AccessKeyID) == "" || strings.TrimSpace(s.SecretAccessKey) == "" {
		return fmt.Errorf("storage access credentials are required")
	}
	return nil
}

type IntegrationConfig struct {
	ResendAPIKey    string `koanf:"resend_api_key" validate:"required"`
	FromEmail       string `koanf:"from_email" validate:"required,email"`
	FromName        string `koanf:"from_name" validate:"required"`
	ResendFromEmail string `koanf:"resend_from_email"`
	ResendFromName  string `koanf:"resend_from_name"`
}

// AuthConfig configures embedded Limen with Google and GitHub OAuth only (no email/password).
type AuthConfig struct {
	AppBaseURL         string `koanf:"app_base_url" validate:"required"`
	FrontendBaseURL    string `koanf:"frontend_base_url" validate:"required"`
	LimenSecret        string `koanf:"limen_secret" validate:"required,min=32"`
	GoogleClientID     string `koanf:"google_client_id" validate:"required"`
	GoogleClientSecret string `koanf:"google_client_secret" validate:"required"`
	GitHubClientID     string `koanf:"github_client_id" validate:"required"`
	GitHubClientSecret string `koanf:"github_client_secret" validate:"required"`
}

func LoadConfig() (*Config, error) {
	// Load root .env if present (never overrides already-set environment variables)
	_ = godotenv.Load(".env", "../../.env")

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	k := koanf.New(".")

	err := k.Load(env.Provider("TUTORGRID_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "TUTORGRID_"))
	}), nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load initial env variables")
	}

	// Pre-populate non-secret operational defaults before unmarshaling so partial
	// dotted env overrides (e.g. only NEW_RELIC.LICENSE_KEY) keep valid defaults.
	mainConfig := &Config{
		Primary: Primary{
			Env: "development",
		},
		Server: ServerConfig{
			Port:         "8080",
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  60,
		},
		Database: DatabaseConfig{
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 300,
			ConnMaxIdleTime: 300,
		},
		Redis: RedisConfig{
			Address: "localhost:6379",
		},
		NATS: NATSConfig{
			URL: "nats://localhost:4222",
		},
		Storage: StorageConfig{
			UsePathStyle: true,
		},
		Integration: IntegrationConfig{
			FromEmail: "onboarding@resend.dev",
			FromName:  "TutorGrid Circle",
		},
		Observability: DefaultObservabilityConfig(),
	}

	err = k.Unmarshal("", mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not unmarshal main config")
	}

	applyFlatEnvCompatibility(mainConfig)

	mainConfig.Observability.ServiceName = "tutorgrid"
	mainConfig.Observability.Environment = mainConfig.Primary.Env

	validate := validator.New()

	err = validate.Struct(mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("config validation failed")
	}

	if err := mainConfig.Observability.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("invalid observability config")
	}
	if err := mainConfig.Storage.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("invalid storage config")
	}

	return mainConfig, nil
}

func applyFlatEnvCompatibility(cfg *Config) {
	if cfg.Integration.ResendFromEmail != "" {
		cfg.Integration.FromEmail = cfg.Integration.ResendFromEmail
	}
	if cfg.Integration.ResendFromName != "" {
		cfg.Integration.FromName = cfg.Integration.ResendFromName
	}
	if v := os.Getenv("TUTORGRID_NATS_URL"); v != "" && os.Getenv("TUTORGRID_NATS.URL") == "" {
		cfg.NATS.URL = v
	}
	if v := os.Getenv("TUTORGRID_STORAGE_ENDPOINT"); v != "" && os.Getenv("TUTORGRID_STORAGE.ENDPOINT") == "" {
		cfg.Storage.Endpoint = v
	}
	if v := os.Getenv("TUTORGRID_STORAGE_REGION"); v != "" && os.Getenv("TUTORGRID_STORAGE.REGION") == "" {
		cfg.Storage.Region = v
	}
	if v := os.Getenv("TUTORGRID_STORAGE_BUCKET"); v != "" && os.Getenv("TUTORGRID_STORAGE.BUCKET") == "" {
		cfg.Storage.Bucket = v
	}
	if v := os.Getenv("TUTORGRID_STORAGE_ACCESS_KEY_ID"); v != "" && os.Getenv("TUTORGRID_STORAGE.ACCESS_KEY_ID") == "" {
		cfg.Storage.AccessKeyID = v
	}
	if v := os.Getenv("TUTORGRID_STORAGE_SECRET_ACCESS_KEY"); v != "" && os.Getenv("TUTORGRID_STORAGE.SECRET_ACCESS_KEY") == "" {
		cfg.Storage.SecretAccessKey = v
	}
	if v := os.Getenv("TUTORGRID_APP_ENV"); v != "" && os.Getenv("TUTORGRID_PRIMARY.ENV") == "" {
		cfg.Primary.Env = v
	}
	if v := os.Getenv("TUTORGRID_HTTP_ADDR"); v != "" && os.Getenv("TUTORGRID_SERVER.PORT") == "" {
		cfg.Server.Port = strings.TrimPrefix(v, ":")
	}

	// Auth flat env compatibility
	if cfg.Auth.AppBaseURL == "" {
		cfg.Auth.AppBaseURL = os.Getenv("TUTORGRID_APP_BASE_URL")
	}
	if cfg.Auth.FrontendBaseURL == "" {
		cfg.Auth.FrontendBaseURL = os.Getenv("TUTORGRID_FRONTEND_BASE_URL")
	}
	if cfg.Auth.LimenSecret == "" {
		cfg.Auth.LimenSecret = os.Getenv("TUTORGRID_LIMEN_SECRET")
	}
	if cfg.Auth.GoogleClientID == "" {
		cfg.Auth.GoogleClientID = os.Getenv("TUTORGRID_GOOGLE_CLIENT_ID")
	}
	if cfg.Auth.GoogleClientSecret == "" {
		cfg.Auth.GoogleClientSecret = os.Getenv("TUTORGRID_GOOGLE_CLIENT_SECRET")
	}
	if cfg.Auth.GitHubClientID == "" {
		cfg.Auth.GitHubClientID = os.Getenv("TUTORGRID_GITHUB_CLIENT_ID")
	}
	if cfg.Auth.GitHubClientSecret == "" {
		cfg.Auth.GitHubClientSecret = os.Getenv("TUTORGRID_GITHUB_CLIENT_SECRET")
	}

	if len(cfg.Server.CORSAllowedOrigins) == 0 && cfg.Auth.FrontendBaseURL != "" {
		cfg.Server.CORSAllowedOrigins = []string{cfg.Auth.FrontendBaseURL}
	}

	// Database URL flat compatibility
	if cfg.Database.Host == "" {
		if rawDSN := os.Getenv("TUTORGRID_DATABASE_URL"); rawDSN != "" {
			if u, err := url.Parse(rawDSN); err == nil {
				cfg.Database.Host = u.Hostname()
				if p, err := strconv.Atoi(u.Port()); err == nil && p > 0 {
					cfg.Database.Port = p
				} else {
					cfg.Database.Port = 5432
				}
				if u.User != nil {
					cfg.Database.User = u.User.Username()
					if pw, ok := u.User.Password(); ok {
						cfg.Database.Password = pw
					}
				}
				cfg.Database.Name = strings.TrimPrefix(u.Path, "/")
				if sslMode := u.Query().Get("sslmode"); sslMode != "" {
					cfg.Database.SSLMode = sslMode
				}
			}
		}
	}
}
