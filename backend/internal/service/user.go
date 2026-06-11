package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/model"
	"github.com/nuonuo/nuonetdisk/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

type UserDTO struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func userToDTO(u *model.User) UserDTO {
	return UserDTO{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   u.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *UserService) GetMe(ctx context.Context, userID uuid.UUID) (*UserDTO, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	dto := userToDTO(user)
	return &dto, nil
}

func (s *UserService) UpdateMe(ctx context.Context, userID uuid.UUID, displayName string) (*UserDTO, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.DisplayName = displayName

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	dto := userToDTO(user)
	return &dto, nil
}
