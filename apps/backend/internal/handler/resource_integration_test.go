package handler_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/thecodearcher/limen"

	"github.com/tutorgrid/tutorgrid/internal/config"
	"github.com/tutorgrid/tutorgrid/internal/database"
	"github.com/tutorgrid/tutorgrid/internal/handler"
	"github.com/tutorgrid/tutorgrid/internal/lib/job"
	"github.com/tutorgrid/tutorgrid/internal/lib/storage"
	"github.com/tutorgrid/tutorgrid/internal/middleware"
	"github.com/tutorgrid/tutorgrid/internal/model"
	"github.com/tutorgrid/tutorgrid/internal/repository"
	"github.com/tutorgrid/tutorgrid/internal/server"
	"github.com/tutorgrid/tutorgrid/internal/service"
)

type memoryObjectStore struct {
	mu           sync.Mutex
	objects      map[string]memObj
	failDelete   bool
	deleteCalls  int
	lastDispName string
}

type memObj struct {
	data        []byte
	contentType string
	etag        string
}

func newMemoryObjectStore() *memoryObjectStore {
	return &memoryObjectStore{objects: make(map[string]memObj)}
}

func (m *memoryObjectStore) putDirect(key, contentType string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.objects[key] = memObj{
		data:        cp,
		contentType: contentType,
		etag:        fmt.Sprintf("etag-%d", len(cp)),
	}
}

func (m *memoryObjectStore) exists(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[key]
	return ok
}

func (m *memoryObjectStore) PresignPut(_ context.Context, key, _ string, _ int64, _ time.Duration) (string, error) {
	return "https://mem-b2.test/put/" + key, nil
}

func (m *memoryObjectStore) PresignGet(_ context.Context, key, downloadFilename string, _ time.Duration) (string, error) {
	m.mu.Lock()
	m.lastDispName = downloadFilename
	m.mu.Unlock()
	return "https://mem-b2.test/get/" + key + "?response-content-disposition=attachment", nil
}

func (m *memoryObjectStore) HeadObject(_ context.Context, key string) (storage.ObjectInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[key]
	if !ok {
		return storage.ObjectInfo{Exists: false}, nil
	}
	return storage.ObjectInfo{
		Exists:      true,
		SizeBytes:   int64(len(obj.data)),
		ETag:        obj.etag,
		ContentType: obj.contentType,
	}, nil
}

func (m *memoryObjectStore) ReadRange(_ context.Context, key string, start, endInclusive int64) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[key]
	if !ok {
		return nil, errors.New("not found")
	}
	if start >= int64(len(obj.data)) {
		return nil, io.EOF
	}
	if endInclusive >= int64(len(obj.data)) {
		endInclusive = int64(len(obj.data)) - 1
	}
	cp := make([]byte, endInclusive-start+1)
	copy(cp, obj.data[start:endInclusive+1])
	return cp, nil
}

func (m *memoryObjectStore) DeleteObject(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteCalls++
	if m.failDelete {
		return errors.New("simulated B2 transient 503 error")
	}
	delete(m.objects, key)
	return nil
}

func (m *memoryObjectStore) ObjectURL(key string) string {
	return "https://mem-b2.test/raw/" + key
}

func makeValidPPTXBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"[Content_Types].xml":  `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"></Types>`,
		"ppt/presentation.xml": `<?xml version="1.0" encoding="UTF-8"?><p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"></p:presentation>`,
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func makePlainZipBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("evil.txt")
	_, _ = w.Write([]byte("plain zip disguised as pptx"))
	_ = zw.Close()
	return buf.Bytes()
}

