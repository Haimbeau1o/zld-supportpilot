package identity

import (
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresUserRepositorySaveAndFindByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresUserRepository(db)
	user := User{Name: "Alice", Email: "alice@example.com", PasswordHash: "hashed-secret"}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO identity_users (id, name, email, password_hash) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email, password_hash = EXCLUDED.password_hash")).
		WithArgs(sqlmock.AnyArg(), user.Name, user.Email, user.PasswordHash).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(user)
	if saved.ID == "" {
		t.Fatalf("expected generated user id")
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email, password_hash FROM identity_users WHERE email = $1")).
		WithArgs(user.Email).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password_hash"}).AddRow(saved.ID, user.Name, user.Email, user.PasswordHash))

	stored, ok := repository.FindByEmail(user.Email)
	if !ok {
		t.Fatalf("expected stored user to be found")
	}
	if stored.Email != user.Email {
		t.Fatalf("expected stored email %q, got %q", user.Email, stored.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPostgresMembershipRepositorySaveAndFindByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresMembershipRepository(db)
	membership := Membership{UserID: "user-1", OrganizationID: "org-1", Role: RoleTenantAdmin}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO identity_memberships (user_id, organization_id, role) VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET organization_id = EXCLUDED.organization_id, role = EXCLUDED.role")).
		WithArgs(membership.UserID, membership.OrganizationID, string(membership.Role)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	saved := repository.Save(membership)
	if saved.UserID != membership.UserID {
		t.Fatalf("expected membership user id %q, got %q", membership.UserID, saved.UserID)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id, organization_id, role FROM identity_memberships WHERE user_id = $1")).
		WithArgs(membership.UserID).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "organization_id", "role"}).AddRow(membership.UserID, membership.OrganizationID, string(membership.Role)))

	stored, ok := repository.FindByUserID(membership.UserID)
	if !ok {
		t.Fatalf("expected membership to be found")
	}
	if stored.OrganizationID != membership.OrganizationID {
		t.Fatalf("expected organization id %q, got %q", membership.OrganizationID, stored.OrganizationID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
