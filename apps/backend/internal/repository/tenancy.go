package repository

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tutorgrid/tutorgrid/internal/database"
	"github.com/tutorgrid/tutorgrid/internal/model"
)

var (
	ErrNotMember                 = errors.New("tenancy: user is not a member of the requested tenant")
	ErrNoActiveTenant            = errors.New("tenancy: no active tenant selected")
	ErrSlugConflict              = errors.New("tenancy: workspace slug already exists")
	ErrInvalidInvitation         = errors.New("tenancy: invitation not found")
	ErrInvitationExpired         = errors.New("tenancy: invitation has expired")
	ErrInvitationRevoked         = errors.New("tenancy: invitation has been revoked")
	ErrInvitationAlreadyAccepted = errors.New("tenancy: invitation has already been accepted")
	ErrInvitationEmailMismatch   = errors.New("tenancy: invitation is restricted to a different email address")
)

type TenancyRepository struct {
	pool *pgxpool.Pool
}

func NewTenancyRepository(pool *pgxpool.Pool) *TenancyRepository {
	return &TenancyRepository{pool: pool}
}

func (r *TenancyRepository) UpsertUserBySubject(ctx context.Context, externalSubject, displayName, email string) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (external_subject, display_name, email)
		VALUES (@external_subject, @display_name, LOWER(TRIM(@email)))
		ON CONFLICT (external_subject) DO UPDATE
			SET display_name = EXCLUDED.display_name,
			    email = EXCLUDED.email,
			    updated_at = NOW()
		RETURNING id, external_subject, display_name, email
	`, pgx.StrictNamedArgs{
		"external_subject": externalSubject,
		"display_name":     displayName,
		"email":            email,
	}).Scan(&u.ID, &u.ExternalSubject, &u.DisplayName, &u.Email)
	if err != nil {
		return model.User{}, fmt.Errorf("repository: upsert user: %w", err)
	}
	return u, nil
}

func (r *TenancyRepository) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]model.Membership, error) {
	memberships := make([]model.Membership, 0)
	err := database.RunInUserTx(ctx, r.pool, userID, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT t.id, t.name, t.slug, tm.role
			FROM tenant_memberships tm
			INNER JOIN tenants t ON t.id = tm.tenant_id
			WHERE tm.user_id = @user_id
			ORDER BY t.created_at ASC
		`, pgx.StrictNamedArgs{
			"user_id": userID,
		})
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var m model.Membership
			var roleStr string
			if err := rows.Scan(&m.TenantID, &m.TenantName, &m.TenantSlug, &roleStr); err != nil {
				return err
			}
			m.Role = model.Role(roleStr)
			memberships = append(memberships, m)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("repository: list memberships: %w", err)
	}
	return memberships, nil
}

func (r *TenancyRepository) GetActiveTenantID(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	var tenantID uuid.NullUUID
	err := database.RunInUserTx(ctx, r.pool, userID, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT uat.active_tenant_id
			FROM user_active_tenants uat
			INNER JOIN tenant_memberships tm
				ON tm.user_id = uat.user_id
			   AND tm.tenant_id = uat.active_tenant_id
			WHERE uat.user_id = @user_id
			LIMIT 1
		`, pgx.StrictNamedArgs{
			"user_id": userID,
		}).Scan(&tenantID)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: get active tenant: %w", err)
	}
	if !tenantID.Valid {
		return nil, nil
	}
	id := tenantID.UUID
	return &id, nil
}

// ResolveVerifiedTenantContext atomically loads active_tenant_id and verifies current membership
// inside a single user-scoped transaction. A revoked/deleted membership immediately fails closed.
func (r *TenancyRepository) ResolveVerifiedTenantContext(ctx context.Context, userID uuid.UUID) (model.TenantContext, error) {
	var tc model.TenantContext
	err := database.RunInUserTx(ctx, r.pool, userID, func(ctx context.Context, tx pgx.Tx) error {
		var tenantID uuid.UUID
		var roleStr string
		err := tx.QueryRow(ctx, `
			SELECT uat.active_tenant_id, tm.role
			FROM user_active_tenants uat
			INNER JOIN tenant_memberships tm
				ON tm.user_id = uat.user_id
			   AND tm.tenant_id = uat.active_tenant_id
			WHERE uat.user_id = @user_id
			LIMIT 1
		`, pgx.StrictNamedArgs{
			"user_id": userID,
		}).Scan(&tenantID, &roleStr)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNoActiveTenant
			}
			return err
		}
		tc = model.TenantContext{
			TenantID: tenantID,
			UserID:   userID,
			Role:     model.Role(roleStr),
		}
		return nil
	})
	if err != nil {
		return model.TenantContext{}, err
	}
	return tc, nil
}

// SetActiveTenant atomically verifies current membership before updating user_active_tenants.
func (r *TenancyRepository) SetActiveTenant(ctx context.Context, userID, tenantID uuid.UUID) (model.Role, error) {
	var role model.Role
	err := database.RunInUserTx(ctx, r.pool, userID, func(ctx context.Context, tx pgx.Tx) error {
		var roleStr string
		err := tx.QueryRow(ctx, `
			SELECT tm.role
			FROM tenant_memberships tm
			WHERE tm.user_id = @user_id AND tm.tenant_id = @tenant_id
			FOR SHARE
		`, pgx.StrictNamedArgs{
			"user_id":   userID,
			"tenant_id": tenantID,
		}).Scan(&roleStr)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotMember
			}
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO user_active_tenants (user_id, active_tenant_id, updated_at)
			VALUES (@user_id, @tenant_id, NOW())
			ON CONFLICT (user_id) DO UPDATE
				SET active_tenant_id = EXCLUDED.active_tenant_id,
				    updated_at = NOW()
		`, pgx.StrictNamedArgs{
			"user_id":   userID,
			"tenant_id": tenantID,
		})
		if err != nil {
			return err
		}
		role = model.Role(roleStr)
		return nil
	})
	if err != nil {
		return "", err
	}
	return role, nil
}

