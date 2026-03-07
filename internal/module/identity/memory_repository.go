package identity

import (
	"fmt"
	"strings"
	"sync"
)

// MemoryUserRepository 使用内存存储用户，当前阶段用于先稳定服务边界和测试行为。
type MemoryUserRepository struct {
	mutex       sync.RWMutex
	usersByID   map[string]User
	idByEmail   map[string]string
	nextUserSeq uint64
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		usersByID: make(map[string]User),
		idByEmail: make(map[string]string),
	}
}

func (repository *MemoryUserRepository) FindByEmail(email string) (User, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	userID, ok := repository.idByEmail[strings.ToLower(email)]
	if !ok {
		return User{}, false
	}

	user, ok := repository.usersByID[userID]
	return user, ok
}

func (repository *MemoryUserRepository) Save(user User) User {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	if user.ID == "" {
		repository.nextUserSeq++
		user.ID = fmt.Sprintf("user-%d", repository.nextUserSeq)
	}

	repository.usersByID[user.ID] = user
	repository.idByEmail[strings.ToLower(user.Email)] = user.ID
	return user
}

// MemoryMembershipRepository 使用内存维护成员关系。
type MemoryMembershipRepository struct {
	mutex             sync.RWMutex
	membershipsByUser map[string]Membership
}

func NewMemoryMembershipRepository() *MemoryMembershipRepository {
	return &MemoryMembershipRepository{
		membershipsByUser: make(map[string]Membership),
	}
}

func (repository *MemoryMembershipRepository) FindByUserID(userID string) (Membership, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	membership, ok := repository.membershipsByUser[userID]
	return membership, ok
}

func (repository *MemoryMembershipRepository) Save(membership Membership) Membership {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	repository.membershipsByUser[membership.UserID] = membership
	return membership
}
