---
trigger: glob
globs: "**/*.go"
description: "TutorGrid Go backend conventions: modular monolith, secure tenant-aware APIs, tests, and approved dependencies."
---
# Go backend rules
- Keep Go backend a modular monolith. Do not create a microservice without an approved ADR.
- Separate transport handlers, application services, domain rules and infrastructure adapters.
- Use Echo only at HTTP boundary; domain logic must not depend on Echo.
- Validate input and authorize tenant/resource access in backend on every protected operation.
- Do not trust client-supplied tenant, role, ownership or permission claims.
- Use context.Context, bounded timeouts, explicit error handling and structured logs.
- Avoid global mutable state and unbounded goroutines/work queues.
- Write unit tests for domain logic and integration tests for DB/auth/tenant behavior.
- Do not add a dependency without explaining need, maintenance/license implications and approval for material dependencies.
- Never trust a client-supplied tenant ID header as authorization, even in early slices. Derive tenant context strictly from a verified identity and authorized membership.
- Keep development-only or test-only tenant/identity helpers strictly isolated from production binaries and HTTP middleware (e.g., `*_test.go` or dedicated `*test` packages).
- Do not label a slice or endpoint "authenticated" until real Limen authentication is integrated and enforced.
