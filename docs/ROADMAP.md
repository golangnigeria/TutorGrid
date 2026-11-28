# TutorGrid Strategic Roadmap — MVP to First 1,000 Users

**Long-Term Vision Reference:** [`docs/MASTER_BLUEPRINT.md`](./MASTER_BLUEPRINT.md) · [`docs/PRODUCT_CAPABILITY_MAP.md`](./PRODUCT_CAPABILITY_MAP.md) · [`docs/TECHNOLOGY_DECISIONS.md`](./TECHNOLOGY_DECISIONS.md)  
**Scope Note:** This roadmap sequences the **MVP and first 1,000 registered users** (capacity defined by measured pilot concurrency, not raw registration count). Full long-term capabilities (Elasticsearch, ClickHouse, full white-label, external APIs/SDKs, AI co-pilot, Hetzner self-hosted LiveKit) remain preserved in `docs/PRODUCT_CAPABILITY_MAP.md` and are phased after MVP.

---

## Phase 0 — Product & Repository Foundation *(Completed — `slice/01-foundation`)*
- [x] Establish `AGENTS.md`, constitution, master blueprint, capability map, technology decision register, and ADR process.
- [x] Initialize Go + Echo + Bun + Goose backend skeleton, React + Vite responsive web shell, `Taskfile.yml` (no `Makefile` or `scripts/`), and GitHub Actions CI.
- [x] Establish local PostgreSQL (`deployments/docker-compose.dev.yml`) with non-superuser `tutorgrid_app` (`NOSUPERUSER NOBYPASSRLS`) and `tutorgrid_test`.
- [x] Prove fail-closed `FORCE ROW LEVEL SECURITY` and pooled-connection tenant isolation via real PostgreSQL integration tests.

**Exit gate:** Repo is reproducible via `Taskfile.yml`, rules/docs are reconciled, CI runs real PostgreSQL RLS tests, and architecture boundaries are documented.

---

## Phase 1 — Secure Platform Skeleton *(Auth Done — Slice 2b; NATS + B2 Storage In Progress — Slice 3)*
- [x] **Slice 2 / 2b (Accepted — `ADR 0002`):** Embedded Limen (`github.com/thecodearcher/limen`) Google & GitHub OAuth (no email/password), `limen_*` schema coexistence, PostgreSQL Dual RLS (zero `SECURITY DEFINER`), server-bound active workspace switching (`user_active_tenants`), RBAC (`platform_admin`, `org_admin`, `teacher`, `learner`), atomic email-restricted invitations with transactional outbox + Resend, in-app pending invitation discovery, and TutorGrid Circle dark-only React shell.
- [ ] **Slice 3 (Stage 1: `ADR 0006` NATS JetStream; Stage 2: `ADR 0003` Backblaze B2-Only Resource Library):**
  - Restore **NATS JetStream** as the single durable job/event system (removing Asynq and migrating Slice 2b invitation emails onto JetStream).
  - Single S3-compatible Go storage adapter (`aws-sdk-go-v2/service/s3`) targeting **Backblaze B2 exclusively** across dev, test/CI, staging, and production (no MinIO; separate private dev & test buckets; opt-in live B2 tests that never fall back to dev bucket).
  - Two-step direct presigned PUT upload (`pending` → `HeadObject` untrusted-size check + magic-byte + deep OOXML PPTX validation → idempotent `ready`), 25 MB cap, MIME allowlist (PNG, JPEG, WebP, PDF, PPT, PPTX; no raw SVG), 5-min `Content-Disposition: attachment` presigned GET (zero inline rendering), durable partial-failure reconciliation (`deleting` state + JetStream retries + bounded periodic reconciler via non-login/`NOBYPASSRLS` pointer function), and `malware_scan_status = 'pending'` extension point (**mandatory real malware-scanning gate required before broader learner-facing resource delivery**).
  - Dedicated `/resources` Resource Library route in TutorGrid Circle's dark-only shell (teachers/admins only; uploader-or-admin delete).

**Exit gate:** NATS JetStream is the sole durable job system, authenticated users access only authorized tenant resources, direct B2 presigned uploads, magic-byte + PPTX OOXML validation, partial-failure reconciliation, tenant RLS isolation, and CI checks pass.

---

## Phase 2 — Lesson Studio Vertical Slice *(Slice 4)*
- Create, edit, save, duplicate, and archive lessons with extensible metadata (title, subject, grade/age band, topic, objectives, duration).
- Ordered sections/steps with add, edit, reorder, and delete (structured content blocks—never one opaque HTML blob).
- Attach uploaded PDF/PowerPoint/image resources as reference materials (no silent auto-conversion into editable lessons).
- Explicit draft autosave, preview, and **immutable `lesson_versions` publishing**.
- *Open decision before coding:* Confirm math/equation renderer (`OD-06`, KaTeX recommended).

