# ADR 0003 — Direct-to-Backblaze B2 Presigned Object Storage (No MinIO)

**Status:** Accepted (Explicit Product-Owner Decision — 2026-11-28)  
**References:** [`docs/MASTER_BLUEPRINT.md`](../MASTER_BLUEPRINT.md) (§9, §23, §24 #10), [`docs/ADR/0001-approved-stack-and-mvp.md`](./0001-approved-stack-and-mvp.md), [`docs/ADR/0006-nats-jetstream-durable-jobs.md`](./0006-nats-jetstream-durable-jobs.md)

---

## 1. Context
TutorGrid requires secure, tenant-isolated file storage for lesson attachments, PDFs, PowerPoint presentations, images, and future whiteboard/activity assets. An earlier proposed draft considered running local MinIO alongside Backblaze B2, which risked CORS and presigned-header parity drift between MinIO and real Backblaze B2 (`OD-11`).

## 2. Decision
1. **Backblaze B2 Exclusively (MinIO Removed from Local Dev & CI):**
   - Use **Backblaze B2** via its S3-compatible API (`aws-sdk-go-v2/service/s3`, configured with `RequestChecksumCalculationWhenRequired` and `ResponseChecksumValidationWhenRequired` for browser presigned `PUT` compatibility) across **local development, automated tests, staging, and production**.
   - **Remove MinIO** from `deployments/docker-compose.dev.yml` and GitHub Actions CI.
   - Use separate private Backblaze B2 buckets with bucket-scoped Application Keys for **dev** (`TUTORGRID_STORAGE.*`) and **test/CI** (`TUTORGRID_TEST_STORAGE_*`). Automated tests hard-fail if configured to point at the dev bucket and never fall back to it.
   - Unconditional config validation rejects `http://`, `localhost`, `127.0.0.1`, `::1`, `minio`, and `*.local` endpoints in all environments.
2. **Two-Step Direct Presigned PUT Upload & Deep Verification:**
   - `POST /api/v1/resources/upload-url`: Enforces `RequireAuth`, `RequireTenant`, and teacher/admin RBAC (`resource:write`). Validates 25 MB MVP size cap and strict MIME allowlist (`image/png`, `image/jpeg`, `image/webp`, `application/pdf`, `application/vnd.ms-powerpoint`, `application/vnd.openxmlformats-officedocument.presentationml.presentation`; no raw SVG). Generates server-owned tenant-prefixed key (`tenants/{tenant_id}/resources/{resource_id}/{safe_name}`), creates an RLS-scoped `pending` row, and returns a 5-minute presigned PUT URL bound to key, content type, and length.
   - `POST /api/v1/resources/:id/complete`: Idempotent completion treating browser-declared size as untrusted. Verifies actual Backblaze B2 object existence and size (`0 < size <= 25 MiB`) via `HeadObject`, validates magic bytes (`bytes=0-511`), and for PPTX validates the actual OOXML ZIP package structure (`[Content_Types].xml` and `ppt/presentation.xml` with bounded entry count). Marks `ready` **only** after all checks succeed; deletes invalid/spoofed objects from B2 and marks `failed`.
3. **Safe Handling of Unscanned Files (MVP Gate):**
   - Malware scanning is **not yet active** in Slice 3. Completed resources retain `malware_scan_status = 'pending'` (never marked `'clean'`) and invoke a no-op `MalwareScanEnqueuer` extension point ready for a NATS JetStream scanning worker.
   - All presigned GET download URLs enforce 5-minute TTL and `Content-Disposition: attachment; filename="<sanitized>"`.
   - The web application **never renders uploaded files inline** (`<img>`, `<iframe>`, `<object>`, or embedded PDF preview).
   - **Mandatory Pre-Learner Gate:** A real malware-scanning pipeline and scan-status enforcement gate must be enabled before broader learner-facing resource delivery.
4. **Authoritative RBAC & Partial-Failure Recovery:**
   - Resource Library list/download/upload are restricted to `teacher`, `org_admin`, and `platform_admin` (`learner` receives `403 Forbidden`). Deletion requires uploader ownership (`uploaded_by_user_id == user_id`) or `org_admin`/`platform_admin`.
   - Deletes transition `status = 'deleting'` under `RunInTenantTx` (immediately hiding the resource), publish a durable NATS JetStream delete job (`tutorgrid.jobs.resources.delete`), and execute idempotent B2 `DeleteObject` (`NotFound` = OK) followed by DB row deletion.
   - A bounded periodic reconciler recovers stale `pending` uploads and stuck `deleting` rows using a tightly constrained `SECURITY DEFINER` pointer function owned by a dedicated `NOLOGIN NOSUPERUSER NOBYPASSRLS` role (`tutorgrid_reconcile_definer`) with column-level `SELECT` on `(id, tenant_id, status, created_at, updated_at)` and an RLS policy limited to `status IN ('pending', 'deleting')`. All actual reads, B2 deletes, and DB mutations execute strictly inside `RunInTenantTx` under runtime RLS.
5. **Opt-In Real B2 Tests & No Silent Conversion:**
   - Unit tests and real PostgreSQL/RLS + NATS JetStream tests run without B2 credentials (safe on forked PRs); real Backblaze B2 E2E integration tests run whenever `TUTORGRID_TEST_STORAGE_*` secrets are configured.
   - Uploaded PDFs and PowerPoints remain attached reference materials and are never silently converted into editable lessons.

## 3. Consequences
- Real Backblaze B2 presigned PUT, CORS, `HeadObject`, byte-range reads, and `Content-Disposition: attachment` downloads are verified continuously from Slice 3 onward (resolving `OD-11`).
- Zero MinIO operational overhead in local dev or CI.
- Partial failures between PostgreSQL and Backblaze B2 are durably reconciled without weakening runtime RLS.
