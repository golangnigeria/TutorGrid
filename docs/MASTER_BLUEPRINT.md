# TUTORGRID — MASTER PRODUCT & TECHNICAL BLUEPRINT

## Complete product vision, capabilities, architecture and confirmed technology decisions

**Purpose:** This document consolidates the TutorGrid product and technical decisions made during the planning discussions, including the extensive architecture decision process covering more than 140 questions.

**Scope exclusion:** This document intentionally excludes the MVP delivery plan, MVP milestones, first-1,000-users roadmap, MVP feature prioritisation and MVP implementation sequence. It describes the complete intended TutorGrid platform. Keep MVP planning in a separate document.

Use this as the consolidated reference for product intent and explicitly recorded decisions. If an older document conflicts with a confirmed decision here, flag the conflict and propose a documentation correction. Do not silently discard requirements.

---

# 1. INSTRUCTIONS TO GOOGLE ANTIGRAVITY

You are working on **TutorGrid**, an educational creation, teaching, interaction and learning platform.

Before changing project files or writing application code:

1. Read this document fully.
2. Inspect the existing repository and all relevant product, architecture, ADR, agent-rule and workflow documents.
3. Compare the repository against this blueprint.
4. Identify contradictions, missing capabilities, duplicate requirements, assumptions and decisions that are not confirmed.
5. Produce a documentation reconciliation plan.
6. Preserve the distinction between:
   - **Confirmed** — explicitly selected or approved by the product owner.
   - **Recommended** — a technical recommendation not explicitly approved.
   - **Open** — a decision requiring product-owner input.
   - **Future capability** — part of the intended complete platform, not necessarily required immediately.
7. Do not invent approval for a recommendation.
8. Do not remove a long-term capability because it is not needed in the first release.
9. Do not introduce unnecessary microservices, Kubernetes, multi-region infrastructure, distributed databases or operational complexity without a demonstrated requirement.
10. Do not begin application implementation until the product owner has approved documentation reconciliation and the specific implementation task.

If an important decision is missing, record it as open and ask one focused question at a time. Do not ask again about decisions already answered here or in approved ADRs.

The product owner wants a sophisticated educational platform. Do not simplify the vision into a generic LMS, video-conferencing product, tutoring CRM, school management system or basic lesson editor.

---

# 2. PRODUCT IDENTITY AND VISION

## 2.1 What TutorGrid is

TutorGrid is an integrated educational content creation, instructional design, collaborative teaching, interactive learning and classroom delivery platform.

Its purpose is to let educators and education businesses create sophisticated educational experiences, deliver them live or asynchronously, and integrate them into their own products and workflows.

TutorGrid is not merely:
- A video meeting application.
- A conventional learning management system.
- A school administration system.
- A tutoring-company CRM.
- A teacher recruitment or payroll platform.
- A basic presentation or document editor.
- A collection of disconnected educational games.

TutorGrid brings together three major educational creation engines:

1. **Lesson Studio** — sophisticated instructional design and lesson authoring.
2. **Whiteboard Engine** — collaborative visual teaching, demonstration and interaction.
3. **Educational Game and Activity Engine** — reusable engine for educational games, interactive exercises, simulations and activities.

The **Live Classroom** brings these engines together for real-time teaching and learning.

The platform must also support standalone use, embedding, APIs, SDKs, sharing and partner integrations.

## 2.2 Educational coverage

TutorGrid is intended to support:
- Pre-K and early childhood learning.
- Kindergarten.
- Primary/elementary education.
- Middle school.
- Secondary/high school.
- Grades 9–12 and equivalent levels.
- A broad range of subjects and educational disciplines.
- Different national, regional, institutional and custom curricula.

The platform must not be locked to one national curriculum. Curriculum mapping should be flexible and extensible.

The experience must be age-appropriate. High-school and advanced learners must not be presented with an unnecessarily childish interface or activity style.

## 2.3 Devices

Support desktop computers, laptops, tablets and mobile phones.

The selected mobile strategy is **responsive web**. Design the application to adapt to different screen sizes and interaction methods. Do not assume a separate native mobile application is required.

## 2.4 Product boundaries

TutorGrid is not intended to become the system of record for day-to-day administration of schools or tutoring companies.

Out of scope as core TutorGrid responsibilities:
- Teacher recruitment and HR.
- Learner admissions and enrolment administration.
- Payroll.
- School fees and fee collection administration.
- Staff management.
- School timetable administration.
- General school administration.
- Full tutoring-company CRM.
- General-purpose business operations.

Tutoring companies and schools can continue managing teachers and learners in their existing systems. They can grant those people access to TutorGrid through accounts, invitations, integrations or authorized identity flows.

TutorGrid may record classroom participation, activity responses, learning interactions and results where those records support teaching, feedback and educational evaluation.

---

# 3. PRODUCT DISTRIBUTION AND BUSINESS MODEL CAPABILITIES

TutorGrid must support two complementary distribution models.

## 3.1 Standalone TutorGrid

Users can register and authenticate, create and manage educational content, build lessons and activities, upload and manage resources, use whiteboard capabilities, run standalone activities, host or join live classrooms, share lessons and activities where authorized, and manage their TutorGrid workspace and authorized organization access.

## 3.2 Embedded and integrated TutorGrid

External platforms, tutoring companies and education businesses must be able to integrate TutorGrid capabilities into their own products.

The long-term integration surface should include:
- REST APIs.
- OpenAPI contracts and documentation.
- SDKs for embedding and application integration.
- Embedded lesson experiences.
- Embedded whiteboards.
- Embedded activity/game players.
- Embedded live classroom experiences.
- Secure classroom launch and participant access.
- Webhooks and event notifications.
- External identity integration.
- Tenant-aware access and authorization.
- Branding configuration.

The same underlying lesson, whiteboard, activity and classroom engines must power both standalone and embedded experiences. Do not build separate product logic for embedded mode.

## 3.3 Lesson creation through multiple channels

Lessons must be creatable directly inside TutorGrid Lesson Studio and through authorized external applications using TutorGrid APIs. The internal domain model must not assume all content originates from the TutorGrid user interface.

## 3.4 Branding and white-label

Support three branding levels:
1. **TutorGrid branding** — standard TutorGrid identity.
2. **Customized branding** — customer logo, colors and selected brand settings.
3. **Full white-label** — embedded experience presented under the customer's brand, subject to licensing and product-plan rules.

Branding must be tenant-aware and consistently applied across supported embedded experiences.

---

# 4. CORE PRODUCT ENGINES

## 4.1 Lesson Studio

Lesson Studio is a sophisticated instructional design and lesson-authoring studio. It must be treated as a major product engine, not a simple form or text editor.

It combines:
- Lesson planning.
- Instructional design.
- Rich document authoring.
- Slide/page authoring.
- Lesson sequencing and timeline design.
- Whiteboard preparation.
- Interactive activity and game authoring.
- Assessment design.
- Resource management.
- Preview, publishing and sharing.
- Reusable content libraries.
- Future AI-assisted authoring.

A lesson is a structured teaching experience, not merely a file or page of text.

### 4.1.1 Lesson planning and instructional design

A lesson may contain:
- Title and description.
- Subject.
- Grade or educational level.
- Age range.
- Topic and subtopics.
- Learning objectives.
- Expected learning outcomes.
- Prerequisite knowledge.
- Teaching methodology.
- Required resources.
- Estimated duration.
- Assessment criteria.
- Teacher preparation notes.
- Curriculum references or mappings where configured.

These fields should be extensible. Do not hard-code one curriculum's terminology or structure into the domain model.

### 4.1.2 Structured lesson composition

A lesson must support nested and ordered structures, including sections, subsections, steps, pages/slides, content blocks, activities, assessments, whiteboard documents and attached resources.

Typical instructional stages may include:
- Introduction.
- Explanation.
- Demonstration.
- Guided practice.
- Independent practice.
- Assessment.
- Conclusion or recap.

