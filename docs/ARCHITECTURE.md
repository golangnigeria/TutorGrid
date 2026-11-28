# TutorGrid MVP Architecture

## Architecture approach
A modular monolith with clear internal domain boundaries, deployed as a small number of workloads. Extract services only when operational or team needs justify it. Keep future extension points, but do not build distributed systems for hypothetical scale.

## Approved technology decisions
- React 19 + TypeScript + Vite 7 + Tailwind CSS v4 + `shadcn/ui` responsive web frontend (`apps/frontend`, dark-only TutorGrid Circle shell; see `docs/ADR/0004-frontend-ui-and-toolchain.md`).
- Go 1.26 + Echo v4 backend (`apps/backend`, Clean Architecture).
- PostgreSQL 16 primary durable database; `pgx/v5` (`pgxpool`, `pgx.StrictNamedArgs` `@snake_case`) access layer; `tern` embedded migrations (per `docs/ADR/0005-go-boilerplate-pgx-tern-asynq-newrelic.md`).
- Embedded Limen auth (`github.com/thecodearcher/limen`) with Google/GitHub OAuth; no email/password in current scope (per `docs/ADR/0002-embedded-limen-oauth-and-rls-tenancy.md`).
- RBAC + resource permissions; tenant-aware backend checks; PostgreSQL Dual RLS (`FORCE ROW LEVEL SECURITY` under non-superuser `tutorgrid_app`) as defense in depth.
- Redis 7 for disposable cache, rate limiting, ephemeral session coordination, and Centrifugo pub/sub.
- **NATS JetStream** for durable async background jobs and events (`TUTORGRID_JOBS` stream; single durable job system per `docs/ADR/0006-nats-jetstream-durable-jobs.md`).
- Centrifugo for application realtime and presence.
- LiveKit (mandatory) — LiveKit Cloud for audio/video; Hetzner Cloud is the future self-hosting candidate, not MVP.
- **Backblaze B2** S3-compatible object storage exclusively across dev, test/CI, staging, and production (no MinIO; separate private dev and test buckets; per `docs/ADR/0003-direct-s3-b2-presigned-storage.md`).
- Transactional email via Resend (`resend-go/v2`) with PostgreSQL transactional outbox.
- REST/OpenAPI public interface; gRPC only for a justified internal boundary.
- Northflank initial deployment candidate, Cloudflare DNS/CDN, Docker, GitHub Actions, GHCR, Terraform.
- Observability via New Relic APM + Zerolog structured logging (per `ADR 0005`).

## Logical modules
1. Identity & Tenancy: profile, organization membership, active workspace selection, invitations, tenant context.
2. Authorization: role/resource policy, access decisions, classroom/media permissions.
3. Lesson Studio: lessons, drafts, sections, content blocks, publishing/versioning.
4. Resource Library: file metadata, signed B2 upload/download, deep content verification, partial-failure reconciliation.
5. Whiteboard: document schema, pages, objects, commands, operations, snapshots.
6. Activities: definitions, template configuration, runtime contracts, attempts/results.
7. Classroom: session lifecycle, participant access, lesson navigation, permissions.
8. Realtime: Centrifugo integration, channel authorization, event delivery.
9. Media: LiveKit room/token/permission adapter.
10. Jobs: NATS JetStream durable streams, idempotent pull consumers, and bounded reconciliation.
11. Audit/Observability: security audit events, New Relic metrics/traces, Zerolog structured logs.

## Dependency direction
- Domain modules own business rules and interfaces.
- HTTP handlers are adapters; they call application/domain services.
- Domain modules do not depend on Echo, Redis, NATS, Centrifugo, or LiveKit SDKs directly.
- Provider-specific implementations sit behind interfaces/adapters (`ObjectStore`, `JobPublisher`, `MalwareScanEnqueuer`, etc.).
- Avoid circular module dependencies. Shared types must be genuinely stable domain concepts, not a dumping-ground `common` package.

