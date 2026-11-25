---
trigger: glob
globs: "**/*.ts, **/*.tsx, **/*.css"
description: "TutorGrid React/Vite responsive frontend conventions, accessibility, explicit UI states and API boundaries."
---
# Frontend rules
- Use React + Vite and the established project UI conventions; do not replace the framework or component system.
- Keep API/business rules on backend; frontend permissions are presentation only.
- Build responsive interfaces for desktop, tablet and mobile; support touch targets and visible focus.
- Every async interaction must handle loading, success, error and retry/empty states as appropriate.
- Never show a successful save/permission change before the API confirms it.
- Do not leave dead buttons, empty handlers, fake data in production paths or hardcoded secrets.
- Reuse existing components and design tokens; avoid introducing a second design system.
- Add component/flow tests for critical authoring, classroom, permissions and submission paths.