Educators must be able to add/remove sections and steps, reorder them, duplicate them, nest them where appropriate, move content between sections, save reusable sections or structures, and preview the entire lesson as a coherent experience.

Do not force every lesson into one fixed teaching sequence.

### 4.1.3 Rich content authoring

Support appropriate combinations of:
- Rich text.
- Headings and formatted paragraphs.
- Ordered and unordered lists.
- Tables.
- Images.
- Audio and video.
- Mathematical notation and equations.
- Diagrams.
- Charts and graphs.
- Code snippets where appropriate.
- Interactive content.
- Activities and games.
- Assessments.
- Whiteboard content.
- Attached resources and reference files.

Use structured content blocks rather than storing the entire lesson as an opaque HTML blob.

### 4.1.4 Multiple authoring interfaces over one lesson model

Provide multiple complementary ways to work with the same structured lesson:
- Document-oriented editor.
- Slide/page-oriented editor.
- Instructional sequence or timeline workspace.
- Whiteboard authoring workspace.
- Activity/game builder.
- Assessment designer.

These are different authoring interfaces over one coherent lesson model. Avoid disconnected lesson formats that cannot work together.

### 4.1.5 Whiteboard authoring within Lesson Studio

Teachers must be able to prepare diagrams, worked examples, annotations, shapes, visual objects, mathematical working, science illustrations, labels, explanations and interactive objects before class.

Prepared whiteboard content must be available during live teaching.

A live classroom must work with a session-specific copy or version of prepared content. Live annotations and learner interactions must not accidentally overwrite the original prepared lesson.

### 4.1.6 Activity and game authoring within Lesson Studio

Educators should be able to select a built-in activity template, add or edit questions and educational content, add images or assets, configure instructions, difficulty, rules, interaction and scoring options, preview and test the activity, save it for reuse, insert it into a lesson, and use an advanced visual builder to combine reusable components and interactions without code.

### 4.1.7 Assessment designer

Support:
- Multiple-choice questions.
- True/false questions.
- Short-answer questions.
- Long-answer questions.
- Matching.
- Ordering and sequencing.
- Fill-in-the-blanks.
- Mathematics problems.
- Image-based questions.
- Interactive exercises.
- Quizzes and tests.

Assessment configuration should support scoring, correct answers and accepted alternatives, feedback, explanations, attempts and completion rules, teacher review, manual grading where needed, results and reporting.

### 4.1.8 Resource library

Provide a resource library for images, videos, audio, documents, whiteboard materials, diagrams, activities, games, questions and question banks, content blocks, reusable sections and lesson templates.

Users should be able to reuse authorized resources across lessons without unnecessary duplicate copies.

### 4.1.9 PDF and PowerPoint handling

Teachers must be able to upload and attach PDF and PowerPoint files to lessons, open and display them during a lesson, present them to learners, annotate or draw over them using the whiteboard, and keep them attached as reference material.

Do not automatically convert uploaded PDF or PowerPoint files into editable TutorGrid lesson content. They remain attached reference files unless a separately designed import/conversion feature is approved.

### 4.1.10 Reusable libraries and ownership

Support reusable complete lesson templates, lesson sections, content blocks, whiteboard documents, activities and games, assessments, question banks, diagrams and resource collections.

Personal libraries are required as a product capability. Authorized shared libraries may be introduced as the sharing and permissions model matures.

Design ownership, reuse, versioning and access rules carefully. Reusing an item must not silently grant edit rights to someone else's original resource.

### 4.1.11 Lesson Studio interface direction

The intended workspace is a sophisticated studio with adaptive panels:

- **Left panel:** lesson outline, sections, subsections and pages.
- **Main workspace:** current document, slide, canvas or authoring surface.
- **Right panel:** selected element properties, formatting and settings.
- **Bottom panel:** timeline, sequencing or interaction controls when relevant.
- **Top toolbar:** save status, undo/redo, preview, sharing and publishing controls.

The interface should adapt to the current authoring mode. Do not display every possible tool at once.

The design should feel like a professional creative and instructional design environment while remaining usable by educators.

### 4.1.12 Lesson domain concepts

Keep these concepts distinct:
- **Lesson:** complete structured teaching experience.
- **Resource:** attached file or reusable media asset.
- **Content block:** instructional content element.
- **Whiteboard document:** interactive visual material.
- **Activity:** playable educational experience.
- **Assessment:** structured evaluation.
- **Template:** reusable starting structure.

## 4.2 Whiteboard Engine

The Whiteboard Engine is a core TutorGrid capability, not a small annotation feature. It must support visual explanation, collaborative teaching, learner participation, interactive educational objects and game/activity integration.

### 4.2.1 Renderer strategy

The long-term strategic direction is a hybrid renderer architecture:
- **Open-source option:** Konva.js with React Konva, MIT-licensed.
- **Commercial option:** tldraw SDK, subject to its commercial licensing terms.

The strategic intention is to support both through a TutorGrid-owned document model and shared educational feature architecture, rather than creating two unrelated whiteboard products.

**Open decision:** The initial/MVP renderer has not been explicitly selected. Do not assume Konva or tldraw is approved as the initial renderer. Do not implement both immediately without an approved need and validated architecture.

Before using tldraw commercially, verify in writing that the applicable license permits the intended multi-tenant SaaS use, embedding, redistribution, white-label use, and customer/end-user access model.

### 4.2.2 Whiteboard architecture boundaries

The intended conceptual modules are:
- `whiteboard-core` — document schema, object types, validation, commands and history.
- `whiteboard-konva` — Konva renderer and adapter.
- `whiteboard-tldraw` — tldraw renderer and adapter.
- `whiteboard-collaboration` — synchronization and collaborative state.
- `whiteboard-education` — educational tools and learning-specific objects.
- `whiteboard-sdk` — integration and embedding surface.

These are logical boundaries. They do not automatically require separate deployable services.

The TutorGrid-owned document schema must not be tied to a renderer's internal scene graph. Both renderers should consume and update the same conceptual document format through adapters.

Provide a capability contract and conversion/import/export tests. Be explicit that not every advanced renderer-specific feature may be losslessly portable.

### 4.2.3 Canvas and page model

Support page-based lesson organization, expandable or infinite-like canvas space within a page, zoom, pan, fit-to-screen, page backgrounds, grids and templates, and teacher navigation between pages during a live lesson.

Page operations:
- Create.
- Delete.
- Duplicate.
- Rename.
- Reorder.
- Save.
- Restore.

### 4.2.4 Drawing and object tools

Support pen and pencil, highlighter, eraser, shapes, lines and arrows, polygons, text boxes and formatting, mathematical notation, selection and movement, resize and rotate, duplicate and delete, layers, grouping and ungrouping, locking, alignment, undo/redo, copy and paste.

### 4.2.5 Educational visual tools

The whiteboard should be extensible to mathematical equations, number lines, geometry tools, graphs, science diagrams, mind maps, flowcharts, charts, labels, visual demonstrations and subject-specific manipulatives.

Where useful, these should be structured educational objects rather than untyped drawings.

### 4.2.6 Media and document objects

Support relevant combinations of images, PDF pages/documents, presentation slides, audio, video and other authorized lesson resources.

Media and document objects should integrate with the storage/resource model and have appropriate access permissions.

### 4.2.7 Collaboration and permissions

Support configurable classroom collaboration modes:
- Teacher-only editing.
- Learner view-only.
- Learner drawing or annotation.
- Learner manipulation of selected objects.
- Individual learner workspaces.
- Shared collaborative workspace.

Permissions should be controllable at suitable levels, including classroom/session, page or workspace, activity, and object or selected object groups where appropriate.

Teachers must be able to grant and revoke learner permissions during a session.

Do not rely only on hiding controls in the frontend. The backend and collaboration layer must enforce authorization.

### 4.2.8 Real-time collaboration and resilience

