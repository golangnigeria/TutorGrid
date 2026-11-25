# TutorGrid — Product Vision & Requirements Blueprint
**Product Direction Consolidated — Version 1.0 (Discovery)**  
**Date:** 23 November 2026  
**Product:** TutorGrid  
**Purpose:** Establish the product vision, confirmed decisions, functional requirements, core engines, and technical direction before implementation begins.

---

## 1. Executive Summary

TutorGrid is an interactive educational teaching and learning platform designed for individual tutors, tutoring companies, and educational service providers.

Its purpose is to give teachers the tools to create lessons, prepare educational materials, conduct live teaching sessions, interact with learners, build educational games, and deliver engaging learning experiences.

TutorGrid is built around three major product capabilities:
1. **Lesson Studio:** A sophisticated instructional design and lesson-authoring environment.
2. **Whiteboard Engine:** A collaborative, interactive teaching and visual learning environment.
3. **Educational Game and Activity Engine:** A reusable engine for creating and delivering educational games and interactive activities from Pre-K through Grade 12.

These capabilities work together through the **Live Classroom**, standalone activity experiences, public sharing, embedding, and APIs.

TutorGrid is **not** intended to be a school management system or a tutoring business administration platform. It provides the educational technology infrastructure that tutoring companies and independent tutors need to teach effectively.

---

## 2. Product Vision & Principles

### Product Vision
To build a flexible, interactive, and intelligent educational platform that enables teachers to create, deliver, and share exceptional learning experiences through sophisticated lesson authoring, collaborative whiteboards, and engaging educational games.

TutorGrid makes it possible for a teacher to prepare an entire lesson, teach it online, engage learners through interactive activities, assess understanding, and reuse the material in future lessons. It functions across educational levels, subjects, teaching styles, curricula, and devices.

### Product Mission
To make high-quality interactive teaching tools accessible to independent educators and tutoring organizations, while providing the technology foundation they can integrate into their own products and services.

### Product Positioning
TutorGrid is an educational teaching and learning infrastructure platform. It is not simply:
- A video conferencing application
- A digital whiteboard
- A presentation editor
- A quiz application
- A learning management system (LMS)
- A school administration system

It combines these relevant teaching capabilities into one connected educational environment, powered by dedicated content and activity engines.

### Core Product Principles
1. **The lesson is the central educational document:** It binds content, whiteboards, games, and assessments into a coherent learning journey.
2. **The whiteboard is a first-class engine:** It is a visual teaching workspace with domain-specific pedagogical tools, not merely a freeform sketching pad.
3. **The game engine is reusable:** Activities run seamlessly as standalone experiences, embedded whiteboard objects, or classroom activities.
4. **Teachers remain in control:** Granular permissions, presentation pacing (guided vs. free navigation), and moderation remain under instructor authority.
5. **Pre-K through Grade 12 breadth:** Visual complexity, pedagogical depth, and game mechanics scale  intelligently across age brackets.
6. **Extensible subject coverage:** STEM, humanities, languages, and early learning primitives are supported natively.
7. **Curriculum flexibility:** Metadata models map freely to national and regional curricula (US Common Core, UK National Curriculum, Nigerian NERDC, IB, etc.) without altering lesson or game runtimes.
8. **Author once, deliver anywhere:** Content authored in Lesson Studio deploys across browser windows, mobile views, standalone links, or partner IFrames/SDKs.
9. **Asset reusability:** Granular components (sections, question banks, interactive widgets, templates) are library assets that avoid authoring duplication.
10. **Headless & integration-first:** Robust REST/WebSocket APIs, webhooks, short-lived tokens, and white-labeling allow deep third-party integration.
11. **Strict scope boundaries:** No tuition processing, school enrollment bureaucracy, or generic HR/CRM features.
12. **AI as assistive co-pilot:** Generative AI accelerates scaffolding and question drafting, but content publication always requires human teacher review.

---

## 3. Target Users, Customers & Scope Boundaries

### Primary Customers
- Independent tutors and educators
- Private tutoring companies and franchises
- Online tutoring businesses and test-prep bootcamps
- Educational service providers and content publishers
- Organizations with existing rosters of educators and learners
- EdTech companies seeking an embeddable virtual classroom or activity engine

### Platform Users
- **Teachers / Instructors / Tutors:** Content authors and live session moderators.
- **Learners / Students:** Active participants in interactive lessons and self-paced activities.
- **Authorized Classroom Observers:** Parents, mentors, or pedagogical coordinators with configured read-only rights.
- **External Platform Users:** Users accessing TutorGrid via partner LTI/SSO integrations.

### Explicit Scope Boundaries
TutorGrid explicitly excludes:
- Teacher hiring, contracts, and payroll
- General student admissions and formal registrar functions
- School fees, invoicing, and tuition billing collection
- School-wide CRM and generic parent-teacher conference scheduling
- Comprehensive administrative attendance tracking (session-level participation metrics are supported)

