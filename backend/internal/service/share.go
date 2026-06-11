package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/model"
	"github.com/nuonuo/nuonetdisk/internal/repository"
)

type ShareService struct {
	shareRepo  *repository.ShareRepository
	fileRepo   *repository.FileRepository
	folderRepo *repository.FolderRepository
	maxTTL     time.Duration
}

func NewShareService(
	shareRepo *repository.ShareRepository,
	fileRepo *repository.FileRepository,
	folderRepo *repository.FolderRepository,
	maxTTL time.Duration,
) *ShareService {
	return &ShareService{
		shareRepo:  shareRepo,
		fileRepo:   fileRepo,
		folderRepo: folderRepo,
		maxTTL:     maxTTL,
	}
}

type ShareLinkDTO struct {
	ID           uuid.UUID             `json:"id"`
	UserID       uuid.UUID             `json:"user_id"`
	ResourceType model.ResourceType    `json:"resource_type"`
	ResourceID   uuid.UUID             `json:"resource_id"`
	Token        string                `json:"token"`
	Permission   model.SharePermission `json:"permission"`
	ExpiresAt    *time.Time            `json:"expires_at,omitempty"`
	IsRevoked    bool                  `json:"is_revoked"`
	CreatedAt    string                `json:"created_at"`
}

func shareToDTO(l *model.ShareLink) ShareLinkDTO {
	return ShareLinkDTO{
		ID:           l.ID,
		UserID:       l.UserID,
		ResourceType: l.ResourceType,
		ResourceID:   l.ResourceID,
		Token:        l.Token,
		Permission:   l.Permission,
		ExpiresAt:    l.ExpiresAt,
		IsRevoked:    l.IsRevoked,
		CreatedAt:    l.CreatedAt.Format(time.RFC3339),
	}
}

type CreateShareInput struct {
	UserID       uuid.UUID
	ResourceType model.ResourceType
	ResourceID   uuid.UUID
	Permission   model.SharePermission
	ExpiresAt    *time.Time
}

func (s *ShareService) CreateShareLink(ctx context.Context, input CreateShareInput) (*ShareLinkDTO, error) {
	if input.ResourceType != model.ResourceTypeFile && input.ResourceType != model.ResourceTypeFolder {
		return nil, model.ErrInvalidInput
	}
	if input.Permission != model.SharePermissionRead && input.Permission != model.SharePermissionWrite {
		return nil, model.ErrInvalidInput
	}

	if input.ResourceType == model.ResourceTypeFile {
		file, err := s.fileRepo.GetByID(ctx, input.ResourceID, input.UserID)
		if err != nil {
			return nil, err
		}
		if file.IsDeleted {
			return nil, model.ErrNotFound
		}

	} else {
		folder, err := s.folderRepo.GetByID(ctx, input.ResourceID, input.UserID)
		if err != nil {
			return nil, err
		}
		if folder.IsDeleted {
			return nil, model.ErrNotFound
		}

	}

	if input.ExpiresAt != nil {
		if input.ExpiresAt.Before(time.Now()) {
			return nil, model.ErrInvalidInput
		}
		maxExpiry := time.Now().Add(s.maxTTL)
		if input.ExpiresAt.After(maxExpiry) {
			input.ExpiresAt = &maxExpiry
		}
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes)

	link := &model.ShareLink{
		UserID:       input.UserID,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		Token:        token,
		Permission:   input.Permission,
		ExpiresAt:    input.ExpiresAt,
	}

	if err := s.shareRepo.Insert(ctx, link); err != nil {
		return nil, err
	}

	dto := shareToDTO(link)
	return &dto, nil
}

func (s *ShareService) GetByToken(ctx context.Context, token string) (*model.ShareLink, error) {
	link, err := s.shareRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if link.IsRevoked {
		return nil, model.ErrNotFound
	}

	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		return nil, model.ErrNotFound
	}

	return link, nil
}

func (s *ShareService) Revoke(ctx context.Context, shareID uuid.UUID, userID uuid.UUID) error {
	return s.shareRepo.Revoke(ctx, shareID, userID)
}
