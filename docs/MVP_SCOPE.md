# TutorGrid MVP Scope — First 1,000 Users

## Goal
Ship a reliable, coherent MVP for an initial target of 1,000 registered users, with an initial pilot of real teachers and learners. The target is not 1,000 simultaneous video participants. Capacity planning must separately define peak concurrent classrooms, participants per room, media usage and activity traffic using pilot data.

## MVP product promise
A teacher can create and organize a lesson, prepare visual teaching material, launch a classroom, teach using a whiteboard and LiveKit audio/video, launch a simple educational activity, permit learner interaction, and review basic participation/activity results. A learner can securely join, participate according to permissions, and access the lesson experience on a responsive web app.

## MVP must-have
### Identity and tenant access
- Google and GitHub OAuth through Limen.
- User profile basics and organization membership/invitation/access flow appropriate to the pilot.
- Roles: platform admin, organization admin (minimal), teacher, learner.
- Tenant-aware access checks for every protected resource.
- Resource sharing/ownership rules and short-lived classroom access tokens.

### Lesson Studio — MVP foundation
- Create, edit, save, duplicate and archive a lesson.
- Basic lesson metadata: title, subject, grade/age band, topic, objectives, duration.
- Ordered sections/steps with add, edit, reorder and delete.
- Basic rich content blocks: text, image and attached files; support equation/math content only if it fits the chosen MVP editor component.
- Attach PDF/PowerPoint as reference materials; preview/present them and annotate via whiteboard; do not auto-convert.
- Preview lesson and publish immutable version.
- Basic reusable templates/blocks if low-risk; otherwise include a small curated template set and document the library expansion phase.

### Whiteboard — MVP
- One renderer selected for MVP; do not implement dual-renderer portability yet.
- TutorGrid-owned minimal document schema and adapter boundary so renderer is replaceable.
- Pages with pan/zoom, pen, eraser, text, basic shapes/lines, selection/move/resize, undo/redo, image/file insertion, save/load.
- Teacher-controlled board and learner view-only/drawing permission.
- Basic real-time synchronization and reconnect/reload recovery.
- Separate prepared board from live session copy/state.
- Basic whiteboard objects; advanced geometry, complex layers, broad object libraries, full infinite canvas, sophisticated import/export are phased.

### Live Classroom — MVP
- Create/start/end classroom linked to a published lesson version.
- Join via authorized invite/access flow.
- LiveKit Cloud audio/video; mic/camera controls and participant status.
- Teacher may publish media; learners join with media disabled by default; teacher grants/revokes learner mic/camera/screen share. Backend enforces the policy.
- Lesson page/section navigation, teacher-guided or free navigation at a simple level.
- Centrifugo for application events/presence; not media transport.
- Basic chat only if required by pilot; do not build a large social messaging system.
- Classroom state and participation records persisted.

### Activity/Game Engine — MVP
- Activity runtime contract and controlled activity components.
- A small polished starter set of templates (e.g. quiz, matching, sorting, fill-in-the-blank; finalize count after product review).
- Teacher can create an activity from a template, enter content/questions, preview and attach to lesson.
- Individual gameplay first; basic teacher-led shared activity where feasible.
- Basic score/completion/response results.
- No arbitrary code execution, no full advanced visual game builder in MVP.

### Platform basics
- B2 upload/download using signed URLs and metadata.
- Asynchronous processing only for essential tasks using small NATS JetStream deployment.
- Basic monitoring, error reporting, backups and restore rehearsal.
- CI: tests, lint, typecheck/build, migration checks.
- Terraform for essential infrastructure, not every possible future environment.
- Basic audit records for security-sensitive actions.

## Explicitly defer
- AI lesson/game generation.
- Full white-label and complex partner provisioning.
- Broad public API/SDK and external lesson authoring API; keep domain design API-ready, deliver narrow API only if pilot requires it.
- Advanced visual game builder and user-authored arbitrary logic.
- Dual whiteboard renderer implementation and lossless conversion.
- Kubernetes, microservices, multi-region, DB sharding, Redis Cluster, NATS cluster unless measured requirement emerges.
- LiveKit self-hosting; stay on LiveKit Cloud initially.
- Dedicated Elasticsearch and ClickHouse; use PostgreSQL first.
- Advanced enterprise SSO, dedicated tenant DBs, enterprise analytics.
- Complex assessment engine, question banks, sophisticated gradebook/reporting.
- Recording/transcription unless pilot confirms need and privacy/legal work is completed.

## MVP non-goals
No school management system, LMS course administration, payroll, fees, admissions, scheduling administration, recruitment, CRM, or general-purpose meeting product.

## MVP acceptance gate
- A teacher can complete the core create → prepare → publish → start classroom → teach → launch activity → end → review flow.
- A learner can join only an authorized classroom and cannot access another tenant's content.
- Lesson source and published/classroom version do not get silently overwritten by live changes.
- Classroom permission changes are enforced server-side and reflected in LiveKit.
- Whiteboard state survives refresh/reconnect within documented MVP limits.
- Activity results are saved idempotently and can be reviewed by the teacher.
- Core flows pass automated tests and a real pilot rehearsal on supported devices/browsers.
- Capacity is load-tested against a stated pilot concurrency profile, not a vague registered-user count.
