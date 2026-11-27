package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/tutorgrid/tutorgrid/internal/handler"
	"github.com/tutorgrid/tutorgrid/internal/middleware"
	"github.com/tutorgrid/tutorgrid/internal/model"
	"github.com/tutorgrid/tutorgrid/internal/server"
	"github.com/tutorgrid/tutorgrid/internal/service"
	"golang.org/x/time/rate"
)

func NewRouter(s *server.Server, h *handler.Handlers, services *service.Services) *echo.Echo {
	middlewares := middleware.NewMiddlewares(s, services.Auth, services.Tenancy)

	router := echo.New()

	router.HTTPErrorHandler = middlewares.Global.GlobalErrorHandler

	// global middlewares
	router.Use(
		echoMiddleware.RateLimiterWithConfig(echoMiddleware.RateLimiterConfig{
			Store: echoMiddleware.NewRateLimiterMemoryStore(rate.Limit(20)),
			DenyHandler: func(c echo.Context, identifier string, err error) error {
				if rateLimitMiddleware := middlewares.RateLimit; rateLimitMiddleware != nil {
					rateLimitMiddleware.RecordRateLimitHit(c.Path())
				}

				s.Logger.Warn().
					Str("request_id", middleware.GetRequestID(c)).
					Str("identifier", identifier).
					Str("path", c.Path()).
					Str("method", c.Request().Method).
					Str("ip", c.RealIP()).
					Msg("rate limit exceeded")

				return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
			},
		}),
		middlewares.Global.CORS(),
		middlewares.Global.Secure(),
		middleware.RequestID(),
		middlewares.Tracing.NewRelicMiddleware(),
		middlewares.Tracing.EnhanceTracing(),
		middlewares.ContextEnhancer.EnhanceContext(),
		middlewares.Global.RequestLogger(),
		middlewares.Global.Recover(),
	)

	// register system routes
	registerSystemRoutes(router, h)

	// mount embedded Limen OAuth (Google & GitHub) and session routes
	if services != nil && services.Auth != nil {
		router.Any("/auth/*", echo.WrapHandler(services.Auth.Handler()))
	}

	// register versioned authenticated routes
	v1 := router.Group("/api/v1")
	authed := v1.Group("", middlewares.Auth.RequireAuth)

	authed.GET("/me", h.Tenancy.Me)
	authed.GET("/me/invitations", h.Tenancy.ListMyPendingInvitations)
	authed.POST("/me/invitations/:id/accept", h.Tenancy.AcceptMyInvitation)
	authed.POST("/auth/logout", h.Tenancy.Logout)
	authed.POST("/tenants", h.Tenancy.CreateTenant)
	authed.POST("/session/active-tenant", h.Tenancy.SetActiveTenant)
	authed.POST("/invitations/accept", h.Tenancy.AcceptInvitation)

	// tenant-scoped RLS + RBAC routes
	tenantScoped := authed.Group("/tenants/current", middlewares.Auth.RequireTenant)
	tenantScoped.GET("/members", h.Tenancy.ListCurrentMembers,
		middlewares.Auth.RequirePermission(model.PermissionTenantReadMembers))
	tenantScoped.GET("/invitations", h.Tenancy.ListCurrentInvitations,
		middlewares.Auth.RequirePermission(model.PermissionTenantInviteMember))
	tenantScoped.POST("/invitations", h.Tenancy.CreateInvitation,
		middlewares.Auth.RequirePermission(model.PermissionTenantInviteMember))
	tenantScoped.DELETE("/invitations/:id", h.Tenancy.RevokeInvitation,
		middlewares.Auth.RequirePermission(model.PermissionTenantRevokeInvite))

	return router
}
