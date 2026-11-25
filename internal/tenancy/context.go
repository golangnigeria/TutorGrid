package tenancy

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Role defines a tenant-scoped authorization role in TutorGrid.
type Role string

const (
	RolePlatformAdmin Role = "platform_admin"
	RoleOrgAdmin      Role = "org_admin"
	RoleTeacher       Role = "teacher"
	RoleLearner       Role = "learner"
)

// Valid reports whether the role is one of the recognized MVP roles.
func (r Role) Valid() bool {
	switch r {
	case RolePlatformAdmin, RoleOrgAdmin, RoleTeacher, RoleLearner:
		return true
	default:
		return false
	}
}

var (
	ErrInvalidTenantID = errors.New("tenancy: invalid tenant id")
	ErrInvalidUserID   = errors.New("tenancy: invalid user id")
	ErrInvalidRole     = errors.New("tenancy: invalid role")
)

// Context represents a verified tenant authorization context.
// Production code must construct Context only after verifying user identity
// and checking authorized tenant membership—never from raw client headers.
type Context struct {
	tenantID uuid.UUID
	userID   uuid.UUID
	role     Role
}

// NewVerifiedContext validates and builds a tenant Context.
func NewVerifiedContext(tenantID, userID uuid.UUID, role Role) (Context, error) {
	if tenantID == uuid.Nil {
		return Context{}, ErrInvalidTenantID
	}
	if userID == uuid.Nil {
		return Context{}, ErrInvalidUserID
	}
	if !role.Valid() {
		return Context{}, fmt.Errorf("%w: %q", ErrInvalidRole, role)
	}
	return Context{
		tenantID: tenantID,
		userID:   userID,
		role:     role,
	}, nil
}

// TenantID returns the verified tenant UUID.
func (c Context) TenantID() uuid.UUID { return c.tenantID }

// UserID returns the verified user UUID.
func (c Context) UserID() uuid.UUID { return c.userID }

// Role returns the verified tenant role.
func (c Context) Role() Role { return c.role }

// Valid reports whether Context holds a non-nil tenant ID, user ID, and valid role.
func (c Context) Valid() bool {
	return c.tenantID != uuid.Nil && c.userID != uuid.Nil && c.role.Valid()
}

type contextKey struct{}

// WithContext attaches a verified tenant Context to a Go context.
func WithContext(ctx context.Context, tc Context) context.Context {
	return context.WithValue(ctx, contextKey{}, tc)
}

// FromContext retrieves a verified tenant Context from a Go context.
func FromContext(ctx context.Context) (Context, bool) {
	tc, ok := ctx.Value(contextKey{}).(Context)
	if !ok || !tc.Valid() {
		return Context{}, false
	}
	return tc, true
}