// TestResourceLibraryRLSJetStreamAndRecovery runs without any Backblaze B2 credentials
// against real PostgreSQL (tutorgrid_app NOBYPASSRLS) and real NATS JetStream (if configured).
func TestResourceLibraryRLSJetStreamAndRecovery(t *testing.T) {
	migrationDSN := os.Getenv("TUTORGRID_TEST_DATABASE_MIGRATION_URL")
	appDSN := os.Getenv("TUTORGRID_TEST_DATABASE_URL")
	if migrationDSN == "" || appDSN == "" {
		t.Skip("set TUTORGRID_TEST_DATABASE_MIGRATION_URL and TUTORGRID_TEST_DATABASE_URL to run resource integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	migratorConn, err := pgx.Connect(ctx, migrationDSN)
	if err != nil {
		t.Fatalf("connect migrator: %v", err)
	}
	defer migratorConn.Close(ctx)

	testLog := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.ErrorLevel)
	if err := database.MigrateConn(ctx, &testLog, migratorConn); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	if _, err := migratorConn.Exec(ctx, "TRUNCATE resources, invitation_email_outbox, tenant_invitations, user_active_tenants, tenant_memberships, users, tenants CASCADE"); err != nil {
		t.Fatalf("truncate fixtures: %v", err)
	}

	appPool, err := pgxpool.New(ctx, appDSN)
	if err != nil {
		t.Fatalf("open app pool: %v", err)
	}
	defer appPool.Close()

	// 1. Verify constrained SECURITY DEFINER role & pointer function hardening
	t.Run("reconcile definer role least privilege verification", func(t *testing.T) {
		var canLogin, isSuper, bypassRLS bool
		if err := migratorConn.QueryRow(ctx,
			"SELECT rolcanlogin, rolsuper, rolbypassrls FROM pg_roles WHERE rolname = 'tutorgrid_reconcile_definer'",
		).Scan(&canLogin, &isSuper, &bypassRLS); err != nil {
			t.Fatalf("query tutorgrid_reconcile_definer: %v", err)
		}
		if canLogin || isSuper || bypassRLS {
			t.Fatalf("tutorgrid_reconcile_definer must be NOLOGIN NOSUPERUSER NOBYPASSRLS, got login=%v super=%v bypassrls=%v", canLogin, isSuper, bypassRLS)
		}

		var fnOwner string
		if err := migratorConn.QueryRow(ctx, `
			SELECT r.rolname
			FROM pg_proc p
			JOIN pg_roles r ON r.oid = p.proowner
			WHERE p.proname = 'resources_list_reconcile_candidates'
		`).Scan(&fnOwner); err != nil {
			t.Fatalf("query function owner: %v", err)
		}
		if fnOwner != "tutorgrid_reconcile_definer" {
			t.Fatalf("expected function owner tutorgrid_reconcile_definer, got %q", fnOwner)
		}

		var canReadFilename, canReadObjectKey bool
		_ = migratorConn.QueryRow(ctx, "SELECT has_column_privilege('tutorgrid_reconcile_definer', 'public.resources', 'filename', 'SELECT')").Scan(&canReadFilename)
		_ = migratorConn.QueryRow(ctx, "SELECT has_column_privilege('tutorgrid_reconcile_definer', 'public.resources', 'object_key', 'SELECT')").Scan(&canReadObjectKey)
		if canReadFilename || canReadObjectKey {
			t.Fatalf("reconcile definer must NOT have SELECT privilege on filename or object_key")
		}
	})

	tenancyRepo := repository.NewTenancyRepository(appPool)
	tenancySvc := service.NewTenancyService(tenancyRepo, nil, "http://localhost:5173", &testLog)

	memStore := newMemoryObjectStore()
	resourceRepo := repository.NewResourceRepository(appPool)

	var jobSvc *job.JobService
	if natsURL := os.Getenv("TUTORGRID_TEST_NATS_URL"); natsURL != "" {
		js, err := job.NewJobService(&testLog, &config.Config{NATS: config.NATSConfig{URL: natsURL}})
		if err != nil {
			t.Fatalf("connect NATS JetStream for test: %v", err)
		}
		defer js.Stop()
		jobSvc = js
	}

	resourceSvc := service.NewResourceService(resourceRepo, memStore, jobSvc, &testLog)
	if jobSvc != nil {
		jobSvc.SetResourceJobHandler(resourceSvc)
		if err := jobSvc.Start(); err != nil {
			t.Fatalf("start NATS consumers: %v", err)
		}
	}

	validator := &stubSessionValidator{
		sessions: map[string]*limen.ValidatedSession{
			"admin-a":   {User: &limen.User{ID: "res-admin-a", Email: "admin-a@example.com"}},
			"teacher-a": {User: &limen.User{ID: "res-teacher-a", Email: "teacher-a@example.com"}},
			"teacher-2": {User: &limen.User{ID: "res-teacher-2", Email: "teacher-2@example.com"}},
			"learner-a": {User: &limen.User{ID: "res-learner-a", Email: "learner-a@example.com"}},
			"admin-b":   {User: &limen.User{ID: "res-admin-b", Email: "admin-b@example.com"}},
		},
	}

	srv := &server.Server{Logger: &testLog}
	authMW := middleware.NewAuthMiddleware(srv, validator, tenancySvc)
	tenancyHandler := handler.NewTenancyHandler(srv, tenancySvc, nil)
	resourceHandler := handler.NewResourceHandler(srv, resourceSvc)

	e := echo.New()
	e.HTTPErrorHandler = middleware.NewGlobalMiddlewares(srv).GlobalErrorHandler
	v1 := e.Group("/api/v1")
	authed := v1.Group("", authMW.RequireAuth)
	authed.POST("/tenants", tenancyHandler.CreateTenant)
	authed.GET("/me", tenancyHandler.Me)

	resGroup := authed.Group("/resources", authMW.RequireTenant)
	resGroup.GET("", resourceHandler.List, authMW.RequirePermission(model.PermissionResourceRead))
	resGroup.POST("/upload-url", resourceHandler.RequestUploadURL, authMW.RequirePermission(model.PermissionResourceWrite))
	resGroup.POST("/:id/complete", resourceHandler.CompleteUpload, authMW.RequirePermission(model.PermissionResourceWrite))
	resGroup.GET("/:id/download-url", resourceHandler.GetDownloadURL, authMW.RequirePermission(model.PermissionResourceRead))
	resGroup.DELETE("/:id", resourceHandler.Delete, authMW.RequirePermission(model.PermissionResourceWrite))

	doJSON := func(method, path, sess string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		if sess != "" {
			req.AddCookie(&http.Cookie{Name: "test_session", Value: sess})
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	// Setup Tenant A (admin-a, teacher-a, teacher-2, learner-a) and Tenant B (admin-b)
	recA := doJSON(http.MethodPost, "/api/v1/tenants", "admin-a", map[string]string{"name": "School A", "slug": "res-school-a"})
	if recA.Code != http.StatusCreated {
		t.Fatalf("create tenant A: %d %s", recA.Code, recA.Body.String())
	}
	var respA struct{ Membership model.Membership }
	_ = json.Unmarshal(recA.Body.Bytes(), &respA)

	recB := doJSON(http.MethodPost, "/api/v1/tenants", "admin-b", map[string]string{"name": "School B", "slug": "res-school-b"})
	if recB.Code != http.StatusCreated {
		t.Fatalf("create tenant B: %d %s", recB.Code, recB.Body.String())
	}

	// Trigger user sync for teacher-a, teacher-2, learner-a and add memberships in Tenant A
	for _, pair := range []struct {
		sess  string
		email string
		role  string
	}{
		{"teacher-a", "teacher-a@example.com", "teacher"},
		{"teacher-2", "teacher-2@example.com", "teacher"},
		{"learner-a", "learner-a@example.com", "learner"},
	} {
		_ = doJSON(http.MethodGet, "/api/v1/me", pair.sess, nil)
		_, err := migratorConn.Exec(ctx, `
			WITH u AS (SELECT id FROM users WHERE email = $1)
			INSERT INTO tenant_memberships (tenant_id, user_id, role)
			SELECT $2, id, $3 FROM u;
		`, pair.email, respA.Membership.TenantID, pair.role)
		if err != nil {
			t.Fatalf("insert membership for %s: %v", pair.email, err)
		}
		_, err = migratorConn.Exec(ctx, `
			WITH u AS (SELECT id FROM users WHERE email = $1)
			INSERT INTO user_active_tenants (user_id, active_tenant_id)
			SELECT id, $2 FROM u
			ON CONFLICT (user_id) DO UPDATE SET active_tenant_id = EXCLUDED.active_tenant_id;
		`, pair.email, respA.Membership.TenantID)
		if err != nil {
			t.Fatalf("set active tenant for %s: %v", pair.email, err)
		}
	}

	var uploadedResourceID uuid.UUID
	var uploadedObjectKey string

	// 2. Upload PDF -> Complete -> Idempotent Complete -> Unscanned Pending Check -> Download
	t.Run("presigned upload complete idempotent and attachment download", func(t *testing.T) {
		recUp := doJSON(http.MethodPost, "/api/v1/resources/upload-url", "teacher-a", map[string]any{
			"filename":     "Algebra Notes.pdf",
			"content_type": service.MimePDF,
			"size_bytes":   1024,
		})
		if recUp.Code != http.StatusCreated {
			t.Fatalf("expected 201 upload-url, got %d: %s", recUp.Code, recUp.Body.String())
		}
		var upResp model.PresignedUpload
		_ = json.Unmarshal(recUp.Body.Bytes(), &upResp)
		uploadedResourceID = upResp.Resource.ID

		// Load exact server-generated object key from DB to simulate browser PUT to B2
		if err := migratorConn.QueryRow(ctx, "SELECT object_key FROM resources WHERE id = $1", uploadedResourceID).Scan(&uploadedObjectKey); err != nil {
			t.Fatalf("read object key: %v", err)
		}
		if !strings.HasPrefix(uploadedObjectKey, "tenants/"+respA.Membership.TenantID.String()+"/resources/") {
			t.Fatalf("unexpected key prefix: %s", uploadedObjectKey)
		}

		// Simulate direct browser upload of valid PDF bytes
		memStore.putDirect(uploadedObjectKey, service.MimePDF, []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF"))

		recComp := doJSON(http.MethodPost, "/api/v1/resources/"+uploadedResourceID.String()+"/complete", "teacher-a", nil)
		if recComp.Code != http.StatusOK {
			t.Fatalf("expected 200 complete, got %d: %s", recComp.Code, recComp.Body.String())
		}
		var compResp struct{ Resource model.Resource }
		_ = json.Unmarshal(recComp.Body.Bytes(), &compResp)
		if compResp.Resource.Status != model.ResourceStatusReady {
			t.Fatalf("expected status ready, got %s", compResp.Resource.Status)
		}
		// Adjustment #2: Never mark unscanned file as clean!
		if compResp.Resource.MalwareScanStatus != model.MalwareScanPending {
			t.Fatalf("expected malware_scan_status=pending, got %s", compResp.Resource.MalwareScanStatus)
		}

		// Idempotent second completion returns 200
		if recAgain := doJSON(http.MethodPost, "/api/v1/resources/"+uploadedResourceID.String()+"/complete", "teacher-a", nil); recAgain.Code != http.StatusOK {
			t.Fatalf("expected idempotent 200 on second complete, got %d", recAgain.Code)
		}

		// Download URL forces attachment disposition
		recDL := doJSON(http.MethodGet, "/api/v1/resources/"+uploadedResourceID.String()+"/download-url", "teacher-a", nil)
		if recDL.Code != http.StatusOK {
			t.Fatalf("expected 200 download-url, got %d", recDL.Code)
		}
		if !strings.Contains(recDL.Body.String(), "attachment") {
			t.Fatalf("expected attachment disposition in download URL, got %s", recDL.Body.String())
		}
	})

	// 3. Magic-byte spoofing, plain-ZIP-as-PPTX, valid OOXML PPTX, and untrusted oversized upload
	t.Run("spoofed pdf, fake zip pptx, valid pptx, and untrusted size enforcement", func(t *testing.T) {
		// Spoofed HTML claiming application/pdf
		recBad := doJSON(http.MethodPost, "/api/v1/resources/upload-url", "teacher-a", map[string]any{
			"filename":     "fake.pdf",
			"content_type": service.MimePDF,
			"size_bytes":   100,
		})
		var badUp model.PresignedUpload
		_ = json.Unmarshal(recBad.Body.Bytes(), &badUp)
		var badKey string
		_ = migratorConn.QueryRow(ctx, "SELECT object_key FROM resources WHERE id = $1", badUp.Resource.ID).Scan(&badKey)
		memStore.putDirect(badKey, service.MimePDF, []byte("<html><script>alert(1)</script></html>"))

		if rec := doJSON(http.MethodPost, "/api/v1/resources/"+badUp.Resource.ID.String()+"/complete", "teacher-a", nil); rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for spoofed PDF, got %d: %s", rec.Code, rec.Body.String())
		}
		if memStore.exists(badKey) {
			t.Fatalf("spoofed object should have been deleted from storage")
		}

		// Plain ZIP claiming PPTX must be rejected by OOXML validation
		recFakePPTX := doJSON(http.MethodPost, "/api/v1/resources/upload-url", "teacher-a", map[string]any{
			"filename":     "disguised.pptx",
			"content_type": service.MimePPTX,
			"size_bytes":   200,
		})
		var fakePPTXUp model.PresignedUpload
		_ = json.Unmarshal(recFakePPTX.Body.Bytes(), &fakePPTXUp)
		var fakePPTXKey string
		_ = migratorConn.QueryRow(ctx, "SELECT object_key FROM resources WHERE id = $1", fakePPTXUp.Resource.ID).Scan(&fakePPTXKey)
		memStore.putDirect(fakePPTXKey, service.MimePPTX, makePlainZipBytes(t))

		if rec := doJSON(http.MethodPost, "/api/v1/resources/"+fakePPTXUp.Resource.ID.String()+"/complete", "teacher-a", nil); rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for plain ZIP disguised as PPTX, got %d", rec.Code)
		}
		if memStore.exists(fakePPTXKey) {
			t.Fatalf("plain ZIP disguised as PPTX should have been deleted from storage")
		}

		// Valid OOXML PPTX succeeds
		recGoodPPTX := doJSON(http.MethodPost, "/api/v1/resources/upload-url", "teacher-a", map[string]any{
			"filename":     "slides.pptx",
			"content_type": service.MimePPTX,
			"size_bytes":   500,
		})
		var goodPPTXUp model.PresignedUpload
		_ = json.Unmarshal(recGoodPPTX.Body.Bytes(), &goodPPTXUp)
		var goodPPTXKey string
		_ = migratorConn.QueryRow(ctx, "SELECT object_key FROM resources WHERE id = $1", goodPPTXUp.Resource.ID).Scan(&goodPPTXKey)
		memStore.putDirect(goodPPTXKey, service.MimePPTX, makeValidPPTXBytes(t))

		if rec := doJSON(http.MethodPost, "/api/v1/resources/"+goodPPTXUp.Resource.ID.String()+"/complete", "teacher-a", nil); rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for valid OOXML PPTX, got %d: %s", rec.Code, rec.Body.String())
		}

		// Untrusted browser declared 100 bytes, actually uploaded > 25 MiB
		recOver := doJSON(http.MethodPost, "/api/v1/resources/upload-url", "teacher-a", map[string]any{
			"filename":     "huge.pdf",
			"content_type": service.MimePDF,
			"size_bytes":   100,
		})
		var overUp model.PresignedUpload
		_ = json.Unmarshal(recOver.Body.Bytes(), &overUp)
		var overKey string
		_ = migratorConn.QueryRow(ctx, "SELECT object_key FROM resources WHERE id = $1", overUp.Resource.ID).Scan(&overKey)
		oversized := make([]byte, service.MaxResourceSizeBytes+10)
		copy(oversized, "%PDF-1.7\n")
		memStore.putDirect(overKey, service.MimePDF, oversized)

		if rec := doJSON(http.MethodPost, "/api/v1/resources/"+overUp.Resource.ID.String()+"/complete", "teacher-a", nil); rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for oversized B2 object, got %d", rec.Code)
		}
		if memStore.exists(overKey) {
			t.Fatalf("oversized object should have been deleted from storage")
		}
	})

	// 4. Tenant RLS Isolation & RBAC (learner blocked on all endpoints; teacher-2 cannot delete teacher-a's file)
	t.Run("tenant RLS isolation and strict RBAC", func(t *testing.T) {
		// Tenant B sees 0 of Tenant A's resources and receives 404 on download/delete
		recListB := doJSON(http.MethodGet, "/api/v1/resources", "admin-b", nil)
		if recListB.Code != http.StatusOK || strings.Contains(recListB.Body.String(), uploadedResourceID.String()) {
			t.Fatalf("tenant B leaked tenant A resource: %d %s", recListB.Code, recListB.Body.String())
		}
		if rec := doJSON(http.MethodGet, "/api/v1/resources/"+uploadedResourceID.String()+"/download-url", "admin-b", nil); rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 cross-tenant download, got %d", rec.Code)
		}
		if rec := doJSON(http.MethodDelete, "/api/v1/resources/"+uploadedResourceID.String(), "admin-b", nil); rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 cross-tenant delete, got %d", rec.Code)
		}

		// Learner blocked (403) from browsing, uploading, downloading, and deleting
		for _, tc := range []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/api/v1/resources"},
			{http.MethodPost, "/api/v1/resources/upload-url"},
			{http.MethodPost, "/api/v1/resources/" + uploadedResourceID.String() + "/complete"},
			{http.MethodGet, "/api/v1/resources/" + uploadedResourceID.String() + "/download-url"},
			{http.MethodDelete, "/api/v1/resources/" + uploadedResourceID.String()},
		} {
			if rec := doJSON(tc.method, tc.path, "learner-a", nil); rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403 for learner on %s %s, got %d", tc.method, tc.path, rec.Code)
			}
		}

		// Teacher 2 in Tenant A can list and download Teacher A's file, but CANNOT delete it (403)
		if rec := doJSON(http.MethodGet, "/api/v1/resources/"+uploadedResourceID.String()+"/download-url", "teacher-2", nil); rec.Code != http.StatusOK {
			t.Fatalf("expected teacher-2 200 download, got %d", rec.Code)
		}
		if rec := doJSON(http.MethodDelete, "/api/v1/resources/"+uploadedResourceID.String(), "teacher-2", nil); rec.Code != http.StatusForbidden {
			t.Fatalf("expected teacher-2 403 deleting teacher-a's resource, got %d", rec.Code)
		}
	})

	// 5. Partial-Failure Deletion Recovery + Bounded Periodic Reconciler (Adjustments #4 & #5)
	t.Run("partial failure delete recovery and abandoned pending reconciliation", func(t *testing.T) {
		// Simulate transient B2 delete failure when org_admin deletes teacher-a's resource
		memStore.mu.Lock()
		memStore.failDelete = true
		memStore.mu.Unlock()

		if rec := doJSON(http.MethodDelete, "/api/v1/resources/"+uploadedResourceID.String(), "admin-a", nil); rec.Code != http.StatusOK {
			t.Fatalf("expected 200 marking deleting even when B2 transiently fails, got %d: %s", rec.Code, rec.Body.String())
		}

		// Immediately hidden from List and blocked from Download while status='deleting'
		recList := doJSON(http.MethodGet, "/api/v1/resources", "admin-a", nil)
		if strings.Contains(recList.Body.String(), uploadedResourceID.String()) {
			t.Fatalf("deleting resource must be hidden immediately from list")
		}
		if rec := doJSON(http.MethodGet, "/api/v1/resources/"+uploadedResourceID.String()+"/download-url", "admin-a", nil); rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
			t.Fatalf("deleting resource must not be downloadable, got %d", rec.Code)
		}

		// Also create an abandoned pending upload (aged 30 min) with partial bytes in B2
		recAb := doJSON(http.MethodPost, "/api/v1/resources/upload-url", "teacher-a", map[string]any{
			"filename":     "abandoned.pdf",
			"content_type": service.MimePDF,
			"size_bytes":   512,
		})
		var abUp model.PresignedUpload
		_ = json.Unmarshal(recAb.Body.Bytes(), &abUp)
		var abKey string
		_ = migratorConn.QueryRow(ctx, "SELECT object_key FROM resources WHERE id = $1", abUp.Resource.ID).Scan(&abKey)
		memStore.putDirect(abKey, service.MimePDF, []byte("%PDF-partial"))
		_, _ = migratorConn.Exec(ctx, "UPDATE resources SET created_at = NOW() - INTERVAL '30 minutes', updated_at = NOW() - INTERVAL '30 minutes' WHERE id = $1", abUp.Resource.ID)
		_, _ = migratorConn.Exec(ctx, "UPDATE resources SET updated_at = NOW() - INTERVAL '10 minutes' WHERE id = $1", uploadedResourceID)

		// Now B2 recovers; bounded periodic reconciler repairs BOTH the stuck 'deleting' row AND the stale 'pending' row
		memStore.mu.Lock()
		memStore.failDelete = false
		memStore.mu.Unlock()

		stats, err := resourceSvc.RunReconciliationSweepWithStats(ctx, 20*time.Minute, 2*time.Minute, 50)
		if err != nil {
			t.Fatalf("RunReconciliationSweepWithStats: %v", err)
		}
		if stats.Discovered < 1 || stats.Failed != 0 {
			t.Fatalf("unexpected reconcile stats: %+v", stats)
		}
		if memStore.exists(abKey) {
			t.Fatalf("abandoned pending object must be removed from B2")
		}
		if memStore.exists(uploadedObjectKey) {
			t.Fatalf("deleting object must be removed from B2 after recovery")
		}

		var remaining int
		_ = migratorConn.QueryRow(ctx, "SELECT COUNT(*) FROM resources WHERE id IN ($1, $2)", uploadedResourceID, abUp.Resource.ID).Scan(&remaining)
		if remaining != 0 {
			t.Fatalf("expected both reconciled rows removed from DB, got %d remaining", remaining)
		}
	})
}

