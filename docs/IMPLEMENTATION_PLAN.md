# TutorGrid MVP Implementation Plan

**Status:** Approved Direction (with Corrections Applied) — Awaiting Approval for Slice 1  
**Authoritative References:** [`AGENTS.md`](../AGENTS.md), [`docs/PRODUCT_CONSTITUTION.md`](./PRODUCT_CONSTITUTION.md), [`docs/MVP_SCOPE.md`](./MVP_SCOPE.md), [`docs/PRODUCT_SPEC.md`](./PRODUCT_SPEC.md), [`docs/ARCHITECTURE.md`](./ARCHITECTURE.md), [`docs/ROADMAP.md`](./ROADMAP.md), [`docs/ADR/0001-approved-stack-and-mvp.md`](./ADR/0001-approved-stack-and-mvp.md)

---

## 1. Non-Negotiable Scope & Stack Guardrails

- **Product Focus:** Lesson Studio, Whiteboard Engine, Educational Game/Activity Engine, and Live Classroom. Strictly **no** LMS, SIS, CRM, payroll, admissions, or school administration features.
- **Authoritative Approved Stack (ADR 0001):**
  - Backend: Go + Echo (modular monolith)
  - Data: PostgreSQL + Bun (data access) + Goose (migrations)
  - Auth: Limen (Google & GitHub OAuth; no email/password)
  - Authorization: RBAC + resource-level permissions; backend checks + PostgreSQL RLS (transaction-scoped)
  - Cache/Ephemeral State: Redis
  - Application Realtime: Centrifugo
  - Media: LiveKit Cloud (no self-hosting in MVP)
  - Object Storage: Backblaze B2 (S3-compatible abstraction)
  - Jobs/Events: NATS JetStream (small, non-clustered MVP deployment)
  - Frontend: React + TypeScript + Vite (responsive web)
  - Delivery/Ops: Northflank candidate, Docker, GitHub Actions, GHCR, Terraform, OpenTelemetry, Prometheus/Grafana, Sentry
- **Proportional Infrastructure (First 1,000 Users):** No Kubernetes, multi-region deployments, microservices, Redis Cluster, NATS clustering, or LiveKit self-hosting in MVP unless a demonstrated, measured requirement justifies an approved ADR.
- **Tenant Context Rule:** Never trust a client-supplied tenant ID header as authorization. Tenant context must be derived from a verified identity and authorized membership. Development-only test identity/context must be strictly isolated from production code paths.

---

## 2. Open & Recommended Decisions Register

None of the items below are confirmed. Each remains **Open** or **Recommended** until explicitly approved by the product owner (and recorded in an ADR where applicable).

| ID | Topic | Status | Recommendation / Notes | Required Before |
|---|---|---|---|---|
| **OD-01** | **Whiteboard Renderer** | **Open** | Define TutorGrid-owned whiteboard document schema first, then submit a renderer comparison (e.g., Konva/React-Konva vs. commercial SDK with SaaS/embedding/white-label license review) and recommend one MVP renderer. Do not build both renderers in MVP. | Milestone 3 (Whiteboard) |
| **OD-02** | **Guest Learner Access Flow** | **Recommended (Pending Approval)** | Short-lived, backend-signed room invitation tokens for guest learners without requiring Google/GitHub OAuth, alongside optional persistent learner accounts. | Milestone 4 (Classroom) |
| **OD-03** | **Learner Media Permissions & Room Modes** | **Open / Recommended** | Baseline from `MVP_SCOPE.md`: learners join with media disabled; teacher grants/revokes mic/camera/screen share server-side. Exact room size tiers (e.g., small group vs. lecture/raise-hand promotion) remain Open pending approval. | Milestone 4 (Classroom) |
| **OD-04** | **Math / Equation Renderer** | **Recommended (Pending Approval)** | KaTeX inline/block support if it fits the chosen MVP rich-text editor component. | Milestone 2 (Lesson Studio) |
| **OD-05** | **Pilot Concurrency Targets** | **Open** | Define concrete peak concurrent classrooms, participants per room, whiteboard event rate, and media minutes using pilot data before load testing. | Milestone 6 (Pilot Hardening) |
| **OD-06** | **Centrifugo Channel Topology & Auth** | **Recommended (Pending Approval)** | Backend-signed Centrifugo connection/subscription JWTs scoped to authorized classroom/board channels. | Milestone 3 (Whiteboard) |
| **OD-07** | **MVP Starter Activity Templates** | **Open** | Finalize exact starter template count and types (candidates from `MVP_SCOPE.md`: quiz, matching, sorting, fill-in-the-blank). | Milestone 5 (Activity Engine) |

