package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/auth"
	"github.com/nuonuo/nuonetdisk/internal/model"
	"github.com/nuonuo/nuonetdisk/internal/repository"
)

type AuthService struct {
	userRepo         *repository.UserRepository
	refreshTokenRepo *repository.RefreshTokenRepository
	jwtService       *auth.JWTService
	refreshSvc       *auth.RefreshTokenService
	accessExpiry     time.Duration
	refreshExpiry    time.Duration
}

func NewAuthService(
	userRepo *repository.UserRepository,
	refreshTokenRepo *repository.RefreshTokenRepository,
	jwtService *auth.JWTService,
	refreshSvc *auth.RefreshTokenService,
	accessExpiry time.Duration,
	refreshExpiry time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
		refreshSvc:       refreshSvc,
		accessExpiry:     accessExpiry,
		refreshExpiry:    refreshExpiry,
	}
}

type AuthRegisterInput struct {
	Email    string
	Password string
}

type AuthLoginInput struct {
	Email    string
	Password string
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *AuthService) Register(ctx context.Context, input AuthRegisterInput) (*model.User, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	if email == "" {
		return nil, model.ErrInvalidInput
	}
	if len(input.Password) < 8 {
		return nil, model.ErrInvalidInput
	}

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, model.ErrDuplicate
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  strings.Split(email, "@")[0],
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, input AuthLoginInput) (*TokenPair, *model.User, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, nil, model.ErrUnauthenticated
		}
		return nil, nil, err
	}

	if err := auth.VerifyPassword(user.PasswordHash, input.Password); err != nil {
		return nil, nil, model.ErrUnauthenticated
	}

	pair, err := s.generateTokenPair(ctx, user.ID, user.Email)
	if err != nil {
		return nil, nil, err
	}

	return pair, user, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	tokenID, secret, err := s.refreshSvc.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, model.ErrInvalidInput
	}

	tokenUUID, err := uuid.Parse(tokenID)
	if err != nil {
		return nil, model.ErrInvalidInput
	}

	stored, err := s.refreshTokenRepo.GetByID(ctx, tokenUUID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, model.ErrTokenExpired
		}
		return nil, err
	}

	if stored.IsRevoked {
		s.refreshTokenRepo.RevokeAllForUser(ctx, stored.UserID)
		return nil, model.ErrTokenRevoked
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, model.ErrTokenExpired
	}

	if err := s.refreshSvc.VerifyRefreshToken(secret, stored.TokenHash); err != nil {
		return nil, model.ErrTokenRevoked
	}

	s.refreshTokenRepo.MarkRevoked(ctx, stored.ID)

	user, err := s.userRepo.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	return s.generateTokenPair(ctx, user.ID, user.Email)
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	tokenID, _, err := s.refreshSvc.ParseRefreshToken(refreshToken)
	if err != nil {
		return model.ErrInvalidInput
	}

	tokenUUID, err := uuid.Parse(tokenID)
	if err != nil {
		return model.ErrInvalidInput
	}

	stored, err := s.refreshTokenRepo.GetByID(ctx, tokenUUID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil
		}
		return err
	}

	if stored.UserID != userID {
		return model.ErrNotFound
	}

	return s.refreshTokenRepo.MarkRevoked(ctx, stored.ID)
}

func (s *AuthService) generateTokenPair(ctx context.Context, userID uuid.UUID, email string) (*TokenPair, error) {
	accessToken, err := s.jwtService.GenerateAccessToken(userID, email)
	if err != nil {
		return nil, err
	}

	tokenID, secret, hashedSecret, err := s.refreshSvc.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshTokenModel := &model.RefreshToken{
		UserID:    userID,
		TokenHash: hashedSecret,
		ExpiresAt: time.Now().Add(s.refreshExpiry),
	}

	if err := s.refreshTokenRepo.Insert(ctx, refreshTokenModel); err != nil {
		return nil, err
	}

	combinedToken := tokenID + "." + secret

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: combinedToken,
		ExpiresIn:    int(s.accessExpiry.Seconds()),
	}, nil
}
