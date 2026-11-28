package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/tutorgrid/tutorgrid/internal/model"
)

// InvitationOutboxStore lets the NATS JetStream worker read and update outbox rows without raw tokens in NATS.
type InvitationOutboxStore interface {
	GetOutboxByID(ctx context.Context, tenantID, outboxID uuid.UUID) (model.OutboxEntry, error)
	UpdateOutboxStatus(ctx context.Context, tenantID, outboxID uuid.UUID, status, lastErr string, clearInviteURL bool) error
}

// ResourceJobPayload identifies a tenant-scoped resource for idempotent cleanup or deletion.
type ResourceJobPayload struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	ResourceID uuid.UUID `json:"resource_id"`
}

func (j *JobService) handleWelcomeEmail(_ context.Context, env Envelope) error {
	var p WelcomeEmailPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal welcome email payload: %w", err)
	}
	if j.emailClient == nil {
		return errors.New("email client is not configured")
	}

	j.logger.Info().
		Str("type", "welcome").
		Str("event_id", env.EventID).
		Str("to", p.To).
		Msg("processing welcome email job")

	if err := j.emailClient.SendWelcomeEmail(p.To, p.FirstName); err != nil {
		j.logger.Error().
			Str("type", "welcome").
			Str("event_id", env.EventID).
			Str("to", p.To).
			Err(err).
			Msg("failed to send welcome email")
		return err
	}

	j.logger.Info().
		Str("type", "welcome").
		Str("event_id", env.EventID).
		Str("to", p.To).
		Msg("successfully sent welcome email")
	return nil
}

func (j *JobService) handleTenantInvitationEmail(ctx context.Context, env Envelope) error {
	var p TenantInvitationPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal tenant invitation payload: %w", err)
	}

	j.mu.RLock()
	store := j.outboxStore
	j.mu.RUnlock()

	if store == nil {
		return errors.New("invitation outbox store is not configured")
	}
	if j.emailClient == nil {
		return errors.New("email client is not configured")
	}

	entry, err := store.GetOutboxByID(ctx, p.TenantID, p.OutboxID)
	if err != nil {
		return fmt.Errorf("load invitation outbox entry: %w", err)
	}
	if entry.Status == "sent" || entry.InviteURL == "" {
		return nil
	}

	j.logger.Info().
		Str("type", "tenant_invitation").
		Str("event_id", env.EventID).
		Str("outbox_id", entry.ID.String()).
		Str("invitation_id", entry.InvitationID.String()).
		Str("tenant_id", entry.TenantID.String()).
		Msg("processing tenant invitation email job")

	if err := j.emailClient.SendInvitationEmail(entry.RecipientEmail, entry.TenantName, string(entry.Role), entry.InviteURL); err != nil {
		_ = store.UpdateOutboxStatus(ctx, entry.TenantID, entry.ID, "failed", err.Error(), false)
		j.logger.Error().
			Str("type", "tenant_invitation").
			Str("event_id", env.EventID).
			Str("outbox_id", entry.ID.String()).
			Str("invitation_id", entry.InvitationID.String()).
			Err(err).
			Msg("failed to send tenant invitation email")
		return err
	}

	// Clear invite_url immediately once sent so zero raw tokens remain in PostgreSQL.
	if err := store.UpdateOutboxStatus(ctx, entry.TenantID, entry.ID, "sent", "", true); err != nil {
		j.logger.Error().
			Str("event_id", env.EventID).
			Str("outbox_id", entry.ID.String()).
			Err(err).
			Msg("sent invitation email but failed to mark outbox sent")
		return err
	}

	j.logger.Info().
		Str("type", "tenant_invitation").
		Str("event_id", env.EventID).
		Str("outbox_id", entry.ID.String()).
		Str("invitation_id", entry.InvitationID.String()).
		Msg("successfully sent tenant invitation email")
	return nil
}

func (j *JobService) handleResourceMessage(ctx context.Context, subject string, env Envelope) error {
	var p ResourceJobPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal resource job payload: %w", err)
	}

	j.mu.RLock()
	handler := j.resourceJobHandler
	j.mu.RUnlock()
	if handler == nil {
		return errors.New("resource job handler is not configured")
	}

	switch subject {
	case SubjectResourceDelete:
		return handler.ReconcileDeletingResource(ctx, p.TenantID, p.ResourceID)
	case SubjectResourceCleanup:
		return handler.CleanupAbandonedPending(ctx, p.TenantID, p.ResourceID)
	default:
		return nil
	}
}