Synchronize drawing operations, object creation and changes, page navigation, activity interactions and permission changes.

Account for reconnection, temporary network failure, concurrent editing, operation ordering, recovery of unsaved work, snapshots and incremental operations, and authorization changes during a session.

Selected direction:
- CRDT-based collaboration principles for collaborative document changes.
- Server-authoritative permissions and event ordering.
- Snapshots plus incremental operations for persistence.

The exact CRDT library and detailed conflict-resolution implementation remain open unless separately approved. Do not claim a particular CRDT implementation has been selected.

### 4.2.9 Undo and redo

Selected behavior:
- Individual undo by default.
- Teacher-controlled global undo/redo where appropriate.

Distinguish a user's own operation history from shared document history and define safe behavior for concurrent edits.

### 4.2.10 Games and activities as whiteboard objects

Games and activities are first-class whiteboard objects.

Teachers should be able to insert an activity into the whiteboard, position and resize it, configure it, launch it, monitor learner responses, pause and resume it, end it, and review results.

An activity retains its own runtime logic and communicates with the whiteboard and classroom through defined interfaces. Do not tightly couple the game engine to one whiteboard renderer.

## 4.3 Educational Game and Activity Engine

The Game/Activity Engine is a major reusable educational engine designed for Pre-K through Grade 12 and extensible beyond it.

It supports both TutorGrid-built activities and teacher-created activities.

### 4.3.1 Activity distribution modes

Activities must work individually, in multiplayer, inside a live classroom, inside the whiteboard, as standalone experiences outside a live class, and embedded in an external platform where authorized.

Individual and multiplayer are capabilities of the same engine, not separate products.

### 4.3.2 Subjects and educational coverage

Support mathematics, English and other languages, reading and writing, phonics, science, biology, chemistry, physics, geography, history, computer science, critical thinking, general knowledge and extensible additional subjects.

Curriculum and subject taxonomies must be configurable and extensible.

### 4.3.3 Age and grade appropriateness

Examples of intended coverage:

**Pre-K / early childhood**
- Colors.
- Shapes.
- Alphabet and phonics.
- Counting.
- Matching.
- Sorting.
- Memory activities.

**Kindergarten to Grade 2**
- Basic arithmetic.
- Reading.
- Spelling.
- Patterns.
- Introductory science.

**Grades 3–5**
- Multiplication and division.
- Fractions and decimals.
- Grammar and vocabulary.
- Science and geography.

**Grades 6–8**
- Algebra and geometry foundations.
- Physics and chemistry foundations.
- Biology.
- Critical thinking.

**Grades 9–12**
- Advanced mathematics.
- Calculus and statistics.
- Physics simulations.
- Chemistry and biology.
- Complex problem-solving.

These are examples, not rigid grade restrictions. Content creators must configure grade, age, subject, difficulty and learning objectives.

### 4.3.4 Activity types

**Question and assessment activities:** multiple choice, true/false, short answer, fill in the blanks and timed quizzes.

**Manipulation and organization:** drag and drop, matching, sorting, categorization, sequencing and labeling.

**Games and puzzles:** memory games, puzzles, word games, spelling games, number games and problem-solving games.

**Interactive learning:** simulations, graph interactions, mathematical manipulatives, science experiments, visual demonstrations and interactive diagrams.

The engine must support future activity types without requiring a rewrite of the classroom or whiteboard engines.

### 4.3.5 Reusable engine architecture

Keep these conceptual responsibilities separate:

1. **Educational content** — questions, objectives, instructions, subject, grade and difficulty.
2. **Activity definition** — activity type, configuration, assets, rules and interactions.
3. **Game runtime** — rendering, inputs, state, timers, scoring and feedback.
4. **Multiplayer coordination** — participants, shared state and synchronized interactions.
5. **Delivery integration** — standalone player, whiteboard and live classroom.

### 4.3.6 Activity authoring levels

**Level 1 — Template-based:** select an activity template, add questions/images/instructions, configure rules and difficulty, preview and save.

**Level 2 — Component configuration:** configure reusable activity components, combine supported interactions and customize behavior through controlled configuration.

**Level 3 — Advanced visual builder:** visually combine reusable components and interactions, build sophisticated activities without programming, preview, test and save reusable definitions.

Do not execute arbitrary uploaded code in learner classrooms. Teacher-created activities should use controlled components and validated configuration. A custom activity/plugin SDK may be considered later as a separately governed capability.

### 4.3.7 Gameplay modes

Support individual practice, teacher-led activities, shared collaborative problem-solving, group activities, competitive individual challenges, team challenges, timed challenges where pedagogically appropriate and leaderboards where appropriate.

Teacher-led mode lets the teacher launch and control an activity while learners respond and the teacher monitors participation.

Do not force competition, timers or leaderboards into every activity. These are configurable educational mechanics.

### 4.3.8 Results and learning data

Capture relevant results such as learner responses, correct and incorrect answers, scores, completion status, attempts, time spent where relevant, progress and group outcomes.

Results should be available to authorized teachers for instruction, feedback and evaluation.

Avoid collecting unnecessary personal data. Define retention and access policies for educational records.

---

# 5. LIVE CLASSROOM

Live Classroom is the real-time delivery environment that combines Lesson Studio output, Whiteboard Engine, Game/Activity Engine and media services.

## 5.1 Teacher capabilities

Teachers should be able to start and end classroom sessions, invite learners, control lesson navigation, present lesson content, use the whiteboard, launch and control activities, grant and revoke learner permissions, monitor learner participation and responses, switch between guided and free navigation, move to any lesson section, skip or repeat activities, and change the teaching flow during a live session.

## 5.2 Learner capabilities

Learners should be able to join through an authorized invitation or access flow, view lesson content, participate in audio/video where permitted, interact with the whiteboard where permitted, draw or annotate where permitted, manipulate selected objects where permitted, play activities, respond to assessments and collaborate with other learners where enabled.

## 5.3 Guided and free navigation

Both modes are required:
- **Guided mode:** teacher controls lesson progression.
- **Free mode:** learners may navigate within configured permissions and boundaries.

The teacher must be able to switch modes during a live session and move to any section, skip or repeat activities, and adapt the teaching flow.

## 5.4 Prepared lesson versus live session

A classroom session must pin a specific immutable published lesson version at launch.

Changes made later to the source lesson must not silently change an active classroom.

Live classroom changes and annotations must be session-specific or otherwise isolated from the prepared source lesson. The teacher should deliberately preserve relevant classroom work rather than accidentally mutate the original lesson.

## 5.5 Live Classroom communication features

The intended experience includes audio, video, camera controls, microphone controls, screen sharing, chat, connection status, participant presence, classroom lifecycle controls, and lesson and activity synchronization.

The application should communicate meaningful connection state and recover gracefully from temporary network disruption.

---

# 6. AUTHENTICATION, IDENTITY, TENANTS AND AUTHORIZATION

## 6.1 Authentication technology

The selected authentication library is **Limen**.

Documentation: https://limenauth.dev/docs/

Selected social authentication providers:
- Google OAuth.
- GitHub OAuth.

Traditional email/password authentication was explicitly rejected for the selected approach.

Verify the current Limen Go integration, session handling, supported OAuth flows, React client SDK and SQL adapter behavior against official documentation during implementation.

Limen's SQL adapter uses `database/sql`. Verify how it shares or coexists with the application's Bun/database connection pool. Do not assume compatibility details without testing.

## 6.2 Authentication versus application authorization

Limen handles authentication and session-related responsibilities. TutorGrid still owns:
- User profiles.
- Organizations and tenants.
- Organization memberships.
- Roles.
- Permissions.
- Resource ownership.
- Classroom membership.
- Tenant boundaries.
- Application authorization.

Authentication success does not automatically grant access to a tenant's lessons, resources, classrooms or other data.

## 6.3 Authorization model

Selected direction:
- Role-based access control (RBAC).
- Resource-level permissions.

