package identity

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/thecodearcher/limen"
	sqladapter "github.com/thecodearcher/limen/adapters/sql"
	"github.com/thecodearcher/limen/plugins/oauth"
	oauthgithub "github.com/thecodearcher/limen/plugins/oauth-github"
	oauthgoogle "github.com/thecodearcher/limen/plugins/oauth-google"
)

// ErrUnauthenticated indicates the request does not carry a valid session.
var ErrUnauthenticated = errors.New("identity: unauthenticated")

// VerifiedIdentity represents a verified authenticated user returned by Limen.
type VerifiedIdentity struct {
	ExternalSubject string
	Email           string
	DisplayName     string
}

// SessionAuthenticator abstracts session verification so middleware and tests
// do not couple directly to Limen internals.
type SessionAuthenticator interface {
	AuthenticateRequest(r *http.Request) (VerifiedIdentity, error)
	RevokeRequestSession(r *http.Request) error
}

// LimenConfig holds the required configuration for embedding Limen.
type LimenConfig struct {
	AppBaseURL         string
	FrontendBaseURL    string
	Secret             string
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	CookieSecure       bool
}

// LimenAuth wraps an embedded *limen.Limen instance configured strictly for
// Google and GitHub OAuth (no email/password login per ADR 0001).
type LimenAuth struct {
	engine *limen.Limen
}

// NewLimenAuth initializes Limen against the provided PostgreSQL *sql.DB.
func NewLimenAuth(sqldb *sql.DB, cfg LimenConfig) (*LimenAuth, error) {
	if sqldb == nil {
		return nil, errors.New("identity: sql.DB is required")
	}
	secretBytes, err := parseSecretBytes(cfg.Secret)
	if err != nil {
		return nil, err
	}

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
		limen.WithHTTPTrustedOrigins([]string{cfg.FrontendBaseURL, cfg.AppBaseURL}),
		limen.WithHTTPCookieHTTPOnly(true),
		limen.WithHTTPCookieSameSite(http.SameSiteLaxMode),
		limen.WithHTTPCookieSecure(cfg.CookieSecure),
	)

	engine, err := limen.New(&limen.Config{
		BaseURL:  cfg.AppBaseURL,
		Secret:   secretBytes,
		Database: sqladapter.NewPostgreSQL(sqldb),
		Schema:   schemaCfg,
		HTTP:     httpCfg,
		Plugins: []limen.Plugin{
			oauth.New(
				oauth.WithProviders(
					oauthgoogle.New(
						oauthgoogle.WithClientID(cfg.GoogleClientID),
						oauthgoogle.WithClientSecret(cfg.GoogleClientSecret),
					),
					oauthgithub.New(
						oauthgithub.WithClientID(cfg.GitHubClientID),
						oauthgithub.WithClientSecret(cfg.GitHubClientSecret),
					),
				),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("identity: initialize limen: %w", err)
	}

	return &LimenAuth{engine: engine}, nil
}

// Handler returns Limen's http.Handler mounted at /auth/*.
func (l *LimenAuth) Handler() http.Handler {
	return l.engine.Handler()
}

// AuthenticateRequest validates the incoming Limen session cookie.
func (l *LimenAuth) AuthenticateRequest(r *http.Request) (VerifiedIdentity, error) {
	validated, err := l.engine.GetSession(r)
	if err != nil || validated == nil || validated.User == nil {
		return VerifiedIdentity{}, ErrUnauthenticated
	}

	subject := fmt.Sprintf("limen:%v", validated.User.ID)
	email := strings.TrimSpace(validated.User.Email)
	displayName := deriveDisplayName(validated.User)

	return VerifiedIdentity{
		ExternalSubject: subject,
		Email:           email,
		DisplayName:     displayName,
	}, nil
}

// RevokeRequestSession revokes the active Limen session from the request.
func (l *LimenAuth) RevokeRequestSession(r *http.Request) error {
	validated, err := l.engine.GetSession(r)
	if err != nil || validated == nil || validated.Session == nil {
		return ErrUnauthenticated
	}
	return l.engine.RevokeSession(r.Context(), validated.Session.Token)
}

type uuidIDGenerator struct{}

func (uuidIDGenerator) Generate(_ context.Context) (any, error) {
	return uuid.NewString(), nil
}

func (uuidIDGenerator) GetColumnType() limen.ColumnType {
	return limen.ColumnTypeString
}

func parseSecretBytes(raw string) ([]byte, error) {
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
	return nil, errors.New("identity: TUTORGRID_LIMEN_SECRET must be 32 raw bytes or a 64-character hex string")
}

func deriveDisplayName(u *limen.User) string {
	raw := u.Raw()
	first, _ := raw["first_name"].(string)
	last, _ := raw["last_name"].(string)
	full := strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
	if full != "" {
		return full
	}
	if at := strings.Index(u.Email, "@"); at > 0 {
		return u.Email[:at]
	}
	return "TutorGrid User"
}
