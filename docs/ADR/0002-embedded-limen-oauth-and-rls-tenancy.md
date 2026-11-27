# ADR 0002 — Embedded Limen OAuth, `limen_*` Schema Coexistence & Dual RLS Tenancy

**Status:** Accepted (Product-Owner Decision — 2026-11-27)  
**References:** [`docs/MASTER_BLUEPRINT.md`](../MASTER_BLUEPRINT.md) (§6, §11.6), [`docs/ADR/0005-go-boilerplate-pgx-tern-asynq-newrelic.md`](./0005-go-boilerplate-pgx-tern-asynq-newrelic.md)

---

## 1. Context
TutorGrid embeds **Limen** (`github.com/thecodearcher/limen`) for Google and GitHub OAuth (no email/password) alongside `pgx/v5` + `tern` and PostgreSQL Row-Level Security (`FORCE ROW LEVEL SECURITY`) enforced via the non-superuser role `tutorgrid_app` (`NOSUPERUSER NOBYPASSRLS`, non-owner).

After Limen authenticates a user, the Go backend must resolve that user's memberships, manage server-bound active workspace selection, and support atomic invitation acceptance without relaxing tenant isolation or requiring `SECURITY DEFINER` SQL functions.

## 2. Decision (Dual RLS — Zero `SECURITY DEFINER` Functions)
1. **Prefixed Limen Tables:** Configure Limen with `limen_users`, `limen_sessions`, `limen_accounts`, `limen_verifications`, and `limen_rate_limits` via `pgx/v5/stdlib.OpenDBFromPool`, sharing the `pgxpool.Pool` without colliding with TutorGrid domain tables.
2. **Per-Table Operation-Specific Dual RLS Policies (`003_tenancy_dual_rls_and_invitations.sql`):**
   - `tenants`: `SELECT` allows `id = app.current_tenant_id OR EXISTS (membership for app.current_user_id)`; `INSERT` requires `app.current_user_id IS NOT NULL`; `UPDATE`/`DELETE` strictly require `id = app.current_tenant_id`.
   - `tenant_memberships`: `SELECT` allows `tenant_id = app.current_tenant_id OR user_id = app.current_user_id`; `INSERT`/`UPDATE`/`DELETE` strictly require `tenant_id = app.current_tenant_id`.
   - `user_active_tenants`: Strictly `user_id = app.current_user_id`.
   - `tenant_invitations`: `SELECT`/`UPDATE` allow `tenant_id = app.current_tenant_id` OR `(app.current_user_id IS NOT NULL AND token_hash = decode(app.current_invite_token_hash, 'hex'))`; `INSERT`/`DELETE` strictly require `tenant_id = app.current_tenant_id`.
   - `invitation_email_outbox`: Strictly `tenant_id = app.current_tenant_id` (written atomically with `tenant_invitations`; `invite_url` cleared immediately upon send; raw tokens never logged).
3. **Server-Bound Active Workspace & Atomic Acceptance:**
   - `SetActiveTenant` verifies membership (`FOR SHARE`) before updating `user_active_tenants`.
   - `RequireTenant` atomically joins `user_active_tenants` with `tenant_memberships` inside a user-scoped transaction; deleted/revoked memberships fail immediately.
   - `AcceptInvitation` locks the invitation (`FOR UPDATE`), checks expiry, revocation, prior acceptance, and email restriction, inserts membership, marks `accepted_at`, and sets `active_tenant_id` in one atomic `pgx.Tx`.

## 3. Consequences
- Eliminates `SECURITY DEFINER` privilege-escalation surface completely; all queries stay in Go/`pgx` under `tutorgrid_app` (`NOSUPERUSER NOBYPASSRLS`).
- Verified by real PostgreSQL integration tests covering the 3-state Dual RLS matrix (unset, user-only, both contexts), 500-tx `pgxpool` concurrency, cross-tenant reads/writes, immediate revocation, outbox durability, and concurrent invitation acceptance races.