Authorization must be enforced in the Go backend and supported by PostgreSQL Row-Level Security where appropriate.

The frontend may hide unavailable controls for usability, but the backend is authoritative.

## 6.4 Multi-tenancy

Selected progressive/hybrid strategy:
- Shared database and shared schema with `tenant_id` initially.
- Tenant-aware data access and isolation from the beginning.
- PostgreSQL RLS as defense in depth.
- Dedicated database or stronger isolation for large enterprise tenants later if justified.

For Bun and PostgreSQL RLS, tenant context must be transaction-scoped and reliably set for every relevant database operation. Avoid leaking tenant context across pooled connections or transactions.

## 6.5 External identity and partner access

The broader product vision includes external identity integration for partner platforms, including OIDC/SSO where approved.

The exact scope and timing of external OIDC/SSO remain open. Google and GitHub OAuth are the confirmed selected social login providers; do not assume they alone satisfy every future partner identity requirement.

## 6.6 Learner guest access

Guest learner access or classroom joining without a full OAuth account has been discussed but is not confirmed as a final decision.

Treat it as an open product/security decision. Do not implement unrestricted anonymous access. Any guest or invitation-based flow must use short-lived, scoped, revocable access credentials and clear tenant/classroom authorization.

---

# 7. REAL-TIME APPLICATION COMMUNICATION

## 7.1 Selected technology: Centrifugo

Centrifugo is the selected client-facing real-time communication infrastructure.

Use it for appropriate application-level real-time capabilities such as classroom events, presence, lesson/session updates, whiteboard synchronization delivery, activity interaction updates and notifications/real-time client state.

Centrifugo is not the WebRTC media server and must not be treated as a replacement for LiveKit.

## 7.2 Redis integration

Redis may be used as the Centrifugo broker/presence backend where appropriate.

Design logical workload separation, credentials, capacity and monitoring so Centrifugo does not unintentionally compete with critical application cache or classroom-state workloads.

## 7.3 Channel authorization

Channel access must be authorized by the backend. Do not trust a client to subscribe to arbitrary tenant or classroom channels.

Use short-lived credentials and validate user/participant identity, tenant, classroom/session membership, role, resource scope and current permissions.

The precise Centrifugo token and channel configuration must be validated against the selected deployment and version.

## 7.4 Event architecture

Selected architecture:
- **NATS JetStream:** internal durable event backbone and background processing.
- **Centrifugo:** client-facing real-time event delivery.
- **Go backend:** authoritative business logic and authorization.

Do not duplicate durable job queue responsibilities in Redis when NATS JetStream already owns durable jobs/events.

## 7.5 Event delivery guarantees

Use a hybrid guarantee model:
- Critical business events: effectively-once business outcomes through idempotency, deduplication and durable processing.
- Notifications: at-least-once where appropriate.
- Presence and typing indicators: best effort.

Do not claim end-to-end exactly-once delivery. Design critical business operations to remain correct under retries and duplicate event delivery.

## 7.6 Transactions and events

Use PostgreSQL transactions with a transactional outbox for events that must remain consistent with committed database changes.

NATS consumers should be idempotent and support retries and failure handling.

## 7.7 Synchronization frequency

Use adaptive synchronization:
- Batch or coalesce high-frequency drawing and movement updates.
- Deliver critical permission and classroom events promptly.
- Persist durable state separately from transient high-frequency updates.

Do not write every pointer movement or drawing sample as an individual PostgreSQL transaction.

---

# 8. VIDEO, AUDIO AND WEBRTC — LIVEKIT

## 8.1 Mandatory media technology

**TutorGrid must use LiveKit** for its WebRTC audio/video infrastructure.

The earlier custom-WebRTC approach is superseded. Do not replace LiveKit with an unrelated custom SFU implementation.

## 8.2 Deployment strategy

Selected strategy:
- LiveKit Cloud initially.
- Provider-independent TutorGrid integration.
- Evaluate and move to self-hosted LiveKit later when justified.
- Hetzner Cloud is the selected future hosting candidate for self-hosted LiveKit.

Self-hosted LiveKit infrastructure has a separate media/network budget and should not be hidden inside ordinary application hosting estimates.

## 8.3 Separation of responsibilities

**LiveKit**
- WebRTC media transport.
- SFU functionality.
- Audio/video rooms and participants.
- Screen-sharing media.
- Media publishing and subscription.
- Network adaptation and reconnection support.

**TutorGrid Go backend**
- Classroom lifecycle.
- Room creation and management.
- Participant authorization.
- Short-lived access tokens.
- Media permissions.
- Lesson and session state.
- Educational business events.

**Centrifugo**
- Application-level real-time messaging.
- Presence and classroom state notifications.
- Whiteboard/activity event delivery where appropriate.

## 8.4 Room model

Selected room design:
- One primary LiveKit room per active classroom session.
- Breakout rooms and optional specialized rooms as broader architectural capabilities.
- Go controls room lifecycle, membership and token issuance.
- PostgreSQL stores durable room and session metadata.
- TutorGrid permissions remain authoritative.

## 8.5 Token and security model

Selected direction:
- Go backend issues short-lived LiveKit tokens.
- Validate tenant, classroom, participant identity and role.
- Grant minimal publish/subscribe/data permissions.
- Room creation and administrative authority remain backend-controlled.
- Support authenticated token refresh.
- Remove or revoke access when authorization changes.

Never embed LiveKit API secrets in browser code or expose server credentials to clients.

## 8.6 Media permissions

The following was recommended but not explicitly confirmed as a final user decision:
- Teacher may publish audio, video and screen sharing.
- Learners join with media disabled by default.
- Teacher grants or revokes learner microphone, camera and screen-sharing permissions.
- Backend enforces changes.
- Support individual, group and breakout-room policies.
- Synchronize LiveKit permissions with classroom roles.

Keep this as a recommendation/open decision until approved.

---

# 9. STORAGE, FILES AND MEDIA PROCESSING

## 9.1 Object storage

The selected object storage provider is **Backblaze B2**, using its S3-compatible API.

Build a TutorGrid-owned storage abstraction so application logic is not coupled to one provider's SDK or endpoint.

Support lesson attachments, PDFs, PowerPoint files, images, audio, video, whiteboard assets, activity resources, previews, exports and future recordings.

## 9.2 Direct upload architecture

The selected upload approach is direct browser-to-object-storage upload using presigned URLs.

Agreed flow:
1. Client requests an upload from the Go backend.
2. Backend validates the permitted MIME/type policy and file-size limit.
3. Backend creates a pending, tenant-scoped resource record.
4. Backend creates a short-lived presigned PUT URL.
5. Client uploads directly to Backblaze B2.
6. Client calls the backend completion endpoint.
7. Backend verifies the uploaded object using `HeadObject` or provider-equivalent verification.
8. Backend marks the resource ready only after verification.
9. Downloads use short-lived presigned GET URLs.

The discussed initial allowlist includes PNG, JPEG, WebP, PDF, PPT and PPTX. Raw SVG uploads are not allowed in the discussed initial allowlist.

The discussed initial file-size cap is 25 MB. This is a proposed initial policy, not a permanent platform-wide maximum. Larger video or other media workflows may require separate policies.

## 9.3 Upload security safeguards

Implement:
- Private buckets.
- Restricted CORS origins.
- Unpredictable server-generated object keys.
- No trust in client-supplied storage paths.
- Actual file-content/type verification, not only client MIME headers.
- Tenant ownership validation on upload, completion, retrieval and deletion.
- Idempotent completion handling.
- Cleanup of abandoned pending uploads.
- No exposure of B2 credentials.
- Malware scanning before sensitive files are made available where appropriate.
- Short-lived presigned URLs.
- Audit logging for relevant file operations.

## 9.4 Local development storage

The selected local development/testing strategy is **MinIO** in a local container.

Use the same S3-compatible Go storage adapter for local MinIO and staging/production Backblaze B2.

