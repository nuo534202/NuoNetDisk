package service

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/repository"
	"github.com/nuonuo/nuonetdisk/internal/storage"
)

type RecycleBinService struct {
	fileRepo   *repository.FileRepository
	folderRepo *repository.FolderRepository
	storage    storage.FileStorage
}

func NewRecycleBinService(
	fileRepo *repository.FileRepository,
	folderRepo *repository.FolderRepository,
	minioStorage storage.FileStorage,
) *RecycleBinService {
	return &RecycleBinService{
		fileRepo:   fileRepo,
		folderRepo: folderRepo,
		storage:    minioStorage,
	}
}

func (s *RecycleBinService) ListDeletedFiles(ctx context.Context, userID uuid.UUID, offset, limit int) ([]FileDTO, int, error) {
	files, total, err := s.fileRepo.ListDeleted(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]FileDTO, len(files))
	for i, f := range files {
		dtos[i] = fileToDTO(f)
	}

	return dtos, total, nil
}

func (s *RecycleBinService) RestoreFile(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (*FileDTO, error) {
	file, err := s.fileRepo.GetDeletedByID(ctx, fileID, userID)
	if err != nil {
		return nil, err
	}

	if err := s.fileRepo.Restore(ctx, fileID, userID); err != nil {
		return nil, err
	}

	file.IsDeleted = false
	dto := fileToDTO(file)
	return &dto, nil
}

func (s *RecycleBinService) PermanentDeleteFile(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.fileRepo.GetDeletedByID(ctx, fileID, userID)
	if err != nil {
		return err
	}

	if err := s.storage.Delete(ctx, file.ObjectKey); err != nil {
		log.Printf("failed to delete MinIO object %s: %v", file.ObjectKey, err)
	}

	return s.fileRepo.HardDelete(ctx, fileID)
}

func (s *RecycleBinService) ListDeletedFolders(ctx context.Context, userID uuid.UUID, offset, limit int) ([]FolderDTO, int, error) {
	folders, total, err := s.folderRepo.ListDeleted(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]FolderDTO, len(folders))
	for i, f := range folders {
		dtos[i] = folderToDTO(f)
	}

	return dtos, total, nil
}

func (s *RecycleBinService) RestoreFolder(ctx context.Context, folderID uuid.UUID, userID uuid.UUID) (*FolderDTO, error) {
	folder, err := s.folderRepo.GetDeletedByID(ctx, folderID, userID)
	if err != nil {
		return nil, err
	}

	if err := s.folderRepo.Restore(ctx, folderID, userID); err != nil {
		return nil, err
	}

	folder.IsDeleted = false
	dto := folderToDTO(folder)
	return &dto, nil
}

func (s *RecycleBinService) PermanentDeleteFolder(ctx context.Context, folderID uuid.UUID, userID uuid.UUID) error {
	_, err := s.folderRepo.GetByID(ctx, folderID, userID)
	if err != nil {
		return err
	}

	objectKeys, err := s.fileRepo.GetObjectKeysByFolderID(ctx, folderID)
	if err != nil {
		return err
	}

	for _, key := range objectKeys {
		if err := s.storage.Delete(ctx, key); err != nil {
			log.Printf("failed to delete MinIO object %s: %v", key, err)
		}
	}

	return s.folderRepo.HardDelete(ctx, folderID)
}
