package tests

import (
	"testing"

	"github.com/nuonuo/nuonetdisk/internal/auth"
)

func TestHashAndVerifyPassword(t *testing.T) {
	plaintext := "MySecureP@ssw0rd!"
	hash, err := auth.HashPassword(plaintext)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}

	err = auth.VerifyPassword(hash, plaintext)
	if err != nil {
		t.Fatalf("VerifyPassword with correct password failed: %v", err)
	}
}

func TestVerifyPasswordWrongPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	err = auth.VerifyPassword(hash, "wrong-password")
	if err == nil {
		t.Fatal("VerifyPassword with wrong password should have failed")
	}
}

func TestVerifyPasswordEmptyHash(t *testing.T) {
	err := auth.VerifyPassword("", "password")
	if err == nil {
		t.Fatal("VerifyPassword with empty hash should have failed")
	}
}
