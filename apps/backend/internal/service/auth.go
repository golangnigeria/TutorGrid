package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/thecodearcher/limen"
	sqladapter "github.com/thecodearcher/limen/adapters/sql"
	"github.com/thecodearcher/limen/plugins/oauth"
	oauthgithub "github.com/thecodearcher/limen/plugins/oauth-github"
	oauthgoogle "github.com/thecodearcher/limen/plugins/oauth-google"

	"github.com/tutorgrid/tutorgrid/internal/server"
)

// AuthService wraps embedded Limen configured strictly for Google and GitHub OAuth.
type AuthService struct {
	server *server.Server
	limen  *limen.Limen
}

func NewAuthService(s *server.Server) (*AuthService, error) {
	if s.DB == nil || s.DB.Pool == nil {
		return nil, errors.New("auth: database pool is required")
	}

	secretBytes, err := parseLimenSecret(s.Config.Auth.LimenSecret)
	if err != nil {
		return nil, err
	}

	sqldb := stdlib.OpenDBFromPool(s.DB.Pool)

	schemaCfg := limen.NewDefaultSchemaConfig(
		limen.WithSchemaIDGenerator(uuidIDGenerator{}),
		limen.WithSchemaUser(limen.WithUserTableName("limen_users")),
		limen.WithSchemaSession(limen.WithSessionTableName("limen_sessions")),
		limen.WithSchemaAccount(limen.WithAccountTableName("limen_accounts")),
		limen.WithSchemaVerification(limen.WithVerificationTableName("limen_verifications")),
		limen.WithSchemaRateLimit(limen.WithRateLimitTableName("limen_rate_limits")),
	)

	httpCfg := limen.NewDefaultHTTPConfig(
		limen.WithHTTPBasePath("/auth"),
		limen.WithHTTPTrustedOrigins([]string{
			s.Config.Auth.FrontendBaseURL,
			s.Config.Auth.AppBaseURL,
		}),
		limen.WithHTTPCookieHTTPOnly(true),
		limen.WithHTTPCookieSameSite(http.SameSiteLaxMode),
		limen.WithHTTPCookieSecure(strings.HasPrefix(s.Config.Auth.AppBaseURL, "https://")),
	)

	engine, err := limen.New(&limen.Config{
		BaseURL:  s.Config.Auth.AppBaseURL,
		Secret:   secretBytes,
		Database: sqladapter.NewPostgreSQL(sqldb),
		Schema:   schemaCfg,
		HTTP:     httpCfg,
		Plugins: []limen.Plugin{
			oauth.New(
				oauth.WithProviders(
					oauthgoogle.New(
						oauthgoogle.WithClientID(s.Config.Auth.GoogleClientID),
						oauthgoogle.WithClientSecret(s.Config.Auth.GoogleClientSecret),
					),
					oauthgithub.New(
						oauthgithub.WithClientID(s.Config.Auth.GitHubClientID),
						oauthgithub.WithClientSecret(s.Config.Auth.GitHubClientSecret),
					),
				),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("auth: initialize limen: %w", err)
	}

	return &AuthService{
		server: s,
		limen:  engine,
	}, nil
}

// Handler returns Limen's http.Handler mounted at /auth/*.
func (a *AuthService) Handler() http.Handler {
	return a.limen.Handler()
}

// GetSession validates the incoming Limen session cookie.
func (a *AuthService) GetSession(r *http.Request) (*limen.ValidatedSession, error) {
	return a.limen.GetSession(r)
}

// RevokeSession revokes the active Limen session token.
func (a *AuthService) RevokeSession(ctx context.Context, token string) error {
	return a.limen.RevokeSession(ctx, token)
}

type uuidIDGenerator struct{}

func (uuidIDGenerator) Generate(_ context.Context) (any, error) {
	return uuid.NewString(), nil
}

func (uuidIDGenerator) GetColumnType() limen.ColumnType {
	return limen.ColumnTypeString
}

func parseLimenSecret(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) == 64 {
		decoded, err := hex.DecodeString(trimmed)
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	if len(trimmed) == 32 {
		return []byte(trimmed), nil
	}
	return nil, errors.New("auth: TUTORGRID_LIMEN_SECRET must be 32 raw bytes or a 64-character hex string")
}
