# ADR 0005 — Adopt `sriniously/go-boilerplate` with Limen Auth, `pgx` + `tern`, Asynq/Redis, and New Relic

**Status:** Accepted (Explicit Product-Owner Decision — 2026-11-27)  
**Supersedes (in part):** [`ADR 0001`](./0001-approved-stack-and-mvp.md) (data access, migrations, background jobs, observability, monorepo layout)

---

## Context
To accelerate backend production readiness with Clean Architecture (`handlers` → `services` → `repositories` → `infrastructure`), built-in OpenAPI/Swagger, Zerolog, Testcontainers, rate limiting, and security headers, the product owner elected to adopt [`github.com/sriniously/go-boilerplate`](https://github.com/sriniously/go-boilerplate) as the TutorGrid monorepo foundation (`apps/backend`, `apps/frontend`, `packages/*`), while keeping **Limen** (`github.com/thecodearcher/limen`) with Google & GitHub OAuth instead of Clerk.

## Decisions
1. **Authentication:** Replaced boilerplate's Clerk SDK with embedded **Limen** (`github.com/thecodearcher/limen`, `adapters/sql` via `pgx/v5/stdlib`, `plugins/oauth`, `plugins/oauth-google`, `plugins/oauth-github`; no email/password).
2. **Database Access & Migrations:** Substituted **`pgx/v5` (`pgxpool`) + `tern`** in place of **Bun + Goose**. PostgreSQL Row-Level Security (`FORCE ROW LEVEL SECURITY`) and transaction-scoped `set_config('app.current_tenant_id', $1, true)` remain mandatory via `database.RunInTenantTx`.
3. **Background Jobs:** Substituted **Asynq (Redis-backed queue)** in place of **NATS JetStream** for MVP background processing.
4. **Observability:** Substituted **New Relic APM** (+ Zerolog structured logging) in place of **OpenTelemetry + Prometheus/Grafana + Sentry**. Real `TUTORGRID_OBSERVABILITY.NEW_RELIC.LICENSE_KEY` is required before local/staging/prod execution.
5. **Transactional Email:** Adopted boilerplate's **Resend** integration (`TUTORGRID_INTEGRATION.RESEND_API_KEY` required).
6. **Monorepo & Task Runner:** Adopted `apps/backend` + `apps/frontend` + `packages/*` (Turborepo) coordinated via root [`Taskfile.yml`](../../Taskfile.yml) (no `Makefile` or `scripts/`).

## Consequences
- Faster delivery of production-hardened Echo middleware, OpenAPI docs, and Testcontainers integration tests.
- All environment variables remain standardized under the `TUTORGRID_` prefix in gitignored `.env`.
- Product vision, core engines (Lesson Studio, Whiteboard, Game/Activity Engine, Live Classroom), LiveKit (mandatory), Centrifugo, Backblaze B2 / MinIO, and strict tenant isolation boundaries remain unchanged.
