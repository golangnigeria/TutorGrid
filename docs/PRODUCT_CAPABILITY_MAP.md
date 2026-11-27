# TutorGrid — Complete Product Capability Map

**Canonical Reference:** [`docs/MASTER_BLUEPRINT.md`](./MASTER_BLUEPRINT.md)  
**Scope Note:** This document captures the **complete intended TutorGrid platform** across all horizons. It intentionally excludes MVP release phasing and 1,000-user sequencing—see [`docs/MVP_SCOPE.md`](./MVP_SCOPE.md) and [`docs/ROADMAP.md`](./ROADMAP.md) for release scope.

---

## 1. Platform Identity & Boundaries
- **Coverage:** Pre-K through Grade 12 (and extensible beyond), all major subjects, flexible national/regional/custom curricula, age-appropriate UI density (no childish visuals forced on older learners).
- **Devices:** Desktop, laptop, tablet, mobile via **responsive web** (no separate native mobile app assumed).
- **Distribution Models (Shared Underlying Engines):**
  - Standalone TutorGrid platform.
  - Embedded TutorGrid experiences (lessons, whiteboards, activity/game players, live classrooms).
  - External API & SDK integration (including external lesson creation, webhooks, and partner identity).
- **Branding Levels:** TutorGrid default, customized customer branding (logo/colors), and full white-label (subject to commercial packaging/licensing).
- **Explicit Out-of-Scope Boundaries:** No SIS, admissions/enrolment admin, teacher HR/recruitment, payroll, school fee collection, timetable admin, or tutoring CRM.

---

## 2. Core Engines

### 2.1 Lesson Studio
- **Planning & Instructional Design:** Title, description, subject, grade/age band, topic/subtopics, objectives, outcomes, prerequisites, methodology, duration, assessment criteria, teacher notes, flexible curriculum mappings.
- **Structured Composition:** Nested ordered sections, subsections, steps, pages/slides, structured content blocks (never one opaque HTML blob), whiteboard documents, activities, assessments, attached reference resources.
- **Multiple Authoring Interfaces Over One Model:** Document editor, slide/page editor, instructional timeline/sequence, whiteboard authoring, activity/game builder, assessment designer.
- **Rich Content:** Rich text, lists, tables, images, audio/video, math notation, diagrams, charts/graphs, code snippets, interactive blocks.
- **Reference Attachments:** Uploaded PDF and PowerPoint files remain attached reference materials for presentation and whiteboard annotation—no silent auto-conversion into editable lessons.
- **Versioning & Libraries:** Editable drafts, immutable published versions, automatic revision history, personal and authorized shared reusable libraries (templates, sections, blocks, question banks, activities, boards).
- **Future AI Co-Pilot (Draft-Only):** Outline/objective/explanation/exercise/assessment drafting and grade adaptation—always requires teacher review and explicit approval before publishing.

### 2.2 Whiteboard Engine
- **Hybrid Canvas & Pages:** Multi-page structure with expandable/infinite-like canvas per page, pan, zoom, fit-to-screen, backgrounds, grids, templates, page CRUD/reorder/save/restore.
- **TutorGrid-Owned Document Model:** Renderer-independent schema (`whiteboard-core`) with adapter boundaries (`whiteboard-konva`, `whiteboard-tldraw` long-term; initial MVP renderer remains **Open**).
- **Tools & Educational Objects:** Freehand pen/pencil/highlighter, eraser, shapes/lines/arrows/polygons, text, math notation, select/move/resize/rotate, layers/groups/locking/alignment, subject-specific manipulatives (math, science, language, mind maps, flowcharts), media/PDF/slide projection.
- **Collaboration & Undo:** Configurable teacher-only, learner view-only, learner draw/annotate, selected-object manipulation, individual learner workspaces, shared workspace; CRDT principles + server-authoritative permissions/ordering + snapshots/incremental ops; individual undo by default + teacher global undo controls.
- **First-Class Activity Integration:** Insert, position, resize, launch, monitor, pause/resume, end, and review activities directly on the canvas without coupling the Game Engine to a whiteboard renderer.

### 2.3 Educational Game & Activity Engine
- **Reusable 5-Layer Model:** Educational content, activity definition, game runtime, multiplayer coordination, delivery integration (standalone, whiteboard, live classroom, embedded).
- **Activity Categories:** Question/assessment (MCQ, true/false, short answer, fill-in-the-blank, timed quizzes), manipulation/organization (drag-and-drop, matching, sorting, categorization, sequencing, labeling), games/puzzles (memory, word/spelling, number, problem-solving), interactive learning (simulations, graphs, manipulatives, experiments, interactive diagrams).
- **3 Authoring Levels:** Level 1 templates, Level 2 configurable components, Level 3 advanced visual builder—**never** executing arbitrary teacher-uploaded code in learner sessions.
- **Gameplay Modes & Results:** Individual, teacher-led, collaborative, competitive/team (configurable, not forced); durable responses, scores, completion, attempts, time spent, and group outcomes.

### 2.4 Live Classroom
- **Media (Mandatory LiveKit):** Audio/video, camera/mic controls, screen sharing, breakout rooms (long-term), LiveKit Cloud initially with Hetzner Cloud as future self-hosting candidate.
- **Delivery & Navigation:** Pinned immutable published lesson version, isolated session-specific live state/annotations, guided and free navigation with live switching, participant presence, chat, resilient reconnection.

---

## 3. Platform Infrastructure, Data, Search, Analytics & Governance
- **Storage & Media Processing:** Backblaze B2 (S3-compatible) with direct presigned PUT/GET, MinIO local parity; hybrid self-managed processing (FFmpeg, LibreOffice, Go libraries) via async NATS JetStream workers.
- **Search & Analytics:** PostgreSQL initially; **Elasticsearch** for dedicated search and **ClickHouse** for high-volume historical analytics as scale justifies; real-time analytics via NATS → Redis → Centrifugo.
- **Governance, Security & Recovery:** Tenant isolation (Go authz + PostgreSQL RLS), comprehensive audit trails, configurable retention & legal holds, structured data exports/portability, automated security checks + future pentesting, encrypted backups + PITR + Backblaze B2 backup copies, Terraform IaC, Cloudflare DNS/CDN, Northflank preferred initial hosting.
