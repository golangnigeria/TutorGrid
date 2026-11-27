package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/tutorgrid/tutorgrid/internal/config"
	"github.com/tutorgrid/tutorgrid/internal/lib/email"
	"github.com/tutorgrid/tutorgrid/internal/model"
)

// InvitationOutboxStore lets the Asynq worker read and update outbox rows without raw tokens in Redis.
type InvitationOutboxStore interface {
	GetOutboxByID(ctx context.Context, tenantID, outboxID uuid.UUID) (model.OutboxEntry, error)
	UpdateOutboxStatus(ctx context.Context, tenantID, outboxID uuid.UUID, status, lastErr string, clearInviteURL bool) error
}

var emailClient *email.Client

func (j *JobService) InitHandlers(config *config.Config, logger *zerolog.Logger) {
	emailClient = email.NewClient(config, logger)
}

func (j *JobService) SetInvitationOutboxStore(store InvitationOutboxStore) {
	j.outboxStore = store
}

func (j *JobService) handleWelcomeEmailTask(ctx context.Context, t *asynq.Task) error {
	var p WelcomeEmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal welcome email payload: %w", err)
	}

	j.logger.Info().
		Str("type", "welcome").
		Str("to", p.To).
		Msg("processing welcome email task")

	if err := emailClient.SendWelcomeEmail(p.To, p.FirstName); err != nil {
		j.logger.Error().
			Str("type", "welcome").
			Str("to", p.To).
			Err(err).
			Msg("failed to send welcome email")
		return err
	}

	j.logger.Info().
		Str("type", "welcome").
		Str("to", p.To).
		Msg("successfully sent welcome email")
	return nil
}

func (j *JobService) handleTenantInvitationEmailTask(ctx context.Context, t *asynq.Task) error {
	var p TenantInvitationPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal tenant invitation payload: %w", err)
	}
	if j.outboxStore == nil {
		return fmt.Errorf("invitation outbox store is not configured")
	}

	entry, err := j.outboxStore.GetOutboxByID(ctx, p.TenantID, p.OutboxID)
	if err != nil {
		return fmt.Errorf("load invitation outbox entry: %w", err)
	}
	if entry.Status == "sent" || entry.InviteURL == "" {
		return nil
	}

	j.logger.Info().
		Str("type", "tenant_invitation").
		Str("outbox_id", entry.ID.String()).
		Str("invitation_id", entry.InvitationID.String()).
		Str("tenant_id", entry.TenantID.String()).
		Msg("processing tenant invitation email task")

	if err := emailClient.SendInvitationEmail(entry.RecipientEmail, entry.TenantName, string(entry.Role), entry.InviteURL); err != nil {
		_ = j.outboxStore.UpdateOutboxStatus(ctx, entry.TenantID, entry.ID, "failed", err.Error(), false)
		j.logger.Error().
			Str("type", "tenant_invitation").
			Str("outbox_id", entry.ID.String()).
			Str("invitation_id", entry.InvitationID.String()).
			Err(err).
			Msg("failed to send tenant invitation email")
		return err
	}

	// Clear invite_url immediately once sent so zero raw tokens remain in PostgreSQL.
	if err := j.outboxStore.UpdateOutboxStatus(ctx, entry.TenantID, entry.ID, "sent", "", true); err != nil {
		j.logger.Error().
			Str("outbox_id", entry.ID.String()).
			Err(err).
			Msg("sent invitation email but failed to mark outbox sent")
		return err
	}

	j.logger.Info().
		Str("type", "tenant_invitation").
		Str("outbox_id", entry.ID.String()).
		Str("invitation_id", entry.InvitationID.String()).
		Msg("successfully sent tenant invitation email")
	return nil
}
