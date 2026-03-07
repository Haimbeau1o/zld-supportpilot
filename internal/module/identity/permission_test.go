package identity

import "testing"

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		name        string
		role        Role
		permissions []Permission
	}{
		{
			name: "tenant admin should have tenant and ticket management permissions",
			role: RoleTenantAdmin,
			permissions: []Permission{
				PermissionTenantManage,
				PermissionTicketWrite,
				PermissionKnowledgeWrite,
			},
		},
		{
			name: "agent should have ticket and knowledge read write permissions",
			role: RoleAgent,
			permissions: []Permission{
				PermissionTicketRead,
				PermissionTicketWrite,
				PermissionKnowledgeRead,
			},
		},
		{
			name: "end user should only have own ticket permissions",
			role: RoleEndUser,
			permissions: []Permission{
				PermissionTicketSelfCreate,
				PermissionTicketSelfRead,
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			permissionSet := RolePermissions(testCase.role)
			if len(permissionSet) == 0 {
				t.Fatalf("expected permissions for role %q", testCase.role)
			}

			for _, permission := range testCase.permissions {
				if !permissionSet.Contains(permission) {
					t.Fatalf("expected role %q to contain permission %q", testCase.role, permission)
				}
			}
		})
	}
}
