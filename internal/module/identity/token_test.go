package identity

import (
	"testing"
	"time"
)

func TestTokenManagerIssueAndParse(t *testing.T) {
	tokenManager := NewTokenManager("test-signing-key", time.Hour)
	issuedToken, err := tokenManager.Issue(IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           RoleTenantAdmin,
		Permissions:    RolePermissions(RoleTenantAdmin),
	})
	if err != nil {
		t.Fatalf("expected token issue success, got error: %v", err)
	}

	claims, err := tokenManager.Parse(issuedToken)
	if err != nil {
		t.Fatalf("expected token parse success, got error: %v", err)
	}

	if claims.UserID != "user-1" {
		t.Fatalf("expected user id user-1, got %q", claims.UserID)
	}

	if claims.OrganizationID != "org-1" {
		t.Fatalf("expected organization id org-1, got %q", claims.OrganizationID)
	}
}

func TestTokenManagerRejectsTamperedToken(t *testing.T) {
	tokenManager := NewTokenManager("test-signing-key", time.Hour)
	issuedToken, err := tokenManager.Issue(IdentityContext{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Role:           RoleTenantAdmin,
		Permissions:    RolePermissions(RoleTenantAdmin),
	})
	if err != nil {
		t.Fatalf("expected token issue success, got error: %v", err)
	}

	if _, err := tokenManager.Parse(issuedToken + "tampered"); err == nil {
		t.Fatalf("expected tampered token parse to fail")
	}
}
