# TutorGrid — Consolidated Technology Decision Register

**Canonical Sources:** [`docs/MASTER_BLUEPRINT.md`](./MASTER_BLUEPRINT.md) (§§23–24) and [`docs/ADR/`](./ADR/)  
**Status Legend:**
- **Confirmed** — Explicitly approved by the product owner.
- **Recommended** — Technical recommendation not yet explicitly approved.
- **Open** — Requires explicit product-owner decision before implementation.
- **Future** — Confirmed long-term capability, deferred beyond initial MVP slices.

---

## 1. Confirmed & Active Technology Decisions (With Source Provenance)

| Area | Technology / Direction | Status | Source | ADR |
|---|---|---|---|---|
| **Backend Architecture** | Modular monolith (extract services only when justified) | Confirmed | Master Blueprint §17.1, §23; ADR 0001 | [`ADR 0001`](./ADR/0001-approved-stack-and-mvp.md) |
| **Repository Strategy** | Single monorepo | Confirmed | Master Blueprint §17.4, §23 | [`ADR 0001`](./ADR/0001-approved-stack-and-mvp.md) |
| **Task Runner** | `Taskfile.yml` (go-task / `task`); no `Makefile` or `scripts/` | Confirmed | Explicit Product-Owner Decision (2026-11-25) | — |
| **Backend Language & HTTP** | Go + Echo | Confirmed | Master Blueprint §17.2, §23; ADR 0001 | [`ADR 0001`](./ADR/0001-approved-stack-and-mvp.md) |
| **Primary Database** | PostgreSQL (relational + selective JSONB) | Confirmed | Master Blueprint §11, §23; ADR 0001 | [`ADR 0001`](./ADR/0001-approved-stack-and-mvp.md) |
| **DB Access & Migrations** | `pgx/v5` (`pgxpool`) + `tern` *(supersedes Bun + Goose)* | Confirmed | Explicit Product-Owner Decision (2026-11-27); ADR 0005 | [`ADR 0005`](./ADR/0005-go-boilerplate-pgx-tern-asynq-newrelic.md) |
| **Multi-Tenancy & Isolation** | Shared DB/schema + `tenant_id`, Go authz + PostgreSQL RLS (dedicated DB later if needed) | Confirmed | Master Blueprint §6.4, §11.6, §23 | [`ADR 0002`](./ADR/0002-embedded-limen-oauth-and-rls-tenancy.md) *(Proposed)* |
| **Authentication** | Limen (`github.com/thecodearcher/limen`) — Google & GitHub OAuth only; no email/password | Confirmed | Master Blueprint §6.1, §23; ADR 0001 | [`ADR 0002`](./ADR/0002-embedded-limen-oauth-and-rls-tenancy.md) *(Proposed)* |
| **Authorization** | RBAC + resource-level permissions (backend authoritative) | Confirmed | Master Blueprint §6.3, §23; ADR 0001 | [`ADR 0002`](./ADR/0002-embedded-limen-oauth-and-rls-tenancy.md) *(Proposed)* |
| **Frontend Framework & Tooling** | React + Vite + responsive web | Confirmed | Master Blueprint §17.3, §23; ADR 0001 | [`ADR 0004`](./ADR/0004-frontend-ui-and-toolchain.md) *(Proposed)* |
| **UI Component System** | `shadcn/ui` | Confirmed | Master Blueprint §17.3, §23 | [`ADR 0004`](./ADR/0004-frontend-ui-and-toolchain.md) *(Proposed)* |
| **React 19, TypeScript, Tailwind CSS v4** | React 19 + TypeScript + `@tailwindcss/vite` | Recommended / Pending ADR Sign-Off | Selected in Slice 1 `/grill-me` (2026-11-25); Master Blueprint §17.3/§24 requires explicit ADR | [`ADR 0004`](./ADR/0004-frontend-ui-and-toolchain.md) *(Proposed)* |
| **Cache & Ephemeral State** | Redis (cache-aside, rate limiting, temporary classroom state, Centrifugo broker) | Confirmed | Master Blueprint §12, §23; ADR 0001 | [`ADR 0001`](./ADR/0001-approved-stack-and-mvp.md) |
| **Application Realtime** | Centrifugo (backend-authorized channels; never replaces LiveKit) | Confirmed | Master Blueprint §7, §23; ADR 0001 | [`ADR 0001`](./ADR/0001-approved-stack-and-mvp.md) |
| **Video / Audio / WebRTC** | **LiveKit** (mandatory; LiveKit Cloud initially, Hetzner Cloud candidate for later self-hosting) | Confirmed | Master Blueprint §8, §23; ADR 0001 | [`ADR 0001`](./ADR/0001-approved-stack-and-mvp.md) |
| **Object Storage (All Environments)** | Backblaze B2 exclusively (S3-compatible API, direct presigned PUT/GET; MinIO removed from dev & CI; separate private dev & test buckets) | Confirmed | Master Blueprint §9.1–9.3, §23; Explicit PO Decision (2026-11-28) | [`ADR 0003`](./ADR/0003-direct-s3-b2-presigned-storage.md) |
| **Jobs & Background Queues** | **NATS JetStream** (single durable job & event system; supersedes Asynq from ADR 0005; Redis strictly cache/rate-limits/Centrifugo) | Confirmed | Master Blueprint §12, §23; Explicit PO Decision (2026-11-28) | [`ADR 0006`](./ADR/0006-nats-jetstream-durable-jobs.md) |
| **API Architecture** | Hybrid REST + OpenAPI (URL major versions) + selective internal gRPC | Confirmed | Master Blueprint §16, §23; ADR 0001 | [`ADR 0001`](./ADR/0001-approved-stack-and-mvp.md) |
| **Whiteboard Strategy** | TutorGrid-owned document schema; long-term hybrid Konva + tldraw | Confirmed (long-term) | Master Blueprint §4.2, §23; ADR 0001 | — *(MVP renderer Open)* |
| **Media Processing** | Hybrid self-managed (FFmpeg, LibreOffice, Go libraries) via async NATS workers | Confirmed (Future) | Master Blueprint §9.5 | — |
| **Search** | PostgreSQL initially; **Elasticsearch** when search needs exceed PG | Confirmed (Future) | Master Blueprint §13, §23 | — |
| **Analytics** | PostgreSQL initially; **ClickHouse** + NATS/Redis/Centrifugo real-time pipeline as volume grows | Confirmed (Future) | Master Blueprint §14, §23 | — |
| **Hosting & Delivery** | Docker initially (K8s later if justified), Northflank preferred candidate, GitHub Actions, GHCR, Terraform, Cloudflare DNS/CDN | Confirmed direction (validate Northflank) | Master Blueprint §18, §23; ADR 0001 | [`ADR 0001`](./ADR/0001-approved-stack-and-mvp.md) |
| **Secrets Management** | Northflank secrets initially; Vault later if justified | Confirmed direction | Master Blueprint §18.5, §23 | — |
| **Observability** | New Relic APM + Zerolog *(supersedes OTel/Prometheus/Grafana/Sentry)* | Confirmed | Explicit Product-Owner Decision (2026-11-27); ADR 0005 | [`ADR 0005`](./ADR/0005-go-boilerplate-pgx-tern-asynq-newrelic.md) |
| **Transactional Email** | Resend (`resend-go/v2`) | Confirmed | Explicit Product-Owner Decision (2026-11-27); ADR 0005 | [`ADR 0005`](./ADR/0005-go-boilerplate-pgx-tern-asynq-newrelic.md) |

