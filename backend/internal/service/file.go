package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/model"
	"github.com/nuonuo/nuonetdisk/internal/repository"
	"github.com/nuonuo/nuonetdisk/internal/storage"
)

type FileService struct {
	fileRepo   *repository.FileRepository
	folderRepo *repository.FolderRepository
	storage    storage.FileStorage
	maxSize    int64
}

func NewFileService(
	fileRepo *repository.FileRepository,
	folderRepo *repository.FolderRepository,
	minioStorage storage.FileStorage,
	maxUploadSize int64,
) *FileService {
	return &FileService{
		fileRepo:   fileRepo,
		folderRepo: folderRepo,
		storage:    minioStorage,
		maxSize:    maxUploadSize,
	}
}

type FileDTO struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	Name           string     `json:"name"`
	Size           int64      `json:"size"`
	MimeType       string     `json:"mime_type"`
	SHA256Hash     string     `json:"sha256_hash"`
	ParentFolderID *uuid.UUID `json:"parent_folder_id"`
	Version        int        `json:"version"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
	DeletedAt      *string    `json:"deleted_at,omitempty"`
	ExpiresAt      *string    `json:"expires_at,omitempty"`
}

const recycleBinRetentionDays = 30

func fileToDTO(f *model.File) FileDTO {
	dto := FileDTO{
		ID:             f.ID,
		UserID:         f.UserID,
		Name:           f.Name,
		Size:           f.Size,
		MimeType:       f.MimeType,
		SHA256Hash:     f.SHA256Hash,
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

func (s *FileService) Upload(ctx context.Context, userID uuid.UUID, fileHeader *multipart.FileHeader, parentFolderID *uuid.UUID) (*FileDTO, error) {
	if fileHeader.Size > s.maxSize {
		return nil, model.ErrFileTooLarge
	}

	if parentFolderID != nil {
		folder, err := s.folderRepo.GetByID(ctx, *parentFolderID, userID)
		if err != nil {
			return nil, model.ErrForbidden
		}
		if folder.IsDeleted {
			return nil, model.ErrNotFound
		}

	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	objectKey := fmt.Sprintf("objects/%s", uuid.New().String())
	hasher := sha256.New()
	teeReader := io.TeeReader(file, hasher)

	mimeType := detectMimeType(fileHeader.Filename, fileHeader.Header.Get("Content-Type"))

	if err := s.storage.Upload(ctx, objectKey, teeReader, fileHeader.Size, mimeType); err != nil {
		return nil, fmt.Errorf("failed to upload to storage: %w", err)
	}

	sha256Hash := hex.EncodeToString(hasher.Sum(nil))

	fileModel := &model.File{
		UserID:         userID,
		Name:           fileHeader.Filename,
		ObjectKey:      objectKey,
		Size:           fileHeader.Size,
		MimeType:       mimeType,
		SHA256Hash:     sha256Hash,
		ParentFolderID: parentFolderID,
	}

	if err := s.fileRepo.Insert(ctx, fileModel); err != nil {
		if cleanupErr := s.storage.Delete(ctx, objectKey); cleanupErr != nil {
			log.Printf("failed to clean up MinIO object %s after DB insert failure: %v", objectKey, cleanupErr)
		}
		return nil, err
	}

	dto := fileToDTO(fileModel)
	return &dto, nil
}

func (s *FileService) GetFile(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (*FileDTO, error) {
	file, err := s.fileRepo.GetByID(ctx, fileID, userID)
	if err != nil {
		return nil, err
	}
	if file.IsDeleted {
		return nil, model.ErrNotFound
	}

	dto := fileToDTO(file)
	return &dto, nil
}

func (s *FileService) Download(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (io.ReadCloser, string, int64, error) {
	file, err := s.fileRepo.GetByID(ctx, fileID, userID)
	if err != nil {
		return nil, "", 0, err
	}
	if file.IsDeleted {
		return nil, "", 0, model.ErrNotFound
	}

	reader, err := s.storage.Download(ctx, file.ObjectKey)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to download from storage: %w", err)
	}

	return reader, file.Name, file.Size, nil
}

func (s *FileService) Delete(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.fileRepo.GetByID(ctx, fileID, userID)
	if err != nil {
		return err
	}
	if file.IsDeleted {
		return model.ErrNotFound
	}

	return s.fileRepo.SoftDelete(ctx, fileID, userID)
}

func (s *FileService) Update(ctx context.Context, fileID uuid.UUID, userID uuid.UUID, name *string, parentFolderID *uuid.UUID) (*FileDTO, error) {
	file, err := s.fileRepo.GetByID(ctx, fileID, userID)
	if err != nil {
		return nil, err
	}
	if file.IsDeleted {
		return nil, model.ErrNotFound
	}

	if name != nil {
		file.Name = *name
	}
	if parentFolderID != nil {
		file.ParentFolderID = parentFolderID
	}

	if err := s.fileRepo.Update(ctx, file); err != nil {
		return nil, err
	}

	dto := fileToDTO(file)
	return &dto, nil
}

func (s *FileService) ListFiles(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID, offset, limit int, sort, order string) ([]FileDTO, int, error) {
	files, total, err := s.fileRepo.ListByFolder(ctx, userID, folderID, offset, limit, sort, order)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]FileDTO, len(files))
	for i, f := range files {
		dtos[i] = fileToDTO(f)
	}

	return dtos, total, nil
}

func detectMimeType(filename, declared string) string {
	if declared != "" && declared != "application/octet-stream" {
		return declared
	}

	ext := strings.ToLower(filename[strings.LastIndex(filename, ".")+1:])
	mimeMap := map[string]string{
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"webp": "image/webp",
		"svg":  "image/svg+xml",
		"pdf":  "application/pdf",
		"doc":  "application/msword",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"xls":  "application/vnd.ms-excel",
		"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"zip":  "application/zip",
		"gz":   "application/gzip",
		"tar":  "application/x-tar",
		"mp4":  "video/mp4",
		"mp3":  "audio/mpeg",
		"txt":  "text/plain",
		"html": "text/html",
		"css":  "text/css",
		"js":   "application/javascript",
		"json": "application/json",
		"xml":  "application/xml",
		"csv":  "text/csv",
	}

	if mime, ok := mimeMap[ext]; ok {
		return mime
	}

	return "application/octet-stream"
}
