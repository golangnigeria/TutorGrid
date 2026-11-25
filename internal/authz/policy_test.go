package authz_test

import (
	"testing"

	"github.com/tutorgrid/tutorgrid/internal/authz"
	"github.com/tutorgrid/tutorgrid/internal/tenancy"
)

func TestRBACPolicyMatrix(t *testing.T) {
	tests := []struct {
		role tenancy.Role
		perm authz.Permission
		want bool
	}{
		{tenancy.RolePlatformAdmin, authz.PermissionTenantReadMembers, true},
		{tenancy.RolePlatformAdmin, authz.PermissionTenantInviteMember, true},
		{tenancy.RolePlatformAdmin, authz.PermissionLessonAuthor, true},
		{tenancy.RolePlatformAdmin, authz.PermissionClassroomHost, true},
		{tenancy.RolePlatformAdmin, authz.PermissionClassroomJoin, true},

		{tenancy.RoleOrgAdmin, authz.PermissionTenantReadMembers, true},
		{tenancy.RoleOrgAdmin, authz.PermissionTenantInviteMember, true},
		{tenancy.RoleOrgAdmin, authz.PermissionLessonAuthor, true},
		{tenancy.RoleOrgAdmin, authz.PermissionClassroomHost, true},
		{tenancy.RoleOrgAdmin, authz.PermissionClassroomJoin, true},

		{tenancy.RoleTeacher, authz.PermissionTenantReadMembers, true},
		{tenancy.RoleTeacher, authz.PermissionTenantInviteMember, false},
		{tenancy.RoleTeacher, authz.PermissionLessonAuthor, true},
		{tenancy.RoleTeacher, authz.PermissionClassroomHost, true},
		{tenancy.RoleTeacher, authz.PermissionClassroomJoin, true},

		{tenancy.RoleLearner, authz.PermissionTenantReadMembers, false},
		{tenancy.RoleLearner, authz.PermissionTenantInviteMember, false},
		{tenancy.RoleLearner, authz.PermissionLessonAuthor, false},
		{tenancy.RoleLearner, authz.PermissionClassroomHost, false},
		{tenancy.RoleLearner, authz.PermissionClassroomJoin, true},

		{tenancy.Role("unknown"), authz.PermissionClassroomJoin, false},
	}

	for _, tc := range tests {
		if got := authz.Can(tc.role, tc.perm); got != tc.want {
			t.Errorf("Can(%q, %q) = %v, want %v", tc.role, tc.perm, got, tc.want)
		}
	}
}