Configure endpoint, bucket, credentials and region through environment configuration.

Requirements:
- Add MinIO to local development Compose setup.
- Keep credentials out of source control.
- Test presigned PUT, completion verification, GET, deletion and tenant isolation.
- Provide a Backblaze B2 setup guide covering private bucket, S3-compatible application key, CORS and staging verification.
- Ensure production cannot accidentally use a MinIO endpoint.
- Do not create separate MinIO and B2 application implementations.

MinIO testing does not fully prove real Backblaze B2 CORS and presigned URL behavior. A real B2 integration test is required before relying on the staging/production configuration.

## 9.5 Media processing

Selected hybrid self-managed processing:
- FFmpeg.
- LibreOffice.
- Go libraries.
- Optional external processing services later where they provide a clear benefit.

Use asynchronous NATS workers for heavier processing. Perform lightweight metadata and validation immediately where practical.

---

# 10. BACKGROUND JOBS AND EVENT PROCESSING

## 10.1 Selected technology: NATS JetStream

NATS JetStream is the selected background job and durable event technology.

Use it for appropriate asynchronous work such as media processing, file scanning, document processing, resource lifecycle tasks, notifications, analytics event pipelines and other durable background workflows.

## 10.2 Design principles

- Durable streams and consumers where required.
- Explicit retry and dead-letter/failure handling.
- Idempotent consumers.
- Observability for queue/consumer lag and failures.
- Transactional outbox for database-to-event consistency.
- Avoid duplicating NATS durable queue responsibilities in Redis.

A small worker allocation or deployment is sufficient when justified; do not introduce an unnecessarily elaborate distributed worker fleet without workload evidence.

---

# 11. DATABASE AND DATA ARCHITECTURE

## 11.1 Primary database

**PostgreSQL** is the selected primary operational database.

## 11.2 Go database access

**Bun** is the selected Go database access/ORM layer. Use it consistently for application data access while retaining the ability to use carefully scoped SQL where necessary.

## 11.3 Migrations

**Goose** is the selected database migration tool.

Migrations must be version-controlled, reviewed, tested in CI, run as a dedicated deployment step, compatible with rolling application deployment where applicable, and backed by backup/rollback planning.

Prefer roll-forward fixes for production schema changes when safe. Use expand-and-contract migration practices for compatibility.

## 11.4 PostgreSQL data modelling

Selected direction is hybrid relational modelling plus JSONB where appropriate.

Use relational columns, constraints and relationships for core domain invariants. Use JSONB for genuinely flexible structured content and configuration, not as an excuse to store the entire application as unvalidated blobs.

## 11.5 Schema organization

Selected direction:
- Shared schema initially.
- Clear domain/module boundaries.
- Separate schemas for major bounded contexts later if justified.

Do not mistake logical module boundaries for a requirement to create separate databases or microservices.

## 11.6 Tenant isolation

Use `tenant_id` on tenant-owned data, Go application authorization, PostgreSQL RLS as defense in depth, transaction-scoped tenant context and tenant-aware tests/query patterns.

The application must not trust a tenant identifier supplied by the browser without validating the authenticated user's membership and authorization.

## 11.7 Connection pooling

Use Go's `database/sql` connection pool initially. Add PgBouncer when connection pressure, deployment topology or provider requirements justify it.

## 11.8 Database transactions and integrity

Use database constraints, domain validation, transactions, idempotency, transactional outbox and reconciliation processes where required.

Use optimistic locking for suitable concurrent updates and selective row locks where required. Whiteboard collaboration should use the selected synchronization model rather than forcing every edit through long-running database locks.

## 11.9 Database performance and scaling

Progressive scaling direction:
- Vertical scaling first.
- Read replicas when justified.
- Partitioning when data volume/query patterns justify it.
- Materialized views where useful.
- Dedicated tenant databases or sharding only when justified by enterprise or scale requirements.

Do not pre-emptively add partitioning, replicas or sharding without measurable need.

## 11.10 PostgreSQL availability and backups

Selected direction:
- Managed PostgreSQL initially.
- Backups and point-in-time recovery where supported.
- Continuous WAL/PITR where supported.
- Daily backups.
- Integrity checks and restore tests.
- Provider backups plus encrypted Backblaze B2 copies.
- Future cross-region disaster recovery as the platform matures.

## 11.11 Recovery targets

Planning targets recorded:

| Stage | RPO target | RTO target |
|---|---:|---:|
| Initial pilot | 24 hours | 24 hours |
| Growth | 1 hour | 4 hours |
| Mature platform | 15 minutes or better | 1 hour or better |

These are planning targets, not a claim that the current hosting configuration already meets them. Validate provider capabilities and test restoration.

## 11.12 Database security and audit

Use TLS in transit, provider/database encryption at rest, application-level encryption for selected sensitive fields where justified, key management and rotation, encrypted backups, comprehensive application audit trail, selective PostgreSQL-level auditing, and monitoring for queries, locks, deadlocks, pool pressure and indexes.

## 11.13 Data retention and portability

Selected direction:
- Configurable retention policies by data category and tenant.
- Backblaze B2 archival where appropriate.
- Legal holds where required.
- Controlled deletion.
- Complete structured exports initially.
- Enterprise APIs and bulk migration capabilities later.

Data export must preserve useful structure and relationships, not only provide a folder of unlabelled files.

---

# 12. REDIS ARCHITECTURE

## 12.1 Selected technology and hosting direction

Redis is the selected caching and temporary-state technology.

Hosting direction:
- Northflank Managed Redis initially.
- Logical workload separation.
- Dedicated instances as capacity or isolation needs grow.
- Redis Cluster later only when justified.

## 12.2 Workload responsibilities

Use Redis for appropriate shared cache, short-lived classroom state, game/activity temporary state, rate limiting, temporary coordination and locks, and presence/broker workloads where appropriate for Centrifugo.

PostgreSQL remains authoritative for durable application records. NATS JetStream remains the durable event/job backbone.

## 12.3 Persistence and eviction

Use workload-specific persistence:
- Disposable caches do not need to become authoritative durable data.
- RDB/AOF selectively where the workload benefits.
- Critical classroom/game state must have a deliberate recovery and durability design.
- Workload-specific eviction policies.

Use LRU/LFU for suitable cache workloads and TTL/controlled eviction for temporary classroom/game state. Protect critical state from accidental eviction.

## 12.4 Key design

Use structured key names including relevant tenant, module, resource, version or namespace, and expiration policy.

Do not put secrets in keys. Use Redis Cluster hash tags deliberately if Cluster is later adopted.

Tenant prefixes are organizational boundaries, not a security boundary. Backend authorization remains mandatory.

## 12.5 Security

Use private networking, TLS where supported, strong authentication, ACLs and per-workload credentials, credential rotation, monitoring and no public exposure.

## 12.6 Caching and invalidation

Selected direction:
- Cache-aside for lessons, resources and templates.
- TTL as a safety mechanism.
- Explicit invalidation.
- Transactional outbox and NATS events for reliable invalidation where needed.
- Versioned keys where useful.

PostgreSQL remains authoritative.

## 12.7 Atomic operations and rate limiting

Use native atomic commands, MULTI/EXEC/WATCH and Lua scripts where appropriate.

Rate limiting direction:
- Token bucket generally.
- Sliding-window or stricter limits for sensitive endpoints.
- Policies scoped by user, tenant, API key, endpoint and plan.

## 12.8 Monitoring and capacity

Use Prometheus, Grafana, OpenTelemetry and Sentry as appropriate.

Monitor workload-specific memory, latency, evictions, connections, hit rates and saturation. Scale vertically, split workloads or add Cluster only when evidence justifies it.

---

# 13. SEARCH

The selected dedicated search engine is **Elasticsearch**.

Use it for search needs that exceed appropriate PostgreSQL search capabilities.

Keep the operational database authoritative. Search indexes should be rebuildable and updated through reliable event-driven indexing.