---

## 4. Conceptual Architecture

```
+-------------------------------------------------------------------------+
|                           TutorGrid Platform                            |
+-------------------------------------------------------------------------+
|  [ Lesson Studio ]     |  [ Whiteboard Engine ]  | [ Educational Engine]|
|  - Instructional Design|  - Visual Teaching      | - Games & Activities |
|  - Content Authoring   |  - Infinite/Hybrid      | - Simulations        |
|  - Templates & Reusables| - Domain Tools         | - Multiplayer State  |
|  - AI Assistance       |  - Realtime Canvas      | - Scoring & Evals    |
+------------------------+-------------------------+----------------------+
|                         Live Classroom Delivery                         |
|  - WebRTC A/V Mesh/SFU | Realtime Sync Protocol | Permissions & States  |
+-------------------------------------------------------------------------+
|                         Platform Infrastructure                         |
|  - REST & WS Gateway   | Tenant Isolation       | S3 Storage           |
|  - Short-Lived Tokens  | White-Label Theming    | Audit & Observability|
+-------------------------------------------------------------------------+
```

Each engine is decoupled:
- An educational game can run in a standalone browser tab, inside an iframe, on a whiteboard canvas, or in a Live Classroom.
- A whiteboard page can be authored offline, populated with interactive widgets, and presented synchronously without state mutation collisions.

---

## 5. Detailed Engine Breakdown

### 5.1 Lesson Studio
- **Editing Philosophy:** Unified authoring environment unifying document-based structure, slide-based pacing, visual canvas elements, and interactive component insertion.
- **Lesson Planning Structure:** Metadata (title, objectives, prerequisite knowledge, target age/grade, subject taxonomy, duration, teacher notes).
- **Hierarchical Composition:** Sections -> Subsections -> Steps (e.g., Introduction -> Explanation -> Demonstration -> Practice -> Assessment -> Conclusion).
- **Rich Content Blocks:** Markdown/WYSIWYG typography, KaTeX mathematical formulas, embedded video/audio, code blocks, data tables, and dynamic interactive blocks.
- **Dual-State Separation:** Lesson definitions remain immutable snapshots during live delivery; classroom session state runs independently.
- **Import Handling:** PDF and PowerPoint files are treated as high-fidelity reference attachments with canvas projection and annotation support (not lossy vector-to-block converters).
- **Assessment Designer:** Formative and summative assessment formats (MCQ, true/false, fill-in-blank, ordering, matching, open response, KaTeX equation solvers) with question banks and rubrics.

### 5.2 Whiteboard Engine
- **Canvas Model:** Hybrid multi-page architecture combining discrete slide pagination with expandable 2D spatial panning and multi-level zooming.
- **Drawing & Vector Primitives:** Pens, highlighters, geometric shapes, directional connectors, rich text cards, layer management (z-index, grouping, locking).
- **Pedagogical Toolsets:**
  - *Mathematics:* Dynamic fraction bars, coordinate grids, number lines, protractor/ruler overlays, equation blocks.
  - *Sciences:* Circuit builders, molecular/cell diagram templates, chemistry balance tools.
  - *Languages & Humanities:* Phonics tiles, sentence diagramming containers, graphic organizers, mind maps.
- **Collaboration & Access Control:** Instructor broadcast mode, global collaborative mode, per-student isolated scratchpads, and selective object-level permissions.
- **Activity Host:** Seamless mounting of active Game Engine components directly onto whiteboard coordinates.

### 5.3 Educational Game & Activity Engine
- **Age Scope:** Pre-K (sensory, matching, shapes) to Grade 12 (advanced calculus simulations, physics mechanics, language synthesis).
- **Authoring Tiers:**
  - *Level 1 (Templates):* Form-based parameterization of existing activity archetypes.
  - *Level 2 (Configurable Builder):* Assembling modular interaction components (triggers, containers, rules, timers).
  - *Level 3 (Visual Interaction Builder):* Node- or rule-based behavior construction without raw code execution.
- **Runtime Archetypes:**
  - *Assessment:* Timed quizzes, flashcards, rapid response buzzers.
  - *Manipulative & Spatial:* Drag-and-drop categorization, card matching, spatial sorting, puzzle assembly.
  - *Simulations:* Physical/mathematical models (e.g., gravity, pendulums, balancing chemical reactions).
- **Multiplayer Coordination:** Single-player practice, turn-based turnarounds, competitive leaderboards, and collaborative team arenas.
- **Security & Sandboxing:** Declarative state machines and schema-validated component trees; strict avoidance of arbitrary student/teacher client code execution.

### 5.4 Live Classroom
- **AV Infrastructure:** WebRTC audio/video feeds, screen sharing, instructor pinning, low-bandwidth audio fallback.
- **Navigation Control:**
  - *Guided Mode:* Teacher locks learner viewports to current lesson sequence and canvas views.
  - *Free Exploration Mode:* Learners navigate slides or complete activities at self-paced intervals.