// TestResourceLibraryRealBackblazeB2 is an OPT-IN end-to-end test against a real private
// Backblaze B2 test bucket. It skips safely when TUTORGRID_TEST_STORAGE_* secrets are unset
// (e.g., on forked PRs) and HARD-FAILS if pointed at the dev bucket (Adjustment #3).
func TestResourceLibraryRealBackblazeB2(t *testing.T) {
	migrationDSN := os.Getenv("TUTORGRID_TEST_DATABASE_MIGRATION_URL")
	appDSN := os.Getenv("TUTORGRID_TEST_DATABASE_URL")
	endpoint := strings.TrimSpace(os.Getenv("TUTORGRID_TEST_STORAGE_ENDPOINT"))
	region := strings.TrimSpace(os.Getenv("TUTORGRID_TEST_STORAGE_REGION"))
	bucket := strings.TrimSpace(os.Getenv("TUTORGRID_TEST_STORAGE_BUCKET"))
	accessKey := strings.TrimSpace(os.Getenv("TUTORGRID_TEST_STORAGE_ACCESS_KEY_ID"))
	secretKey := strings.TrimSpace(os.Getenv("TUTORGRID_TEST_STORAGE_SECRET_ACCESS_KEY"))

	if migrationDSN == "" || appDSN == "" || endpoint == "" || region == "" || bucket == "" || accessKey == "" || secretKey == "" {
		t.Skip("opt-in real Backblaze B2 test skipped: configure TUTORGRID_TEST_STORAGE_* env vars to run")
	}

	// Safety guard: NEVER allow automated tests to target or fall back to the dev bucket!
	devBucket := strings.TrimSpace(os.Getenv("TUTORGRID_STORAGE.BUCKET"))
	if devBucket == "" {
		devBucket = strings.TrimSpace(os.Getenv("TUTORGRID_STORAGE_BUCKET"))
	}
	if devBucket != "" && strings.EqualFold(bucket, devBucket) {
		t.Fatalf("SECURITY GUARD: TUTORGRID_TEST_STORAGE_BUCKET (%q) must not match dev bucket (%q)", bucket, devBucket)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	b2Store, err := storage.NewB2Store(ctx, config.StorageConfig{
		Endpoint:        endpoint,
		Region:          region,
		Bucket:          bucket,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatalf("init real Backblaze B2 test store: %v", err)
	}

	migratorConn, err := pgx.Connect(ctx, migrationDSN)
	if err != nil {
		t.Fatalf("connect migrator: %v", err)
	}
	defer migratorConn.Close(ctx)

	testLog := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.ErrorLevel)
	if err := database.MigrateConn(ctx, &testLog, migratorConn); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	if _, err := migratorConn.Exec(ctx, "TRUNCATE resources, invitation_email_outbox, tenant_invitations, user_active_tenants, tenant_memberships, users, tenants CASCADE"); err != nil {
		t.Fatalf("truncate fixtures: %v", err)
	}

	appPool, err := pgxpool.New(ctx, appDSN)
	if err != nil {
		t.Fatalf("open app pool: %v", err)
	}
	defer appPool.Close()

	tenancyRepo := repository.NewTenancyRepository(appPool)
	tenancySvc := service.NewTenancyService(tenancyRepo, nil, "http://localhost:5173", &testLog)
	resourceRepo := repository.NewResourceRepository(appPool)
	resourceSvc := service.NewResourceService(resourceRepo, b2Store, nil, &testLog)

	user, err := tenancySvc.UpsertUserBySubject(ctx, "limen:b2-e2e", "B2 Teacher", "b2-e2e@example.com")
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}
	membership, err := tenancySvc.CreateTenantWithAdmin(ctx, user.ID, "B2 E2E School", "b2-e2e-school")
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	tc := model.TenantContext{TenantID: membership.TenantID, UserID: user.ID, Role: model.RoleOrgAdmin}

	var keysToCleanup []string
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanCancel()
		for _, k := range keysToCleanup {
			_ = b2Store.DeleteObject(cleanCtx, k)
		}
	})

	httpClient := &http.Client{Timeout: 30 * time.Second}

	// 1. Real presigned PUT of valid PDF -> Complete -> Real presigned GET (attachment) -> Unsigned GET denied -> Delete
	pdfBytes := []byte("%PDF-1.7\n%TutorGrid real Backblaze B2 E2E test\n%%EOF\n")
	up, err := resourceSvc.RequestUploadURL(ctx, tc, "real-b2-lesson.pdf", service.MimePDF, int64(len(pdfBytes)))
	if err != nil {
		t.Fatalf("RequestUploadURL: %v", err)
	}
	var pdfKey string
	_ = migratorConn.QueryRow(ctx, "SELECT object_key FROM resources WHERE id = $1", up.Resource.ID).Scan(&pdfKey)
	keysToCleanup = append(keysToCleanup, pdfKey)

	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, up.UploadURL, bytes.NewReader(pdfBytes))
	if err != nil {
		t.Fatalf("new PUT request: %v", err)
	}
	putReq.Header.Set("Content-Type", service.MimePDF)
	putReq.ContentLength = int64(len(pdfBytes))
	putResp, err := httpClient.Do(putReq)
	if err != nil {
		t.Fatalf("real B2 presigned PUT failed: %v", err)
	}
	putBody, _ := io.ReadAll(putResp.Body)
	putResp.Body.Close()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		t.Fatalf("real B2 presigned PUT returned %d: %s", putResp.StatusCode, string(putBody))
	}

	completed, err := resourceSvc.CompleteUpload(ctx, tc, up.Resource.ID)
	if err != nil {
		t.Fatalf("CompleteUpload against real B2 failed: %v", err)
	}
	if completed.Status != model.ResourceStatusReady || completed.MalwareScanStatus != model.MalwareScanPending {
		t.Fatalf("unexpected completed state: status=%s scan=%s", completed.Status, completed.MalwareScanStatus)
	}

	// Verify presigned GET returns exact bytes AND Content-Disposition: attachment
	dl, err := resourceSvc.GetDownloadURL(ctx, tc, completed.ID)
	if err != nil {
		t.Fatalf("GetDownloadURL: %v", err)
	}
	getResp, err := httpClient.Get(dl.DownloadURL)
	if err != nil {
		t.Fatalf("real B2 presigned GET failed: %v", err)
	}
	gotBytes, _ := io.ReadAll(getResp.Body)
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("real B2 presigned GET returned %d", getResp.StatusCode)
	}
	if !bytes.Equal(gotBytes, pdfBytes) {
		t.Fatalf("downloaded bytes mismatch")
	}
	if disp := getResp.Header.Get("Content-Disposition"); !strings.Contains(strings.ToLower(disp), "attachment") {
		t.Fatalf("expected Content-Disposition: attachment from real B2, got %q", disp)
	}

	// Verify private bucket rejects unsigned direct GET
	rawResp, err := httpClient.Get(b2Store.ObjectURL(pdfKey))
	if err == nil {
		rawResp.Body.Close()
		if rawResp.StatusCode == http.StatusOK {
			t.Fatalf("SECURITY FAILURE: private Backblaze B2 test bucket served unsigned direct GET with 200 OK")
		}
	}

	// Verify real OOXML PPTX upload + range-based central directory inspection against real B2
	pptxBytes := makeValidPPTXBytes(t)
	upPPTX, err := resourceSvc.RequestUploadURL(ctx, tc, "real-slides.pptx", service.MimePPTX, int64(len(pptxBytes)))
	if err != nil {
		t.Fatalf("RequestUploadURL pptx: %v", err)
	}
	var pptxKey string
	_ = migratorConn.QueryRow(ctx, "SELECT object_key FROM resources WHERE id = $1", upPPTX.Resource.ID).Scan(&pptxKey)
	keysToCleanup = append(keysToCleanup, pptxKey)

	putPPTXReq, _ := http.NewRequestWithContext(ctx, http.MethodPut, upPPTX.UploadURL, bytes.NewReader(pptxBytes))
	putPPTXReq.Header.Set("Content-Type", service.MimePPTX)
	putPPTXReq.ContentLength = int64(len(pptxBytes))
	putPPTXResp, err := httpClient.Do(putPPTXReq)
	if err != nil {
		t.Fatalf("real B2 PPTX PUT failed: %v", err)
	}
	putPPTXResp.Body.Close()
	if putPPTXResp.StatusCode < 200 || putPPTXResp.StatusCode >= 300 {
		t.Fatalf("real B2 PPTX PUT returned %d", putPPTXResp.StatusCode)
	}
	if _, err := resourceSvc.CompleteUpload(ctx, tc, upPPTX.Resource.ID); err != nil {
		t.Fatalf("real B2 OOXML PPTX completion failed: %v", err)
	}

	// Delete both resources and confirm HeadObject reports gone on real B2
	for _, id := range []uuid.UUID{completed.ID, upPPTX.Resource.ID} {
		if err := resourceSvc.DeleteResource(ctx, tc, id); err != nil {
			t.Fatalf("DeleteResource %s: %v", id, err)
		}
	}
	info, err := b2Store.HeadObject(ctx, pdfKey)
	if err != nil {
		t.Fatalf("HeadObject after delete: %v", err)
	}
	if info.Exists {
		t.Fatalf("expected PDF object to be deleted from real Backblaze B2")
	}
}
