package httputil

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// DBPinger abstracts a database readiness check.
type DBPinger interface {
	Ping(ctx context.Context) error
}

// RegisterHealthRoutes mounts unauthenticated operational probes on Echo.
func RegisterHealthRoutes(e *echo.Echo, pinger DBPinger) {
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	e.GET("/readyz", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()

		if err := pinger.Ping(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status":   "not_ready",
				"database": "unreachable",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status":   "ready",
			"database": "ok",
		})
	})
}
