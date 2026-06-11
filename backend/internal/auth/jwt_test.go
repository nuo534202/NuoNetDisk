package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_GenerateAndValidate(t *testing.T) {
	svc := NewJWTService("test-secret-key-that-is-32-chars-long!", 15*time.Minute)

	userID := uuid.New()
	email := "test@example.com"

	token, err := svc.GenerateAccessToken(userID, email)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := svc.ValidateAccessToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
}

func TestJWTService_RejectExpiredToken(t *testing.T) {
	svc := NewJWTService("test-secret-key-that-is-32-chars-long!", -1*time.Minute)

	userID := uuid.New()
	token, err := svc.GenerateAccessToken(userID, "test@example.com")
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestJWTService_RejectInvalidSignature(t *testing.T) {
	svc1 := NewJWTService("test-secret-key-that-is-32-chars-long!", 15*time.Minute)
	svc2 := NewJWTService("different-secret-key-that-is-also-32-chars!!", 15*time.Minute)

	userID := uuid.New()
	token, err := svc1.GenerateAccessToken(userID, "test@example.com")
	require.NoError(t, err)

	_, err = svc2.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestJWTService_RejectMalformedToken(t *testing.T) {
	svc := NewJWTService("test-secret-key-that-is-32-chars-long!", 15*time.Minute)

	_, err := svc.ValidateAccessToken("malformed-token-string")
	assert.Error(t, err)
}
