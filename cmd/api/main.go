package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/uptrace/bun"

	"github.com/tutorgrid/tutorgrid/internal/authz"
	"github.com/tutorgrid/tutorgrid/internal/identity"
	"github.com/tutorgrid/tutorgrid/internal/platform/config"
	"github.com/tutorgrid/tutorgrid/internal/platform/database"
	"github.com/tutorgrid/tutorgrid/internal/platform/httputil"
	"github.com/tutorgrid/tutorgrid/internal/tenancy"
)

type bunPinger struct {
	db *bun.DB
}

func (p bunPinger) Ping(ctx context.Context) error {
	return database.Ping(ctx, p.db)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	db, err := database.Open(database.PoolConfig{
		DSN:             cfg.DatabaseURL,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
	})
	if err != nil {
		logger.Error("database open error", "error", err)
		os.Exit(1)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Error("database close error", "error", cerr)
		}
	}()

	limenAuth, err := identity.NewLimenAuth(db.DB, identity.LimenConfig{
		AppBaseURL:         cfg.AppBaseURL,
		FrontendBaseURL:    cfg.FrontendBaseURL,
		Secret:             cfg.LimenSecret,
		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
		GitHubClientID:     cfg.GitHubClientID,
		GitHubClientSecret: cfg.GitHubClientSecret,
		CookieSecure:       strings.HasPrefix(cfg.AppBaseURL, "https://"),
	})
	if err != nil {
		logger.Error("limen initialization error", "error", err)
		os.Exit(1)
	}

	tenancySvc := tenancy.NewService(db)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{cfg.FrontendBaseURL},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}))

	httputil.RegisterHealthRoutes(e, bunPinger{db: db})

	// Mount Limen's embedded OAuth + session routes at /auth/*.
	e.Any("/auth/*", echo.WrapHandler(limenAuth.Handler()))

	// Mount authenticated TutorGrid v1 API routes.
	v1 := e.Group("/api/v1")
	tenancy.RegisterRoutes(v1, tenancySvc, tenancy.RouteMiddlewares{
		RequireAuth:         authz.RequireAuth(limenAuth, tenancySvc),
		RequireTenant:       authz.RequireTenant(tenancySvc),
		RequireReadMembers:  authz.RequirePermission(authz.PermissionTenantReadMembers),
		RequireInviteMember: authz.RequirePermission(authz.PermissionTenantInviteMember),
		ExtractUser:         authz.VerifiedUserFromContext,
		SessionRevoker:      limenAuth,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting TutorGrid API server", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
		if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutting down TutorGrid API server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := e.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}
	}
}