// CreateTenantWithAdmin atomically creates a tenant, sets app.current_tenant_id inside the tx,
// inserts the org_admin membership, and sets user_active_tenants.
func (r *TenancyRepository) CreateTenantWithAdmin(ctx context.Context, userID uuid.UUID, name, slug string) (model.Membership, error) {
	var m model.Membership
	err := database.RunInUserTx(ctx, r.pool, userID, func(ctx context.Context, tx pgx.Tx) error {
		tenantID := uuid.New()
		if err := database.SetTxTenantID(ctx, tx, tenantID); err != nil {
			return err
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO tenants (id, name, slug)
			VALUES (@id, @name, @slug)
		`, pgx.StrictNamedArgs{
			"id":   tenantID,
			"name": name,
			"slug": slug,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return ErrSlugConflict
			}
			return err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO tenant_memberships (tenant_id, user_id, role)
			VALUES (@tenant_id, @user_id, 'org_admin')
		`, pgx.StrictNamedArgs{
			"tenant_id": tenantID,
			"user_id":   userID,
		}); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO user_active_tenants (user_id, active_tenant_id, updated_at)
			VALUES (@user_id, @tenant_id, NOW())
			ON CONFLICT (user_id) DO UPDATE
				SET active_tenant_id = EXCLUDED.active_tenant_id,
				    updated_at = NOW()
		`, pgx.StrictNamedArgs{
			"user_id":   userID,
			"tenant_id": tenantID,
		}); err != nil {
			return err
		}

		m = model.Membership{
			TenantID:   tenantID,
			TenantName: name,
			TenantSlug: slug,
			Role:       model.RoleOrgAdmin,
		}
		return nil
	})
	if err != nil {
		return model.Membership{}, err
	}
	return m, nil
}

// CreateInvitationWithOutbox atomically inserts tenant_invitations and invitation_email_outbox
// inside one RLS-scoped tenant transaction so a committed invite never loses its delivery record.
func (r *TenancyRepository) CreateInvitationWithOutbox(
	ctx context.Context,
	tc model.TenantContext,
	recipientEmail string,
	role model.Role,
	tokenHash []byte,
	inviteURL string,
	expiresAt time.Time,
) (uuid.UUID, model.OutboxEntry, error) {
	var invID uuid.UUID
	var outbox model.OutboxEntry

	err := database.RunInTenantTx(ctx, r.pool, tc.TenantID, func(ctx context.Context, tx pgx.Tx) error {
		var tenantName string
		if err := tx.QueryRow(ctx, `
			SELECT name FROM tenants WHERE id = @tenant_id
		`, pgx.StrictNamedArgs{
			"tenant_id": tc.TenantID,
		}).Scan(&tenantName); err != nil {
			return err
		}

		if err := tx.QueryRow(ctx, `
			INSERT INTO tenant_invitations (tenant_id, email, role, token_hash, invited_by_user_id, expires_at)
			VALUES (@tenant_id, @email, @role, @token_hash, @invited_by_user_id, @expires_at)
			RETURNING id
		`, pgx.StrictNamedArgs{
			"tenant_id":          tc.TenantID,
			"email":              recipientEmail,
			"role":               string(role),
			"token_hash":         tokenHash,
			"invited_by_user_id": tc.UserID,
			"expires_at":         expiresAt,
		}).Scan(&invID); err != nil {
			return err
		}

		var outboxID uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO invitation_email_outbox (invitation_id, tenant_id, recipient_email, tenant_name, role, invite_url, status)
			VALUES (@invitation_id, @tenant_id, @recipient_email, @tenant_name, @role, @invite_url, 'pending')
			RETURNING id
		`, pgx.StrictNamedArgs{
			"invitation_id":   invID,
			"tenant_id":       tc.TenantID,
			"recipient_email": recipientEmail,
			"tenant_name":     tenantName,
			"role":            string(role),
			"invite_url":      inviteURL,
		}).Scan(&outboxID); err != nil {
			return err
		}

		outbox = model.OutboxEntry{
			ID:             outboxID,
			InvitationID:   invID,
			TenantID:       tc.TenantID,
			RecipientEmail: recipientEmail,
			TenantName:     tenantName,
			Role:           role,
			InviteURL:      inviteURL,
			Status:         "pending",
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, model.OutboxEntry{}, fmt.Errorf("repository: create invitation with outbox: %w", err)
	}
	return invID, outbox, nil
}

func (r *TenancyRepository) UpdateOutboxStatus(
	ctx context.Context,
	tenantID, outboxID uuid.UUID,
	status string,
	lastErr string,
	clearInviteURL bool,
) error {
	return database.RunInTenantTx(ctx, r.pool, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		var errPtr *string
		if lastErr != "" {
			errPtr = &lastErr
		}
		args := pgx.StrictNamedArgs{
			"status":     status,
			"last_error": errPtr,
			"id":         outboxID,
		}
		if clearInviteURL {
			_, err := tx.Exec(ctx, `
				UPDATE invitation_email_outbox
				SET status = @status,
				    attempts = attempts + 1,
				    last_error = @last_error,
				    invite_url = NULL,
				    sent_at = CASE WHEN @status::text = 'sent' THEN NOW() ELSE sent_at END,
				    updated_at = NOW()
				WHERE id = @id
			`, args)
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE invitation_email_outbox
			SET status = @status,
			    attempts = attempts + 1,
			    last_error = @last_error,
			    updated_at = NOW()
			WHERE id = @id
		`, args)
		return err
	})
}

func (r *TenancyRepository) GetOutboxByID(ctx context.Context, tenantID, outboxID uuid.UUID) (model.OutboxEntry, error) {
	var e model.OutboxEntry
	err := database.RunInTenantTx(ctx, r.pool, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		var roleStr string
		var urlPtr *string
		err := tx.QueryRow(ctx, `
			SELECT id, invitation_id, tenant_id, recipient_email, tenant_name, role, invite_url, status, attempts
			FROM invitation_email_outbox
			WHERE id = @id
		`, pgx.StrictNamedArgs{
			"id": outboxID,
		}).Scan(&e.ID, &e.InvitationID, &e.TenantID, &e.RecipientEmail, &e.TenantName, &roleStr, &urlPtr, &e.Status, &e.Attempts)
		if err != nil {
			return err
		}
		e.Role = model.Role(roleStr)
		if urlPtr != nil {
			e.InviteURL = *urlPtr
		}
		return nil
	})
	return e, err
}

func (r *TenancyRepository) RevokeInvitation(ctx context.Context, tc model.TenantContext, invitationID uuid.UUID) error {
	return database.RunInTenantTx(ctx, r.pool, tc.TenantID, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE tenant_invitations
			SET revoked_at = NOW(), updated_at = NOW()
			WHERE id = @id AND accepted_at IS NULL AND revoked_at IS NULL
		`, pgx.StrictNamedArgs{
			"id": invitationID,
		})
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrInvalidInvitation
		}
		return nil
	})
}

