package tenancy

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tutorgrid/tutorgrid/internal/platform/httputil"
)

// UserExtractor retrieves the authenticated User from a request context.
type UserExtractor func(ctx context.Context) (User, bool)

// SessionRevoker revokes the active Limen session on sign-out.
type SessionRevoker interface {
	RevokeRequestSession(r *http.Request) error
}

// RouteMiddlewares groups the auth, tenant, and permission middlewares needed by tenancy routes.
type RouteMiddlewares struct {
	RequireAuth         echo.MiddlewareFunc
	RequireTenant       echo.MiddlewareFunc
	RequireReadMembers  echo.MiddlewareFunc
	RequireInviteMember echo.MiddlewareFunc
	ExtractUser         UserExtractor
	SessionRevoker      SessionRevoker
}

// RegisterRoutes mounts the authenticated identity/tenancy v1 API routes.
func RegisterRoutes(v1 *echo.Group, svc *Service, mw RouteMiddlewares) {
	authed := v1.Group("", mw.RequireAuth)

	authed.GET("/me", func(c echo.Context) error {
		user, ok := mw.ExtractUser(c.Request().Context())
		if !ok {
			return httputil.WriteError(c, http.StatusUnauthorized, httputil.CodeUnauthorized, "authentication required")
		}
		memberships, err := svc.ListUserMemberships(c.Request().Context(), user.ID)
		if err != nil {
			return httputil.WriteError(c, http.StatusInternalServerError, httputil.CodeInternalError, "failed to load memberships")
		}
		activeTenantID, err := svc.GetActiveTenantID(c.Request().Context(), user.ID)
		if err != nil {
			return httputil.WriteError(c, http.StatusInternalServerError, httputil.CodeInternalError, "failed to load active tenant")
		}
		return c.JSON(http.StatusOK, map[string]any{
			"user":             user,
			"memberships":      memberships,
			"active_tenant_id": activeTenantID,
		})
	})

	authed.POST("/auth/logout", func(c echo.Context) error {
		if mw.SessionRevoker != nil {
			_ = mw.SessionRevoker.RevokeRequestSession(c.Request())
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "signed_out"})
	})

	authed.POST("/tenants", func(c echo.Context) error {
		user, ok := mw.ExtractUser(c.Request().Context())
		if !ok {
			return httputil.WriteError(c, http.StatusUnauthorized, httputil.CodeUnauthorized, "authentication required")
		}
		var req struct {
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		if err := c.Bind(&req); err != nil {
			return httputil.WriteBadRequest(c, "invalid request body")
		}
		membership, err := svc.CreateTenantWithAdmin(c.Request().Context(), user.ID, req.Name, req.Slug)
		if err != nil {
			switch {
			case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidSlug):
				return httputil.WriteBadRequest(c, err.Error())
			case errors.Is(err, ErrSlugConflict):
				return httputil.WriteError(c, http.StatusConflict, httputil.CodeConflict, "that workspace slug is already taken")
			default:
				return httputil.WriteError(c, http.StatusInternalServerError, httputil.CodeInternalError, "failed to create teaching space")
			}
		}
		return c.JSON(http.StatusCreated, map[string]any{"membership": membership})
	})

	authed.POST("/session/active-tenant", func(c echo.Context) error {
		user, ok := mw.ExtractUser(c.Request().Context())
		if !ok {
			return httputil.WriteError(c, http.StatusUnauthorized, httputil.CodeUnauthorized, "authentication required")
		}
		var req struct {
			TenantID string `json:"tenant_id"`
		}
		if err := c.Bind(&req); err != nil {
			return httputil.WriteBadRequest(c, "invalid request body")
		}
		tenantID, err := uuid.Parse(req.TenantID)
		if err != nil {
			return httputil.WriteBadRequest(c, "invalid tenant_id")
		}
		role, err := svc.SetActiveTenant(c.Request().Context(), user.ID, tenantID)
		if err != nil {
			if errors.Is(err, ErrNotMember) {
				return httputil.WriteError(c, http.StatusForbidden, httputil.CodeForbidden, "not a member of the requested teaching space")
			}
			return httputil.WriteError(c, http.StatusInternalServerError, httputil.CodeInternalError, "failed to switch teaching space")
		}
		return c.JSON(http.StatusOK, map[string]any{
			"active_tenant_id": tenantID,
			"role":             role,
		})
	})

	authed.POST("/invitations/accept", func(c echo.Context) error {
		user, ok := mw.ExtractUser(c.Request().Context())
		if !ok {
			return httputil.WriteError(c, http.StatusUnauthorized, httputil.CodeUnauthorized, "authentication required")
		}
		var req struct {
			Token string `json:"token"`
		}
		if err := c.Bind(&req); err != nil {
			return httputil.WriteBadRequest(c, "invalid request body")
		}
		membership, err := svc.AcceptInvitation(c.Request().Context(), user.ID, req.Token)
		if err != nil {
			if errors.Is(err, ErrInvalidInvitation) {
				return httputil.WriteError(c, http.StatusNotFound, httputil.CodeNotFound, "invitation is invalid, expired, or already accepted")
			}
			return httputil.WriteError(c, http.StatusInternalServerError, httputil.CodeInternalError, "failed to accept invitation")
		}
		return c.JSON(http.StatusOK, map[string]any{"membership": membership})
	})

	// Tenant-scoped RLS-protected routes
	tenantScoped := authed.Group("/tenants/current", mw.RequireTenant)

	tenantScoped.GET("/members", func(c echo.Context) error {
		tc, ok := FromContext(c.Request().Context())
		if !ok {
			return httputil.WriteError(c, http.StatusForbidden, httputil.CodeForbidden, "tenant context required")
		}
		members, err := svc.ListCurrentTenantMembers(c.Request().Context(), tc)
		if err != nil {
			return httputil.WriteError(c, http.StatusInternalServerError, httputil.CodeInternalError, "failed to list members")
		}
		return c.JSON(http.StatusOK, map[string]any{"members": members})
	}, mw.RequireReadMembers)

	tenantScoped.POST("/invitations", func(c echo.Context) error {
		tc, ok := FromContext(c.Request().Context())
		if !ok {
			return httputil.WriteError(c, http.StatusForbidden, httputil.CodeForbidden, "tenant context required")
		}
		var req struct {
			Email string `json:"email"`
			Role  Role   `json:"role"`
		}
		if err := c.Bind(&req); err != nil {
			return httputil.WriteBadRequest(c, "invalid request body")
		}
		inv, err := svc.CreateInvitation(c.Request().Context(), tc, req.Email, req.Role)
		if err != nil {
			if errors.Is(err, ErrInvalidEmail) || errors.Is(err, ErrInvalidRole) {
				return httputil.WriteBadRequest(c, err.Error())
			}
			return httputil.WriteError(c, http.StatusInternalServerError, httputil.CodeInternalError, "failed to create invitation")
		}
		return c.JSON(http.StatusCreated, map[string]any{"invitation": inv})
	}, mw.RequireInviteMember)
}
