# TutorGrid MVP Architecture

## Architecture approach
A modular monolith with clear internal domain boundaries, deployed as a small number of workloads. Extract services only when operational or team needs justify it. Keep future extension points, but do not build distributed systems for hypothetical scale.

## Approved technology decisions
- React + Vite responsive web frontend.
- Go + Echo backend.
- PostgreSQL primary durable database; Bun access layer; Goose migrations.
- Limen auth with Google/GitHub OAuth; no email/password in current scope.
- RBAC + resource permissions; tenant-aware checks; PostgreSQL RLS defense in depth.
- Redis for cache and ephemeral session state.
- Centrifugo for application realtime and presence.
- LiveKit Cloud for audio/video; no self-hosted media in MVP.
- Backblaze B2 S3-compatible storage.
- NATS JetStream for durable async jobs/events, small deployment.
- REST/OpenAPI public interface; gRPC only for a justified internal boundary.
- Northflank initial deployment candidate, Docker, GitHub Actions, GHCR, Terraform.
- OpenTelemetry + Prometheus/Grafana + Sentry.

## Logical modules
1. Identity & Tenancy: profile, organization membership, invitations, tenant context.
2. Authorization: role/resource policy, access decisions, classroom/media permissions.
3. Lesson Studio: lessons, drafts, sections, content blocks, publishing/versioning.
4. Resource Library: file metadata, signed upload/download, B2 references.
5. Whiteboard: document schema, pages, objects, commands, operations, snapshots.
6. Activities: definitions, template configuration, runtime contracts, attempts/results.
7. Classroom: session lifecycle, participant access, lesson navigation, permissions.
8. Realtime: Centrifugo integration, channel authorization, event delivery.
9. Media: LiveKit room/token/permission adapter.
10. Jobs: NATS JetStream workers and idempotent processing.
11. Audit/Observability: security audit events, metrics, traces, logs.

## Dependency direction
- Domain modules own business rules and interfaces.
- HTTP handlers are adapters; they call application/domain services.
- Domain modules do not depend on Echo, Redis, Centrifugo or LiveKit SDKs directly.
- Provider-specific implementations sit behind interfaces/adapters.
- Avoid circular module dependencies. Shared types must be genuinely stable domain concepts, not a dumping-ground `common` package.

## MVP deployment shape
- One frontend deployment.
- One Go API application (modular monolith).
- One small NATS JetStream deployment for required background work.
- Managed PostgreSQL and Redis.
- Backblaze B2 object storage.
- Centrifugo deployment sized for pilot usage.
- LiveKit Cloud.
- Basic monitoring/error reporting and backups.
No Kubernetes, service mesh, multi-region or database sharding in MVP.

## Data and event rules
- PostgreSQL is source of truth for durable product records.
- Redis is disposable/temporary except for explicitly designed short-lived coordination; do not make Redis the only durable home of learner results or lesson data.
- NATS events/jobs use stable event IDs, schema version, tenant ID, correlation/causation IDs and timestamps.
- Use transactional outbox for database changes that must publish events.
- Consumers must be idempotent; retries are bounded and failed jobs are observable.
- Large files, video, board snapshots and bulky payloads belong in B2/Postgres as appropriate; NATS carries references and small event payloads.
- Centrifugo is client delivery/presence, not durable business authority.

## Multi-tenancy and security
- Every tenant-owned row/resource has tenant ownership and is checked in backend service layer.
- RLS provides defense in depth; set tenant context transaction-safely for pooled connections.
- Never trust tenant ID, role, classroom ID or permissions supplied by client without verifying ownership.
- Signed B2 URLs are short-lived and scoped to one object/action.
- Secrets are never committed or printed. Use platform secret storage.
- Audit privileged changes, membership changes, permission grants/revocations, publishing and classroom lifecycle.

## Whiteboard strategy
- MVP uses one renderer, selected after a short technical spike and licensing review.
- Maintain TutorGrid-owned schema and adapter boundary from day one.
- Do not implement both Konva and tldraw adapters in MVP.
- Prior discovery selected a hybrid long-term option: open-source Konva/React Konva and paid tldraw SDK. This is a strategic option, not a requirement to ship both now.
- Before choosing commercial SDK, confirm license covers multi-tenant SaaS, embedding, redistribution and white-label.
- CRDT-inspired collaboration principles and server-authoritative permissions are architectural direction; exact CRDT library is not yet frozen.

## LiveKit strategy
- LiveKit Cloud first; self-hosting on Hetzner is a later cost/operations decision.
- Go backend controls room creation, token issuance, participant authorization and permission updates.
- Learners start media disabled; teacher grants/revokes media permissions.
- Centrifugo carries classroom app events; LiveKit carries media.

## Scaling triggers
Do not scale by user-registration count alone. Measure active concurrency, classroom participants, websocket connections, media minutes, event rate, database load, storage, and error/latency SLOs.
Possible future actions only after evidence: increase managed DB/cache size, add worker replicas, isolate workloads, add read replicas, deploy NATS cluster, self-host LiveKit, extract service, or adopt Kubernetes.
