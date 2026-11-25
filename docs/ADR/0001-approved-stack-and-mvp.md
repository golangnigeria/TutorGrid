# ADR 0001 — Approved MVP Stack and Scope

Status: Accepted baseline; provider capability, pricing and licensing checks remain implementation-time validation items.

## Context
TutorGrid targets a coherent MVP and first 1,000 registered users, with capacity defined by measured concurrent classroom profile rather than total registrations. Avoid premature distributed infrastructure.

## Decisions
React/Vite; Go/Echo modular monolith; PostgreSQL/Bun/Goose; Limen with Google/GitHub OAuth; Redis; Centrifugo; LiveKit Cloud; Backblaze B2; NATS JetStream; REST/OpenAPI with selective internal gRPC; Northflank candidate; Docker/GitHub Actions/GHCR/Terraform; OpenTelemetry/Prometheus/Grafana/Sentry.

Long-term whiteboard strategy is hybrid (Konva open source and tldraw commercial option), but MVP ships one renderer behind TutorGrid-owned document schema. No dual renderer in MVP.

## Consequences
Simple deployment and clear boundaries; some advanced capabilities and scale-out work deferred. Validate provider features, cost, SDK compatibility and license before committing production usage. Any stack substitution requires a new ADR and explicit approval.
