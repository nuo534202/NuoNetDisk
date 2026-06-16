package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/auth"
)

func TestJWTGenerateAndValidate(t *testing.T) {
	secret := "this-is-a-test-secret-that-is-32-chars!"
	svc := auth.NewJWTService(secret, 15*time.Minute)

	userID := uuid.New()
	email := "test@example.com"

	token, err := svc.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateAccessToken returned empty token")
	}

	claims, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected Email %q, got %q", email, claims.Email)
	}
}

func TestJWTInvalidToken(t *testing.T) {
	svc := auth.NewJWTService("this-is-a-test-secret-that-is-32-chars!", 15*time.Minute)

	_, err := svc.ValidateAccessToken("invalid-token-string")
	if err == nil {
		t.Fatal("ValidateAccessToken with invalid token should have failed")
	}
}

func TestJWTWrongSecret(t *testing.T) {
	svc1 := auth.NewJWTService("this-is-a-test-secret-that-is-32-chars!", 15*time.Minute)
	svc2 := auth.NewJWTService("a-different-secret-that-is-also-32-chars!!", 15*time.Minute)

	userID := uuid.New()
	token, err := svc1.GenerateAccessToken(userID, "test@example.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = svc2.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("ValidateAccessToken with wrong secret should have failed")
	}
}

func TestJWTExpiredToken(t *testing.T) {
	svc := auth.NewJWTService("this-is-a-test-secret-that-is-32-chars!", -1*time.Minute)

	userID := uuid.New()
	token, err := svc.GenerateAccessToken(userID, "test@example.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = svc.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("ValidateAccessToken with expired token should have failed")
	}
}

func TestJWTShortSecret(t *testing.T) {
	svc := auth.NewJWTService("short", 15*time.Minute)

	userID := uuid.New()
	token, err := svc.GenerateAccessToken(userID, "test@example.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken with short secret failed: %v", err)
	}

	claims, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken with short secret token failed: %v", err)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected Email %q, got %q", "test@example.com", claims.Email)
	}
}
