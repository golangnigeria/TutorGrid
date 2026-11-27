# ADR 0004 — Frontend Toolchain & UI Component System

**Status:** **Proposed — Pending Approval**  
**Date:** 2026-11-26  
**References:** [`docs/MASTER_BLUEPRINT.md`](../MASTER_BLUEPRINT.md) (§17.3, §23, §24 #7–8), [`docs/ADR/0001-approved-stack-and-mvp.md`](./0001-approved-stack-and-mvp.md)

---

## 1. Context
[`docs/MASTER_BLUEPRINT.md`](../MASTER_BLUEPRINT.md) (§17.3 and §23) explicitly confirms **React**, **Vite**, **responsive web**, and **`shadcn/ui`** as the frontend foundation, while noting (§17.3, §24 #7–8) that **TypeScript** and **Tailwind CSS / Radix UI** should be explicitly recorded in an approved ADR rather than silently assumed.

During Slice 1 planning (2026-11-25), the product owner selected **React 19 + TypeScript + Vite + Tailwind CSS v4 (`@tailwindcss/vite`) + ESLint + Vitest**, which is now implemented in `web/`.

## 2. Proposed Decision
Formally adopt the following unified frontend toolchain in `web/`:
- **Framework:** React 19
- **Language:** TypeScript (`strict: true`)
- **Build & Test Tooling:** Vite + Vitest + React Testing Library + ESLint
- **Styling:** Tailwind CSS v4 (`@tailwindcss/vite`) with semantic CSS design tokens
- **UI Component System:** `shadcn/ui` (Radix UI primitives where required by `shadcn/ui` components), adopted progressively starting in **Slice 3** (Resource Library UI) without introducing a second competing design system.

## 3. Consequences
- Resolves Master Blueprint §24 #7 and #8 cleanly in a single ADR.
- Ensures consistent accessibility (visible focus, keyboard navigation, touch targets) and age-appropriate density across Pre-K–12 views.
- Existing Slice 2 auth views remain functional and can migrate incrementally to shared `shadcn/ui` primitives as needed.
