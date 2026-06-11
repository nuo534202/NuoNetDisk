package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_Success(t *testing.T) {
	password := "my-secure-password-123"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	assert.NotEqual(t, password, hash)
	assert.NoError(t, VerifyPassword(hash, password))
}

func TestVerifyPassword_Correct(t *testing.T) {
	password := "correct-password"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	err = VerifyPassword(hash, password)
	assert.NoError(t, err)
}

func TestVerifyPassword_Wrong(t *testing.T) {
	hash, err := HashPassword("correct-password")
	require.NoError(t, err)

	err = VerifyPassword(hash, "wrong-password")
	assert.Error(t, err)
}

func TestHashPassword_EmptyString(t *testing.T) {
	hash, err := HashPassword("")
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	assert.NoError(t, VerifyPassword(hash, ""))
}
