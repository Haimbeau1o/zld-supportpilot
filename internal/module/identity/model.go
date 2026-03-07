package identity

// User 表示系统中的登录用户。
type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
}

// Organization 表示一个租户组织。
type Organization struct {
	ID   string
	Name string
}

// Membership 表示用户与组织之间的成员关系。
type Membership struct {
	UserID         string
	OrganizationID string
	Role           Role
}

// IdentityContext 是后续模块最关心的身份上下文。
type IdentityContext struct {
	UserID         string
	OrganizationID string
	Role           Role
	Permissions    PermissionSet
}