// AcceptInvitation atomically locks the invitation (FOR UPDATE), validates expiry, revocation,
// prior acceptance, and email restriction, creates/confirms membership, marks accepted_at,
// and switches active_tenant_id—all inside a single pgx transaction.
func (r *TenancyRepository) AcceptInvitation(ctx context.Context, userID uuid.UUID, userEmail string, tokenHash []byte) (model.Membership, error) {
	tokenHashHex := hex.EncodeToString(tokenHash)
	var m model.Membership

	err := database.RunInInviteTx(ctx, r.pool, userID, tokenHashHex, func(ctx context.Context, tx pgx.Tx) error {
		var (
			invID      uuid.UUID
			tenantID   uuid.UUID
			invEmail   string
			roleStr    string
			expiresAt  time.Time
			acceptedAt *time.Time
			revokedAt  *time.Time
		)

		err := tx.QueryRow(ctx, `
			SELECT id, tenant_id, email, role, expires_at, accepted_at, revoked_at
			FROM tenant_invitations
			WHERE token_hash = @token_hash
			FOR UPDATE
		`, pgx.StrictNamedArgs{
			"token_hash": tokenHash,
		}).Scan(&invID, &tenantID, &invEmail, &roleStr, &expiresAt, &acceptedAt, &revokedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidInvitation
			}
			return err
		}

		if revokedAt != nil {
			return ErrInvitationRevoked
		}
		if acceptedAt != nil {
			return ErrInvitationAlreadyAccepted
		}
		if !expiresAt.After(time.Now().UTC()) {
			return ErrInvitationExpired
		}
		if !strings.EqualFold(strings.TrimSpace(invEmail), strings.TrimSpace(userEmail)) {
			return ErrInvitationEmailMismatch
		}

		// Elevate transaction-local tenant scope strictly to the verified invitation's tenant.
		if err := database.SetTxTenantID(ctx, tx, tenantID); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO tenant_memberships (tenant_id, user_id, role)
			VALUES (@tenant_id, @user_id, @role)
			ON CONFLICT (tenant_id, user_id) DO NOTHING
		`, pgx.StrictNamedArgs{
			"tenant_id": tenantID,
			"user_id":   userID,
			"role":      roleStr,
		}); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			UPDATE tenant_invitations
			SET accepted_at = NOW(), updated_at = NOW()
			WHERE id = @id
		`, pgx.StrictNamedArgs{
			"id": invID,
		}); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO user_active_tenants (user_id, active_tenant_id, updated_at)
			VALUES (@user_id, @tenant_id, NOW())
			ON CONFLICT (user_id) DO UPDATE
				SET active_tenant_id = EXCLUDED.active_tenant_id,
				    updated_at = NOW()
		`, pgx.StrictNamedArgs{
			"user_id":   userID,
			"tenant_id": tenantID,
		}); err != nil {
			return err
		}

		var tenantName, tenantSlug string
		if err := tx.QueryRow(ctx, `
			SELECT name, slug FROM tenants WHERE id = @tenant_id
		`, pgx.StrictNamedArgs{
			"tenant_id": tenantID,
		}).Scan(&tenantName, &tenantSlug); err != nil {
			return err
		}

		m = model.Membership{
			TenantID:   tenantID,
			TenantName: tenantName,
			TenantSlug: tenantSlug,
			Role:       model.Role(roleStr),
		}
		return nil
	})
	if err != nil {
		return model.Membership{}, err
	}
	return m, nil
}

func (r *TenancyRepository) ListUserPendingInvitations(ctx context.Context, userID uuid.UUID, userEmail string) ([]model.UserPendingInvitation, error) {
	invitations := make([]model.UserPendingInvitation, 0)
	normalizedEmail := strings.ToLower(strings.TrimSpace(userEmail))
	if normalizedEmail == "" {
		return invitations, nil
	}

	err := database.RunInUserEmailTx(ctx, r.pool, userID, normalizedEmail, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT ti.id, ti.tenant_id, t.name, t.slug, ti.role, ti.expires_at, ti.created_at
			FROM tenant_invitations ti
			INNER JOIN tenants t ON t.id = ti.tenant_id
			WHERE LOWER(ti.email) = LOWER(@user_email)
			  AND ti.accepted_at IS NULL
			  AND ti.revoked_at IS NULL
			  AND ti.expires_at > NOW()
			ORDER BY ti.created_at DESC
		`, pgx.StrictNamedArgs{
			"user_email": normalizedEmail,
		})
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var inv model.UserPendingInvitation
			var roleStr string
			if err := rows.Scan(&inv.ID, &inv.TenantID, &inv.TenantName, &inv.TenantSlug, &roleStr, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
				return err
			}
			inv.Role = model.Role(roleStr)
			invitations = append(invitations, inv)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("repository: list user pending invitations: %w", err)
	}
	return invitations, nil
}

