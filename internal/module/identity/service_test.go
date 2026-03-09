package identity

import "testing"

type fakePasswordManager struct{}

func (fakePasswordManager) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (fakePasswordManager) Compare(hashedPassword string, plainPassword string) bool {
	return hashedPassword == "hashed:"+plainPassword
}

func TestRegister(t *testing.T) {
	service := NewService(
		NewMemoryUserRepository(),
		NewMemoryMembershipRepository(),
		fakePasswordManager{},
	)

	registeredUser, membership, err := service.Register(RegisterInput{
		Name:        "Alice",
		Email:       "alice@example.com",
		Password:    "secret123",
		TenantName:  "中联数据支持中心",
		DefaultRole: RoleTenantAdmin,
	})
	if err != nil {
		t.Fatalf("expected register success, got error: %v", err)
	}

	if registeredUser.Email != "alice@example.com" {
		t.Fatalf("expected registered email to be alice@example.com, got %q", registeredUser.Email)
	}

	if membership.Role != RoleTenantAdmin {
		t.Fatalf("expected default role to be %q, got %q", RoleTenantAdmin, membership.Role)
	}

	if registeredUser.PasswordHash == "secret123" {
		t.Fatalf("expected password to be hashed before storage")
	}
}

func TestRegisterGeneratesUniqueOrganizationIDAcrossRestart(t *testing.T) {
	nextOrganizationSeq.Store(0)
	t.Cleanup(func() { nextOrganizationSeq.Store(0) })

	firstService := NewService(
		NewMemoryUserRepository(),
		NewMemoryMembershipRepository(),
		fakePasswordManager{},
	)
	_, firstMembership, err := firstService.Register(RegisterInput{
		Name:        "Alice",
		Email:       "alice@example.com",
		Password:    "secret123",
		TenantName:  "中联数据支持中心",
		DefaultRole: RoleTenantAdmin,
	})
	if err != nil {
		t.Fatalf("expected first register success, got error: %v", err)
	}

	// 模拟服务重启后重新从零开始分配租户 ID，持久化模式下不应产生重复组织标识。
	nextOrganizationSeq.Store(0)
	secondService := NewService(
		NewMemoryUserRepository(),
		NewMemoryMembershipRepository(),
		fakePasswordManager{},
	)
	_, secondMembership, err := secondService.Register(RegisterInput{
		Name:        "Bob",
		Email:       "bob@example.com",
		Password:    "secret456",
		TenantName:  "中联数据支持中心",
		DefaultRole: RoleTenantAdmin,
	})
	if err != nil {
		t.Fatalf("expected second register success, got error: %v", err)
	}

	if firstMembership.OrganizationID == secondMembership.OrganizationID {
		t.Fatalf("expected organization ids to remain unique across restart, got duplicated id %q", firstMembership.OrganizationID)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	service := NewService(
		NewMemoryUserRepository(),
		NewMemoryMembershipRepository(),
		fakePasswordManager{},
	)

	_, _, err := service.Register(RegisterInput{
		Name:        "Alice",
		Email:       "alice@example.com",
		Password:    "secret123",
		TenantName:  "中联数据支持中心",
		DefaultRole: RoleTenantAdmin,
	})
	if err != nil {
		t.Fatalf("expected first register success, got error: %v", err)
	}

	_, _, err = service.Register(RegisterInput{
		Name:        "Alice2",
		Email:       "alice@example.com",
		Password:    "secret456",
		TenantName:  "另一个租户",
		DefaultRole: RoleEndUser,
	})
	if err != ErrEmailAlreadyExists {
		t.Fatalf("expected duplicate email error, got %v", err)
	}
}

func TestLogin(t *testing.T) {
	service := NewService(
		NewMemoryUserRepository(),
		NewMemoryMembershipRepository(),
		fakePasswordManager{},
	)

	registeredUser, membership, err := service.Register(RegisterInput{
		Name:        "Alice",
		Email:       "alice@example.com",
		Password:    "secret123",
		TenantName:  "中联数据支持中心",
		DefaultRole: RoleTenantAdmin,
	})
	if err != nil {
		t.Fatalf("expected register success, got error: %v", err)
	}

	identityContext, err := service.Login(LoginInput{
		Email:    "alice@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("expected login success, got error: %v", err)
	}

	if identityContext.UserID != registeredUser.ID {
		t.Fatalf("expected user id %q, got %q", registeredUser.ID, identityContext.UserID)
	}

	if identityContext.OrganizationID != membership.OrganizationID {
		t.Fatalf("expected organization id %q, got %q", membership.OrganizationID, identityContext.OrganizationID)
	}

	if identityContext.Role != membership.Role {
		t.Fatalf("expected role %q, got %q", membership.Role, identityContext.Role)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	service := NewService(
		NewMemoryUserRepository(),
		NewMemoryMembershipRepository(),
		fakePasswordManager{},
	)

	_, _, err := service.Register(RegisterInput{
		Name:        "Alice",
		Email:       "alice@example.com",
		Password:    "secret123",
		TenantName:  "中联数据支持中心",
		DefaultRole: RoleTenantAdmin,
	})
	if err != nil {
		t.Fatalf("expected register success, got error: %v", err)
	}

	_, err = service.Login(LoginInput{
		Email:    "alice@example.com",
		Password: "wrong-password",
	})
	if err != ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
}
