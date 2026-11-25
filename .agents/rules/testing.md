---
trigger: model_decision
description: "Apply when adding features, fixing bugs, changing APIs, schemas, permissions, realtime behavior, or preparing a release."
---
# Verification rules
- Define observable acceptance criteria before implementation.
- Test success, validation failure, unauthorized access, cross-tenant access, retries/duplicates and relevant reconnect/error cases.
- Use integration tests for PostgreSQL, auth, tenant/RLS, NATS/Centrifugo/LiveKit adapters when behavior depends on them.
- Do not rely only on mocks for authorization or persistence correctness.
- Run exact project scripts; report command and outcome. Never claim tests passed if not run.
- Add regression tests for bugs.
