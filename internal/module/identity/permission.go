package identity

// Role 定义当前阶段支持的固定角色集合。
// 这里先使用固定角色而不是可配置角色，是为了优先稳定后续业务模块依赖的权限边界。
type Role string

const (
	RoleTenantAdmin Role = "tenant_admin"
	RoleAgent       Role = "agent"
	RoleEndUser     Role = "end_user"
)

// Permission 描述系统中的最小权限粒度。
type Permission string

const (
	PermissionTenantManage     Permission = "tenant:manage"
	PermissionTicketRead       Permission = "ticket:read"
	PermissionTicketWrite      Permission = "ticket:write"
	PermissionTicketSelfCreate Permission = "ticket:self:create"
	PermissionTicketSelfRead   Permission = "ticket:self:read"
	PermissionKnowledgeRead    Permission = "knowledge:read"
	PermissionKnowledgeWrite   Permission = "knowledge:write"
)

// PermissionSet 作为权限集合使用，方便后续做快速判定。
type PermissionSet map[Permission]struct{}

// Contains 用于判断某个权限是否存在于集合中。
func (permissionSet PermissionSet) Contains(permission Permission) bool {
	_, ok := permissionSet[permission]
	return ok
}

// RolePermissions 返回角色对应的权限集合。
func RolePermissions(role Role) PermissionSet {
	switch role {
	case RoleTenantAdmin:
		// 租户管理员负责管理租户、处理工单并维护知识库，因此给予完整的租户管理和业务写权限。
		return newPermissionSet(
			PermissionTenantManage,
			PermissionTicketRead,
			PermissionTicketWrite,
			PermissionKnowledgeRead,
			PermissionKnowledgeWrite,
		)
	case RoleAgent:
		// 客服 / 处理人主要围绕工单和知识库工作，不应该拥有租户级管理权限。
		return newPermissionSet(
			PermissionTicketRead,
			PermissionTicketWrite,
			PermissionKnowledgeRead,
			PermissionKnowledgeWrite,
		)
	case RoleEndUser:
		// 终端用户只允许创建和查看自己的工单，避免越权访问他人数据。
		return newPermissionSet(
			PermissionTicketSelfCreate,
			PermissionTicketSelfRead,
		)
	default:
		return PermissionSet{}
	}
}

func newPermissionSet(permissions ...Permission) PermissionSet {
	permissionSet := make(PermissionSet, len(permissions))
	for _, permission := range permissions {
		permissionSet[permission] = struct{}{}
	}

	return permissionSet
}