- **Session Pinning:** Classroom sessions latch onto a specific version hash of a lesson, guaranteeing runtime immutability against background teacher revisions.

---

## 6. Approved Technical Stack (Superseded by ADR 0001)

> **Note:** Initial discovery stack ideas (`Chi`, `pgx + sqlc`, raw WebSockets) have been superseded by the authoritative approved stack in [`docs/ADR/0001-approved-stack-and-mvp.md`](file:///c:/Users/HomePC/Desktop/TutorGrid/docs/ADR/0001-approved-stack-and-mvp.md), [`docs/ARCHITECTURE.md`](file:///c:/Users/HomePC/Desktop/TutorGrid/docs/ARCHITECTURE.md), and [`AGENTS.md`](file:///c:/Users/HomePC/Desktop/TutorGrid/AGENTS.md). Do not substitute without an approved ADR.

| Domain | Authoritative Approved Technology | Status |
|---|---|---|
| **Backend Core** | Go + Echo (modular monolith) | Approved (ADR 0001) |
| **Database** | PostgreSQL + Bun (data access) + Goose (migrations) | Approved (ADR 0001) |
| **Authentication** | Limen (Google and GitHub OAuth; no email/password) | Approved (ADR 0001) |
| **Authorization** | RBAC + resource-level permissions; backend checks + PostgreSQL RLS | Approved (ADR 0001) |
| **Cache & Ephemeral State** | Redis | Approved (ADR 0001) |
| **Application Realtime** | Centrifugo | Approved (ADR 0001) |
| **Media Engine** | LiveKit Cloud (no self-hosting in MVP) | Approved (ADR 0001) |
| **Object Storage** | Backblaze B2 (via S3-compatible abstraction) | Approved (ADR 0001) |
| **Jobs / Async Events** | NATS JetStream (small MVP deployment, non-clustered) | Approved (ADR 0001) |
| **API Contract** | REST + OpenAPI (selective internal gRPC only if justified) | Approved (ADR 0001) |
| **Frontend App** | React + TypeScript + Vite (responsive web) | Approved (ADR 0001) |
| **Hosting & Delivery** | Northflank candidate, Docker, GitHub Actions, GHCR, Terraform | Approved (ADR 0001) |
| **Observability** | OpenTelemetry, Prometheus/Grafana, Sentry | Approved (ADR 0001) |
| **Whiteboard Renderer** | Single MVP renderer behind TutorGrid-owned document schema | **Open (Pending Schema + Renderer Comparison & Approval)** |
| **Math Notation Renderer** | KaTeX (candidate) | **Recommended (Pending Approval)** |

---

## 7. Confirmed Decisions Baseline

1. TutorGrid serves independent educators, tutoring firms, and third-party integrators.
2. TutorGrid is both a standalone destination app and an embeddable component via APIs/SDKs.
3. Administrative HR, payroll, and tuition fees are explicitly out of scope.
4. Whiteboard and Game Engines are core parallel priorities, not secondary plugins.
5. Grade coverage spans Pre-K through Grade 12 across all major disciplines.
6. Activities support teacher-created variants as well as system-curated presets.
7. Gameplay natively accommodates single-player, collaborative, and competitive formats.
8. Curriculum classification uses a decoupled, extensible metadata taxonomy.
9. Lesson delivery supports seamless toggling between Guided and Free Navigation.
10. Whiteboard canvas adopts a hybrid multi-page + expandable pan/zoom model.
11. PDFs/PPTs remain attached reference documents with projection and annotation capabilities.
12. Strict isolation between prepared lesson templates and active live classroom state.
13. Modular monolith architecture as the launch baseline for simplified deployment and domain clarity.

---

## 8. Next Deliverables Schedule

1. **Deliverable 1: Product Requirements Document (PRD)** — Exhaustive functional and non-functional requirements, acceptance criteria, personas, and edge cases.
2. **Deliverable 2: Whiteboard Engine Specification** — Canvas coordinates, object graphs, delta sync protocol, CRDT/OT strategies, tool interfaces.
3. **Deliverable 3: Educational Game Engine Specification** — Schema-driven activity runtime, component contracts, multiplayer state machine, scoring SDK.
4. **Deliverable 4: Lesson Studio Specification** — Content block schemas, hierarchical planning model, revision history, preview pipelines.
5. **Deliverable 5: System Architecture Document** — Service boundaries, WebSocket protocol specification, WebRTC signaling, security & tenant boundaries.
6. **Deliverable 6: Database & Data Model Design** — Relational schema, indexes, JSONB structures, versioning tables, tenant isolation policies.
7. **Deliverable 7: Implementation Roadmap & Milestones** — Phased sprint breakdown, critical path analysis, and verification checklist.