The exact search scope, mappings, indexing strategy, hosting and cost model should be refined as searchable content and query requirements become concrete.

---

# 14. ANALYTICS AND EDUCATIONAL REPORTING

## 14.1 Analytics architecture

Selected hybrid direction:
- PostgreSQL for operational records and initial/basic reporting.
- ClickHouse for high-volume educational analytics as event volume grows.

## 14.2 Real-time educational analytics

Selected real-time direction:
- NATS for event flow.
- Redis for temporary/live dashboard state.
- Centrifugo for live dashboard updates.
- ClickHouse for historical and analytical queries.

## 14.3 Educational information

Where authorized and appropriate, analytics may include participation, activity completion, responses, correct/incorrect answers, scores, attempts, time spent where relevant, learning progress and group outcomes.

Do not turn TutorGrid into a general school administration or student information system. Analytics should support the teaching and learning experiences TutorGrid provides.

Design data minimization, tenant access, retention and export policies into analytics.

---

# 15. CONTENT VERSIONING AND PUBLISHING

Selected versioning direction:
- Editable drafts.
- Immutable published lesson versions.
- Automatic revision history.
- Active classroom sessions pinned to a specific lesson version.
- Advanced branching and merge capabilities later if justified.

Publishing must create a stable version that can be used consistently by a classroom.

A change to a draft or a newer published version must not silently alter an active session already using an earlier version.

Apply versioning concepts to other reusable educational assets where appropriate, including activities and templates.

---

# 16. API, CONTRACTS AND INTEGRATION ARCHITECTURE

## 16.1 Selected API strategy

Selected API architecture is hybrid:
- REST APIs.
- OpenAPI documentation/contracts.
- gRPC for appropriate internal or service-to-service use cases.

Do not use gRPC indiscriminately for browser-facing integration where REST is more appropriate.

## 16.2 Versioning

Selected direction:
- URL-based major API versions.
- Backward-compatible additions within a version.
- Explicit deprecation and migration policies.

## 16.3 API scope

The long-term API surface may include lesson creation and updates, lesson publishing and version retrieval, resource management, whiteboard documents, activity creation and management, assessments, classroom lifecycle, participants and permissions, lesson sharing, session events, activity results, branding configuration, and tenant/integration management where authorized.

## 16.4 Webhooks and SDKs

Provide webhooks for relevant lifecycle and event integrations.

Provide SDKs and embedding interfaces that make it practical for external education products to use TutorGrid engines without recreating TutorGrid's internal business logic.

Security, tenant isolation, short-lived access and clear API contracts are mandatory.

## 16.5 Rate limiting

Use tiered rate limiting by user, tenant, API key, endpoint and plan. Redis is the selected shared rate-limiting technology.

---

# 17. APPLICATION ARCHITECTURE AND CODE ORGANIZATION

## 17.1 Backend architecture

The selected backend architecture is a **modular monolith**, with potential service extraction later when justified.

Do not start with microservices simply because TutorGrid contains multiple product engines.

Keep domain boundaries explicit so components can evolve independently and selected workloads can be extracted later if needed.

## 17.2 Backend language and framework

- Language: **Go**.
- HTTP framework: **Echo**.
- Database: **PostgreSQL**.
- Database access: **Bun**.
- Migrations: **Goose**.

## 17.3 Frontend technology

- Frontend framework: **React**.
- Build/application tooling: **Vite**.
- UI component system: **shadcn/ui**.
- Mobile strategy: **responsive web**.

TypeScript was discussed as the natural pairing for React, but was not separately recorded as an explicit product-owner decision in the consolidated decisions. Treat TypeScript as a strong recommendation unless the repository contains a later explicit approval.

Tailwind CSS and Radix UI are commonly associated with the selected shadcn/ui approach, but do not represent them as independently approved decisions unless confirmed in the repository's ADR.

## 17.4 Repository strategy

The selected repository approach is a **single repository/monorepo**.

Use clear separation between backend application, frontend application, shared contracts and generated clients where appropriate, database migrations, infrastructure, documentation, agent rules and workflows, and tests.

A monorepo does not mean every module must be tightly coupled.

## 17.5 Deployment architecture

Selected progressive direction:
- Docker initially.
- Kubernetes later if justified by operational scale and requirements.

Do not introduce Kubernetes prematurely.

---

# 18. CLOUD HOSTING AND INFRASTRUCTURE

## 18.1 Application hosting

The selected hosting direction is a managed cloud platform with provider flexibility.

**Northflank is the preferred hosting candidate** for the initial application infrastructure. This is a preferred candidate, not an irrevocable claim that every component or feature has already been validated.

## 18.2 Managed data services and containers

Recorded hosting direction:
- PostgreSQL: Northflank Managed PostgreSQL.
- Redis: Northflank Managed Redis.
- NATS: Docker deployment on Northflank planned.
- Centrifugo: Docker deployment on Northflank planned.
- LiveKit Cloud initially.
- Future self-hosted LiveKit: Hetzner Cloud.
- Object storage: Backblaze B2.

Validate current provider features, networking, persistence, regions, backups, monitoring, pricing and operational constraints before production commitments.

## 18.3 CI/CD and registry

Selected:
- CI/CD: GitHub Actions plus Northflank deployment integration.
- Container registry: GitHub Container Registry (GHCR).

Pipelines should include suitable build, test, security and deployment checks. Database migrations must be handled as a deliberate deployment step.

## 18.4 Infrastructure as Code

**Terraform** is the selected Infrastructure as Code tool.

Keep infrastructure definitions version-controlled, reviewable and reproducible.

## 18.5 Secrets management

Selected hybrid direction:
- Northflank secrets initially.
- Dedicated secrets manager such as Vault later when justified.

Never commit credentials, signing keys, API secrets or production environment files to the repository.

## 18.6 Domain, DNS and edge

Selected hybrid direction:
- Domain registration through a registrar.
- Cloudflare for DNS, CDN and security capabilities.

## 18.7 Environments

Maintain distinct development, testing/staging and production environments, plus isolated temporary test databases/environments where useful.

Production credentials and data must not be reused in development or ordinary test environments.

---

# 19. OBSERVABILITY, SECURITY AND QUALITY

## 19.1 Observability

Selected hybrid observability stack:
- OpenTelemetry.
- Prometheus.
- Grafana.
- Sentry.

Use appropriate logs, metrics and traces across Go services, workers, real-time infrastructure, media integrations and frontend experiences.

Monitor both technical health and educational workflow reliability.

## 19.2 Testing strategy

The selected testing approach is comprehensive:
- Unit tests.
- Integration tests.
- End-to-end tests.
- API contract tests.
- Performance tests.
- Security tests.
- Regression tests.
- Browser tests.

Tests should cover tenant isolation, authorization, classroom lifecycle, lesson version pinning, whiteboard collaboration, activity behavior, uploads and recovery paths.

## 19.3 Security testing

Selected hybrid direction:
- Comprehensive automated security checks from the beginning.
- External penetration testing as the platform grows.

Include dependency and container scanning, secret detection, static analysis and appropriate application security tests.

## 19.4 Audit logging

Comprehensive audit trails are selected.

Record important security- and business-relevant actions such as access changes, publishing, sharing, classroom lifecycle changes, resource operations and administrative actions.

Audit data must be protected against unauthorized modification and access.

## 19.5 Encryption and secrets

Use TLS for network communication, provider encryption at rest, application-level encryption for selected sensitive fields where appropriate, encrypted backups, key rotation and restricted service credentials.

Do not treat encryption as a replacement for access control.

## 19.6 Backup and disaster recovery

Selected hybrid direction:
- Scheduled comprehensive backups initially.
- Provider backup and PITR capabilities where supported.
- Encrypted backup copies in Backblaze B2.
- Terraform-based infrastructure recreation.
- GitHub Actions/Northflank redeployment.
- Cloudflare DNS recovery.
- Restore and disaster-recovery simulations.
- Cross-region recovery capabilities later.

