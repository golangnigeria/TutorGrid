package httputil_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/tutorgrid/tutorgrid/internal/platform/httputil"
)

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(_ context.Context) error { return s.err }

func TestHealthAndReadinessEndpoints(t *testing.T) {
	t.Run("healthz returns 200", func(t *testing.T) {
		e := echo.New()
		httputil.RegisterHealthRoutes(e, stubPinger{})

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("readyz returns 200 when db is reachable", func(t *testing.T) {
		e := echo.New()
		httputil.RegisterHealthRoutes(e, stubPinger{})

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("readyz returns 503 when db ping fails", func(t *testing.T) {
		e := echo.New()
		httputil.RegisterHealthRoutes(e, stubPinger{err: errors.New("connection refused")})

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
