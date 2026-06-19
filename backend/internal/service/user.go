package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/auth"
	"github.com/nuonuo/nuonetdisk/internal/model"
	"github.com/nuonuo/nuonetdisk/internal/repository"
	"github.com/nuonuo/nuonetdisk/internal/storage"
)

const maxAvatarSizeBytes = 5 * 1024 * 1024

type UserService struct {
	userRepo *repository.UserRepository
	storage  storage.FileStorage
}

func NewUserService(userRepo *repository.UserRepository, storage storage.FileStorage) *UserService {
	return &UserService{userRepo: userRepo, storage: storage}
}

type UserDTO struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	Bio         string    `json:"bio"`
	Gender      string    `json:"gender"`
	UserHash    string    `json:"user_hash"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func userToDTO(u *model.User) UserDTO {
	dto := UserDTO{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Bio:         u.Bio,
		Gender:      u.Gender,
		UserHash:    auth.UserIDToHash(u.ID),
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   u.UpdatedAt.Format(time.RFC3339),
	}
	if u.AvatarURL != "" {
		dto.AvatarURL = "/api/v1/user/me/avatar"
	}
	return dto
}

func (s *UserService) GetMe(ctx context.Context, userID uuid.UUID) (*UserDTO, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	dto := userToDTO(user)
	return &dto, nil
}

type UpdateMeInput struct {
	DisplayName *string
	AvatarURL   *string
	Bio         *string
	Gender      *string
}

func (s *UserService) UpdateMe(ctx context.Context, userID uuid.UUID, input UpdateMeInput) (*UserDTO, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if input.DisplayName != nil {
		user.DisplayName = *input.DisplayName
	}
	if input.AvatarURL != nil {
		user.AvatarURL = *input.AvatarURL
	}
	if input.Bio != nil {
		user.Bio = *input.Bio
	}
	if input.Gender != nil {
		user.Gender = *input.Gender
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	dto := userToDTO(user)
	return &dto, nil
}

func (s *UserService) UpdateAvatar(ctx context.Context, userID uuid.UUID, fileHeader *multipart.FileHeader) (*UserDTO, error) {
	if fileHeader.Size > maxAvatarSizeBytes {
		return nil, model.ErrFileTooLarge
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open avatar file: %w", err)
	}
	defer file.Close()

	objectKey := fmt.Sprintf("avatars/%s/%s", userID.String(), uuid.New().String())
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := s.storage.Upload(ctx, objectKey, file, fileHeader.Size, contentType); err != nil {
		return nil, fmt.Errorf("failed to upload avatar to storage: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.storage.Delete(ctx, objectKey)
		return nil, err
	}

	oldKey := user.AvatarURL
	user.AvatarURL = objectKey

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.storage.Delete(ctx, objectKey)
		return nil, err
	}

	if oldKey != "" {
		s.storage.Delete(ctx, oldKey)
	}

	dto := userToDTO(user)
	return &dto, nil
}

func (s *UserService) GetAvatar(ctx context.Context, userID uuid.UUID) (io.ReadCloser, string, int64, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, "", 0, err
	}

	if user.AvatarURL == "" {
		return nil, "", 0, model.ErrNotFound
	}

	reader, err := s.storage.Download(ctx, user.AvatarURL)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to download avatar: %w", err)
	}

	size, contentType, err := s.storage.Stat(ctx, user.AvatarURL)
	if err != nil {
		reader.Close()
		return nil, "", 0, fmt.Errorf("failed to stat avatar: %w", err)
	}

	return reader, contentType, size, nil
}
