---
trigger: model_decision
description: "Apply when implementing auth, authorization, tenancy, uploads, APIs, classrooms, or handling credentials."
---
# TutorGrid Security & Tenant Isolation Rules
- Backend authorization is authoritative; frontend hiding is presentation only.
- Never trust a client-supplied tenant ID, role, ownership, or object key. Derive tenant context strictly from verified identity and authorized membership.
- Enforce PostgreSQL RLS (`ENABLE` + `FORCE ROW LEVEL SECURITY`) via a non-superuser role (`NOSUPERUSER NOBYPASSRLS`) and transaction-scoped `set_config('app.current_tenant_id', ..., true)` through Bun.
- Direct uploads must use private buckets, restricted CORS, server-generated tenant-prefixed object keys, MIME + magic-byte validation, size caps, `HeadObject` completion verification, and short-lived presigned URLs.
- Production configuration must reject local MinIO / `localhost` / non-HTTPS storage endpoints.
- Never execute arbitrary teacher-uploaded code in learner classrooms or browsers.
- Never commit, print, or invent secrets, OAuth credentials, or external links. Always ask the engineer for real external inputs.
- Record audit trails for privileged, membership, publishing, sharing, resource, and classroom lifecycle actions.