---

## 3. Proposed Repository Structure (MVP Modular Monolith)

Services and folders are added progressively only when their vertical slice is approved.

```text
TutorGrid/
├── .agents/                        # Antigravity rules and workflows
├── .github/workflows/              # CI pipelines (Go lint/test/build, React lint/typecheck/build)
├── api/openapi/v1/                 # OpenAPI REST contracts
├── cmd/
│   ├── api/                        # Go + Echo modular monolith HTTP server
│   └── worker/                     # NATS JetStream async worker (added when needed)
├── deployments/
│   └── docker-compose.dev.yml      # Local backing services (added proportionally per slice)
├── docs/                           # Constitution, specs, ADRs, implementation plan
├── internal/
│   ├── identity/                   # Users, profiles, Limen OAuth integration (Milestone 1)
│   ├── tenancy/                    # Tenants/organizations, memberships, tenant context
│   ├── authz/                      # RBAC + resource-level policies
│   ├── lessons/                    # Lesson Studio: drafts, sections, blocks, immutable versions
│   ├── resources/                  # Backblaze B2 metadata & signed URLs
│   ├── whiteboard/                 # TutorGrid board schema, pages, objects, snapshots
│   ├── activities/                 # Activity runtime contract, templates, attempts/results
│   ├── classroom/                  # Session lifecycle, pinned lesson version, participant state
│   ├── realtime/                   # Centrifugo adapter & channel authorization
│   ├── media/                      # LiveKit Cloud room/token/permission adapter
│   ├── audit/                      # Security-sensitive audit logging
│   └── platform/                   # Shared infrastructure adapters (no domain logic)
│       ├── config/                 # Typed config loading & validation
│       ├── database/               # PostgreSQL + Bun pool, RLS transaction helper, Goose runner
│       └── httputil/               # Consistent error envelopes, request IDs, health handlers
├── migrations/                     # Goose SQL migrations (append-only)
├── web/                            # React + TypeScript + Vite responsive web app
├── go.mod
├── go.sum
└── Taskfile.yml
```

---

## 4. Milestone Plan (First 1,000 Users)

1. **Milestone 0 / Slice 1 — Repository Foundation + Database + Tenant-Security Test Harness** *(Awaiting Approval)*
   - Go + Echo skeleton, React + Vite skeleton, local PostgreSQL (`docker-compose.dev.yml`), Goose migrations, Bun pool with transaction-scoped RLS helper, non-superuser DB app role, and pooled PostgreSQL integration tests proving zero cross-tenant leakage.
2. **Milestone 1 — Identity (Limen), Tenant Membership & B2 Storage**
   - Limen Google/GitHub OAuth integration, verified user/membership lookup deriving tenant context server-side, RBAC policy baseline, Backblaze B2 signed URL adapter, Redis integration.
3. **Milestone 2 — Lesson Studio Vertical Slice**
   - Lesson CRUD, metadata, ordered sections/content blocks, PDF/PPT reference attachments (no auto-conversion), draft autosave, preview, and immutable `lesson_versions` publishing.
4. **Milestone 3 — Whiteboard Engine Vertical Slice**
   - **Mandatory Pre-Slice Gate:** Define TutorGrid-owned whiteboard document schema + submit renderer comparison + receive approval on one MVP renderer.
   - Implement chosen single renderer adapter, basic tools, Centrifugo realtime sync, reconnect recovery, and separate live session copy from prepared board.
