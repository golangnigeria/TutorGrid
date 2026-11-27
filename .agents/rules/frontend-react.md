---
trigger: glob
globs: "**/*.ts, **/*.tsx, **/*.css"
description: "TutorGrid React/Vite responsive frontend conventions, accessibility, explicit UI states and API boundaries."
---
# Frontend rules
- Use React + Vite + `shadcn/ui` (see `docs/ADR/0004-frontend-ui-and-toolchain.md` for React 19, TypeScript, and Tailwind CSS v4); do not introduce a second competing design system.
- Keep API/business rules on backend; frontend permissions are presentation only.
- Build responsive interfaces for desktop, tablet and mobile; support touch targets, keyboard navigation, visible focus, and age-appropriate density (Pre-K through Grade 12 without childish styling for older learners).
- Every async interaction must handle loading, success, error and retry/empty states as appropriate.
- Never show a successful save/permission change before the API confirms it.
- Do not leave dead buttons, empty handlers, fake data in production paths or hardcoded secrets.
- Reuse existing components and design tokens; avoid introducing a second design system.
- Add component/flow tests for critical authoring, classroom, permissions and submission paths.
