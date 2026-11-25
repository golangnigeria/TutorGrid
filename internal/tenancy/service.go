package tenancy

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/tutorgrid/tutorgrid/internal/platform/database"
)

var (
	ErrNotMember         = errors.New("tenancy: user is not a member of the requested tenant")
	ErrNoActiveTenant    = errors.New("tenancy: no active tenant selected")
	ErrInvalidSlug       = errors.New("tenancy: slug must be 3-48 lowercase alphanumeric characters or hyphens")
	ErrInvalidName       = errors.New("tenancy: name must be between 2 and 100 characters")
	ErrInvalidEmail      = errors.New("tenancy: valid email is required")
	ErrInvalidInvitation = errors.New("tenancy: invitation is invalid, expired, or already accepted")
	ErrSlugConflict      = errors.New("tenancy: tenant slug already exists")
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,46}[a-z0-9]$`)

// User represents a TutorGrid user synchronized from a verified Limen identity.
type User struct {
	ID              uuid.UUID `json:"id"`
	ExternalSubject string    `json:"-"`
	DisplayName     string    `json:"display_name"`
	Email           string    `json:"email"`
}

// Membership summarizes a user's authorized membership in a tenant.
type Membership struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	TenantName string    `json:"tenant_name"`
	TenantSlug string    `json:"tenant_slug"`
	Role       Role      `json:"role"`
}

// Member represents a tenant member returned from an RLS-scoped query.
type Member struct {
	MembershipID uuid.UUID `json:"membership_id"`
	UserID       uuid.UUID `json:"user_id"`
	DisplayName  string    `json:"display_name"`
	Email        string    `json:"email"`
	Role         Role      `json:"role"`
}

// CreatedInvitation contains the one-time invite token returned to an org_admin.
type CreatedInvitation struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Service manages verified user sync, pre-tenant membership lookup, onboarding,
// active tenant switching, and RLS-scoped tenant operations.
type Service struct {
	db *bun.DB
}

// NewService constructs a tenancy Service backed by Bun.
func NewService(db *bun.DB) *Service {
	return &Service{db: db}
}

// UpsertUserBySubject creates or updates a TutorGrid user from a verified identity.
func (s *Service) UpsertUserBySubject(ctx context.Context, externalSubject, displayName, email string) (User, error) {
	if strings.TrimSpace(externalSubject) == "" || strings.TrimSpace(email) == "" {
		return User{}, errors.New("tenancy: external subject and email are required")
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = email
	}

	var u User
	err := s.db.QueryRowContext(ctx,
		"SELECT id, external_subject, display_name, email FROM auth_upsert_user_by_subject(?, ?, ?)",
		externalSubject, displayName, email,
	).Scan(&u.ID, &u.ExternalSubject, &u.DisplayName, &u.Email)
	if err != nil {
		return User{}, fmt.Errorf("tenancy: upsert user: %w", err)
	}
	return u, nil
}

// ListUserMemberships returns all authorized tenant memberships for a verified user.
func (s *Service) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]Membership, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT tenant_id, tenant_name, tenant_slug, role FROM auth_list_user_memberships(?)",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("tenancy: list memberships: %w", err)
	}
	defer rows.Close()

	memberships := make([]Membership, 0)
	for rows.Next() {
		var m Membership
		var roleStr string
		if err := rows.Scan(&m.TenantID, &m.TenantName, &m.TenantSlug, &roleStr); err != nil {
			return nil, fmt.Errorf("tenancy: scan membership: %w", err)
		}
		m.Role = Role(roleStr)
		memberships = append(memberships, m)
	}
	return memberships, rows.Err()
}

// GetActiveTenantID returns the server-bound active tenant ID for a verified user, if any.
func (s *Service) GetActiveTenantID(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	var tenantID uuid.NullUUID
	err := s.db.QueryRowContext(ctx, "SELECT auth_get_active_tenant(?)", userID).Scan(&tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("tenancy: get active tenant: %w", err)
	}
	if !tenantID.Valid {
		return nil, nil
	}
	id := tenantID.UUID
	return &id, nil
}

// ResolveVerifiedTenantContext derives a verified tenancy.Context strictly from
// the user's server-bound active tenant and current PostgreSQL membership.
func (s *Service) ResolveVerifiedTenantContext(ctx context.Context, userID uuid.UUID) (Context, error) {
	activeTenantID, err := s.GetActiveTenantID(ctx, userID)
	if err != nil {
		return Context{}, err
	}
	if activeTenantID == nil {
		return Context{}, ErrNoActiveTenant
	}

	var roleStr string
	err = s.db.QueryRowContext(ctx,
		"SELECT role FROM auth_verify_user_membership(?, ?)",
		userID, *activeTenantID,
	).Scan(&roleStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Context{}, ErrNotMember
		}
		return Context{}, fmt.Errorf("tenancy: verify membership: %w", err)
	}

	return NewVerifiedContext(*activeTenantID, userID, Role(roleStr))
}

// SetActiveTenant verifies membership and updates the user's server-bound active tenant.
func (s *Service) SetActiveTenant(ctx context.Context, userID, tenantID uuid.UUID) (Role, error) {
	var tid uuid.UUID
	var roleStr string
	err := s.db.QueryRowContext(ctx,
		"SELECT tenant_id, role FROM auth_set_active_tenant(?, ?)",
		userID, tenantID,
	).Scan(&tid, &roleStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotMember
		}
		return "", fmt.Errorf("tenancy: set active tenant: %w", err)
	}
	return Role(roleStr), nil
}

// CreateTenantWithAdmin creates a new Teaching Space / Organization and assigns
// the verified user as org_admin and sets it as their active tenant.
func (s *Service) CreateTenantWithAdmin(ctx context.Context, userID uuid.UUID, name, slug string) (Membership, error) {
	trimmedName := strings.TrimSpace(name)
	if len(trimmedName) < 2 || len(trimmedName) > 100 {
		return Membership{}, ErrInvalidName
	}
	normalizedSlug := strings.ToLower(strings.TrimSpace(slug))
	if !slugRegex.MatchString(normalizedSlug) {
		return Membership{}, ErrInvalidSlug
	}

	var m Membership
	var roleStr string
	err := s.db.QueryRowContext(ctx,
		"SELECT tenant_id, tenant_name, tenant_slug, role FROM auth_create_tenant_with_admin(?, ?, ?)",
		userID, trimmedName, normalizedSlug,
	).Scan(&m.TenantID, &m.TenantName, &m.TenantSlug, &roleStr)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return Membership{}, ErrSlugConflict
		}
		return Membership{}, fmt.Errorf("tenancy: create tenant: %w", err)
	}
	m.Role = Role(roleStr)
	return m, nil
}

// CreateInvitation creates an RLS-scoped invitation inside RunInTenantTx.
func (s *Service) CreateInvitation(ctx context.Context, tc Context, email string, role Role) (CreatedInvitation, error) {
	trimmedEmail := strings.ToLower(strings.TrimSpace(email))
	if trimmedEmail == "" || !strings.Contains(trimmedEmail, "@") {
		return CreatedInvitation{}, ErrInvalidEmail
	}
	if role != RoleOrgAdmin && role != RoleTeacher && role != RoleLearner {
		return CreatedInvitation{}, ErrInvalidRole
	}

	rawBytes := make([]byte, 24)
	if _, err := rand.Read(rawBytes); err != nil {
		return CreatedInvitation{}, fmt.Errorf("tenancy: generate invite token: %w", err)
	}
	rawToken := hex.EncodeToString(rawBytes)
	tokenHash := sha256.Sum256([]byte(rawToken))
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)

	var invID uuid.UUID
	err := database.RunInTenantTx(ctx, s.db, tc, func(ctx context.Context, tx bun.Tx) error {
		return tx.QueryRowContext(ctx, `
			INSERT INTO tenant_invitations (tenant_id, email, role, token_hash, invited_by_user_id, expires_at)
			VALUES (?, ?, ?, ?, ?, ?)
			RETURNING id
		`, tc.TenantID(), trimmedEmail, string(role), tokenHash[:], tc.UserID(), expiresAt).Scan(&invID)
	})
	if err != nil {
		return CreatedInvitation{}, fmt.Errorf("tenancy: create invitation: %w", err)
	}

	return CreatedInvitation{
		ID:        invID,
		Email:     trimmedEmail,
		Role:      role,
		Token:     rawToken,
		ExpiresAt: expiresAt,
	}, nil
}

// AcceptInvitation validates a one-time invitation token and joins the tenant.
func (s *Service) AcceptInvitation(ctx context.Context, userID uuid.UUID, rawToken string) (Membership, error) {
	trimmed := strings.TrimSpace(rawToken)
	if trimmed == "" {
		return Membership{}, ErrInvalidInvitation
	}
	tokenHash := sha256.Sum256([]byte(trimmed))

	var m Membership
	var roleStr string
	err := s.db.QueryRowContext(ctx,
		"SELECT tenant_id, tenant_name, tenant_slug, role FROM auth_accept_tenant_invitation(?, ?)",
		userID, tokenHash[:],
	).Scan(&m.TenantID, &m.TenantName, &m.TenantSlug, &roleStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Membership{}, ErrInvalidInvitation
		}
		return Membership{}, fmt.Errorf("tenancy: accept invitation: %w", err)
	}
	m.Role = Role(roleStr)
	return m, nil
}

// ListCurrentTenantMembers queries members strictly inside an RLS-scoped transaction.
func (s *Service) ListCurrentTenantMembers(ctx context.Context, tc Context) ([]Member, error) {
	members := make([]Member, 0)
	err := database.RunInTenantTx(ctx, s.db, tc, func(ctx context.Context, tx bun.Tx) error {
		rows, err := tx.QueryContext(ctx, `
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
			var m Member
			var roleStr string
			if err := rows.Scan(&m.MembershipID, &m.UserID, &m.DisplayName, &m.Email, &roleStr); err != nil {
				return err
			}
			m.Role = Role(roleStr)
			members = append(members, m)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("tenancy: list current members: %w", err)
	}
	return members, nil
}
