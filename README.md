# TutorGrid Circle

**TutorGrid Circle** is a multi-tenant teaching and learning creation-and-delivery platform for **Pre-K through Grade 12** educators, tutoring organizations, and learners.

Its core product is built around four first-class engines:
- **Lesson Studio** — Structured, versioned lesson authoring with immutable published versions and attached reference materials.
- **Whiteboard Engine** — Collaborative teaching canvas with isolated board surfaces inside a calm dark studio chrome.
- **Educational Activity & Game Engine** — Declarative, sandboxed interactive learning activities (zero arbitrary user-uploaded code execution).
- **Live Classroom** — Real-time audio/video/screen-sharing sessions pinned to published lesson versions without mutating source lessons.

---

## Current Implementation Status

| Phase / Slice | Scope | Status |
| :--- | :--- | :--- |
| **Slice 1** | Modular monolith foundation (`go-boilerplate` + `pgx/v5` + `tern`), `Taskfile.yml`, PostgreSQL 16 + Redis 7 Docker dev stack, non-superuser `tutorgrid_app` (`NOBYPASSRLS`) RLS concurrency harness | Completed |
| **Slice 2 / 2b** | Embedded **Limen** Google & GitHub OAuth (no email/password), PostgreSQL **Dual RLS** (zero `SECURITY DEFINER`), server-bound active workspace switching (`user_active_tenants`), backend RBAC (`platform_admin`, `org_admin`, `teacher`, `learner`), atomic invitations with **Transactional Outbox + Asynq + Resend**, in-app pending invitation discovery, and TutorGrid Circle dark-only React shell (`/`, `/dashboard`, `/sign-in`, `/sign-out`, `/invite`) | Completed |
| **Slice 3** | **NATS JetStream** single durable job/event backbone (`ADR 0006`, replacing Asynq) + **Backblaze B2-only Resource Library** (`ADR 0003`, no MinIO): 2-step presigned PUT, untrusted-size + magic-byte + OOXML PPTX validation, `Content-Disposition: attachment` downloads (zero inline rendering), constrained `SECURITY DEFINER` reconciler, and `/resources` dark-only UI | Completed |
| **Slice 4–7** | Lesson Studio, Whiteboard Engine, Live Classroom (LiveKit Cloud + Centrifugo), Activity/Game Engine | Planned ([`docs/ROADMAP.md`](docs/ROADMAP.md)) |

---

## Technology Stack

Approved stack (updated by [`docs/ADR/0005-go-boilerplate-pgx-tern-asynq-newrelic.md`](docs/ADR/0005-go-boilerplate-pgx-tern-asynq-newrelic.md)):

| Layer | Technology |
| :--- | :--- |
| **Backend API** | Go `1.26` + Echo v4 (`apps/backend`), Clean Architecture (`handler` → `service` → `repository` → `database`) |
| **Database & Migrations** | PostgreSQL 16, `pgx/v5` (`pgxpool`, `pgx.StrictNamedArgs` `@snake_case`), `tern` embedded SQL migrations |
| **Authentication** | Embedded **Limen** (`github.com/thecodearcher/limen`) — Google & GitHub OAuth only |
| **Authorization & Isolation** | Backend RBAC + per-table PostgreSQL `FORCE ROW LEVEL SECURITY` under non-superuser `tutorgrid_app` |
| **Cache & Ephemeral State** | Redis 7 (cache, rate limiting, ephemeral coordination, future Centrifugo broker) |
| **Durable Background Jobs & Events** | **NATS JetStream** (`github.com/nats-io/nats.go`, `ADR 0006`) |
| **Object Storage** | **Backblaze B2** exclusively via S3-compatible API (`aws-sdk-go-v2/service/s3`, `ADR 0003`; no MinIO) |
| **Transactional Email** | **Resend** (`resend-go/v2`) via PostgreSQL transactional outbox (`invitation_email_outbox`) |
| **Observability** | **New Relic APM** (`go-agent/v3`, `nrecho-v4`, `nrpgx5`, `nrredis-v9`) + **Zerolog** structured logging |
| **Frontend** | React 19 + TypeScript + Vite 7 + Tailwind CSS v4 + `shadcn/ui` + `motion/react` (`apps/frontend`, Bun + Turborepo) |
| **Task Runner & CI** | `Taskfile.yml` (`go-task`), GitHub Actions CI (`go test -race`, migration rollback, `govulncheck`, Bun checks, Gitleaks), Dependabot |

---

## Repository Structure

