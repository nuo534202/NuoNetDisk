package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	secretBytesLen = 32
	refreshCost    = 10
)

type RefreshTokenService struct{}

func NewRefreshTokenService() *RefreshTokenService {
	return &RefreshTokenService{}
}

// GenerateRefreshToken creates a new refresh token in the format: tokenID.secret
func (s *RefreshTokenService) GenerateRefreshToken() (tokenID string, secret string, hashedSecret string, err error) {
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate token ID: %w", err)
	}

	secretBytes := make([]byte, secretBytesLen)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate token secret: %w", err)
	}

	tokenID = hex.EncodeToString(idBytes)
	secret = hex.EncodeToString(secretBytes)

	hashed, err := bcrypt.GenerateFromPassword([]byte(secret), refreshCost)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to hash token: %w", err)
	}

	return tokenID, secret, string(hashed), nil
}

// ParseRefreshToken splits a combined token string into tokenID and secret
func (s *RefreshTokenService) ParseRefreshToken(combined string) (tokenID string, secret string, err error) {
	parts := strings.SplitN(combined, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid refresh token format")
	}
	return parts[0], parts[1], nil
}

// VerifyRefreshToken checks if the provided secret matches the stored hash
func (s *RefreshTokenService) VerifyRefreshToken(secret, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)); err != nil {
		return fmt.Errorf("invalid refresh token")
	}
	return nil
}

// HashTokenID creates a SHA-256 hash of the token ID for DB lookup
func (s *RefreshTokenService) HashTokenID(tokenID string) string {
	sum := sha256.Sum256([]byte(tokenID))
	return hex.EncodeToString(sum[:])
}
