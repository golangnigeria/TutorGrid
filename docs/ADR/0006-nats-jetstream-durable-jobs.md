# ADR 0006 — Restore NATS JetStream as Single Durable Job & Event System (Superseding Asynq in ADR 0005)

**Status:** Accepted (Explicit Product-Owner Decision — 2026-11-28)  
**Supersedes (in part):** [`ADR 0005`](./0005-go-boilerplate-pgx-tern-asynq-newrelic.md) (Decision #3 — Background Jobs)  
**References:** [`docs/MASTER_BLUEPRINT.md`](../MASTER_BLUEPRINT.md) (§12, §23), [`docs/ARCHITECTURE.md`](../ARCHITECTURE.md)

---

## 1. Context
During the `go-boilerplate` migration (`ADR 0005`), Redis-backed **Asynq** was temporarily carried over from the boilerplate for MVP background jobs. However, TutorGrid's core architecture designates **Redis** as disposable cache, rate-limiting, ephemeral coordination, and Centrifugo broker state—not as a durable job store—while **NATS JetStream** is the canonical durable background job and event backbone for invitations, storage cleanup/deletes, future malware scanning, media processing, and classroom lifecycle events.

Keeping Asynq alongside a future NATS JetStream deployment would create two overlapping job systems and weaken durability guarantees for storage reconciliation and transactional outbox delivery.

## 2. Decision
1. **Single Durable Job & Event System:** Adopt **NATS JetStream** (`github.com/nats-io/nats.go`, `nats:2.10-alpine` with `-js`) as TutorGrid's sole durable background job and event system across local development, CI, staging, and production.
2. **Remove Asynq Completely:** Remove `github.com/hibiken/asynq` from `apps/backend` and migrate Slice 2b's transactional invitation email worker (`tutorgrid.jobs.email.tenant_invitation`) onto JetStream stream `TUTORGRID_JOBS`.
3. **Standard Event Envelope & Durability Rules:**
   - Every job message uses a versioned JSON envelope: `event_id`, `schema_version`, `tenant_id`, `correlation_id`, `timestamp`, and `payload` (never carrying raw secrets or binary payloads).
   - Consumers are idempotent with bounded retries (`MaxDeliver`) and observable error logging.
   - PostgreSQL remains the durable source of truth (transactional outboxes and durable lifecycle states such as `pending` / `deleting`), backed by bounded periodic reconciliation to recover any database-commit/publish gap.
4. **Redis Role Restored:** Redis remains strictly disposable for caching, rate limiting, ephemeral session coordination, and Centrifugo pub/sub.

## 3. Consequences
- Eliminates dual-queue complexity while aligning the running codebase with `docs/MASTER_BLUEPRINT.md` and `docs/ARCHITECTURE.md`.
- Adds one lightweight `nats:2.10-alpine` container (`-js`) to local Docker Compose and GitHub Actions CI.
- Positions Slice 3 (Resource Library cleanup/deletion), Slice 5 (Whiteboard), and future media/malware-scanning workers on a single durable event bus.
