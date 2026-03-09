package identity

// UserRepository 定义用户数据访问能力。
type UserRepository interface {
	FindByEmail(email string) (User, bool)
	FindByID(userID string) (User, bool)
	Save(user User) User
}

// MembershipRepository 定义成员关系数据访问能力。
type MembershipRepository interface {
	FindByUserID(userID string) (Membership, bool)
	Save(membership Membership) Membership
}
