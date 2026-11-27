# ADR 0003 — Direct-to-S3/Backblaze B2 Presigned Upload Architecture & MinIO Local Parity

**Status:** **Proposed — Pending Approval**  
**Date:** 2026-11-26  
**References:** [`docs/MASTER_BLUEPRINT.md`](../MASTER_BLUEPRINT.md) (§9, §23, §24 #10), [`docs/ADR/0001-approved-stack-and-mvp.md`](./0001-approved-stack-and-mvp.md)

---

## 1. Context
TutorGrid requires secure, tenant-isolated file storage for lesson attachments, PDFs, PowerPoint presentations, images, and future whiteboard/activity assets. Backblaze B2 (S3-compatible API) is the confirmed staging/production object storage provider.

## 2. Proposed Decision
1. **Single S3-Compatible Adapter:** Implement one Go `ObjectStore` adapter (`aws-sdk-go-v2/service/s3`) configured purely via `TUTORGRID_STORAGE_*` environment variables. Use it identically for **MinIO** in local `deployments/docker-compose.dev.yml` and **Backblaze B2** in staging/production. Never create separate MinIO vs. B2 application implementations.
2. **Two-Step Direct Presigned PUT Flow:**
   - `POST /api/v1/resources/upload-url`: Validate RBAC, 25 MB MVP size cap, and strict MIME allowlist (`image/png`, `image/jpeg`, `image/webp`, `application/pdf`, `application/vnd.ms-powerpoint`, `application/vnd.openxmlformats-officedocument.presentationml.presentation`; no raw SVG). Generate server-owned tenant-prefixed key (`tenants/{tenant_id}/resources/{resource_id}/{safe_name}`). Create RLS-scoped `pending` record and return 5-minute presigned PUT URL.
   - `POST /api/v1/resources/:id/complete`: Idempotent completion performing `HeadObject` existence/size verification + `GetObject` byte-range (`bytes=0-511`) magic-byte validation. Reject and delete spoofed objects. Mark `ready` and trigger the async `MalwareScanEnqueuer` extension point (ready for NATS JetStream).
3. **Downloads, Cleanup & Production Guard:**
   - Downloads use 5-minute presigned GET URLs after RLS tenant verification (private bucket only).
   - Provide abandoned `pending` upload cleanup.
   - Config validation rejects `localhost`, `127.0.0.1`, `minio`, or non-HTTPS storage endpoints whenever `TUTORGRID_APP_ENV` is not `development`/`test`.
4. **No Silent Conversion:** Uploaded PDFs and PowerPoints remain attached reference materials.

## 3. Consequences & Verification Requirement
- Eliminates Go API memory/bandwidth bottlenecks for file uploads and enforces tenant ownership before any URL is signed.
- **Mandatory Staging Gate (Master Blueprint §9.4, §24 #10):** Local MinIO testing does **not** prove real Backblaze B2 CORS and presigned header behavior. A real Backblaze B2 integration test in staging is required before production reliance.
