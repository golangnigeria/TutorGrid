package job

import "github.com/google/uuid"

type WelcomeEmailPayload struct {
	To        string `json:"to"`
	FirstName string `json:"first_name"`
}

// TenantInvitationPayload carries only IDs—never raw tokens—so NATS JetStream logs
// and streams never expose invitation secrets. The worker reads the outbox row from
// PostgreSQL under RLS and clears invite_url immediately after sending.
type TenantInvitationPayload struct {
	TenantID uuid.UUID `json:"tenant_id"`
	OutboxID uuid.UUID `json:"outbox_id"`
}
