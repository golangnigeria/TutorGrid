# TutorGrid Product Constitution

## Product definition
TutorGrid is an educational content creation and teaching-delivery platform for teachers, tutors, tutoring companies, and partner applications. It enables educators to design sophisticated lessons, teach using a collaborative whiteboard, and create/play educational activities and games. It is available as a standalone platform and as embeddable/API-integrated capabilities.

## What TutorGrid is not
TutorGrid does not manage school administration, admissions, recruitment, payroll, fees, schedules, teacher employment, learner enrollment administration, or customer relationship management. Partner organizations own their teacher/learner records and operational administration. TutorGrid may show classroom participation and learning activity results because those directly support teaching.

## Core engines
### 1. Lesson Studio
A sophisticated instructional design workspace—not a basic text editor. It combines lesson planning, structured outline/sections, document editing, page/slide editing, sequencing/timeline, whiteboard authoring, activity/game creation, assessment design, resource libraries, preview, publishing and versioning.

### 2. Whiteboard Engine
A reusable collaborative visual teaching engine supporting prepared pages with expandable canvas, teacher presentation, learner collaboration, education-specific objects and embedded activities. It must have a TutorGrid-owned document model independent of any renderer.

### 3. Educational Activity/Game Engine
A reusable engine for built-in and teacher-created activities, from templates to advanced visual composition. It supports individual and multiplayer use, classroom integration and standalone play, Pre-K–12 and broad subject coverage.

### Live Classroom
A delivery environment combining lesson navigation, whiteboard, activities, assessment, participation and LiveKit audio/video. It is not the primary content authoring system.

## User groups and responsibilities
- Teacher/tutor: authors lessons and activities, teaches, manages classroom interaction and reviews results.
- Learner: joins authorized classrooms, views content, collaborates when permitted, plays activities and submits responses.
- Organization/partner: manages its own identities and people externally, grants TutorGrid access, configures permitted branding/integration.
- TutorGrid platform operator: operates the platform, tenant controls and safety/security functions.

## Product principles
- Teacher agency: teacher can guide or freely navigate a lesson and change the teaching flow.
- Learner collaboration: configurable, permission-based, age-appropriate participation.
- Reuse: reusable lesson templates, sections, content blocks, whiteboard documents, activities, assessments and question banks.
- Accessibility and responsive design: usable across desktop/tablet/mobile, with age-appropriate interface density.
- Separation of concerns: authored content, published version, active session state, learner responses and media transport are distinct concepts.
- Extensibility without premature complexity: design clear module boundaries; do not deploy every future service in MVP.
- Teacher review: future AI suggestions remain drafts requiring teacher review and approval.

## Distribution and branding
- Standalone TutorGrid experience.
- Embedded experiences and API/SDK integration for partner apps.
- Branding levels planned: TutorGrid default, partner logo/colors, full white-label. Full white-label and broad partner APIs may be phased after MVP.
- External sharing must respect owner, tenant, publication and access permissions.

## Curriculum and age range
Pre-K through Grade 12; flexible curriculum; all major subjects. Activities and UI must adapt to developmental level and not make older learners' content appear childish.

## Product boundary rule
If a feature does not directly support lesson creation, teaching delivery, whiteboard collaboration, educational activities/assessment, content sharing/integration, or learning-result review, it is not automatically part of TutorGrid. Treat it as out of scope until product owner approves it.
