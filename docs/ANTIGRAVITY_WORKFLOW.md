# Antigravity Operating Workflow

## Golden loop
1. Read project constitution, MVP scope, product spec, architecture and relevant ADRs.
2. Inspect the repository and identify existing patterns.
3. Restate the requested user outcome and acceptance criteria.
4. Propose a small plan: files/modules, data/API effects, risks, tests.
5. Wait for approval when the change is broad, architectural, destructive, dependency-changing or scope-expanding.
6. Implement one vertical slice only.
7. Run tests/lint/typecheck/build/migrations as relevant.
8. Review diff for unrelated changes, security/tenant gaps, incomplete states and placeholders.
9. Update docs/ADR/tests.
10. Report exactly what changed, what was verified, failures, limitations and next smallest task.

## Feature prompt template
"Implement [FEATURE ID + name] from docs/PRODUCT_SPEC.md. First inspect the repo and relevant rules. Do not change scope or architecture. Present a brief plan and acceptance criteria before editing. Implement only this vertical slice. Add/update tests. Run the relevant checks and report actual results. If requirements conflict or a key decision is missing, stop and ask instead of guessing."

## Bug-fix prompt template
"Investigate [BUG]. Do not edit initially. Reproduce or identify a reliable failing test, explain root cause and propose the smallest safe fix. After approval, fix it, add a regression test, run checks, and report evidence. Avoid unrelated refactoring."

## Architecture-change prompt template
"Evaluate [PROPOSED CHANGE] against product constitution, MVP, current architecture, cost/operational complexity and first-1,000-user target. Do not implement. Produce options, tradeoffs, migration impact, security/licensing implications and a proposed ADR. Wait for explicit approval."

## Review prompt template
"Review the current diff against AGENTS.md, product spec, acceptance criteria and security/tenant boundaries. Do not modify files. Report only concrete findings, ranked critical/high/medium/low, with file/line references and recommended fixes. Check for product drift, missing tests, fake success states, placeholders, unsafe permissions, migration hazards and unnecessary infrastructure."

## Release prompt template
"Run the MVP release checklist for this branch. Verify tests, build, migrations, config/secrets, tenant isolation, backup/restore evidence, LiveKit permission behavior, whiteboard reconnect, activity result idempotency and monitoring. Do not claim a check passed unless it was run. List blockers and evidence."

## Working constraints
- One feature/task per conversation or clearly bounded task.
- Prefer 1–3 related files/modules per small change; larger changes must be split into milestones.
- Use a plan before implementation for multi-file work.
- Avoid parallel agents modifying overlapping files. Use independent read-only audits in parallel only when helpful.
- Review the diff yourself before accepting it.
- Commit/checkpoint after each verified vertical slice.
- Never allow the agent to automatically deploy to production or run destructive DB operations without explicit human approval.
