package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/thecodearcher/limen"

	"github.com/tutorgrid/tutorgrid/internal/database"
	"github.com/tutorgrid/tutorgrid/internal/handler"
	"github.com/tutorgrid/tutorgrid/internal/middleware"
	"github.com/tutorgrid/tutorgrid/internal/model"
	"github.com/tutorgrid/tutorgrid/internal/repository"
	"github.com/tutorgrid/tutorgrid/internal/server"
	"github.com/tutorgrid/tutorgrid/internal/service"
)

type stubSessionValidator struct {
	sessions map[string]*limen.ValidatedSession
}

func (s *stubSessionValidator) GetSession(r *http.Request) (*limen.ValidatedSession, error) {
	c, err := r.Cookie("test_session")
	if err != nil {
		return nil, errors.New("no session")
	}
	vs, ok := s.sessions[c.Value]
	if !ok {
		return nil, errors.New("invalid session")
	}
	return vs, nil
}

type failingEnqueuer struct {
	fail bool
	err  error
}

func (f *failingEnqueuer) EnqueueInvitationEmail(_ context.Context, _, _ uuid.UUID) error {
	if f.fail {
		return f.err
	}
	return nil
}

func TestTenancyDualRLSInvitationsOutboxAndRBAC(t *testing.T) {
	migrationDSN := os.Getenv("TUTORGRID_TEST_DATABASE_MIGRATION_URL")
	appDSN := os.Getenv("TUTORGRID_TEST_DATABASE_URL")
	if migrationDSN == "" || appDSN == "" {
		t.Skip("set TUTORGRID_TEST_DATABASE_MIGRATION_URL and TUTORGRID_TEST_DATABASE_URL to run tenancy integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	migratorConn, err := pgx.Connect(ctx, migrationDSN)
	if err != nil {
		t.Fatalf("connect migrator: %v", err)
	}
	defer migratorConn.Close(ctx)

	nopLog := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.ErrorLevel)
	if err := database.MigrateConn(ctx, &nopLog, migratorConn); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	if _, err := migratorConn.Exec(ctx, "TRUNCATE invitation_email_outbox, tenant_invitations, user_active_tenants, tenant_memberships, users, tenants CASCADE"); err != nil {
		t.Fatalf("truncate fixtures: %v", err)
	}

	appPool, err := pgxpool.New(ctx, appDSN)
	if err != nil {
		t.Fatalf("open app pool: %v", err)
	}
	defer appPool.Close()

	// 1. Verify tutorgrid_app is NOSUPERUSER, NOBYPASSRLS, and NOT table owner.
	t.Run("role privileges and non-ownership", func(t *testing.T) {
		var rolSuper, rolBypassRLS bool
		if err := appPool.QueryRow(ctx,
			"SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user",
		).Scan(&rolSuper, &rolBypassRLS); err != nil {
			t.Fatalf("query role: %v", err)
		}
		if rolSuper || rolBypassRLS {
			t.Fatalf("tutorgrid_app must be NOSUPERUSER NOBYPASSRLS, got super=%v bypassrls=%v", rolSuper, rolBypassRLS)
		}

		for _, tbl := range []string{"tenants", "tenant_memberships", "user_active_tenants", "tenant_invitations", "invitation_email_outbox"} {
			var owner string
			if err := appPool.QueryRow(ctx, "SELECT tableowner FROM pg_tables WHERE schemaname = 'public' AND tablename = $1", tbl).Scan(&owner); err != nil {
				t.Fatalf("query tableowner for %s: %v", tbl, err)
			}
			if owner == "tutorgrid_app" {
				t.Fatalf("tutorgrid_app must not own table %s", tbl)
			}
		}
	})

	repo := repository.NewTenancyRepository(appPool)
	enq := &failingEnqueuer{}
	tenancySvc := service.NewTenancyService(repo, nil, "http://localhost:5173", &nopLog)
	tenancySvc.SetTaskEnqueuer(enq)

	validator := &stubSessionValidator{
		sessions: map[string]*limen.ValidatedSession{
			"sess-a": {User: &limen.User{ID: "limen-a", Email: "teacher-a@example.com"}},
			"sess-b": {User: &limen.User{ID: "limen-b", Email: "learner-b@example.com"}},
			"sess-c": {User: &limen.User{ID: "limen-c", Email: "other-c@example.com"}},
		},
	}

	srv := &server.Server{Logger: &nopLog}
	authMW := middleware.NewAuthMiddleware(srv, validator, tenancySvc)
	tenancyHandler := handler.NewTenancyHandler(srv, tenancySvc, nil)

	e := echo.New()
	e.HTTPErrorHandler = middleware.NewGlobalMiddlewares(srv).GlobalErrorHandler
	v1 := e.Group("/api/v1")
	authed := v1.Group("", authMW.RequireAuth)
	authed.GET("/me", tenancyHandler.Me)
	authed.GET("/me/invitations", tenancyHandler.ListMyPendingInvitations)
	authed.POST("/me/invitations/:id/accept", tenancyHandler.AcceptMyInvitation)
	authed.POST("/tenants", tenancyHandler.CreateTenant)
	authed.POST("/session/active-tenant", tenancyHandler.SetActiveTenant)
	authed.POST("/invitations/accept", tenancyHandler.AcceptInvitation)

	tenantScoped := authed.Group("/tenants/current", authMW.RequireTenant)
	tenantScoped.GET("/members", tenancyHandler.ListCurrentMembers, authMW.RequirePermission(model.PermissionTenantReadMembers))
	tenantScoped.GET("/invitations", tenancyHandler.ListCurrentInvitations, authMW.RequirePermission(model.PermissionTenantInviteMember))
	tenantScoped.POST("/invitations", tenancyHandler.CreateInvitation, authMW.RequirePermission(model.PermissionTenantInviteMember))
	tenantScoped.DELETE("/invitations/:id", tenancyHandler.RevokeInvitation, authMW.RequirePermission(model.PermissionTenantRevokeInvite))

	doJSON := func(method, path, sess string, body any, headers map[string]string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		if sess != "" {
			req.AddCookie(&http.Cookie{Name: "test_session", Value: sess})
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	// 2. Route middleware audit: all protected routes reject unauthenticated requests with 401.
	t.Run("unauthenticated rejected on all protected routes", func(t *testing.T) {
		routes := [][2]string{
			{http.MethodGet, "/api/v1/me"},
			{http.MethodPost, "/api/v1/tenants"},
			{http.MethodPost, "/api/v1/session/active-tenant"},
			{http.MethodPost, "/api/v1/invitations/accept"},
			{http.MethodGet, "/api/v1/tenants/current/members"},
			{http.MethodPost, "/api/v1/tenants/current/invitations"},
			{http.MethodDelete, "/api/v1/tenants/current/invitations/" + uuid.NewString()},
		}
		for _, rt := range routes {
			rec := doJSON(rt[0], rt[1], "", nil, nil)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s: expected 401, got %d", rt[0], rt[1], rec.Code)
			}
		}
	})

	// 3. Authenticated User A with 0 memberships gets 428 TENANT_REQUIRED.
	if rec := doJSON(http.MethodGet, "/api/v1/tenants/current/members", "sess-a", nil, nil); rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("expected 428 before onboarding, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. Create Tenant A (User A) and Tenant B (User B).
	recA := doJSON(http.MethodPost, "/api/v1/tenants", "sess-a", map[string]string{"name": "Alpha Math", "slug": "alpha-math"}, nil)
	if recA.Code != http.StatusCreated {
		t.Fatalf("create tenant A: expected 201, got %d: %s", recA.Code, recA.Body.String())
	}
	var respA struct{ Membership model.Membership }
	_ = json.Unmarshal(recA.Body.Bytes(), &respA)

	recB := doJSON(http.MethodPost, "/api/v1/tenants", "sess-b", map[string]string{"name": "Beta Science", "slug": "beta-science"}, nil)
	if recB.Code != http.StatusCreated {
		t.Fatalf("create tenant B: expected 201, got %d: %s", recB.Code, recB.Body.String())
	}
	var respB struct{ Membership model.Membership }
	_ = json.Unmarshal(recB.Body.Bytes(), &respB)

	// 5. Dual RLS 3-state matrix (no context, user-only context, both contexts).
	t.Run("dual RLS 3-state matrix", func(t *testing.T) {
		// (a) No user & no tenant context -> 0 rows everywhere
		for _, tbl := range []string{"tenants", "tenant_memberships", "user_active_tenants", "tenant_invitations", "invitation_email_outbox"} {
			var cnt int
			if err := appPool.QueryRow(ctx, "SELECT COUNT(*) FROM "+tbl).Scan(&cnt); err != nil {
				t.Fatalf("count %s: %v", tbl, err)
			}
			if cnt != 0 {
				t.Fatalf("expected 0 rows in %s with unset context, got %d", tbl, cnt)
			}
		}

		// (b) User A context ONLY (no tenant context): sees own membership/tenant, cannot UPDATE/DELETE Tenant B
		var userA model.User
		_ = migratorConn.QueryRow(ctx, "SELECT id FROM users WHERE email = 'teacher-a@example.com'").Scan(&userA.ID)
		err := database.RunInUserTx(ctx, appPool, userA.ID, func(ctx context.Context, tx pgx.Tx) error {
			var cnt int
			if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM tenant_memberships").Scan(&cnt); err != nil {
				return err
			}
			if cnt != 1 {
				t.Fatalf("expected 1 own membership under user-only context, got %d", cnt)
			}
			tag, err := tx.Exec(ctx, "UPDATE tenants SET name = 'Hacked' WHERE id = $1", respB.Membership.TenantID)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 0 {
				t.Fatalf("expected 0 rows affected updating tenant B under user-only context, got %d", tag.RowsAffected())
			}
			return nil
		})
		if err != nil {
			t.Fatalf("user-only Dual RLS check failed: %v", err)
		}

		// (c) Tenant A context: cannot read or write Tenant B
		err = database.RunInTenantTx(ctx, appPool, respA.Membership.TenantID, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "INSERT INTO tenant_memberships (tenant_id, user_id, role) VALUES ($1, $2, 'learner')",
				respB.Membership.TenantID, userA.ID)
			return err
		})
		if err == nil {
			t.Fatal("expected cross-tenant INSERT into tenant B under tenant A context to fail RLS")
		}
	})

	// 6. Cross-tenant switch rejected (403) and spoofed X-Tenant-ID ignored.
	if rec := doJSON(http.MethodPost, "/api/v1/session/active-tenant", "sess-a",
		map[string]string{"tenant_id": respB.Membership.TenantID.String()}, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 switching to unauthorized tenant, got %d", rec.Code)
	}
	recMem := doJSON(http.MethodGet, "/api/v1/tenants/current/members", "sess-a", nil,
		map[string]string{"X-Tenant-ID": respB.Membership.TenantID.String()})
	if recMem.Code != http.StatusOK {
		t.Fatalf("expected 200 listing own members, got %d", recMem.Code)
	}
	var memResp struct{ Members []model.Member }
	_ = json.Unmarshal(recMem.Body.Bytes(), &memResp)
	if len(memResp.Members) != 1 || memResp.Members[0].Email != "teacher-a@example.com" {
		t.Fatalf("spoofed header leaked wrong members: %+v", memResp.Members)
	}

	// 7. Outbox durability when Asynq enqueue fails.
	t.Run("invitation outbox survives enqueue failure", func(t *testing.T) {
		enq.fail = true
		enq.err = errors.New("redis connection refused")
		defer func() { enq.fail = false; enq.err = nil }()

		rec := doJSON(http.MethodPost, "/api/v1/tenants/current/invitations", "sess-a",
			map[string]string{"email": "queued-check@example.com", "role": "teacher"}, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 even when enqueue fails, got %d: %s", rec.Code, rec.Body.String())
		}
		var created struct{ Invitation model.CreatedInvitation }
		_ = json.Unmarshal(rec.Body.Bytes(), &created)
		if created.Invitation.EmailStatus != "pending" {
			t.Fatalf("expected email_status=pending on enqueue failure, got %q", created.Invitation.EmailStatus)
		}

		var status, lastErr string
		if err := migratorConn.QueryRow(ctx,
			"SELECT status, COALESCE(last_error, '') FROM invitation_email_outbox WHERE invitation_id = $1",
			created.Invitation.ID,
		).Scan(&status, &lastErr); err != nil {
			t.Fatalf("verify outbox row: %v", err)
		}
		if status != "pending" || lastErr == "" {
			t.Fatalf("expected durable pending outbox with last_error, got status=%q lastErr=%q", status, lastErr)
		}
	})

	// 8. Invitation edge cases: no raw token in HTTP response, list pending invitations, wrong user (403), expired (410), revoked (410), concurrent race, replay (409).
	t.Run("invitation security edge cases and concurrent acceptance", func(t *testing.T) {
		extractEmailToken := func(invitationID uuid.UUID) string {
			t.Helper()
			var inviteURL string
			if err := migratorConn.QueryRow(ctx,
				"SELECT invite_url FROM invitation_email_outbox WHERE invitation_id = $1",
				invitationID,
			).Scan(&inviteURL); err != nil {
				t.Fatalf("read outbox invite_url: %v", err)
			}
			parsed, err := url.Parse(inviteURL)
			if err != nil {
				t.Fatalf("parse invite_url: %v", err)
			}
			tok := parsed.Query().Get("token")
			if tok == "" {
				t.Fatalf("expected token query param in invite_url %q", inviteURL)
			}
			return tok
		}

		// Create invite for learner-b@example.com
		recInv := doJSON(http.MethodPost, "/api/v1/tenants/current/invitations", "sess-a",
			map[string]string{"email": "learner-b@example.com", "role": "learner"}, nil)
		if recInv.Code != http.StatusCreated {
			t.Fatalf("create invite: %d", recInv.Code)
		}
		if strings.Contains(recInv.Body.String(), `"token"`) {
			t.Fatalf("raw invitation token must never be exposed in HTTP response: %s", recInv.Body.String())
		}
		var invResp struct{ Invitation model.CreatedInvitation }
		_ = json.Unmarshal(recInv.Body.Bytes(), &invResp)
		invToken := extractEmailToken(invResp.Invitation.ID)

		// Admin can list pending invitations
		recList := doJSON(http.MethodGet, "/api/v1/tenants/current/invitations", "sess-a", nil, nil)
		if recList.Code != http.StatusOK {
			t.Fatalf("expected 200 listing invitations for org_admin, got %d", recList.Code)
		}

		// Wrong user (other-c@example.com) must be rejected with 403.
		if rec := doJSON(http.MethodPost, "/api/v1/invitations/accept", "sess-c",
			map[string]string{"token": invToken}, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for wrong-user invite acceptance, got %d: %s", rec.Code, rec.Body.String())
		}

		// Concurrent acceptance race by 5 goroutines for learner-b@example.com -> exactly 1 succeeds (200), 4 fail (409).
		var okCount, conflictCount atomic.Int32
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rec := doJSON(http.MethodPost, "/api/v1/invitations/accept", "sess-b",
					map[string]string{"token": invToken}, nil)
				switch rec.Code {
				case http.StatusOK:
					okCount.Add(1)
				case http.StatusConflict:
					conflictCount.Add(1)
				}
			}()
		}
		wg.Wait()
		if okCount.Load() != 1 || conflictCount.Load() != 4 {
			t.Fatalf("concurrent invite race: expected 1 OK and 4 Conflict, got ok=%d conflict=%d", okCount.Load(), conflictCount.Load())
		}

		// Revoked invitation -> 410 Gone
		recInv2 := doJSON(http.MethodPost, "/api/v1/tenants/current/invitations", "sess-a",
			map[string]string{"email": "other-c@example.com", "role": "teacher"}, nil)
		var inv2 struct{ Invitation model.CreatedInvitation }
		_ = json.Unmarshal(recInv2.Body.Bytes(), &inv2)
		inv2Token := extractEmailToken(inv2.Invitation.ID)

		// Cross-tenant revoke by User B (currently learner in Tenant A) -> 403 RBAC!
		if rec := doJSON(http.MethodDelete, "/api/v1/tenants/current/invitations/"+inv2.Invitation.ID.String(), "sess-b", nil, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for learner revoking invite, got %d", rec.Code)
		}
		// Valid revoke by org_admin User A -> 200
		if rec := doJSON(http.MethodDelete, "/api/v1/tenants/current/invitations/"+inv2.Invitation.ID.String(), "sess-a", nil, nil); rec.Code != http.StatusOK {
			t.Fatalf("expected 200 revoking invite, got %d", rec.Code)
		}
		if rec := doJSON(http.MethodPost, "/api/v1/invitations/accept", "sess-c",
			map[string]string{"token": inv2Token}, nil); rec.Code != http.StatusGone {
			t.Fatalf("expected 410 for revoked invite, got %d: %s", rec.Code, rec.Body.String())
		}

		// Expired invitation -> 410 Gone
		recInv3 := doJSON(http.MethodPost, "/api/v1/tenants/current/invitations", "sess-a",
			map[string]string{"email": "other-c@example.com", "role": "teacher"}, nil)
		var inv3 struct{ Invitation model.CreatedInvitation }
		_ = json.Unmarshal(recInv3.Body.Bytes(), &inv3)
		inv3Token := extractEmailToken(inv3.Invitation.ID)
		_, _ = migratorConn.Exec(ctx, "UPDATE tenant_invitations SET expires_at = NOW() - INTERVAL '1 hour' WHERE id = $1", inv3.Invitation.ID)
		if rec := doJSON(http.MethodPost, "/api/v1/invitations/accept", "sess-c",
			map[string]string{"token": inv3Token}, nil); rec.Code != http.StatusGone {
			t.Fatalf("expected 410 for expired invite, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verified-email in-app discovery & 1-click accept by ID
		recInv4 := doJSON(http.MethodPost, "/api/v1/tenants/current/invitations", "sess-a",
			map[string]string{"email": "other-c@example.com", "role": "teacher"}, nil)
		var inv4 struct{ Invitation model.CreatedInvitation }
		_ = json.Unmarshal(recInv4.Body.Bytes(), &inv4)

		// User B (learner-b@example.com) must NOT see or accept User C's pending invitation.
		recListB := doJSON(http.MethodGet, "/api/v1/me/invitations", "sess-b", nil, nil)
		if recListB.Code != http.StatusOK || strings.Contains(recListB.Body.String(), inv4.Invitation.ID.String()) {
			t.Fatalf("user B must not see user C's pending invitation: code=%d body=%s", recListB.Code, recListB.Body.String())
		}
		if rec := doJSON(http.MethodPost, "/api/v1/me/invitations/"+inv4.Invitation.ID.String()+"/accept", "sess-b", nil, nil); rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 when wrong user tries to accept by ID under RLS, got %d", rec.Code)
		}

		// User C (other-c@example.com) sees it in /me/invitations and accepts in 1 click.
		recListC := doJSON(http.MethodGet, "/api/v1/me/invitations", "sess-c", nil, nil)
		if recListC.Code != http.StatusOK || !strings.Contains(recListC.Body.String(), inv4.Invitation.ID.String()) {
			t.Fatalf("expected user C to see pending invitation in /me/invitations: code=%d body=%s", recListC.Code, recListC.Body.String())
		}
		if rec := doJSON(http.MethodPost, "/api/v1/me/invitations/"+inv4.Invitation.ID.String()+"/accept", "sess-c", nil, nil); rec.Code != http.StatusOK {
			t.Fatalf("expected 200 accepting pending invitation by ID for user C, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 9. Learner RBAC denial + immediate membership revocation invalidates active workspace.
	t.Run("learner RBAC denial and immediate membership revocation", func(t *testing.T) {
		// User B is currently active in Tenant A as learner -> denied reading members & creating invites (403)
		if rec := doJSON(http.MethodGet, "/api/v1/tenants/current/members", "sess-b", nil, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for learner reading members, got %d", rec.Code)
		}
		if rec := doJSON(http.MethodPost, "/api/v1/tenants/current/invitations", "sess-b",
			map[string]string{"email": "x@example.com", "role": "learner"}, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for learner creating invitation, got %d", rec.Code)
		}

		// Delete User B's membership in Tenant A -> active workspace must be unusable immediately!
		_, _ = migratorConn.Exec(ctx, "DELETE FROM tenant_memberships WHERE tenant_id = $1 AND user_id = (SELECT id FROM users WHERE email = 'learner-b@example.com')", respA.Membership.TenantID)
		if rec := doJSON(http.MethodGet, "/api/v1/tenants/current/members", "sess-b", nil, nil); rec.Code != http.StatusPreconditionRequired {
			t.Fatalf("expected 428 immediately after membership deletion, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
