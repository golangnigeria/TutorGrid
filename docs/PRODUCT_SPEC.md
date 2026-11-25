# TutorGrid Product Specification (MVP baseline)

## Primary teacher journey
1. Sign in using Google or GitHub.
2. Enter or create an organization/teaching space for the pilot.
3. Create a lesson with title, subject, grade/age, topic, objectives and duration.
4. Add sections/steps and instructional content.
5. Attach PDF/PowerPoint or other supported resources as reference material.
6. Prepare a whiteboard page or attach an existing whiteboard document.
7. Add a template-based activity and configure its content.
8. Preview and publish a version.
9. Start a classroom from that published version.
10. Invite learners using the authorized join flow.
11. Teach with LiveKit audio/video, whiteboard, lesson navigation and activities.
12. Grant/revoke learner media and collaboration permissions.
13. End the classroom and review participation and activity results.

## Primary learner journey
1. Open a valid classroom invitation and authenticate/identify according to the pilot access policy.
2. Join only the specified authorized classroom.
3. View lesson content and media according to access rights.
4. Use mic/camera only after teacher permission; permission revocation takes effect.
5. Draw/interact only where enabled.
6. Participate in activities and submit responses.
7. Reconnect safely after temporary network interruption without duplicate submissions.

## Lesson domain model
- Lesson: logical authored lesson identity and metadata.
- LessonDraft: editable working state.
- LessonVersion: immutable published snapshot.
- LessonSection: ordered section/step hierarchy.
- ContentBlock: typed instructional element with validated payload.
- Resource: metadata and B2 object reference for uploaded material.
- WhiteboardDocument: TutorGrid-owned board schema and pages.
- ActivityDefinition: configured educational activity and its content.
- Assessment/ActivityAttempt: learner response and result records as needed for MVP.
- ClassroomSession: active delivery session referencing a fixed LessonVersion.
- ClassroomParticipant: user/guest identity, role, status and permissions.
- ClassroomEvent: meaningful session event; do not store high-frequency cursor/drawing samples as ordinary relational rows.

## Lesson Studio interaction requirements
- Left: outline/sections/pages.
- Center: content editor/canvas.
- Right: properties/settings when relevant.
- Top: save state, undo/redo where applicable, preview, publish.
- Keep advanced tools contextual; do not crowd MVP with all future authoring controls.
- Autosave should show explicit saving/saved/error state. Never show success before persistence confirms.
- Reordering must persist and maintain stable IDs.
- Published version is immutable; editing creates/updates a draft and publishing creates a new version.

## Whiteboard MVP requirements
- Board document has stable IDs, version/schema version, page list, object list, and ownership/access metadata.
- Object types are explicit and validated: stroke, text, basic shape, line/arrow, image/resource reference.
- Renderer-specific data must be isolated in adapter/extension fields; core document is not a direct renderer export.
- Operations include create/update/delete/move/transform and page navigation, with authorization.
- Teacher is authoritative for permissions. Learner collaboration is opt-in by room/board/object policy.
- Persist snapshots and incremental operations where appropriate; coalesce high-frequency drawing updates.
- Reconnect path loads authoritative snapshot and applies recoverable updates; duplicate operations are safely deduplicated.

## Activity runtime contract
An activity definition includes: stable ID, type/version, subject/grade/difficulty metadata, instructions, content/configuration, asset references, allowed interaction mode, and scoring/result schema.
The runtime receives a controlled configuration and emits validated events/results. It must not receive arbitrary executable scripts from user content.
MVP activities must support: start, pause/resume where needed, submit/complete, reset/retry policy, and result persistence. Scoring is server-validated when it affects durable outcomes.

## Classroom state and permissions
- ClassroomSession references tenant, teacher, published lesson version, LiveKit room identifier and lifecycle state.
- Backend owns room lifecycle, token issuance, participant authorization and permission changes.
- LiveKit tokens are short-lived and narrowly scoped.
- Teacher can grant/revoke learner mic/camera/screen share. Learners start media-disabled.
- A permission change is validated, applied to LiveKit, and recorded; do not update UI as successful until backend confirms.
- Postgres stores durable session/participant/results. Redis holds temporary active state. Centrifugo broadcasts application updates. LiveKit carries audio/video/screen media.

## APIs and errors
- Use REST endpoints with OpenAPI for external-facing contracts.
- Validate input at boundaries and return consistent error envelopes with stable machine-readable codes.
- Use pagination for lists, idempotency keys for retryable submissions/session commands, and explicit authorization failures.
- Avoid exposing internal database structures as public API contracts.

## UX and accessibility
- Responsive layouts across desktop/tablet/mobile.
- Clear loading, empty, saved, error, offline/reconnecting, permission-denied and success states.
- Keyboard-accessible controls where practical; visible focus; labels for controls.
- Avoid tiny controls on touch devices.
- Use age-appropriate content styling; provide configurable density, not one childish theme for all grades.

## Acceptance criteria format
Every feature ticket must state: user/role, goal, preconditions, happy path, error/permission cases, data changes, API/events, UI states, security/tenant checks, and testable acceptance criteria. Do not code from a vague feature name alone.
