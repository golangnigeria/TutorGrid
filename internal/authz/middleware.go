package authz

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/tutorgrid/tutorgrid/internal/identity"
	"github.com/tutorgrid/tutorgrid/internal/platform/httputil"
	"github.com/tutorgrid/tutorgrid/internal/tenancy"
)

type userContextKey struct{}

// WithVerifiedUser attaches a verified TutorGrid user to ctx.
func WithVerifiedUser(ctx context.Context, u tenancy.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, u)
}

// VerifiedUserFromContext retrieves the verified TutorGrid user from ctx.
func VerifiedUserFromContext(ctx context.Context) (tenancy.User, bool) {
	u, ok := ctx.Value(userContextKey{}).(tenancy.User)
	return u, ok
}

// RequireAuth verifies the incoming Limen session and attaches the synchronized
// TutorGrid user to the request context.
func RequireAuth(authenticator identity.SessionAuthenticator, tenancySvc *tenancy.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			verifiedID, err := authenticator.AuthenticateRequest(c.Request())
			if err != nil {
				return httputil.WriteError(c, http.StatusUnauthorized, httputil.CodeUnauthorized, "authentication required")
			}

			user, err := tenancySvc.UpsertUserBySubject(
				c.Request().Context(),
				verifiedID.ExternalSubject,
				verifiedID.DisplayName,
				verifiedID.Email,
			)
			if err != nil {
				return httputil.WriteError(c, http.StatusInternalServerError, httputil.CodeInternalError, "failed to load user profile")
			}

			ctx := WithVerifiedUser(c.Request().Context(), user)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// RequireTenant derives tenancy.Context strictly from the verified user's
// server-bound active tenant and PostgreSQL membership check.
// It never reads or trusts any client-supplied tenant ID header.
func RequireTenant(tenancySvc *tenancy.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := VerifiedUserFromContext(c.Request().Context())
			if !ok {
				return httputil.WriteError(c, http.StatusUnauthorized, httputil.CodeUnauthorized, "authentication required")
			}

			tc, err := tenancySvc.ResolveVerifiedTenantContext(c.Request().Context(), user.ID)
			if err != nil {
				if errors.Is(err, tenancy.ErrNoActiveTenant) {
					return httputil.WriteError(c, http.StatusPreconditionRequired, httputil.CodeTenantRequired, "an active teaching space is required")
				}
				if errors.Is(err, tenancy.ErrNotMember) {
					return httputil.WriteError(c, http.StatusForbidden, httputil.CodeForbidden, "not a member of the active teaching space")
				}
				return httputil.WriteError(c, http.StatusInternalServerError, httputil.CodeInternalError, "failed to resolve tenant context")
			}

			ctx := tenancy.WithContext(c.Request().Context(), tc)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// RequirePermission checks RBAC authorization against the verified tenancy.Context.
func RequirePermission(perm Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tc, ok := tenancy.FromContext(c.Request().Context())
			if !ok {
				return httputil.WriteError(c, http.StatusForbidden, httputil.CodeForbidden, "tenant context required")
			}
			if !Can(tc.Role(), perm) {
				return httputil.WriteError(c, http.StatusForbidden, httputil.CodeForbidden, "insufficient permissions")
			}
			return next(c)
		}
	}
}
