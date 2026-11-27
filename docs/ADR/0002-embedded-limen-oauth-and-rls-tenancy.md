# ADR 0002 — Embedded Limen OAuth, `limen_*` Schema Coexistence & Pre-Tenant Lookup with PostgreSQL RLS

**Status:** **Proposed — Pending Security Review**  
**Date:** 2026-11-26  
**Implemented on Branch:** `slice/02-limen-auth` (requires explicit security review before acceptance)  
**References:** [`docs/MASTER_BLUEPRINT.md`](../MASTER_BLUEPRINT.md) (§6, §11.6), [`docs/ADR/0001-approved-stack-and-mvp.md`](./0001-approved-stack-and-mvp.md)

---

## 1. Context
TutorGrid uses **Limen** (`github.com/thecodearcher/limen`) for Google and GitHub OAuth (no email/password), **Bun + Goose** on PostgreSQL, and **PostgreSQL Row-Level Security (RLS)** as defense in depth on tenant-owned tables (`tenants`, `tenant_memberships`, `tenant_invitations`, and future domain tables).

Two concrete design tensions arise when integrating Limen with fail-closed PostgreSQL RLS:
1. **Schema Coexistence:** Limen's default table names include `users`, which collides with TutorGrid's domain `users` table (`00001_init_tenancy_and_rls.sql`). Limen's SQL adapter (`limen/adapters/sql`) operates directly on `*sql.DB` outside tenant RLS transactions.
2. **Pre-Tenant Chicken-and-Egg Lookup:** Under fail-closed `FORCE ROW LEVEL SECURITY` on `tenants` and `tenant_memberships` for the non-superuser role `tutorgrid_app` (`NOSUPERUSER NOBYPASSRLS`), a query without `app.current_tenant_id` set returns **0 rows**. However, after Limen verifies a user's session, the backend must resolve which tenants that verified user belongs to and verify their active tenant *before* setting `app.current_tenant_id`.

---

## 2. Proposed Design (Current Slice 2 Implementation)
1. **Prefixed Limen Tables:** Configure Limen via `limen.NewDefaultSchemaConfig` to use `limen_users`, `limen_sessions`, `limen_accounts`, `limen_verifications`, and `limen_rate_limits`, managed in Goose migration `00002_limen_auth_and_tenancy_lookup.sql`. Limen shares Bun's `*sql.DB` pool without introducing GORM.
2. **Fail-Closed `FORCE ROW LEVEL SECURITY`:** Keep `tenants`, `tenant_memberships`, and `tenant_invitations` under `ENABLE ROW LEVEL SECURITY` + `FORCE ROW LEVEL SECURITY` for `tutorgrid_app` (`NOSUPERUSER NOBYPASSRLS`).
3. **Narrow `SECURITY DEFINER` Pre-Tenant Lookup Functions:** Define 7 narrow PostgreSQL functions owned by `tutorgrid_migrator` (`SET search_path = public, pg_temp`, `REVOKE ALL FROM PUBLIC`, `GRANT EXECUTE TO tutorgrid_app`):
   - `auth_upsert_user_by_subject(p_external_subject, p_display_name, p_email)`
   - `auth_list_user_memberships(p_user_id)`
   - `auth_verify_user_membership(p_user_id, p_tenant_id)`
   - `auth_get_active_tenant(p_user_id)`
   - `auth_set_active_tenant(p_user_id, p_tenant_id)`
   - `auth_create_tenant_with_admin(p_user_id, p_name, p_slug)`
   - `auth_accept_tenant_invitation(p_user_id, p_token_hash)`

---

## 3. Threat Model & Security Implications

| Threat | Risk | Mitigation in Proposal | Residual Risk / Review Question |
|---|---|---|---|
| **`search_path` hijacking in `SECURITY DEFINER`** | High if unpinned | Every function pins `SET search_path = public, pg_temp` and revokes `PUBLIC` execute. | Verify `public` schema `CREATE` privilege is revoked from `tutorgrid_app` / `PUBLIC` on PG 15+. |
| **Over-broad `SECURITY DEFINER` owner privileges** | High | Functions are owned by `tutorgrid_migrator` (currently superuser in local Docker). | **Review required:** In staging/prod, should `tutorgrid_migrator` be a non-superuser table owner with `BYPASSRLS` or should a dedicated minimal-privilege `tutorgrid_auth_definer` role own these functions? |
| **Cross-user enumeration via `p_user_id`** | Medium | Go `RequireAuth` middleware passes `user.ID` strictly from verified Limen session, never client input. | If SQL injection or buggy Go code passes an arbitrary `p_user_id`, `SECURITY DEFINER` functions do not independently verify session ownership. |
| **Limen table compromise / pool leakage** | Medium | `limen_*` tables have no RLS and are accessed directly via `*sql.DB` by Limen; tenant queries use `RunInTenantTx` (`set_config(..., true)`). | Confirm Limen never runs inside a tenant `bun.Tx` and never leaves session-level `SET` parameters on pooled connections. |
| **Invitation token brute force / replay** | Medium | 24-byte random hex tokens stored as SHA-256 `token_hash`; `SELECT ... FOR UPDATE` marks `accepted_at = NOW()` atomically. | Add Redis/API rate limiting on `/api/v1/invitations/accept` in a later slice. |

---

## 4. Alternatives Considered

### Alternative A: Dedicated `tutorgrid_auth_lookup` PostgreSQL Role + Separate Connection Pool
- Create a second non-superuser DB role (`tutorgrid_auth_lookup`) granted `BYPASSRLS` (or narrow `SELECT` views) strictly on `users`, `tenants`, `tenant_memberships`, `user_active_tenants`, and `tenant_invitations`, used only by `internal/tenancy` auth lookup via a separate Bun pool.
- **Pros:** Eliminates `SECURITY DEFINER` PL/pgSQL functions; logic stays in Go/Bun.
- **Cons:** Doubles DB connection pools per API instance; a misconfigured pool pass could accidentally bypass RLS for domain queries.

### Alternative B: Dual RLS Policy (`tenant_id = app.current_tenant_id OR user_id = app.current_user_id`)
- Add a second transaction-scoped setting `app.current_user_id` and allow `tenant_memberships` reads when `user_id = current_user_id`.
- **Pros:** No `SECURITY DEFINER` functions for listing own memberships.
- **Cons:** Weakens strict single-tenant RLS on `tenant_memberships` and still requires a separate bypass for creating a new tenant or accepting an invitation before membership exists.

---

## 5. Required Security Review Checklist Before Changing Status to `Accepted`
- [ ] Security review of all 7 `SECURITY DEFINER` functions in `migrations/00002_limen_auth_and_tenancy_lookup.sql`.
- [ ] Decision on function owner role in staging/production (`tutorgrid_migrator` vs. dedicated non-superuser `tutorgrid_auth_definer` role).
- [ ] Verification that `REVOKE CREATE ON SCHEMA public FROM PUBLIC` holds in all environments.
- [ ] Explicit product-owner / security sign-off approving `SECURITY DEFINER` over Alternative A or B.
