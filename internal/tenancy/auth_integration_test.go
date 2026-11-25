package tenancy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tutorgrid/tutorgrid/internal/authz"
	"github.com/tutorgrid/tutorgrid/internal/identity"
	"github.com/tutorgrid/tutorgrid/internal/platform/database"
	"github.com/tutorgrid/tutorgrid/internal/tenancy"
)

// testSessionAuth is strictly test-only (never compiled into production binaries).
type testSessionAuth struct {
	sessions map[string]identity.VerifiedIdentity
}

func (a *testSessionAuth) AuthenticateRequest(r *http.Request) (identity.VerifiedIdentity, error) {
	cookie, err := r.Cookie("test_limen_session")
	if err != nil {
		return identity.VerifiedIdentity{}, identity.ErrUnauthenticated
	}
	id, ok := a.sessions[cookie.Value]
	if !ok {
		return identity.VerifiedIdentity{}, identity.ErrUnauthenticated
	}
	return id, nil
}

func (a *testSessionAuth) RevokeRequestSession(r *http.Request) error {
	cookie, err := r.Cookie("test_limen_session")
	if err != nil {
		return identity.ErrUnauthenticated
	}
	delete(a.sessions, cookie.Value)
	return nil
}

func TestAuthenticatedTenancyRBACAndRLSFlow(t *testing.T) {
	migrationDSN := os.Getenv("TUTORGRID_TEST_DATABASE_MIGRATION_URL")
	appDSN := os.Getenv("TUTORGRID_TEST_DATABASE_URL")
	if migrationDSN == "" || appDSN == "" {
		t.Skip("set TUTORGRID_TEST_DATABASE_MIGRATION_URL and TUTORGRID_TEST_DATABASE_URL to run PostgreSQL auth/RLS integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	migratorDB, err := database.Open(database.PoolConfig{DSN: migrationDSN, MaxOpenConns: 2, MaxIdleConns: 1})
	if err != nil {
		t.Fatalf("open migrator db: %v", err)
	}
	defer migratorDB.Close()

	if err := database.MigrateUp(ctx, migratorDB.DB); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	if _, err := migratorDB.ExecContext(ctx, "TRUNCATE tenant_invitations, user_active_tenants, tenant_memberships, users, tenants CASCADE"); err != nil {
		t.Fatalf("truncate fixtures: %v", err)
	}

	appDB, err := database.Open(database.PoolConfig{DSN: appDSN, MaxOpenConns: 4, MaxIdleConns: 2})
	if err != nil {
		t.Fatalf("open app db: %v", err)
	}
	defer appDB.Close()

	svc := tenancy.NewService(appDB)
	stubAuth := &testSessionAuth{
		sessions: map[string]identity.VerifiedIdentity{
			"sess-a": {ExternalSubject: "limen:user-a", DisplayName: "Teacher A", Email: "teacher-a@example.com"},
			"sess-b": {ExternalSubject: "limen:user-b", DisplayName: "Learner B", Email: "learner-b@example.com"},
		},
	}

	e := echo.New()
	v1 := e.Group("/api/v1")
	tenancy.RegisterRoutes(v1, svc, tenancy.RouteMiddlewares{
		RequireAuth:         authz.RequireAuth(stubAuth, svc),
		RequireTenant:       authz.RequireTenant(svc),
		RequireReadMembers:  authz.RequirePermission(authz.PermissionTenantReadMembers),
		RequireInviteMember: authz.RequirePermission(authz.PermissionTenantInviteMember),
		ExtractUser:         authz.VerifiedUserFromContext,
		SessionRevoker:      stubAuth,
	})

	doJSON := func(method, path, sessionCookie string, body any, extraHeaders map[string]string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&buf).Encode(body); err != nil {
				t.Fatalf("encode body: %v", err)
			}
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		if sessionCookie != "" {
			req.AddCookie(&http.Cookie{Name: "test_limen_session", Value: sessionCookie})
		}
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	// 1. Unauthenticated /api/v1/me is rejected with 401.
	if rec := doJSON(http.MethodGet, "/api/v1/me", "", nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated /me, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Authenticated User A with 0 memberships gets 428 on tenant-scoped route.
	if rec := doJSON(http.MethodGet, "/api/v1/tenants/current/members", "sess-a", nil, nil); rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("expected 428 when no active tenant, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. User A creates Tenant A; User B creates Tenant B.
	recA := doJSON(http.MethodPost, "/api/v1/tenants", "sess-a", map[string]string{"name": "Alpha Academy", "slug": "alpha-academy"}, nil)
	if recA.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating Tenant A, got %d: %s", recA.Code, recA.Body.String())
	}
	var createdA struct {
		Membership tenancy.Membership `json:"membership"`
	}
	_ = json.Unmarshal(recA.Body.Bytes(), &createdA)
	if createdA.Membership.TenantID == uuid.Nil || createdA.Membership.Role != tenancy.RoleOrgAdmin {
		t.Fatalf("unexpected Tenant A membership: %+v", createdA.Membership)
	}

	recB := doJSON(http.MethodPost, "/api/v1/tenants", "sess-b", map[string]string{"name": "Beta Academy", "slug": "beta-academy"}, nil)
	if recB.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating Tenant B, got %d: %s", recB.Code, recB.Body.String())
	}
	var createdB struct {
		Membership tenancy.Membership `json:"membership"`
	}
	_ = json.Unmarshal(recB.Body.Bytes(), &createdB)

	// 4. User A cannot switch active tenant to Tenant B (403 FORBIDDEN).
	recSwitch := doJSON(http.MethodPost, "/api/v1/session/active-tenant", "sess-a", map[string]string{
		"tenant_id": createdB.Membership.TenantID.String(),
	}, nil)
	if recSwitch.Code != http.StatusForbidden {
		t.Fatalf("expected 403 switching to unauthorized tenant, got %d: %s", recSwitch.Code, recSwitch.Body.String())
	}

	// 5. Spoofed X-Tenant-ID header for Tenant B is completely ignored; RLS returns only Tenant A members.
	recMembers := doJSON(http.MethodGet, "/api/v1/tenants/current/members", "sess-a", nil, map[string]string{
		"X-Tenant-ID": createdB.Membership.TenantID.String(),
	})
	if recMembers.Code != http.StatusOK {
		t.Fatalf("expected 200 listing Tenant A members, got %d: %s", recMembers.Code, recMembers.Body.String())
	}
	var membersResp struct {
		Members []tenancy.Member `json:"members"`
	}
	_ = json.Unmarshal(recMembers.Body.Bytes(), &membersResp)
	if len(membersResp.Members) != 1 || membersResp.Members[0].Email != "teacher-a@example.com" {
		t.Fatalf("expected only Teacher A in Tenant A members, got %+v", membersResp.Members)
	}

	// 6. User A (org_admin) invites User B as learner; User B accepts invitation.
	recInv := doJSON(http.MethodPost, "/api/v1/tenants/current/invitations", "sess-a", map[string]string{
		"email": "learner-b@example.com",
		"role":  "learner",
	}, nil)
	if recInv.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating invitation, got %d: %s", recInv.Code, recInv.Body.String())
	}
	var invResp struct {
		Invitation tenancy.CreatedInvitation `json:"invitation"`
	}
	_ = json.Unmarshal(recInv.Body.Bytes(), &invResp)
	if invResp.Invitation.Token == "" {
		t.Fatal("expected non-empty invitation token")
	}

	recAccept := doJSON(http.MethodPost, "/api/v1/invitations/accept", "sess-b", map[string]string{
		"token": invResp.Invitation.Token,
	}, nil)
	if recAccept.Code != http.StatusOK {
		t.Fatalf("expected 200 accepting invitation, got %d: %s", recAccept.Code, recAccept.Body.String())
	}

	// 7. Re-using the accepted invitation token must fail (404).
	recReplay := doJSON(http.MethodPost, "/api/v1/invitations/accept", "sess-b", map[string]string{
		"token": invResp.Invitation.Token,
	}, nil)
	if recReplay.Code != http.StatusNotFound {
		t.Fatalf("expected 404 replaying accepted invite token, got %d", recReplay.Code)
	}

	// 8. User B (now learner in Tenant A) is denied org_admin invitation creation (403 FORBIDDEN).
	recDenied := doJSON(http.MethodPost, "/api/v1/tenants/current/invitations", "sess-b", map[string]string{
		"email": "someone@example.com",
		"role":  "teacher",
	}, nil)
	if recDenied.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for learner creating invitation, got %d: %s", recDenied.Code, recDenied.Body.String())
	}

	// 9. Logout revokes session so subsequent /me returns 401.
	if recOut := doJSON(http.MethodPost, "/api/v1/auth/logout", "sess-b", nil, nil); recOut.Code != http.StatusOK {
		t.Fatalf("expected 200 on logout, got %d", recOut.Code)
	}
	if recAfterOut := doJSON(http.MethodGet, "/api/v1/me", "sess-b", nil, nil); recAfterOut.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", recAfterOut.Code)
	}
}
