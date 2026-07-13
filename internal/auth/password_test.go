package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	const password = "correct horse battery staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	match, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !match {
		t.Fatal("expected the correct password to match")
	}

	match, err = VerifyPassword("wrong password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword (wrong): %v", err)
	}
	if match {
		t.Fatal("expected a wrong password to be rejected")
	}
}
