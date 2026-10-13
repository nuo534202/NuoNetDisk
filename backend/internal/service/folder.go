package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/model"
	"github.com/nuonuo/nuonetdisk/internal/repository"
	"github.com/nuonuo/nuonetdisk/internal/storage"
	"github.com/nuonuo/nuonetdisk/pkg/nullable"
)

type FolderService struct {
	folderRepo *repository.FolderRepository
	fileRepo   *repository.FileRepository
	storage    storage.FileStorage
}

func NewFolderService(
	folderRepo *repository.FolderRepository,
	fileRepo *repository.FileRepository,
	minioStorage storage.FileStorage,
) *FolderService {
	return &FolderService{
		folderRepo: folderRepo,
		fileRepo:   fileRepo,
		storage:    minioStorage,
	}
}

type FolderDTO struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	Name           string     `json:"name"`
	ParentFolderID *uuid.UUID `json:"parent_folder_id"`
	Version        int        `json:"version"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
	DeletedAt      *string    `json:"deleted_at,omitempty"`
	ExpiresAt      *string    `json:"expires_at,omitempty"`
}

func folderToDTO(f *model.Folder) FolderDTO {
	dto := FolderDTO{
		ID:             f.ID,
		UserID:         f.UserID,
		Name:           f.Name,
		ParentFolderID: f.ParentFolderID,
		Version:        f.Version,
		CreatedAt:      f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      f.UpdatedAt.Format(time.RFC3339),
	}
	if f.DeletedAt != nil {
		deletedAt := f.DeletedAt.Format(time.RFC3339)
		dto.DeletedAt = &deletedAt
		expiresAt := f.DeletedAt.Add(recycleBinRetentionDays * 24 * time.Hour).Format(time.RFC3339)
		dto.ExpiresAt = &expiresAt
	}
	return dto
}

func (s *FolderService) Create(ctx context.Context, userID uuid.UUID, name string, parentFolderID *uuid.UUID) (*FolderDTO, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, model.ErrInvalidInput
	}

	if parentFolderID != nil {
		parent, err := s.folderRepo.GetByID(ctx, *parentFolderID, userID)
		if err != nil {
			return nil, err
		}
		if parent.IsDeleted {
			return nil, model.ErrNotFound
		}

		isDescendant, err := s.folderRepo.IsDescendant(ctx, *parentFolderID, *parentFolderID, userID)
		if err != nil {
			return nil, err
		}
		if isDescendant {
			return nil, model.ErrInvalidInput
		}
	}

	folder := &model.Folder{
		UserID:         userID,
		Name:           name,
		ParentFolderID: parentFolderID,
	}

	if err := s.folderRepo.Insert(ctx, folder); err != nil {
		return nil, err
	}

	dto := folderToDTO(folder)
	return &dto, nil
}

type AncestorDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (s *FolderService) GetAncestors(ctx context.Context, folderID uuid.UUID, userID uuid.UUID) ([]AncestorDTO, error) {
	rows, err := s.folderRepo.GetAncestors(ctx, folderID, userID)
	if err != nil {
		return nil, err
	}
	dtos := make([]AncestorDTO, len(rows))
	for i, r := range rows {
		dtos[i] = AncestorDTO{ID: r.ID, Name: r.Name}
	}
	return dtos, nil
}

type ResolvedFolder struct {
	Folder    FolderDTO     `json:"folder"`
	Ancestors []AncestorDTO `json:"ancestors"`
}

func (s *FolderService) ResolveByPath(ctx context.Context, userID uuid.UUID, path string) (*ResolvedFolder, error) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 || (len(segments) == 1 && segments[0] == "") {
		return nil, model.ErrInvalidInput
	}

	var ancestors []AncestorDTO
	var currentParentID *uuid.UUID

	for i, seg := range segments {
		var folder *model.Folder
		var err error
		if currentParentID == nil {
			folder, err = s.folderRepo.GetByName(ctx, userID, seg)
		} else {
			folder, err = s.folderRepo.GetByNameAndParent(ctx, userID, seg, *currentParentID)
		}
		if err != nil {
			return nil, err
		}
		if folder.IsDeleted {
			return nil, model.ErrNotFound
		}
		if i < len(segments)-1 {
			ancestors = append(ancestors, AncestorDTO{ID: folder.ID, Name: folder.Name})
		}
		currentParentID = &folder.ID
	}

	lastFolder, err := s.folderRepo.GetByID(ctx, *currentParentID, userID)
	if err != nil {
		return nil, err
	}
	if lastFolder.IsDeleted {
		return nil, model.ErrNotFound
	}

	return &ResolvedFolder{
		Folder:    folderToDTO(lastFolder),
		Ancestors: ancestors,
	}, nil
}

func (s *FolderService) GetByName(ctx context.Context, userID uuid.UUID, name string) (*FolderDTO, error) {
	folder, err := s.folderRepo.GetByName(ctx, userID, name)
	if err != nil {
		return nil, err
	}
	if folder.IsDeleted {
		return nil, model.ErrNotFound
	}
	dto := folderToDTO(folder)
	return &dto, nil
}

func (s *FolderService) GetByID(ctx context.Context, folderID uuid.UUID, userID uuid.UUID) (*FolderDTO, error) {
	folder, err := s.folderRepo.GetByID(ctx, folderID, userID)
	if err != nil {
		return nil, err
	}
	if folder.IsDeleted {
		return nil, model.ErrNotFound
	}

	dto := folderToDTO(folder)
	return &dto, nil
}

func (s *FolderService) ListByParent(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID) ([]FolderDTO, error) {
	folders, err := s.folderRepo.ListByParent(ctx, userID, parentID)
	if err != nil {
		return nil, err
	}

	dtos := make([]FolderDTO, len(folders))
	for i, f := range folders {
		dtos[i] = folderToDTO(f)
	}

	return dtos, nil
}

func (s *FolderService) Update(ctx context.Context, folderID uuid.UUID, userID uuid.UUID, name *string, parentFolderID nullable.UUID) (*FolderDTO, error) {
	folder, err := s.folderRepo.GetByID(ctx, folderID, userID)
	if err != nil {
		return nil, err
	}
	if folder.IsDeleted {
		return nil, model.ErrNotFound
	}

	if parentFolderID.Valid {
		if parentFolderID.UUID != nil {
			isDescendant, err := s.folderRepo.IsDescendant(ctx, *parentFolderID.UUID, folderID, userID)
			if err != nil {
				return nil, err
			}
			if isDescendant {
				return nil, model.ErrInvalidInput
			}
		}
		folder.ParentFolderID = parentFolderID.UUID
	}

	if name != nil {
		folder.Name = *name
	}

	if err := s.folderRepo.Update(ctx, folder); err != nil {
		return nil, err
	}

	dto := folderToDTO(folder)
	return &dto, nil
}

func (s *FolderService) Delete(ctx context.Context, folderID uuid.UUID, userID uuid.UUID) error {
	folder, err := s.folderRepo.GetByID(ctx, folderID, userID)
	if err != nil {
		return err
	}
	if folder.IsDeleted {
		return model.ErrNotFound
	}

	return s.folderRepo.SoftDeleteCascade(ctx, folderID, userID)
}

func (s *FolderService) HardDeleteCascade(ctx context.Context, folderID uuid.UUID, userID uuid.UUID) error {
	folder, err := s.folderRepo.GetByID(ctx, folderID, userID)
	if err != nil {
		return err
	}

	folderDeleteErr := s.folderRepo.HardDelete(ctx, folder.ID)
	if folderDeleteErr != nil && !errors.Is(folderDeleteErr, model.ErrNotFound) {
		return folderDeleteErr
	}

	return nil
}

func (s *FolderService) IsDescendant(ctx context.Context, folderID, ancestorID uuid.UUID, userID uuid.UUID) (bool, error) {
	return s.folderRepo.IsDescendant(ctx, folderID, ancestorID, userID)
}