---

## 2. Open & Recommended Decisions (Do NOT Treat as Confirmed)

| ID | Topic | Status | Source / Notes | Decision Needed Before |
|---|---|---|---|---|
| **OD-01** | **Initial MVP Whiteboard Renderer** | **Open** | Master Blueprint §4.2.1, §24 #1. Define TutorGrid-owned schema + submit Konva vs. tldraw comparison before coding whiteboard. | Phase 3 (Whiteboard) |
| **OD-02** | **tldraw Commercial Licensing** | **Open** | Master Blueprint §4.2.1, §24 #2. Requires written confirmation for multi-tenant SaaS, embedding, redistribution, and white-label. | Before any commercial tldraw use |
| **OD-03** | **Whiteboard CRDT Library & Algorithm** | **Open** | Master Blueprint §4.2.8, §24 #3. CRDT principles confirmed; exact library/algorithm open. | Phase 3 (Whiteboard) |
| **OD-04** | **Learner Guest Access** | **Open / Recommended** | Master Blueprint §6.6, §24 #4. Short-lived, scoped, revocable invite tokens discussed; not confirmed. | Phase 4 (Classroom) |
| **OD-05** | **LiveKit Learner Media Defaults & Permissions** | **Recommended (Pending Approval)** | Master Blueprint §8.6, §24 #6. Learners media-disabled by default with teacher grant/revoke is recommended, not confirmed. | Phase 4 (Classroom) |
| **OD-06** | **Math / Equation Renderer** | **Recommended (Pending Approval)** | KaTeX candidate; not confirmed in Master Blueprint. | Phase 2 (Lesson Studio) |
| **OD-07** | **Pilot Concurrency Profile** | **Open** | `docs/MVP_SCOPE.md`. Must define peak concurrent rooms, participants/room, and event rates before load testing. | Phase 6 (Pilot Hardening) |
| **OD-08** | **MVP Starter Activity Template List** | **Open** | `docs/MVP_SCOPE.md`. Finalize exact starter count/types (quiz, matching, sorting, fill-in-the-blank). | Phase 5 (Activity Engine) |
| **OD-09** | **External Partner OIDC/SSO** | **Open (Future)** | Master Blueprint §6.5, §24 #5. Exact provider support and timing open. | Phase 4+ / Partner API |
| **OD-10** | **Northflank Provider Validation** | **Open** | Master Blueprint §18.2, §24 #9. Validate managed PG PITR, Redis ACLs, NATS/Centrifugo persistence, pricing. | Before Staging/Prod deploy |
| **OD-11** | **Real Backblaze B2 Staging Verification** | **Open** | Master Blueprint §9.4, §24 #10. MinIO alone is insufficient; must test real B2 presigned URL + CORS in staging. | Before Staging/Prod deploy |
| **OD-12** | **AI Provider, White-Label Packaging, Retention Periods, Search Mappings, Partner SDK** | **Open (Future)** | Master Blueprint §24 #13–18. Long-term capabilities; policy/commercial design deferred. | Post-MVP phases |
