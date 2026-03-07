package identity

import "testing"

func TestPasswordManager(t *testing.T) {
	passwordManager := NewPasswordManager()
	plainPassword := "secret123"

	hashedPassword, err := passwordManager.Hash(plainPassword)
	if err != nil {
		t.Fatalf("expected hash success, got error: %v", err)
	}

	if hashedPassword == plainPassword {
		t.Fatalf("expected hashed password to differ from plain password")
	}

	if !passwordManager.Compare(hashedPassword, plainPassword) {
		t.Fatalf("expected password comparison to succeed")
	}

	if passwordManager.Compare(hashedPassword, "wrong-password") {
		t.Fatalf("expected wrong password comparison to fail")
	}
}
