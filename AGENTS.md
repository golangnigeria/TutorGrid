# TutorGrid — Antigravity Project Constitution

## Mission
Build TutorGrid as a teaching and learning creation-and-delivery platform. TutorGrid is not an LMS, school administration system, tutor CRM, or generic video-meeting clone. Its core product is the Lesson Studio, Whiteboard Engine, and Educational Activity/Game Engine, delivered through Live Classroom and available standalone or embedded.

## Source of truth
Before changing code, read `docs/PRODUCT_CONSTITUTION.md`, `docs/MVP_SCOPE.md`, `docs/PRODUCT_SPEC.md`, and `docs/ARCHITECTURE.md`. For the task's module, read the relevant `.agents/rules/*.md` file and existing ADRs. Existing code and tests are evidence of implementation, not permission to change product decisions.

## Non-negotiable product rules
1. Do not invent, remove, simplify, or materially reinterpret product requirements. If a requirement is unclear or conflicts with a source document, stop and ask one focused question or propose an ADR; do not silently decide.
2. MVP and the first 1,000 users are the planning horizon. Do not implement speculative enterprise-scale infrastructure, microservices, Kubernetes, multi-region systems, or advanced AI unless explicitly approved.
3. TutorGrid's three product engines are first-class: Lesson Studio, Whiteboard Engine, and Educational Activity/Game Engine. Live Classroom integrates them; it does not replace them.
4. Support Pre-K through Grade 12, all major subjects, and flexible curriculum. Avoid childish visual treatment for older learners.
5. Lessons can be created in TutorGrid and eventually through external APIs. Standalone and embedded distribution share the same underlying domain model.
6. Uploaded PDF/PowerPoint files remain attached reference materials; do not silently convert them into editable lesson content.
7. AI is future scope. Do not add AI-generated content or AI infrastructure to MVP without approval.
8. Teacher-prepared lesson content and live classroom session state must be separate. A classroom pins a published lesson version; live changes must not overwrite the source lesson.
9. Tenant boundaries and backend authorization are required from the first release. Never trust frontend-only permissions.
10. Never execute arbitrary teacher-uploaded code in a learner's browser/classroom. Activities use controlled components and configuration.

## Approved stack (do not substitute without an approved ADR)
- Frontend: React + Vite; responsive web for desktop, laptop, tablet and mobile.
- Backend: Go + Echo, modular monolith.
- Data: PostgreSQL, Bun for data access, Goose migrations.
- Authentication: Limen; Google and GitHub OAuth; no email/password login in the current decision.
- Authorization: RBAC plus resource-level permissions; tenant-aware backend checks and PostgreSQL RLS as defense in depth.
- Cache/temporary state: Redis.
- Application realtime: Centrifugo.
- Media: LiveKit Cloud initially; self-hosting on Hetzner is a later evaluation, not MVP.
- Object storage: Backblaze B2 via S3-compatible abstraction.
- Jobs/events: NATS JetStream, small MVP deployment.
- API: REST + OpenAPI for external/public use; gRPC only where useful internally.
- Hosting: Northflank candidate/initial deployment; Docker, GitHub Actions, GHCR, Terraform.
- Observability: OpenTelemetry, Prometheus/Grafana and Sentry, with practical MVP dashboards/alerts.
- Search/analytics: PostgreSQL initially; Elasticsearch and ClickHouse are later growth choices, not required MVP services.

## Engineering behavior
- Work in small, reviewable vertical slices. Do not generate the whole application in one pass.
- First inspect repository state, docs, package scripts, existing patterns and tests. Do not overwrite existing work.
- Before coding, provide: scope, files/modules likely to change, data/API impact, risks, and verification plan. Wait for approval for architectural changes, broad refactors, new infrastructure, dependency changes, or scope expansion.
- Implement only the requested slice. Do not opportunistically refactor unrelated code.
- Prefer the simplest maintainable solution that satisfies documented acceptance criteria and the 1,000-user target.
- No placeholder buttons, fake success states, mock data in production paths, empty handlers, or TODO-only implementations presented as complete.
- Do not claim tests pass unless they were run and the result was observed. If unable to run, state why and give exact commands.
- Update tests, docs, and ADRs when behavior or contracts change.
- Preserve backward compatibility unless the task explicitly authorizes a breaking change.
- Use secure defaults, input validation, authorization checks, bounded resource usage, and clear error handling.
- Never silently confirm unresolved recommendations. Keep unapproved choices (e.g., guest learner access, learner media permission tiers, whiteboard renderer, math renderer, pilot concurrency targets) explicitly labeled **Open** or **Recommended (Pending Approval)** until explicitly approved.
- Before implementing the Whiteboard Engine, define the TutorGrid-owned document schema and submit a renderer comparison recommending one MVP renderer for explicit approval.

## Definition of done for every task
1. Acceptance criteria are satisfied.
2. Relevant tests are added/updated and run.
3. Lint/typecheck/build (where applicable) are run.
4. No unrelated files or dependencies changed.
5. API/schema/config/docs changes are documented.
6. A concise report lists files changed, behavior delivered, checks run/results, known limitations, and next step.

## Stop conditions
Stop and ask before: changing approved technologies; adding a new service; changing tenancy/authentication; changing lesson or whiteboard document formats; changing media provider; expanding MVP; deleting user data; destructive migrations; or making a decision that materially affects product UX, licensing, cost, or security.