// AcceptInvitationByID atomically accepts a pending invitation addressed to the user's verified email.
func (r *TenancyRepository) AcceptInvitationByID(ctx context.Context, userID uuid.UUID, userEmail string, invitationID uuid.UUID) (model.Membership, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(userEmail))
	var m model.Membership

	err := database.RunInUserEmailTx(ctx, r.pool, userID, normalizedEmail, func(ctx context.Context, tx pgx.Tx) error {
		var (
			invID      uuid.UUID
			tenantID   uuid.UUID
			invEmail   string
			roleStr    string
			expiresAt  time.Time
			acceptedAt *time.Time
			revokedAt  *time.Time
		)

		err := tx.QueryRow(ctx, `
			SELECT id, tenant_id, email, role, expires_at, accepted_at, revoked_at
			FROM tenant_invitations
			WHERE id = @id
			FOR UPDATE
		`, pgx.StrictNamedArgs{
			"id": invitationID,
		}).Scan(&invID, &tenantID, &invEmail, &roleStr, &expiresAt, &acceptedAt, &revokedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidInvitation
			}
			return err
		}

		if revokedAt != nil {
			return ErrInvitationRevoked
		}
		if acceptedAt != nil {
			return ErrInvitationAlreadyAccepted
		}
		if !expiresAt.After(time.Now().UTC()) {
			return ErrInvitationExpired
		}
		if !strings.EqualFold(strings.TrimSpace(invEmail), normalizedEmail) {
			return ErrInvitationEmailMismatch
		}

		if err := database.SetTxTenantID(ctx, tx, tenantID); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO tenant_memberships (tenant_id, user_id, role)
			VALUES (@tenant_id, @user_id, @role)
			ON CONFLICT (tenant_id, user_id) DO NOTHING
		`, pgx.StrictNamedArgs{
			"tenant_id": tenantID,
			"user_id":   userID,
			"role":      roleStr,
		}); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			UPDATE tenant_invitations
			SET accepted_at = NOW(), updated_at = NOW()
			WHERE id = @id
		`, pgx.StrictNamedArgs{
			"id": invID,
		}); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO user_active_tenants (user_id, active_tenant_id, updated_at)
			VALUES (@user_id, @tenant_id, NOW())
			ON CONFLICT (user_id) DO UPDATE
				SET active_tenant_id = EXCLUDED.active_tenant_id,
				    updated_at = NOW()
		`, pgx.StrictNamedArgs{
			"user_id":   userID,
			"tenant_id": tenantID,
		}); err != nil {
			return err
		}

		var tenantName, tenantSlug string
		if err := tx.QueryRow(ctx, `
			SELECT name, slug FROM tenants WHERE id = @tenant_id
		`, pgx.StrictNamedArgs{
			"tenant_id": tenantID,
		}).Scan(&tenantName, &tenantSlug); err != nil {
			return err
		}

		m = model.Membership{
			TenantID:   tenantID,
			TenantName: tenantName,
			TenantSlug: tenantSlug,
			Role:       model.Role(roleStr),
		}
		return nil
	})
	if err != nil {
		return model.Membership{}, err
	}
	return m, nil
}

func (r *TenancyRepository) ListCurrentTenantInvitations(ctx context.Context, tc model.TenantContext) ([]model.PendingInvitation, error) {
	invitations := make([]model.PendingInvitation, 0)
	err := database.RunInTenantTx(ctx, r.pool, tc.TenantID, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT ti.id, ti.email, ti.role, COALESCE(o.status, 'pending'), ti.expires_at, ti.created_at
			FROM tenant_invitations ti
			LEFT JOIN invitation_email_outbox o ON o.invitation_id = ti.id
			WHERE ti.accepted_at IS NULL
			  AND ti.revoked_at IS NULL
			  AND ti.expires_at > NOW()
			ORDER BY ti.created_at DESC
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var inv model.PendingInvitation
			var roleStr string
			if err := rows.Scan(&inv.ID, &inv.Email, &roleStr, &inv.EmailStatus, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
				return err
			}
			inv.Role = model.Role(roleStr)
			invitations = append(invitations, inv)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("repository: list current invitations: %w", err)
	}
	return invitations, nil
}

func (r *TenancyRepository) ListCurrentTenantMembers(ctx context.Context, tc model.TenantContext) ([]model.Member, error) {
	members := make([]model.Member, 0)
	err := database.RunInTenantTx(ctx, r.pool, tc.TenantID, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT tm.id, u.id, u.display_name, u.email, tm.role
			FROM tenant_memberships tm
			INNER JOIN users u ON u.id = tm.user_id
			ORDER BY tm.created_at ASC
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var m model.Member
			var roleStr string
			if err := rows.Scan(&m.MembershipID, &m.UserID, &m.DisplayName, &m.Email, &roleStr); err != nil {
				return err
			}
			m.Role = model.Role(roleStr)
			members = append(members, m)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("repository: list current members: %w", err)
	}
	return members, nil
}
