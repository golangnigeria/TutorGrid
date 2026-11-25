package authz

import "github.com/tutorgrid/tutorgrid/internal/tenancy"

// Permission represents an action gated by TutorGrid's RBAC policy.
type Permission string

const (
	PermissionTenantReadMembers  Permission = "tenant:members:read"
	PermissionTenantInviteMember Permission = "tenant:members:invite"
	PermissionLessonAuthor       Permission = "lesson:author"
	PermissionClassroomHost      Permission = "classroom:host"
	PermissionClassroomJoin      Permission = "classroom:join"
)

var rolePermissions = map[tenancy.Role]map[Permission]bool{
	tenancy.RolePlatformAdmin: {
		PermissionTenantReadMembers:  true,
		PermissionTenantInviteMember: true,
		PermissionLessonAuthor:       true,
		PermissionClassroomHost:      true,
		PermissionClassroomJoin:      true,
	},
	tenancy.RoleOrgAdmin: {
		PermissionTenantReadMembers:  true,
		PermissionTenantInviteMember: true,
		PermissionLessonAuthor:       true,
		PermissionClassroomHost:      true,
		PermissionClassroomJoin:      true,
	},
	tenancy.RoleTeacher: {
		PermissionTenantReadMembers: true,
		PermissionLessonAuthor:      true,
		PermissionClassroomHost:     true,
		PermissionClassroomJoin:     true,
	},
	tenancy.RoleLearner: {
		PermissionClassroomJoin: true,
	},
}

// Can reports whether the given role is authorized for perm.
func Can(role tenancy.Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
}