```text
TutorGrid/
├── apps/
│   ├── backend/
│   │   ├── cmd/
│   │   │   ├── tutorgrid/          # Main HTTP API + Asynq worker entrypoint
│   │   │   └── migrate/            # Embedded tern migration CLI (supports -to <version>)
│   │   ├── internal/
│   │   │   ├── config/             # Koanf + validator typed config
│   │   │   ├── database/           # pgxpool, Dual RLS tx helpers, tern migrations (001–004)
│   │   │   ├── handler/            # Echo HTTP handlers + real Postgres integration tests
│   │   │   ├── lib/                # Resend email client, Asynq jobs & handlers
│   │   │   ├── logger/             # Zerolog console/JSON + New Relic log forwarding
│   │   │   ├── middleware/         # RequireAuth, RequireTenant, RequirePermission, CORS, Secure
│   │   │   ├── model/              # Domain types & RBAC permission matrix
│   │   │   ├── repository/         # pgx.StrictNamedArgs (@snake_case) repositories
│   │   │   ├── router/             # Versioned REST routes + Limen /auth/* mount
│   │   │   ├── server/             # Echo server, Redis, Asynq, graceful shutdown
│   │   │   └── service/            # Limen auth & tenancy business logic
│   │   └── templates/emails/       # Branded HTML email templates (Resend)
│   └── frontend/                   # React 19 + Vite 7 + Tailwind v4 + shadcn/ui dark-only app
├── deployments/
│   ├── docker-compose.dev.yml      # Local PostgreSQL 16 + Redis 7
│   └── postgres/init-roles.sql     # Creates non-superuser tutorgrid_app (NOBYPASSRLS) & tutorgrid_test
├── docs/                           # Master Blueprint, Constitution, Architecture, Roadmap & ADRs
├── packages/                       # Shared monorepo packages (openapi, zod, emails)
├── .github/
│   ├── workflows/ci.yml            # Backend (RLS + rollback + govulncheck), Frontend, Gitleaks
│   └── dependabot.yml              # Weekly dependency updates (Go, npm, GitHub Actions)
├── AGENTS.md                       # Engineering constitution & conventions
├── Taskfile.yml                    # Canonical local dev, migration, build & test commands
└── .env.example                    # Environment variable template (no real secrets)
```

---

## Security & Multi-Tenancy Architecture

1. **Zero Client-Side Tenant Trust:** The active workspace is stored server-side in `user_active_tenants`. `RequireTenant` atomically joins `user_active_tenants` with `tenant_memberships` inside a PostgreSQL transaction on every tenant-scoped request; revoked or deleted memberships immediately fail closed (`428 Precondition Required`).
2. **PostgreSQL Dual RLS (Zero `SECURITY DEFINER`):**
   - Runtime queries run as the non-superuser, non-owner `tutorgrid_app` role (`NOSUPERUSER NOBYPASSRLS`).
   - Every tenant-bearing table enforces `ENABLE ROW LEVEL SECURITY` and `FORCE ROW LEVEL SECURITY` with per-operation policies scoped via transaction-local `set_config(..., true)`.
3. **Atomic Invitations & Transactional Outbox:**
   - Creating an invitation writes `tenant_invitations` (SHA-256 token hash only) and `invitation_email_outbox` in one `pgx.Tx`.
   - Raw invitation tokens are never returned in HTTP JSON responses or logged.
   - Asynq delivery payloads carry only `(tenant_id, outbox_id)`; the worker clears `invite_url` immediately upon successful Resend delivery.
   - Acceptance uses `SELECT ... FOR UPDATE`, enforces expiry, revocation, single-use, and case-insensitive verified-email matching inside one transaction.

---

## Local Development Quick Start

### Prerequisites

- **Go** `1.26+`
- **Bun** `1.3+`
- **Docker Desktop** (for PostgreSQL 16 and Redis 7)
- **go-task (`task`)** `3.x`

### 1. Clone & Install Dependencies

```bash
git clone https://github.com/golangnigeria/TutorGrid.git
cd TutorGrid
bun install
```

### 2. Configure Environment Variables

Copy the template to a local gitignored `.env` at the repository root:

```bash
cp .env.example .env
```

Populate the required real values in `.env` (never commit `.env`):

| Variable | Description |
| :--- | :--- |
| `TUTORGRID_LIMEN_SECRET` | 32-byte (64-char hex) secret for Limen session signing |
| `TUTORGRID_GOOGLE_CLIENT_ID` / `TUTORGRID_GOOGLE_CLIENT_SECRET` | Google OAuth credentials (callback: `http://localhost:8080/auth/oauth/google/callback`) |
| `TUTORGRID_GITHUB_CLIENT_ID` / `TUTORGRID_GITHUB_CLIENT_SECRET` | GitHub OAuth credentials (callback: `http://localhost:8080/auth/oauth/github/callback`) |
| `TUTORGRID_INTEGRATION.RESEND_API_KEY` | Resend API key for transactional invitation emails |
| `TUTORGRID_INTEGRATION.FROM_EMAIL` / `FROM_NAME` | Verified Resend sender address and display name |
| `TUTORGRID_OBSERVABILITY.NEW_RELIC.LICENSE_KEY` | New Relic APM license key |
| `TUTORGRID_DATABASE_MIGRATION_URL` | Migrator DSN (`tutorgrid_migrator`) for `task migrate:up` |
| `TUTORGRID_TEST_DATABASE_MIGRATION_URL` / `TUTORGRID_TEST_DATABASE_URL` | Test DB DSNs (`tutorgrid_test`) for `task test:go` |