5. **Milestone 4 — Live Classroom + LiveKit Cloud Vertical Slice**
   - Classroom session pinned to published `lesson_version_id`, authorized join flow (per approved OD-02/OD-03), LiveKit Cloud room/token/permission enforcement, guided/free navigation, persisted participation records.
6. **Milestone 5 — Educational Activity/Game Engine Vertical Slice**
   - Declarative activity runtime contract (no arbitrary code execution), approved starter templates (OD-07), teacher configuration/preview, classroom launch, server-validated idempotent result persistence.
7. **Milestone 6 — Pilot Hardening**
   - Usability sessions, responsive device checks, security/tenant-isolation review, load testing against explicit pilot concurrency profile (OD-05), backup/restore rehearsal, OpenTelemetry/Prometheus/Grafana/Sentry alerts.
8. **Milestone 7 — First 1,000 Users Cohort Rollout**
   - Gradual cohort onboarding, tracking activation, classroom reliability, reconnect rate, media minutes, and cost per active classroom.

---

## 5. Slice 1 Plan: Repository Foundation + Database + Tenant-Security Test Harness

*(Do not implement until explicitly approved.)*

### Scope
Establish the reproducible repository foundation for the Go/Echo/Bun/Goose backend and React/Vite frontend, stand up local PostgreSQL, create the foundational tenant/user/membership schema with PostgreSQL Row-Level Security (RLS), and prove cross-tenant isolation using real PostgreSQL integration tests over pooled connections.

**Explicitly NOT in Slice 1:**
- No Limen OAuth authentication (this slice is **not** authenticated).
- No client-supplied `X-Tenant-ID` HTTP header middleware.
- No Redis, Centrifugo, NATS, LiveKit, or B2 services yet (kept proportional to what Slice 1 exercises).

### Affected Files (All New)
- `go.mod`, `go.sum` — Go module with Echo v4, Bun (`uptrace/bun`, `bun/dialect/pgdialect`, `bun/driver/pgdriver`), Goose (`pressly/goose/v3`).
- `deployments/docker-compose.dev.yml` — Local PostgreSQL 16 service with init script creating a dedicated non-superuser application role (`tutorgrid_app`) so PostgreSQL RLS is genuinely enforced (superusers bypass RLS).
- `deployments/postgres/init-roles.sql` — Creates `tutorgrid_app` (non-superuser, `NOBYPASSRLS`) for runtime and integration tests; `tutorgrid_migrator` for Goose migrations.
- `migrations/00001_init_tenancy_and_rls.sql` — Goose migration creating:
  - `tenants` (`id`, `name`, `slug`, `created_at`, `updated_at`)
  - `users` (`id`, `external_subject`, `display_name`, `email`, `created_at`, `updated_at`)
  - `tenant_memberships` (`id`, `tenant_id`, `user_id`, `role`, `created_at`, `updated_at`)
  - `tenant_scoped_records` (or minimal `tenant_settings` / `audit_probe` table or `tenant_memberships` itself) with `ENABLE ROW LEVEL SECURITY` and `FORCE ROW LEVEL SECURITY` using `current_setting('app.current_tenant_id', true)::uuid`.
- `internal/platform/config/config.go` — Environment-based configuration with validation (DB URLs, port, environment mode).
- `internal/platform/database/db.go` — Bun DB connection pool setup (`pgdriver`, bounded `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`).
- `internal/platform/database/rls.go` — Transaction-scoped tenant context helper (`RunInTenantTx(ctx, db, tenantCtx, fn)`):
  - Requires a verified `tenancy.Context` value (not raw strings from HTTP headers).
  - Begins a `bun.Tx`, executes `SELECT set_config('app.current_tenant_id', ?, true)` (`is_local = true`, transaction-scoped), runs `fn(ctx, tx)`, and commits/rolls back cleanly so pooled connections never retain tenant state.
