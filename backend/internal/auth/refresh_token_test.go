package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenService_Generate(t *testing.T) {
	svc := NewRefreshTokenService()

	tokenID, secret, hashed, err := svc.GenerateRefreshToken()
	require.NoError(t, err)
	require.NotEmpty(t, tokenID)
	require.NotEmpty(t, secret)
	require.NotEmpty(t, hashed)

	assert.NotEqual(t, secret, hashed)
}

func TestRefreshTokenService_ParseAndVerify(t *testing.T) {
	svc := NewRefreshTokenService()

	tokenID, secret, hashed, err := svc.GenerateRefreshToken()
	require.NoError(t, err)

	combined := tokenID + "." + secret

	parsedID, parsedSecret, err := svc.ParseRefreshToken(combined)
	require.NoError(t, err)

	assert.Equal(t, tokenID, parsedID)
	assert.Equal(t, secret, parsedSecret)

	err = svc.VerifyRefreshToken(parsedSecret, hashed)
	assert.NoError(t, err)
}

func TestRefreshTokenService_VerifyWrongSecret(t *testing.T) {
	svc := NewRefreshTokenService()

	_, _, hashed, err := svc.GenerateRefreshToken()
	require.NoError(t, err)

	err = svc.VerifyRefreshToken("wrong-secret", hashed)
	assert.Error(t, err)
}

func TestRefreshTokenService_ParseInvalid(t *testing.T) {
	svc := NewRefreshTokenService()

	_, _, err := svc.ParseRefreshToken("invalid-format")
	assert.Error(t, err)

	_, _, err = svc.ParseRefreshToken("")
	assert.Error(t, err)
}

func TestRefreshTokenService_HashTokenID(t *testing.T) {
	svc := NewRefreshTokenService()

	tokenID, _, _, err := svc.GenerateRefreshToken()
	require.NoError(t, err)

	hash := svc.HashTokenID(tokenID)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)
}