### 3. Start Everything

```bash
task dev
```

This starts PostgreSQL 16 and Redis 7, applies pending `tern` migrations, and launches both services:
- **Backend API:** `http://localhost:8080` (OpenAPI UI at `/docs`, health at `/status`)
- **Frontend App:** `http://localhost:5173`

---

## Common Commands

All workflow commands live in [`Taskfile.yml`](Taskfile.yml):

| Command | Description |
| :--- | :--- |
| `task dev` | Start Postgres + Redis, run migrations, and launch API + frontend |
| `task dev:api` | Run only the Go backend (`apps/backend`) |
| `task dev:web` | Run only the React frontend (`apps/frontend`) |
| `task db:up` / `task db:down` | Start or stop local PostgreSQL 16 and Redis 7 containers |
| `task migrate:up` | Apply pending `tern` migrations to latest version |
| `task vet` | Run `go vet ./...` in `apps/backend` |
| `task build:api` | Compile `cmd/tutorgrid` and `cmd/migrate` binaries |
| `task test:go` | Run all Go unit and real PostgreSQL RLS integration tests (`-p 1`) |
| `task tidy` | Run `go mod tidy` and `go mod verify` |

**Migration rollback (example):**
```bash
cd apps/backend
go run ./cmd/migrate -to 2
```

**Frontend verification:**
```bash
cd apps/frontend
bun run typecheck
bun run lint
bun run build
```

---

## Implemented API Endpoints

| Method & Path | Auth / Scope | Purpose |
| :--- | :--- | :--- |
| `GET /status` | Public | Health check (PostgreSQL + Redis readiness) |
| `GET /docs` | Public | OpenAPI documentation UI |
| `ANY /auth/*` | Public | Embedded Limen Google & GitHub OAuth and session endpoints |
| `GET /api/v1/me` | `RequireAuth` | Verified user profile, memberships, and server-bound `active_tenant_id` |
| `GET /api/v1/me/invitations` | `RequireAuth` | Pending invitations addressed to the user's verified OAuth email |
| `POST /api/v1/me/invitations/:id/accept` | `RequireAuth` | Atomically accept a pending invitation by ID |
| `POST /api/v1/auth/logout` | `RequireAuth` | Revoke active Limen session |
| `POST /api/v1/tenants` | `RequireAuth` | Create a new Teaching Space and assign creator as `org_admin` |
| `POST /api/v1/session/active-tenant` | `RequireAuth` | Atomically verify membership and switch active Teaching Space |
| `POST /api/v1/invitations/accept` | `RequireAuth` | Atomically accept an invitation via email token |
| `GET /api/v1/tenants/current/members` | `RequireTenant` + `tenant:members:read` | List members of the active Teaching Space under Dual RLS |
| `GET /api/v1/tenants/current/invitations` | `RequireTenant` + `tenant:members:invite` | List pending invitations + outbox email delivery status |
| `POST /api/v1/tenants/current/invitations` | `RequireTenant` + `tenant:members:invite` | Create invitation + enqueue Resend email via transactional outbox |
| `DELETE /api/v1/tenants/current/invitations/:id` | `RequireTenant` + `tenant:invitations:revoke` | Revoke a pending invitation |

---

## CI & Quality Gates

Every push and pull request runs three checks in [`.github/workflows/ci.yml`](.github/workflows/ci.yml):
1. **`Backend (Lint, Migrations & RLS Tests)`** — `go vet`, `go build`, migration `up → -to 2 → up` rollback verification, `go test -p 1 -race` against PostgreSQL 16 + Redis 7 under `tutorgrid_app` (`NOBYPASSRLS`), and `govulncheck`.
2. **`Frontend (Typecheck, Lint & Build)`** — `bun install --frozen-lockfile`, `tsc`, `eslint`, and `vite build`.
3. **`Security (Secret Scan)`** — Gitleaks full-history secret scan.

---

## Documentation & ADRs

- [`AGENTS.md`](AGENTS.md) — Project constitution, approved stack, and coding conventions
- [`docs/MASTER_BLUEPRINT.md`](docs/MASTER_BLUEPRINT.md) — Long-term product and technical blueprint
- [`docs/PRODUCT_CONSTITUTION.md`](docs/PRODUCT_CONSTITUTION.md) — Non-negotiable product principles
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — System architecture and module boundaries
- [`docs/TECHNOLOGY_DECISIONS.md`](docs/TECHNOLOGY_DECISIONS.md) — Approved stack and open decision register
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — Sequenced MVP to first 1,000 users roadmap
- [`docs/ADR/`](docs/ADR/) — Architecture Decision Records (`0001`–`0005`)

---

## License

See [`LICENSE`](LICENSE) for license details.