## MVP deployment shape
- One frontend deployment (`apps/frontend`).
- One Go API + worker modular monolith (`apps/backend`).
- One small NATS JetStream deployment (`-js`) for durable background jobs and events.
- Managed PostgreSQL 16 and Redis 7.
- Backblaze B2 private buckets (separate dev, test, staging, production buckets).
- Centrifugo deployment sized for pilot usage.
- LiveKit Cloud.
- New Relic APM + structured logging and backups.
No Kubernetes, service mesh, multi-region, or database sharding in MVP.

## Data and event rules
- PostgreSQL is the durable source of truth for all product records.
- Redis is disposable/temporary; never make Redis the sole durable home of jobs, learner results, or lesson data.
- NATS JetStream jobs/events use stable `event_id`, `schema_version`, `tenant_id`, `correlation_id`, and `timestamp` envelopes.
- Use transactional outboxes or durable lifecycle states (`pending`, `deleting`) for database changes that trigger async side effects, paired with bounded periodic reconciliation for publish-gap recovery.
- Consumers must be idempotent; retries are bounded and failed jobs are observable.
- Large files, video, board snapshots, and bulky payloads belong in Backblaze B2/PostgreSQL; NATS carries references and small event envelopes only (never raw secrets).
- Centrifugo is client delivery/presence, not durable business authority.

## Multi-tenancy and security
- Every tenant-owned row/resource has tenant ownership verified in the backend service layer.
- PostgreSQL RLS (`ENABLE` + `FORCE ROW LEVEL SECURITY`) provides defense in depth under `tutorgrid_app` (`NOSUPERUSER NOBYPASSRLS`) via transaction-local `set_config(..., true)`.
- Any cross-tenant reconciliation discovery uses a tightly constrained read-only `SECURITY DEFINER` pointer function owned by a dedicated `NOLOGIN NOSUPERUSER NOBYPASSRLS` role returning only `(tenant_id, resource_id, status)`; all actual data reads and mutations execute inside `RunInTenantTx`.
- Never trust client-supplied tenant ID, role, ownership, object key, or file size.
- Signed B2 URLs are short-lived (5 min), scoped to one object/action, and downloads enforce `Content-Disposition: attachment` with zero inline browser rendering of unscanned files.
- Secrets are never committed or printed. Use local gitignored `.env` and platform secret storage.
- Audit privileged changes, membership changes, permission grants/revocations, publishing, resource lifecycle, and classroom lifecycle.

## Whiteboard strategy
- MVP uses one renderer, selected after a short technical spike and licensing review.
- Maintain TutorGrid-owned schema and adapter boundary from day one.
- Do not implement both Konva and tldraw adapters in MVP.
- Prior discovery selected a hybrid long-term option: open-source Konva/React Konva and paid tldraw SDK. This is a strategic option, not a requirement to ship both now.
- Before choosing commercial SDK, confirm license covers multi-tenant SaaS, embedding, redistribution, and white-label.
- CRDT-inspired collaboration principles and server-authoritative permissions are architectural direction; exact CRDT library is not yet frozen.

## LiveKit strategy
- LiveKit is mandatory for WebRTC media; LiveKit Cloud first, Hetzner Cloud self-hosting is a later cost/operations decision.
- Go backend controls room creation, token issuance, participant authorization, and permission changes.
- **Recommended (Pending Approval — Master Blueprint §8.6):** Learners start media disabled; teacher grants/revokes media permissions.
- Centrifugo carries classroom app events; LiveKit carries media.

## Scaling triggers
Do not scale by user-registration count alone. Measure active concurrency, classroom participants, websocket connections, media minutes, event rate, database load, storage, and error/latency SLOs.
Possible future actions only after evidence: increase managed DB/cache size, add worker replicas, isolate workloads, add read replicas, deploy NATS cluster, self-host LiveKit, extract service, or adopt Kubernetes.
