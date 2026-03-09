package identity

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/persistence"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

var nextOrganizationSeq atomic.Uint64

// PasswordManager 抽象密码哈希和校验能力，方便当前阶段先做服务边界，后续再接入正式实现。
type PasswordManager interface {
	Hash(password string) (string, error)
	Compare(hashedPassword string, plainPassword string) bool
}

type RegisterInput struct {
	Name        string
	Email       string
	Password    string
	TenantName  string
	DefaultRole Role
}

type LoginInput struct {
	Email    string
	Password string
}

type Service struct {
	userRepository       UserRepository
	membershipRepository MembershipRepository
	passwordManager      PasswordManager
}

func NewService(userRepository UserRepository, membershipRepository MembershipRepository, passwordManager PasswordManager) *Service {
	return &Service{
		userRepository:       userRepository,
		membershipRepository: membershipRepository,
		passwordManager:      passwordManager,
	}
}

func (service *Service) Register(input RegisterInput) (User, Membership, error) {
	normalizedEmail := strings.TrimSpace(strings.ToLower(input.Email))
	if _, exists := service.userRepository.FindByEmail(normalizedEmail); exists {
		return User{}, Membership{}, ErrEmailAlreadyExists
	}

	hashedPassword, err := service.passwordManager.Hash(input.Password)
	if err != nil {
		return User{}, Membership{}, fmt.Errorf("hash password: %w", err)
	}

	user := service.userRepository.Save(User{
		Name:         strings.TrimSpace(input.Name),
		Email:        normalizedEmail,
		PasswordHash: hashedPassword,
	})

	defaultRole := input.DefaultRole
	if defaultRole == "" {
		// 当前阶段默认把首个注册用户视为租户管理员，用于快速建立租户级管理入口。
		defaultRole = RoleTenantAdmin
	}

	membership := service.membershipRepository.Save(Membership{
		UserID:         user.ID,
		OrganizationID: nextOrganizationID(),
		Role:           defaultRole,
	})

	return user, membership, nil
}

func (service *Service) GetUser(userID string) (User, bool) {
	return service.userRepository.FindByID(strings.TrimSpace(userID))
}

func (service *Service) Login(input LoginInput) (IdentityContext, error) {
	user, exists := service.userRepository.FindByEmail(strings.TrimSpace(strings.ToLower(input.Email)))
	if !exists {
		// 这里统一返回凭证错误，避免暴露“用户不存在”和“密码错误”的差异。
		return IdentityContext{}, ErrInvalidCredentials
	}

	if !service.passwordManager.Compare(user.PasswordHash, input.Password) {
		return IdentityContext{}, ErrInvalidCredentials
	}

	membership, exists := service.membershipRepository.FindByUserID(user.ID)
	if !exists {
		return IdentityContext{}, ErrInvalidCredentials
	}

	return IdentityContext{
		UserID:         user.ID,
		OrganizationID: membership.OrganizationID,
		Role:           membership.Role,
		Permissions:    RolePermissions(membership.Role),
	}, nil
}

func nextOrganizationID() string {
	// 租户 ID 一旦落入持久化存储，就不能依赖进程内自增序列，否则服务重启会与历史组织发生碰撞。
	return persistence.NewID("org")
}
