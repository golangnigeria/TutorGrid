package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/thecodearcher/limen"

	"github.com/tutorgrid/tutorgrid/internal/errs"
	"github.com/tutorgrid/tutorgrid/internal/model"
	"github.com/tutorgrid/tutorgrid/internal/repository"
	"github.com/tutorgrid/tutorgrid/internal/server"
)

// SessionValidator abstracts Limen session verification for middleware and tests.
type SessionValidator interface {
	GetSession(r *http.Request) (*limen.ValidatedSession, error)
}

// TenancyResolver abstracts verified user sync and server-bound tenant resolution.
type TenancyResolver interface {
	UpsertUserBySubject(ctx context.Context, externalSubject, displayName, email string) (model.User, error)
	ResolveVerifiedTenantContext(ctx context.Context, userID uuid.UUID) (model.TenantContext, error)
}

type AuthMiddleware struct {
	server    *server.Server
	validator SessionValidator
	tenancy   TenancyResolver
}

func NewAuthMiddleware(s *server.Server, validator SessionValidator, tenancy TenancyResolver) *AuthMiddleware {
	return &AuthMiddleware{
		server:    s,
		validator: validator,
		tenancy:   tenancy,
	}
}

// RequireAuth validates the incoming Limen session cookie and synchronizes the verified TutorGrid user.
func (auth *AuthMiddleware) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		if auth.validator == nil || auth.tenancy == nil {
			return errs.NewUnauthorizedError("Unauthorized", false)
		}

		validated, err := auth.validator.GetSession(c.Request())
		if err != nil || validated == nil || validated.User == nil {
			auth.server.Logger.Warn().
				Str("function", "RequireAuth").
				Str("request_id", GetRequestID(c)).
				Dur("duration", time.Since(start)).
				Msg("unauthenticated request rejected")
			return errs.NewUnauthorizedError("Unauthorized", false)
		}

		userSubject := fmt.Sprintf("limen:%v", validated.User.ID)
		displayName := deriveDisplayName(validated.User)

		user, err := auth.tenancy.UpsertUserBySubject(c.Request().Context(), userSubject, displayName, validated.User.Email)
		if err != nil {
			auth.server.Logger.Error().Err(err).Str("request_id", GetRequestID(c)).Msg("failed to sync verified user")
			return errs.NewInternalServerError()
		}

		c.Set("user_id", user.ID.String())
		c.Set("verified_user", user)
		c.Set("user_email", user.Email)
		c.Set("user_display_name", user.DisplayName)
		if validated.Session != nil {
			c.Set("session_token", validated.Session.Token)
		}

		return next(c)
	}
}

// RequireTenant atomically resolves and re-verifies the user's server-bound active workspace
// inside PostgreSQL. Never reads or trusts any client-supplied tenant ID header.
func (auth *AuthMiddleware) RequireTenant(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := GetVerifiedUser(c)
		if !ok {
			return errs.NewUnauthorizedError("Unauthorized", false)
		}

		tc, err := auth.tenancy.ResolveVerifiedTenantContext(c.Request().Context(), user.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNoActiveTenant) {
				return c.JSON(http.StatusPreconditionRequired, map[string]any{
					"code":    "TENANT_REQUIRED",
					"message": "An active teaching space is required",
				})
			}
			if errors.Is(err, repository.ErrNotMember) {
				return errs.NewForbiddenError("Not a member of the active teaching space", false)
			}
			return errs.NewInternalServerError()
		}

		c.Set("tenant_context", tc)
		c.Set("user_role", string(tc.Role))
		return next(c)
	}
}

// RequirePermission enforces backend RBAC permissions against the verified TenantContext.
func (auth *AuthMiddleware) RequirePermission(perm model.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tc, ok := GetTenantContext(c)
			if !ok || !tc.Valid() {
				return errs.NewForbiddenError("Tenant context required", false)
			}
			if !model.Can(tc.Role, perm) {
				return errs.NewForbiddenError("Insufficient permissions", false)
			}
			return next(c)
		}
	}
}

func GetVerifiedUser(c echo.Context) (model.User, bool) {
	u, ok := c.Get("verified_user").(model.User)
	return u, ok && u.ID != uuid.Nil
}

func GetTenantContext(c echo.Context) (model.TenantContext, bool) {
	tc, ok := c.Get("tenant_context").(model.TenantContext)
	return tc, ok && tc.Valid()
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
