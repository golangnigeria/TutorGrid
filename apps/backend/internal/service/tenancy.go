package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/tutorgrid/tutorgrid/internal/lib/job"
	"github.com/tutorgrid/tutorgrid/internal/model"
	"github.com/tutorgrid/tutorgrid/internal/repository"
)

var (
	ErrInvalidName  = errors.New("tenancy: workspace name must be between 2 and 100 characters")
	ErrInvalidSlug  = errors.New("tenancy: workspace slug must be 3-48 lowercase letters, numbers, or hyphens")
	ErrInvalidEmail = errors.New("tenancy: valid email address is required")
	ErrInvalidRole  = errors.New("tenancy: role must be org_admin, teacher, or learner")
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,46}[a-z0-9]$`)

// TaskEnqueuer abstracts NATS JetStream job publishing so outbox failure/retry paths are deterministically testable.
type TaskEnqueuer interface {
	EnqueueInvitationEmail(ctx context.Context, tenantID, outboxID uuid.UUID) error
}

type TenancyService struct {
	repo            *repository.TenancyRepository
	enqueuer        TaskEnqueuer
	frontendBaseURL string
	logger          *zerolog.Logger
}

func NewTenancyService(
	repo *repository.TenancyRepository,
	jobSvc *job.JobService,
	frontendBaseURL string,
	logger *zerolog.Logger,
) *TenancyService {
	var enq TaskEnqueuer
	if jobSvc != nil {
		enq = jobSvc
	}
	return &TenancyService{
		repo:            repo,
		enqueuer:        enq,
		frontendBaseURL: strings.TrimRight(frontendBaseURL, "/"),
		logger:          logger,
	}
}

// SetTaskEnqueuer overrides the enqueuer (used by integration tests to simulate queue failures).
func (s *TenancyService) SetTaskEnqueuer(e TaskEnqueuer) {
	s.enqueuer = e
}

func (s *TenancyService) UpsertUserBySubject(ctx context.Context, externalSubject, displayName, email string) (model.User, error) {
	if strings.TrimSpace(externalSubject) == "" || strings.TrimSpace(email) == "" {
		return model.User{}, errors.New("tenancy: external subject and email are required")
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = email
	}
	return s.repo.UpsertUserBySubject(ctx, externalSubject, displayName, email)
}

func (s *TenancyService) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]model.Membership, error) {
	return s.repo.ListUserMemberships(ctx, userID)
}

func (s *TenancyService) GetActiveTenantID(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	return s.repo.GetActiveTenantID(ctx, userID)
}

func (s *TenancyService) ResolveVerifiedTenantContext(ctx context.Context, userID uuid.UUID) (model.TenantContext, error) {
	return s.repo.ResolveVerifiedTenantContext(ctx, userID)
}

func (s *TenancyService) SetActiveTenant(ctx context.Context, userID, tenantID uuid.UUID) (model.Role, error) {
	return s.repo.SetActiveTenant(ctx, userID, tenantID)
}

func (s *TenancyService) CreateTenantWithAdmin(ctx context.Context, userID uuid.UUID, name, slug string) (model.Membership, error) {
	trimmedName := strings.TrimSpace(name)
	if len(trimmedName) < 2 || len(trimmedName) > 100 {
		return model.Membership{}, ErrInvalidName
	}
	normalizedSlug := strings.ToLower(strings.TrimSpace(slug))
	if !slugRegex.MatchString(normalizedSlug) {
		return model.Membership{}, ErrInvalidSlug
	}
	return s.repo.CreateTenantWithAdmin(ctx, userID, trimmedName, normalizedSlug)
}

// CreateInvitation persists the invitation + transactional email outbox row atomically,
// then attempts to enqueue the Asynq delivery task. If enqueueing fails, the outbox
// remains durably pending with last_error recorded for retry without losing the invite.
func (s *TenancyService) CreateInvitation(
	ctx context.Context,
	tc model.TenantContext,
	email string,
	role model.Role,
) (model.CreatedInvitation, error) {
	trimmedEmail := strings.ToLower(strings.TrimSpace(email))
	if trimmedEmail == "" || !strings.Contains(trimmedEmail, "@") {
		return model.CreatedInvitation{}, ErrInvalidEmail
	}
	if role != model.RoleOrgAdmin && role != model.RoleTeacher && role != model.RoleLearner {
		return model.CreatedInvitation{}, ErrInvalidRole
	}

	rawBytes := make([]byte, 24)
	if _, err := rand.Read(rawBytes); err != nil {
		return model.CreatedInvitation{}, fmt.Errorf("tenancy: generate invite token: %w", err)
	}
	rawToken := hex.EncodeToString(rawBytes)
	tokenHash := sha256.Sum256([]byte(rawToken))
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	inviteURL := fmt.Sprintf("%s/invite?token=%s", s.frontendBaseURL, url.QueryEscape(rawToken))

	invID, outbox, err := s.repo.CreateInvitationWithOutbox(ctx, tc, trimmedEmail, role, tokenHash[:], inviteURL, expiresAt)
	if err != nil {
		return model.CreatedInvitation{}, err
	}

	emailStatus := "pending"
	if s.enqueuer != nil {
		if err := s.enqueuer.EnqueueInvitationEmail(ctx, tc.TenantID, outbox.ID); err != nil {
			// Never log rawToken or inviteURL; keep outbox durably pending for retry.
			_ = s.repo.UpdateOutboxStatus(ctx, tc.TenantID, outbox.ID, "pending", err.Error(), false)
			if s.logger != nil {
				s.logger.Warn().
					Str("invitation_id", invID.String()).
					Str("outbox_id", outbox.ID.String()).
					Str("tenant_id", tc.TenantID.String()).
					Err(err).
					Msg("invitation committed; async email enqueue failed and remains pending in outbox")
			}
		} else {
			emailStatus = "enqueued"
			_ = s.repo.UpdateOutboxStatus(ctx, tc.TenantID, outbox.ID, "enqueued", "", false)
		}
	}

	return model.CreatedInvitation{
		ID:          invID,
		Email:       trimmedEmail,
		Role:        role,
		Token:       rawToken,
		EmailStatus: emailStatus,
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *TenancyService) RevokeInvitation(ctx context.Context, tc model.TenantContext, invitationID uuid.UUID) error {
	return s.repo.RevokeInvitation(ctx, tc, invitationID)
}

func (s *TenancyService) AcceptInvitation(ctx context.Context, userID uuid.UUID, userEmail, rawToken string) (model.Membership, error) {
	trimmed := strings.TrimSpace(rawToken)
	if trimmed == "" {
		return model.Membership{}, repository.ErrInvalidInvitation
	}
	tokenHash := sha256.Sum256([]byte(trimmed))
	return s.repo.AcceptInvitation(ctx, userID, userEmail, tokenHash[:])
}

func (s *TenancyService) ListUserPendingInvitations(ctx context.Context, userID uuid.UUID, userEmail string) ([]model.UserPendingInvitation, error) {
	return s.repo.ListUserPendingInvitations(ctx, userID, userEmail)
}

func (s *TenancyService) AcceptInvitationByID(ctx context.Context, userID uuid.UUID, userEmail string, invitationID uuid.UUID) (model.Membership, error) {
	return s.repo.AcceptInvitationByID(ctx, userID, userEmail, invitationID)
}

func (s *TenancyService) ListCurrentTenantInvitations(ctx context.Context, tc model.TenantContext) ([]model.PendingInvitation, error) {
	return s.repo.ListCurrentTenantInvitations(ctx, tc)
}

func (s *TenancyService) ListCurrentTenantMembers(ctx context.Context, tc model.TenantContext) ([]model.Member, error) {
	return s.repo.ListCurrentTenantMembers(ctx, tc)
}
