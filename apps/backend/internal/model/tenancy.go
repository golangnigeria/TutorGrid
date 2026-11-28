package model

import (
	"time"

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

func (r Role) Valid() bool {
	switch r {
	case RolePlatformAdmin, RoleOrgAdmin, RoleTeacher, RoleLearner:
		return true
	default:
		return false
	}
}

// Permission represents an action gated by TutorGrid's RBAC policy.
type Permission string

const (
	PermissionTenantReadMembers  Permission = "tenant:members:read"
	PermissionTenantInviteMember Permission = "tenant:members:invite"
	PermissionTenantRevokeInvite Permission = "tenant:invitations:revoke"
	PermissionResourceRead       Permission = "resource:read"
	PermissionResourceWrite      Permission = "resource:write"
	PermissionLessonAuthor       Permission = "lesson:author"
	PermissionClassroomHost      Permission = "classroom:host"
	PermissionClassroomJoin      Permission = "classroom:join"
)

var rolePermissions = map[Role]map[Permission]bool{
	RolePlatformAdmin: {
		PermissionTenantReadMembers:  true,
		PermissionTenantInviteMember: true,
		PermissionTenantRevokeInvite: true,
		PermissionResourceRead:       true,
		PermissionResourceWrite:      true,
		PermissionLessonAuthor:       true,
		PermissionClassroomHost:      true,
		PermissionClassroomJoin:      true,
	},
	RoleOrgAdmin: {
		PermissionTenantReadMembers:  true,
		PermissionTenantInviteMember: true,
		PermissionTenantRevokeInvite: true,
		PermissionResourceRead:       true,
		PermissionResourceWrite:      true,
		PermissionLessonAuthor:       true,
		PermissionClassroomHost:      true,
		PermissionClassroomJoin:      true,
	},
	RoleTeacher: {
		PermissionTenantReadMembers: true,
		PermissionResourceRead:      true,
		PermissionResourceWrite:     true,
		PermissionLessonAuthor:      true,
		PermissionClassroomHost:     true,
		PermissionClassroomJoin:     true,
	},
	RoleLearner: {
		PermissionClassroomJoin: true,
	},
}

// Can reports whether role is authorized to perform perm.
func Can(role Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
}

// TenantContext is derived strictly server-side from verified identity + membership.
type TenantContext struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	Role     Role
}

func (tc TenantContext) Valid() bool {
	return tc.TenantID != uuid.Nil && tc.UserID != uuid.Nil && tc.Role.Valid()
}

type User struct {
	ID              uuid.UUID `json:"id"`
	ExternalSubject string    `json:"-"`
	DisplayName     string    `json:"display_name"`
	Email           string    `json:"email"`
}

type Membership struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	TenantName string    `json:"tenant_name"`
	TenantSlug string    `json:"tenant_slug"`
	Role       Role      `json:"role"`
}

type Member struct {
	MembershipID uuid.UUID `json:"membership_id"`
	UserID       uuid.UUID `json:"user_id"`
	DisplayName  string    `json:"display_name"`
	Email        string    `json:"email"`
	Role         Role      `json:"role"`
}

type CreatedInvitation struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	Role        Role      `json:"role"`
	Token       string    `json:"-"`
	EmailStatus string    `json:"email_status"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type PendingInvitation struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	Role        Role      `json:"role"`
	EmailStatus string    `json:"email_status"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserPendingInvitation struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	TenantName string    `json:"tenant_name"`
	TenantSlug string    `json:"tenant_slug"`
	Role       Role      `json:"role"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type OutboxEntry struct {
	ID             uuid.UUID
	InvitationID   uuid.UUID
	TenantID       uuid.UUID
	RecipientEmail string
	TenantName     string
	Role           Role
	InviteURL      string
	Status         string
	Attempts       int
}
