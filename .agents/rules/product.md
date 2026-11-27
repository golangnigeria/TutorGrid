---
trigger: model_decision
description: "Apply when planning features, reconciling docs, or making product/architecture decisions for TutorGrid."
---
# TutorGrid Product & Blueprint Rules
- Canonical long-term reference: `docs/MASTER_BLUEPRINT.md` and `docs/PRODUCT_CAPABILITY_MAP.md`.
- Consolidated decision & open-decision register: `docs/TECHNOLOGY_DECISIONS.md` and `docs/ADR/`.
- MVP release scope and sequencing live separately in `docs/MVP_SCOPE.md`, `docs/ROADMAP.md`, and `docs/IMPLEMENTATION_PLAN.md`. Never shrink the complete platform vision to match one release, and never expand MVP without explicit approval.
- TutorGrid's three core engines (**Lesson Studio**, **Whiteboard Engine**, **Educational Game/Activity Engine**) delivered via **Live Classroom** must share one underlying domain model across standalone and embedded modes.
- Strictly out of scope: SIS, admissions/enrolment admin, teacher HR/recruitment, payroll, school fee collection, timetable admin, or tutoring CRM.
- Always preserve the distinction between **Confirmed**, **Recommended**, **Open**, and **Future** decisions. Never silently convert a recommendation into a confirmed decision.
- Uploaded PDF/PowerPoint files remain attached reference materials; never auto-convert them into editable lessons.
- Future AI capabilities are draft-only and always require teacher review and explicit approval.
