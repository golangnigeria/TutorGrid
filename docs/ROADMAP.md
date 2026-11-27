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

## Phase 1 — Secure Platform Skeleton *(Auth Done — `slice/02-limen-auth`; Storage Next — Slice 3)*
- [x] **Slice 2 (Implemented, Pending ADR 0002 Security Review):** Embedded Limen (`github.com/thecodearcher/limen`) Google & GitHub OAuth (no email/password), `limen_*` schema coexistence, verified user/membership lookup deriving `tenancy.Context` server-side (zero client tenant header trust), RBAC baseline (`platform_admin`, `org_admin`, `teacher`, `learner`), onboarding (Create Teaching Space / Accept Invitation), and React sign-in/onboarding shell.
- [ ] **Slice 3 (Next — Planned in `ADR 0003` & `ADR 0004`):**
  - Single S3-compatible Go storage adapter shared by **MinIO** (local Docker Compose) and **Backblaze B2** (staging/production), with production guard blocking local/MinIO endpoints.
  - Two-step direct presigned PUT upload (`pending` → `HeadObject` + magic-byte validation → idempotent `ready`), 25 MB cap, MIME allowlist (PNG, JPEG, WebP, PDF, PPT, PPTX; no raw SVG), 5-min presigned GET, abandoned-upload cleanup, and async malware-scan extension point (ready for NATS JetStream).
  - Begin progressive adoption of confirmed **`shadcn/ui`** components in `web/` (pending `ADR 0004` sign-off).
  - Backblaze B2 setup guide (`docs/STORAGE_BACKBLAZE_B2_GUIDE.md`); **mandatory real B2 staging verification** before production reliance (`OD-11`).

**Exit gate:** Authenticated users access only authorized tenant resources; direct presigned uploads, magic-byte validation, tenant isolation, and CI checks pass; `ADR 0002` security review completed.

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