**Exit gate:** Teacher completes create → edit → attach resource → preview → publish; version immutability and cross-tenant isolation tests pass.

---

## Phase 3 — Whiteboard Engine Vertical Slice *(Slice 5)*
- **Mandatory Pre-Slice Gate (`OD-01`, `OD-02`, `OD-03`):**
  1. Define the **TutorGrid-owned whiteboard document schema** (`whiteboard-core`) independent of any renderer.
  2. Submit a **Konva vs. tldraw** renderer comparison (including written tldraw licensing verification for multi-tenant SaaS, embedding, redistribution, and white-label if commercial SDK is considered) and receive explicit approval for **one MVP renderer**. Do not build both renderers in MVP.
  3. Confirm exact CRDT library/collaboration strategy (`OD-03`).
- Implement single approved renderer adapter (`whiteboard-konva` or `whiteboard-tldraw`), multi-page + expandable pan/zoom canvas, basic tools, save/load, teacher/learner permissions, and **Centrifugo** + **Redis** realtime sync with coalesced updates and reconnect recovery.
- Separate prepared lesson board from isolated live classroom session copy.

**Exit gate:** Two clients collaborate with server-enforced permissions, state survives refresh/reconnect, and prepared boards are protected from live session mutations.

---

## Phase 4 — Live Classroom + LiveKit Cloud Vertical Slice *(Slice 6)*
- Create/start/end classroom session pinned to an immutable published `lesson_version_id`.
- Authorized participant join flow (*Open decision `OD-04`: guest learner access policy*).
- **Mandatory LiveKit Cloud** audio/video/screen-sharing adapter with short-lived backend-issued tokens (Hetzner self-hosting deferred post-MVP).
- Learner media permission controls (*Recommended, Pending Approval `OD-05`: learners join media-disabled by default; teacher grants/revokes server-side*).
- Guided and free lesson navigation, Centrifugo presence/events, and persisted session/participation records.

**Exit gate:** Teacher and learner complete a live test lesson; unauthorized join/media publish is rejected server-side; reconnect and end-session flows are tested.

---

## Phase 5 — Educational Activity/Game Engine Vertical Slice *(Slice 7)*
- Declarative, sandboxed activity runtime contract (zero arbitrary teacher-uploaded code execution).
- Approved starter templates (*Open decision `OD-08`: finalize count/types from quiz, matching, sorting, fill-in-the-blank*).
- Teacher configuration and preview in Lesson Studio; launch inside Live Classroom; individual gameplay first plus basic teacher-led shared activity.
- Server-validated, idempotent result persistence and basic teacher review.

**Exit gate:** Multi-learner classroom activity flow completes with idempotent result persistence and zero duplicate submissions.

---

## Phase 6 — Pilot Hardening & Operational Readiness
- Real teacher/learner usability sessions and responsive desktop/tablet/mobile browser checks.
- Security & tenant-isolation audit (RLS, tokens, signed B2 URLs, permissions, XSS/input validation, audit trail coverage).
- Real **Backblaze B2** staging CORS & presigned URL verification (`OD-11`) and **Northflank** capability validation (`OD-10`).
- Load test against an explicit pilot concurrency profile (`OD-07`), not raw registered-user count.
- Backup/restore rehearsal against pilot RPO/RTO targets (24h/24h initial), deployment rollback rehearsal, and OpenTelemetry/Prometheus/Grafana/Sentry dashboards & alerts.

**Exit gate:** Pilot acceptance criteria pass, known limitations documented, and operational runbooks exist.

---

## Phase 7 — First 1,000 Users Cohort Rollout & Post-MVP Evolution
- Gradually onboard pilot cohorts; measure activation, classroom reliability, learner join success, activity completion, whiteboard reconnect recovery, error rate, and cost per active classroom hour.
- Scale managed resources strictly from measured bottlenecks.
- Post-MVP capabilities (preserved in [`docs/PRODUCT_CAPABILITY_MAP.md`](./PRODUCT_CAPABILITY_MAP.md)): advanced visual game builder, dual whiteboard renderer option, FFmpeg/LibreOffice async NATS media processing, Elasticsearch search, ClickHouse analytics, external partner APIs/SDKs & white-label, Hetzner self-hosted LiveKit evaluation, and teacher-reviewed AI co-pilot.

---

## Scope-Change Process
Any proposed scope change must state: user value, MVP necessity, dependencies, estimate/risk, security/privacy impact, cost/operational impact, and what is deferred if added. Product owner approves all scope changes.
