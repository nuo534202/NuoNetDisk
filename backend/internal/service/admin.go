package service

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nuonuo/nuonetdisk/internal/auth"
	"github.com/nuonuo/nuonetdisk/internal/model"
	"github.com/nuonuo/nuonetdisk/internal/repository"
	"github.com/nuonuo/nuonetdisk/internal/storage"
)

var startTime = time.Now()

type AdminService struct {
	userRepo   *repository.UserRepository
	fileRepo   *repository.FileRepository
	folderRepo *repository.FolderRepository
	shareRepo  *repository.ShareRepository
	storage    storage.FileStorage
	dbPool     *pgxpool.Pool
	adminEmail string
}

func NewAdminService(
	userRepo *repository.UserRepository,
	fileRepo *repository.FileRepository,
	folderRepo *repository.FolderRepository,
	shareRepo *repository.ShareRepository,
	fs storage.FileStorage,
	dbPool *pgxpool.Pool,
	adminEmail string,
) *AdminService {
	return &AdminService{
		userRepo:   userRepo,
		fileRepo:   fileRepo,
		folderRepo: folderRepo,
		shareRepo:  shareRepo,
		storage:    fs,
		dbPool:     dbPool,
		adminEmail: adminEmail,
	}
}

type ProjectInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	GoVersion string `json:"go_version"`
}

type ServiceHealth struct {
	Database bool `json:"database"`
	MinIO    bool `json:"minio"`
	Backend  bool `json:"backend"`
}

type SystemResources struct {
	NumGoroutine     int     `json:"num_goroutine"`
	AllocatedMB      float64 `json:"allocated_mb"`
	TotalAllocatedMB float64 `json:"total_allocated_mb"`
	SysMB            float64 `json:"sys_mb"`
	NumGC            uint32  `json:"num_gc"`
}

type StorageStats struct {
	TotalUsers        int   `json:"total_users"`
	TotalFiles        int   `json:"total_files"`
	TotalFolders      int   `json:"total_folders"`
	TotalStorageBytes int64 `json:"total_storage_bytes"`
	ActiveShares      int   `json:"active_shares"`
	RecycleBinCount   int   `json:"recycle_bin_count"`
}

type Dashboard struct {
	Project  ProjectInfo     `json:"project_info"`
	Services ServiceHealth   `json:"services"`
	System   SystemResources `json:"system"`
	Stats    StorageStats    `json:"stats"`
}

func (s *AdminService) GetDashboard(ctx context.Context) (*Dashboard, error) {
	// Project info
	version := "dev"
	if v := os.Getenv("APP_VERSION"); v != "" {
		version = v
	}
	uptime := time.Since(startTime).Round(time.Second).String()

	// Service health checks
	dbOK := s.userRepo.Ping(ctx) == nil
	minioOK := s.storage.HealthCheck(ctx) == nil

	// System resources
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Storage stats
	totalUsers, _ := s.userRepo.Count(ctx)
	totalFiles, _ := s.fileRepo.Count(ctx)
	totalFolders, _ := s.folderRepo.Count(ctx)
	totalBytes, _ := s.fileRepo.TotalSize(ctx)
	activeShares, _ := s.shareRepo.CountActive(ctx)
	recycleCount, _ := s.fileRepo.CountDeleted(ctx)

	return &Dashboard{
		Project: ProjectInfo{
			Name:      "NuoNetDisk",
			Version:   version,
			Uptime:    uptime,
			GoVersion: runtime.Version(),
		},
		Services: ServiceHealth{
			Database: dbOK,
			MinIO:    minioOK,
			Backend:  true,
		},
		System: SystemResources{
			NumGoroutine:     runtime.NumGoroutine(),
			AllocatedMB:      toMB(m.Alloc),
			TotalAllocatedMB: toMB(m.TotalAlloc),
			SysMB:            toMB(m.Sys),
			NumGC:            m.NumGC,
		},
		Stats: StorageStats{
			TotalUsers:        totalUsers,
			TotalFiles:        totalFiles,
			TotalFolders:      totalFolders,
			TotalStorageBytes: totalBytes,
			ActiveShares:      activeShares,
			RecycleBinCount:   recycleCount,
		},
	}, nil
}

type AdminUserDTO struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	DisplayName  string `json:"display_name"`
	AvatarURL    string `json:"avatar_url"`
	Bio          string `json:"bio"`
	Gender       string `json:"gender"`
	IsAdmin      bool   `json:"is_admin"`
	IsSuperAdmin bool   `json:"is_super_admin"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func (s *AdminService) ListUsers(ctx context.Context, offset, limit int, filter repository.UserFilter) ([]*AdminUserDTO, int, error) {
	users, total, err := s.userRepo.ListAll(ctx, offset, limit, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	dtos := make([]*AdminUserDTO, 0, len(users))
	for _, u := range users {
		isSuperAdmin := s.adminEmail != "" && strings.EqualFold(u.Email, s.adminEmail)
		avatarURL := ""
		if u.AvatarURL != "" {
			avatarURL = fmt.Sprintf("/api/v1/admin/users/%s/avatar", u.ID)
		}
		dtos = append(dtos, &AdminUserDTO{
			ID:           u.ID.String(),
			Email:        u.Email,
			DisplayName:  u.DisplayName,
			AvatarURL:    avatarURL,
			Bio:          u.Bio,
			Gender:       u.Gender,
			IsAdmin:      u.IsAdmin,
			IsSuperAdmin: isSuperAdmin,
			CreatedAt:    u.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    u.UpdatedAt.Format(time.RFC3339),
		})
	}
	return dtos, total, nil
}

func (s *AdminService) RegisterAdmin(ctx context.Context, email, password string) error {
	if email == "" || len(password) < 8 {
		return model.ErrInvalidInput
	}

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && err != model.ErrNotFound {
		return fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return model.ErrDuplicate
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  strings.Split(email, "@")[0],
		IsAdmin:      true,
	}
	return s.userRepo.Create(ctx, user)
}

func toMB(bytes uint64) float64 {
	return math.Round(float64(bytes)/1024/1024*100) / 100
}

// GetUserAvatar streams a specific user's avatar from storage. Returns
// model.ErrNotFound if the user does not exist or has no avatar set.
func (s *AdminService) GetUserAvatar(ctx context.Context, userID uuid.UUID) (io.ReadCloser, string, int64, error) {
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
