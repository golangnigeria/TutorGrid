---
trigger: glob
globs: "**/*.sql, **/*migration*, **/db/**"
description: "TutorGrid PostgreSQL/Bun/Goose rules: tenant isolation, safe migrations, integrity and compatibility."
---
# Database rules
- PostgreSQL is the durable source of truth; Redis is not a substitute for durable lesson/results data.
- Use Bun and Goose as approved. Never edit an already-applied migration; add a new migration.
- Use expand-and-contract for compatibility-sensitive schema changes; avoid destructive migrations without explicit approval and backup plan.
- Every tenant-owned record must have a clear tenant ownership strategy and authorization coverage.
- Use constraints, transactions and idempotency for durable outcomes.
- Apply tenant context safely for pooled connections when using RLS; never leak context between transactions.
- Add indexes based on query patterns and explain query/lock risks for large changes.
- Test migrations up and relevant rollback/recovery strategy in CI or isolated test DB.
- Enforce PostgreSQL RLS using a non-superuser application role (`NOSUPERUSER NOBYPASSRLS`) with `FORCE ROW LEVEL SECURITY` and transaction-scoped `set_config('app.current_tenant_id', ..., true)` through Bun.
- Prove zero cross-tenant leakage with real PostgreSQL integration tests running over pooled connections (including concurrent transactions across tenants and fail-closed behavior when tenant context is unset).