The recovery targets in Section 11 are planning targets and must be validated by testing.

---

# 20. CLASSROOM AND WHITEBOARD STATE OWNERSHIP

Use clear ownership boundaries:
- PostgreSQL: durable authoritative application records and session metadata.
- Redis: temporary active-session and live state where appropriate.
- Centrifugo: client-facing real-time delivery.
- LiveKit: media transport.
- NATS JetStream: durable internal event and background-job backbone.
- Go backend: authoritative business logic, permissions and lifecycle control.

Do not allow two systems to become competing sources of truth for the same durable business record.

Define recovery and reconciliation behavior for temporary state and real-time connections.

---

# 21. DATA GOVERNANCE, PRIVACY AND RETENTION

TutorGrid handles educational content and potentially learner-related activity data. Design privacy and governance into the platform.

Requirements:
- Tenant-scoped access.
- Least privilege.
- Clear resource ownership.
- Configurable retention by data category and tenant.
- Controlled deletion.
- Legal holds where required.
- Structured export and portability.
- Audit trails for sensitive actions.
- Appropriate protection for learner data.
- Data minimization.
- No unnecessary collection of personal data.

Do not infer that every education customer has the same regulatory requirements. The architecture should be capable of supporting applicable obligations without inventing compliance certifications.

---

# 22. AI CAPABILITIES — FUTURE PRODUCT DIRECTION

AI is a future capability and was not required for the initial product foundation.

The intended future AI assistant should be integrated primarily into Lesson Studio and may help educators:
- Generate lesson outlines.
- Suggest age-appropriate learning objectives.
- Draft explanations.
- Create examples.
- Generate practice questions.
- Suggest games and activities.
- Draft assessment questions.
- Adapt content for different grade levels.
- Improve or refine lesson materials.

AI output must be treated as a draft. The teacher reviews, edits and approves it. AI must not automatically publish content or silently replace teacher-authored material.

Design AI integrations behind controlled interfaces so providers can be evaluated or changed without coupling core lesson logic to one model provider.

---

# 23. CONSOLIDATED TECHNOLOGY DECISION REGISTER

| Area | Recorded choice | Status / qualification |
|---|---|---|
| Product type | Educational creation, teaching and learning platform | Confirmed vision |
| Core engines | Lesson Studio, Whiteboard Engine, Game/Activity Engine | Confirmed vision |
| Delivery | Live Classroom | Confirmed vision |
| Education levels | Pre-K through Grade 12 | Confirmed |
| Subjects | All major subjects, extensible | Confirmed |
| Curriculum | Flexible, not locked to one national curriculum | Confirmed |
| Devices | Desktop, laptop, tablet, mobile | Confirmed |
| Mobile approach | Responsive web | Confirmed |
| Distribution | Standalone and embedded/integrated | Confirmed |
| Lesson creation | TutorGrid UI and external APIs | Confirmed |
| Branding | TutorGrid, customized, full white-label | Confirmed capability direction |
| Whiteboard strategy | Hybrid Konva + tldraw long-term | Confirmed strategic direction |
| Whiteboard initial renderer | Not selected | Open |
| Whiteboard document model | TutorGrid-owned shared model | Confirmed direction |
| Whiteboard collaboration | CRDT principles, server-authoritative permissions/order | Confirmed direction; exact library open |
| Whiteboard persistence | Snapshots plus incremental operations | Confirmed |
| Whiteboard undo | Individual by default; teacher global controls | Confirmed |
| Game creation | Templates and advanced visual builder | Confirmed |
| Gameplay | Individual and multiplayer | Confirmed |
| Backend language | Go | Confirmed |
| Backend framework | Echo | Confirmed |
| Frontend framework | React | Confirmed |
| Frontend build tooling | Vite | Confirmed |
| UI system | shadcn/ui | Confirmed |
| TypeScript | Recommended pairing | Not independently confirmed |
| Primary database | PostgreSQL | Confirmed |
| Database access | Bun | Confirmed |
| Migrations | Goose | Confirmed |
| Cache/state | Redis | Confirmed |
| Authentication | Limen | Confirmed |
| Social OAuth | Google and GitHub | Confirmed |
| Email/password | No | Confirmed |
| Authorization | RBAC plus resource-level permissions | Confirmed |
| Real-time messaging | Centrifugo | Confirmed |
| Object storage | Backblaze B2, S3-compatible | Confirmed |
| Local object storage | MinIO container | Confirmed |
| Video/audio | LiveKit | Mandatory confirmed decision |
| LiveKit deployment | Cloud initially, self-host later | Confirmed |
| Future LiveKit hosting | Hetzner Cloud | Confirmed candidate |
| API architecture | REST + OpenAPI + gRPC | Confirmed hybrid |
| Background jobs/events | NATS JetStream | Confirmed |
| Observability | OpenTelemetry + Prometheus + Grafana + Sentry | Confirmed hybrid |
| Deployment | Docker initially, Kubernetes later | Confirmed progressive direction |
| Hosting platform | Managed provider; Northflank preferred | Preferred candidate; validate |
| PostgreSQL hosting | Northflank Managed PostgreSQL | Planned direction; validate |
| Redis hosting | Northflank Managed Redis | Planned direction; validate |
| NATS hosting | Docker on Northflank | Planned direction; validate |
| Centrifugo hosting | Docker on Northflank | Planned direction; validate |
| CI/CD | GitHub Actions + Northflank | Confirmed direction |
| Container registry | GHCR | Confirmed |
| Infrastructure as Code | Terraform | Confirmed |
| Secrets | Northflank initially; Vault later | Confirmed hybrid direction |
| DNS/CDN/security | Registrar + Cloudflare | Confirmed hybrid direction |
| Security testing | Automated checks; external pentest as platform grows | Confirmed hybrid |
| Backups/DR | Scheduled backups; cross-region later | Confirmed hybrid |
| Multi-tenancy | Shared DB/schema + tenant_id; dedicated DB later if needed | Confirmed hybrid |
| Backend architecture | Modular monolith; extract later if justified | Confirmed |
| Repository | Monorepo | Confirmed |
| Testing | Unit, integration, E2E, contract, performance, security, regression, browser | Confirmed |
| API versioning | URL major versions; compatible additions | Confirmed |
| Rate limiting | Tiered by user, tenant, API key, endpoint and plan | Confirmed |
| Audit logging | Comprehensive | Confirmed |
| Caching | Redis + Go memory + Cloudflare CDN + temporary live state | Confirmed hybrid |
| PostgreSQL modelling | Relational + JSONB | Confirmed hybrid |
| Search | Elasticsearch | Confirmed |
| Analytics | PostgreSQL initially; ClickHouse as volume grows | Confirmed hybrid |
| Real-time analytics | NATS → Redis → Centrifugo; ClickHouse history | Confirmed |
| Retention | Configurable policies; B2 archival; controlled deletion | Confirmed hybrid |
| Export | Structured exports initially; enterprise APIs/bulk migration later | Confirmed hybrid |
| Content versioning | Drafts, immutable published versions, revision history, session pinning | Confirmed hybrid |
| DB events | Transactional outbox + idempotent NATS consumers | Confirmed |
| Tenant enforcement | Go authorization + PostgreSQL RLS | Confirmed |
| DB pooling | database/sql pool; PgBouncer when justified | Confirmed |
| DB schema organization | Shared schema; separate schemas later if justified | Confirmed hybrid |
| DB migrations | Goose, CI tests, dedicated deploy step, expand/contract | Confirmed |
| DB environments | Dev, test/staging, production, isolated test DBs | Confirmed |
| DB scaling | Vertical first; replicas/partitioning/sharding when justified | Confirmed progressive |
| DB availability | Managed DB, backups/PITR, HA/failover as needed | Confirmed progressive |
| DB encryption | TLS, at-rest, selected field encryption, key rotation, encrypted backups | Confirmed |
| DB integrity | Constraints, validation, transactions, idempotency, reconciliation | Confirmed |
| DB concurrency | Optimistic locking, selective row locks, CRDT principles, ordering | Confirmed direction |
| DB compatibility | Expand-and-contract and contract tests | Confirmed |
| Redis hosting | Northflank managed initially | Confirmed direction; validate |
| Redis persistence | Workload-specific RDB/AOF | Confirmed |
| Redis eviction | Workload-specific LRU/LFU/TTL/controlled eviction | Confirmed |
| Redis key design | Structured tenant/module/resource namespaces | Confirmed |
| Redis security | Private network, TLS, ACLs, per-workload credentials | Confirmed |
| Redis HA | Managed initially; replicas/failover as growth requires | Confirmed progressive |
| Redis caching | Cache-aside, TTL, explicit invalidation | Confirmed |
| Redis invalidation | TTL + explicit invalidation + outbox/NATS + versioned keys | Confirmed |
| Redis tenant isolation | Namespaces plus backend authorization; dedicated instances later | Confirmed |
| Redis atomic operations | Native commands, transactions, WATCH, Lua | Confirmed |
| Redis rate limiting | Token bucket; sliding window for sensitive endpoints | Confirmed |
| Redis/NATS division | Redis temporary coordination; NATS durable jobs/events | Confirmed |
| Redis monitoring | Prometheus, Grafana, OpenTelemetry, Sentry | Confirmed |

