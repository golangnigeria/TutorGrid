package job

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	TaskWelcome          = "email:welcome"
	TaskTenantInvitation = "email:tenant_invitation"
)

type WelcomeEmailPayload struct {
	To        string `json:"to"`
	FirstName string `json:"first_name"`
}

// TenantInvitationPayload carries only IDs—never raw tokens—so Asynq/Redis logs
// never expose invitation secrets. The worker reads the outbox row from PostgreSQL
// under RLS and clears invite_url immediately after sending.
type TenantInvitationPayload struct {
	TenantID uuid.UUID `json:"tenant_id"`
	OutboxID uuid.UUID `json:"outbox_id"`
}

func NewWelcomeEmailTask(to, firstName string) (*asynq.Task, error) {
	payload, err := json.Marshal(WelcomeEmailPayload{
		To:        to,
		FirstName: firstName,
	})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TaskWelcome, payload,
		asynq.MaxRetry(3),
		asynq.Queue("default"),
		asynq.Timeout(30*time.Second)), nil
}

func NewTenantInvitationEmailTask(tenantID, outboxID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(TenantInvitationPayload{
		TenantID: tenantID,
		OutboxID: outboxID,
	})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TaskTenantInvitation, payload,
		asynq.MaxRetry(5),
		asynq.Queue("critical"),
		asynq.Timeout(30*time.Second)), nil
}
