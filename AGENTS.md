# TutorGrid — Antigravity Project Constitution

## Mission
Build TutorGrid as a teaching and learning creation-and-delivery platform. TutorGrid is not an LMS, school administration system, tutor CRM, or generic video-meeting clone. Its core product is the Lesson Studio, Whiteboard Engine, and Educational Activity/Game Engine, delivered through Live Classroom and available standalone or embedded.

## Source of truth
Before changing code, read `docs/MASTER_BLUEPRINT.md`, `docs/PRODUCT_CONSTITUTION.md`, `docs/PRODUCT_CAPABILITY_MAP.md`, `docs/MVP_SCOPE.md`, `docs/PRODUCT_SPEC.md`, `docs/ARCHITECTURE.md`, and `docs/TECHNOLOGY_DECISIONS.md`. For the task's module, read the relevant `.agents/rules/*.md` file and existing ADRs in `docs/ADR/`. Existing code and tests are evidence of implementation, not permission to change product decisions.

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

## Approved stack (do not substitute without an approved ADR — updated by ADR 0005)
- Frontend: React + Vite + `shadcn/ui`; responsive web in `apps/frontend` (Turborepo monorepo).
- Backend: Go + Echo, modular monolith in `apps/backend` (Clean Architecture from `sriniously/go-boilerplate`).
- Data: PostgreSQL, `pgx/v5` (`pgxpool`) for data access, `tern` migrations (per `ADR 0005`).
- Authentication: Limen (`github.com/thecodearcher/limen`); Google and GitHub OAuth; no email/password login (replaced Clerk).
- Authorization: RBAC plus resource-level permissions; tenant-aware backend checks and PostgreSQL RLS as defense in depth.
- Cache & Ephemeral Coordination: Redis (cache, rate limiting, ephemeral session coordination, Centrifugo broker).
- Durable Background Jobs & Events: NATS JetStream (`github.com/nats-io/nats.go`, per `ADR 0006`; supersedes Asynq from `ADR 0005`).
- Application realtime: Centrifugo.
- Media: LiveKit (mandatory) — LiveKit Cloud initially; Hetzner Cloud self-hosting is a later evaluation, not MVP.
- Object storage: Backblaze B2 exclusively via S3-compatible abstraction (`aws-sdk-go-v2/service/s3`, per `ADR 0003` — no MinIO in local dev or CI; separate private dev & test buckets; non-HTTPS/localhost/MinIO endpoints rejected in all environments).
- Email: Resend (real `TUTORGRID_INTEGRATION.RESEND_API_KEY` required).
- API: REST + OpenAPI for external/public use; gRPC only where useful internally.
- Hosting & Edge: Northflank candidate/initial deployment, Cloudflare DNS/CDN, Docker, GitHub Actions, GHCR, Terraform.
- Observability: New Relic APM + Zerolog structured logging (per `ADR 0005`; real `TUTORGRID_OBSERVABILITY.NEW_RELIC.LICENSE_KEY` required).
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
- Always ask the engineer for real external inputs—never use dummy links, fake credentials, mock bypasses, or shortcuts. When a task requires a real repository URL, service/endpoint URL, secret link, OAuth callback URL, provider account, API key, or credential, stop and ask for the real input explicitly. Ask directly in chat for real links/URLs; for secrets/credentials, list the exact required environment variable names and ask the engineer to configure the real values in local gitignored `.env` or GitHub/platform secret storage and confirm before proceeding (never commit or print secrets).
- Use `Taskfile.yml` (go-task / `task`) for all local development, migration, test, lint, and build commands. Do not add a `Makefile` or a `scripts/` directory to this repository.
- Backend SQL convention: use `pgx.StrictNamedArgs` with `@snake_case` named parameters across `apps/backend/internal/repository/*` and `apps/backend/internal/database/rls.go` (avoid positional `$1`/`$2` placeholders in Go SQL queries).
- Frontend UI convention: strict dark-only app shell for **TutorGrid Circle** using the **Dark Forest-Moss & Warm Stone** palette extracted from Watermelon UI `hero-34` — Deep Pine-Charcoal base (`#101411`), Frosted Moss-Slate surfaces (`#181F1A`/82 `backdrop-blur-md`), Muted Sage primary accent (`#8FAF94`), Warm Sand secondary highlight (`#D6C29F`), Warm Parchment headings/ink (`#F3F0E8`), and Soft Stone-Sage muted ink (`#A6B0A8`), with hairline borders, no heavy drop shadows or AI-style neon glows/gradients, subtle `motion/react` micro-interactions, and no light mode toggle. Uses `/tutorgrid-bg.avif` as the visible full-bleed backdrop behind frosted surfaces, fixed auth-dynamic Watermelon UI `navigation-7` header pill, `hero-34` landing (`/`), fixed centered floating native-mobile-style bottom tab dock (`MobileFooterDock`) across all screen sizes, and routes `/`, `/dashboard`, `/sign-in`, `/sign-out`. Future Lesson Studio and Whiteboard canvases may expose isolated canvas-only background surfaces inside the dark studio chrome. Typography uses `Inter` (400–600, `--font-sans`) for workspace UI, forms, tables, and body copy, and `Fredoka` (500–600, SIL OFL, `--font-display`) strictly for brand wordmark, page headings, onboarding title, and selected educational accents.

## Definition of done for every task
1. Acceptance criteria are satisfied.
2. Relevant tests are added/updated and run.
3. Lint/typecheck/build (where applicable) are run.
4. No unrelated files or dependencies changed.
5. API/schema/config/docs changes are documented.
6. A concise report lists files changed, behavior delivered, checks run/results, known limitations, and next step.

## Stop conditions
Stop and ask before: changing approved technologies; adding a new service; changing tenancy/authentication; changing lesson or whiteboard document formats; changing media provider; expanding MVP; deleting user data; destructive migrations; proceeding without real external links, accounts, or credentials required by the task; or making a decision that materially affects product UX, licensing, cost, or security.
