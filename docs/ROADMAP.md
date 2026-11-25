# TutorGrid Strategic Roadmap — MVP to First 1,000 Users

## Phase 0 — Product and repo foundation
- Confirm MVP feature list and pilot success measures.
- Create repository structure, AGENTS.md, product specs, ADR process.
- Establish lint/typecheck/test/build commands and CI.
- Create dev/staging/prod configuration policy and secret handling.
- Define tenant/resource ownership and access model before protected feature work.
- Select one whiteboard renderer after a small proof-of-concept and license check.

Exit gate: repo is reproducible, rules/docs are loaded, CI runs, architecture boundaries are documented.

## Phase 1 — Secure platform skeleton
- React/Vite shell, routing, responsive layout.
- Go/Echo API, health/readiness endpoints, config validation.
- Limen Google/GitHub auth integration.
- Profile, tenant membership and minimal roles.
- PostgreSQL schema, Bun, Goose migration workflow.
- Tenant authorization middleware/service policy and RLS baseline.
- B2 signed upload/download abstraction.

Exit gate: authenticated users can access only their authorized tenant resources; migration and CI checks pass.

## Phase 2 — Lesson Studio vertical slice
- Create/edit lesson metadata.
- Add/reorder sections and basic content blocks.
- Attach resources.
- Save draft, preview and publish immutable version.
- Lesson listing/detail and ownership/sharing checks.

Exit gate: teacher completes create → edit → preview → publish; version immutability and tenant isolation tests pass.

## Phase 3 — Whiteboard vertical slice
- TutorGrid-owned board schema, pages and basic objects.
- Renderer adapter and basic drawing/selection/undo.
- Save/load, teacher view, learner view-only and learner drawing permission.
- Centrifugo updates and reconnect/recovery.
- Attach board to lesson and create live copy/session state.

Exit gate: two clients collaborate, permissions are enforced server-side, state recovers after reconnect, authored board is protected from live changes.

## Phase 4 — Classroom + LiveKit vertical slice
- Create/start/end session from published lesson version.
- Invite/join and participant lifecycle.
- LiveKit Cloud room/token adapter.
- Mic/camera UI and teacher-controlled learner media permissions.
- Lesson navigation, presence and session persistence.

Exit gate: teacher and learner can run a real test lesson; unauthorized join/media publish is rejected; end/reconnect flows are tested.

## Phase 5 — Activity engine vertical slice
- Activity definition/runtime contract.
- Starter template(s): quiz and one interaction type such as matching/sorting.
- Teacher configures and previews activity.
- Launch in classroom, learner submits, teacher sees result.
- Idempotent result persistence and basic feedback.

Exit gate: complete activity flow works with multiple learners and results persist without duplicate submissions.

## Phase 6 — Pilot hardening
- Real teacher/learner usability sessions.
- Responsive browser/device checks.
- Security review: tenant isolation, access links, tokens, file URLs, permissions, XSS/input validation.
- Load test against explicit pilot profile.
- Backup/restore rehearsal, deployment rollback rehearsal.
- Monitoring dashboards, alerts, incident runbooks.
- Fix critical UX friction and reliability issues; do not expand scope.

Exit gate: pilot acceptance criteria pass, known limitations documented, operational owner/runbooks exist.

## Phase 7 — First 1,000 users
- Gradually onboard cohorts; observe usage and support issues.
- Track weekly active teachers/learners, classrooms/week, lesson completion, activity use, reconnect rate, media minutes, error rates and cost per active classroom.
- Scale managed resources based on measured bottlenecks.
- Prioritize reliability and teacher workflow improvements over new infrastructure.

## Suggested product metrics
- Activation: teacher creates and publishes first lesson.
- Classroom success: session starts and ends without critical failure.
- Learner join success and time-to-join.
- Lesson reuse and repeat classroom usage.
- Activity completion and submission success.
- Whiteboard recovery after reconnect.
- Core-flow error rate and support tickets.
- Cost per active classroom/hour.

## Scope-change process
Any proposed feature must state: user value, MVP necessity, dependencies, estimate/risk, security/privacy impact, cost/operational impact, and what is removed/deferred if it is added. Product owner approves scope changes.
