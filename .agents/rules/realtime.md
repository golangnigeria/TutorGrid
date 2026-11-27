---
trigger: model_decision
description: "Apply when working on Whiteboard, Live Classroom, Centrifugo, LiveKit, Redis, or NATS JetStream."
---
# TutorGrid Realtime, Media & State Ownership Rules
- **PostgreSQL** is the durable operational authority for lessons, versions, resources, memberships, sessions, and activity results.
- **Redis** is ephemeral cache, rate limiting, temporary active-session state, and Centrifugo broker—never the sole durable home of core business records or a duplicate durable job queue.
- **NATS JetStream** owns durable internal events and background jobs, paired with PostgreSQL transactional outbox and idempotent consumers.
- **Centrifugo** delivers client-facing application realtime updates and presence with backend-authorized short-lived channel tokens; never replaces LiveKit.
- **LiveKit** is mandatory for WebRTC audio/video/screen sharing (LiveKit Cloud initially; Hetzner Cloud is the future self-hosting candidate). Never substitute a custom SFU.
- Coalesce high-frequency drawing/cursor updates; never write individual pointer samples as relational PostgreSQL rows.
- Active classrooms pin an immutable published lesson version and maintain isolated live session state so source lessons are never overwritten.
- Whiteboard must use a TutorGrid-owned document schema independent of any renderer; initial MVP renderer and exact CRDT library remain **Open** until explicitly approved.