---

# 24. OPEN DECISIONS — DO NOT ASSUME APPROVAL

Keep these items visibly open or recommended until the product owner confirms them or an approved ADR does so:

1. **Initial whiteboard renderer:** long-term hybrid Konva/tldraw is the strategy, but the initial renderer is not selected.
2. **tldraw licensing:** obtain written confirmation for multi-tenant SaaS, embedding, redistribution and white-label use.
3. **CRDT implementation:** CRDT principles are selected, but the exact library and detailed algorithm are not.
4. **Learner guest access:** invitation-based/guest access without full OAuth was discussed but is not confirmed.
5. **External OIDC/SSO:** included in the broader integration vision, but exact provider support and timing need confirmation.
6. **LiveKit learner media defaults and teacher-granted permissions:** recommended, not confirmed.
7. **TypeScript:** strongly recommended for React, but not independently recorded as an explicit user decision.
8. **Tailwind/Radix details:** validate against the actual approved shadcn/ui setup rather than treating each as a separate confirmed choice.
9. **Northflank capabilities:** validate managed PostgreSQL/Redis, private networking, persistent storage, NATS/Centrifugo deployment, backups, regions, monitoring, pricing and operational limits.
10. **Backblaze B2 integration:** validate real presigned URL and CORS behavior; MinIO alone is insufficient.
11. **Exact service boundaries:** modular monolith is selected; logical modules do not automatically imply separate services.
12. **Enterprise isolation model:** dedicated databases are a future option, not an immediate requirement.
13. **AI provider and model:** no provider has been selected.
14. **White-label licensing and commercial packaging:** product capability is confirmed, but exact entitlement and pricing rules remain open.
15. **Detailed retention periods:** configurable policy is selected, but exact periods by data category and tenant need policy decisions.
16. **Search indexing details:** Elasticsearch is selected; exact mappings, indexing and hosting need implementation design.
17. **Detailed API/SDK surface and partner onboarding model:** long-term capability is established; specific contracts and release sequencing require design.
18. **Exact activity plugin SDK:** future possibility only; arbitrary uploaded code must not execute in learner classrooms.

If the repository contains newer explicit decisions on any item, use the newer approved decision and document the change in an ADR.

---

# 25. ARCHITECTURAL PRINCIPLES

Apply these principles consistently:

1. **Product vision before release scope.** Do not shrink the complete platform definition to match one release.
2. **One coherent educational platform.** Lesson Studio, whiteboard, activities and classroom must integrate through shared domain concepts and contracts.
3. **Shared engines across distribution modes.** Standalone and embedded experiences use the same underlying engines.
4. **Modular monolith first.** Keep clear boundaries; extract services only when justified.
5. **Tenant isolation from the beginning.** Never treat tenant security as a later feature.
6. **Backend authorization is authoritative.** Frontend visibility is not a security control.
7. **PostgreSQL is durable operational authority.**
8. **Redis is not the durable source of truth for core business records.**
9. **NATS JetStream owns durable internal event/job workflows.**
10. **Centrifugo delivers application real-time updates; LiveKit delivers WebRTC media.**
11. **LiveKit is mandatory for media.** Do not substitute a custom SFU.
12. **TutorGrid owns its whiteboard document model.** Do not make the persisted domain format a renderer's private scene graph.
13. **High-frequency interactions must be coalesced appropriately.** Do not persist every pointer movement as an individual relational transaction.
14. **Published lesson versions are immutable.** Active classrooms pin their lesson version.
15. **Prepared content and live session state are separated.**
16. **Activities are first-class reusable educational experiences.**
17. **Do not execute arbitrary teacher-uploaded code in learner sessions.**
18. **Use provider abstractions where they prevent avoidable lock-in**, particularly storage and media integration.
19. **Do not abstract everything prematurely.** Abstractions should solve a real boundary or preserve a strategic option.
20. **Security, observability, testing, recovery and auditability are architectural concerns, not last-minute additions.**
21. **Design for extensibility without implementing every future capability immediately.**
22. **Never silently convert a recommendation into a confirmed product decision.**

---

# 26. REQUIRED ANTIGRAVITY DOCUMENTATION WORK

After reading this blueprint and inspecting the repository, reconcile the project documentation so that it has a clear source of truth.

Recommended document structure:
- `AGENTS.md` — persistent instructions for the coding agent.
- `docs/PRODUCT_CONSTITUTION.md` — product identity, boundaries and non-negotiable principles.
- `docs/PRODUCT_CAPABILITY_MAP.md` — complete long-term product capability map.
- `docs/PRODUCT_SPEC.md` — detailed functional requirements for product engines and experiences.
- `docs/ARCHITECTURE.md` — system architecture, domain boundaries, data flow and infrastructure.
- `docs/TECHNOLOGY_DECISIONS.md` or approved ADR index — consolidated technology decision register.
- `docs/ADR/` — individual architectural decision records, including superseded decisions.
- `.agents/rules/` — focused frontend, backend, database, security, testing, real-time and product rules.
- `.agents/workflows/` — repeatable audit, design, implementation and verification workflows.

Do not create duplicate contradictory sources of truth. If a decision is detailed in an ADR, link to it from the consolidated architecture document.

Review older blueprint files for obsolete proposals. In particular, if older material proposes different Go frameworks, database access layers, raw WebSockets or a custom WebRTC/SFU approach, reconcile it against the confirmed choices here. LiveKit is mandatory; Echo, Bun, PostgreSQL, Goose and Centrifugo are selected technologies.

The repository may contain an older file named `.agents/rules/00_PRODUCT_VISION_AND_BLUEPRINT.md`. Review it carefully. If it conflicts with the approved architecture, update it to reference the current approved documents rather than allowing two competing architecture specifications.

---

# 27. REQUIRED ANTIGRAVITY RESPONSE BEFORE IMPLEMENTATION

After inspection, report:

1. Which existing files were reviewed.
2. Which requirements in this blueprint are already documented.
3. Which requirements are missing.
4. Which files conflict with confirmed decisions.
5. Which assumptions in existing documents are not approved.
6. Which open decisions require product-owner input.
7. A proposed documentation reconciliation plan.
8. Any architectural risks or contradictions.
9. Any provider or licensing claims that require verification.
10. A clear statement that no application code has been changed.

Do not start coding as part of this audit.

The goal is to establish one accurate, comprehensive and internally consistent TutorGrid product and technical blueprint that can guide implementation over time without losing the full vision.