- `internal/platform/database/migrate.go` — Embedded Goose migration runner (`Up`, `Down`, `Status`).
- `internal/tenancy/context.go` — Domain `Context` type (`TenantID`, `UserID`, `Role`) with an unexported construction boundary so production code can only build it via verified membership lookup (plus an explicit `testonly` helper inside `*_test.go` / `internal/tenancy/tenancytest` isolated from production binaries).
- `internal/platform/database/rls_integration_test.go` — Real PostgreSQL integration test suite (run against `tutorgrid_app` non-superuser role with a pooled connection limit):
  1. Verifies migrations apply cleanly up and down.
  2. Seeds Tenant A and Tenant B data.
  3. Proves Tenant A transaction sees only Tenant A rows and cannot read, update, or delete Tenant B rows.
  4. Proves a query without `app.current_tenant_id` set returns zero rows (fail-closed default).
  5. Proves pooled connection safety under concurrency: runs parallel goroutines alternating Tenant A and Tenant B transactions across a small connection pool (e.g., `MaxOpenConns = 2`) and verifies zero cross-tenant leakage.
- `cmd/api/main.go` — Minimal Echo server with structured logging, graceful shutdown, and unauthenticated operational probes: `GET /healthz` (process liveness) and `GET /readyz` (PostgreSQL connectivity ping). No tenant-protected endpoints yet.
- `web/` — Minimal React + TypeScript + Vite responsive shell (`package.json`, `vite.config.ts`, `tsconfig.json`, `src/App.tsx`, lint/typecheck/build scripts).
- `.github/workflows/ci.yml` — CI running Go vet/test/build, migration check, and frontend lint/typecheck/build.
- `Makefile` — Standard commands (`make db-up`, `make migrate-up`, `make test-unit`, `make test-integration`, `make web-build`).

### API & Data Changes
- **HTTP Endpoints:**
  - `GET /healthz` → `200 OK` `{ "status": "ok" }`
  - `GET /readyz` → `200 OK` `{ "status": "ready", "database": "ok" }` (or `503 Service Unavailable` if DB unreachable)
- **Database Schema:** Initial `00001_init_tenancy_and_rls.sql` migration with `tenants`, `users`, `tenant_memberships`, and RLS policies enforced for `tutorgrid_app`.

### Security Implications
- **Fail-Closed RLS:** `current_setting('app.current_tenant_id', true)` returns `NULL`/empty when unset; RLS policies compare `tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid`, which evaluates to false when unset, guaranteeing fail-closed isolation.
- **Non-Superuser Enforcement:** Application and integration tests connect as `tutorgrid_app` (`NOSUPERUSER NOBYPASSRLS`) so PostgreSQL RLS cannot be accidentally bypassed during testing or runtime.
- **Zero Header Spoofing:** No HTTP middleware accepts a tenant ID from request headers. Test tenant contexts exist strictly in test-only helpers (`internal/tenancy/tenancytest`), never imported by `cmd/api`.

### Verification Plan & Acceptance Criteria
- [ ] Local PostgreSQL starts via `docker compose -f deployments/docker-compose.dev.yml up -d` with `tutorgrid_app` (`NOBYPASSRLS`) initialized.
- [ ] Goose migration `00001` applies cleanly up and rolls back cleanly down on an isolated test DB.
- [ ] `go test -race ./...` passes unit tests and real PostgreSQL RLS integration tests (single-tenant isolation, fail-closed unset context, cross-tenant update/delete rejection, and concurrent pooled connection isolation with `MaxOpenConns=2`).
- [ ] `go vet ./...` and `go build ./cmd/api` succeed cleanly.
- [ ] `GET /healthz` and `GET /readyz` return expected status codes against real PostgreSQL.
- [ ] `web/` passes `npm run lint`, `npx tsc --noEmit`, and `npm run build`.
- [ ] No unrelated files, unapproved dependencies, or speculative infrastructure added.
