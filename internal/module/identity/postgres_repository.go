package identity

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/Haimbeau1o/zld-supportpilot/internal/platform/persistence"
)

// PostgresUserRepository 使用 PostgreSQL 持久化用户数据。
type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (repository *PostgresUserRepository) FindByEmail(email string) (User, bool) {
	row := repository.db.QueryRow(`SELECT id, name, email, password_hash FROM identity_users WHERE email = $1`, email)

	user := User{}
	if err := row.Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, false
		}
		panic(fmt.Sprintf("find user by email from postgres: %v", err))
	}

	return user, true
}

func (repository *PostgresUserRepository) Save(user User) User {
	if user.ID == "" {
		user.ID = persistence.NewID("user")
	}

	if _, err := repository.db.Exec(
		`INSERT INTO identity_users (id, name, email, password_hash) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email, password_hash = EXCLUDED.password_hash`,
		user.ID,
		user.Name,
		user.Email,
		user.PasswordHash,
	); err != nil {
		// 当前 repository 接口尚未上抛 error，这里显式失败，避免数据库异常被静默吞掉。
		panic(fmt.Sprintf("save user to postgres: %v", err))
	}

	return user
}

// PostgresMembershipRepository 使用 PostgreSQL 持久化成员关系。
type PostgresMembershipRepository struct {
	db *sql.DB
}

func NewPostgresMembershipRepository(db *sql.DB) *PostgresMembershipRepository {
	return &PostgresMembershipRepository{db: db}
}

func (repository *PostgresMembershipRepository) FindByUserID(userID string) (Membership, bool) {
	row := repository.db.QueryRow(`SELECT user_id, organization_id, role FROM identity_memberships WHERE user_id = $1`, userID)

	membership := Membership{}
	var role string
	if err := row.Scan(&membership.UserID, &membership.OrganizationID, &role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Membership{}, false
		}
		panic(fmt.Sprintf("find membership by user id from postgres: %v", err))
	}
	membership.Role = Role(role)
	return membership, true
}

func (repository *PostgresMembershipRepository) Save(membership Membership) Membership {
	if _, err := repository.db.Exec(
		`INSERT INTO identity_memberships (user_id, organization_id, role) VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET organization_id = EXCLUDED.organization_id, role = EXCLUDED.role`,
		membership.UserID,
		membership.OrganizationID,
		string(membership.Role),
	); err != nil {
		panic(fmt.Sprintf("save membership to postgres: %v", err))
	}

	return membership
}
