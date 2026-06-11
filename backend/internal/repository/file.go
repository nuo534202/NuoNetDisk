package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nuonuo/nuonetdisk/internal/model"
)

type FileRepository struct {
	pool *pgxpool.Pool
}

func NewFileRepository(pool *pgxpool.Pool) *FileRepository {
	return &FileRepository{pool: pool}
}

func (r *FileRepository) Insert(ctx context.Context, file *model.File) error {
	query := `
		INSERT INTO files (user_id, name, object_key, size, mime_type, sha256_hash, parent_folder_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, version, created_at, updated_at`
	return r.pool.QueryRow(ctx, query,
		file.UserID, file.Name, file.ObjectKey, file.Size, file.MimeType, file.SHA256Hash, file.ParentFolderID).
		Scan(&file.ID, &file.Version, &file.CreatedAt, &file.UpdatedAt)
}

func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.File, error) {
	query := `
		SELECT id, user_id, name, object_key, size, mime_type, sha256_hash,
		       parent_folder_id, is_deleted, deleted_at, version, created_at, updated_at
		FROM files WHERE id = $1 AND user_id = $2`
	file := &model.File{}
	err := r.pool.QueryRow(ctx, query, id, userID).
		Scan(&file.ID, &file.UserID, &file.Name, &file.ObjectKey, &file.Size, &file.MimeType,
			&file.SHA256Hash, &file.ParentFolderID, &file.IsDeleted, &file.DeletedAt,
			&file.Version, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return file, nil
}

func (r *FileRepository) ListByFolder(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID, offset, limit int, sort, order string) ([]*model.File, int, error) {
	if order != "asc" && order != "desc" {
		order = "asc"
	}
	validSorts := map[string]bool{"name": true, "size": true, "created_at": true, "updated_at": true}
	if !validSorts[sort] {
		sort = "name"
	}

	countQuery := `
		SELECT COUNT(*) FROM files
		WHERE user_id = $1 AND parent_folder_id IS NOT DISTINCT FROM $2 AND is_deleted = FALSE`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, userID, folderID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, name, object_key, size, mime_type, sha256_hash,
		       parent_folder_id, is_deleted, deleted_at, version, created_at, updated_at
		FROM files
		WHERE user_id = $1 AND parent_folder_id IS NOT DISTINCT FROM $2 AND is_deleted = FALSE
		ORDER BY %s %s
		OFFSET $3 LIMIT $4`, sort, order)

	rows, err := r.pool.Query(ctx, query, userID, folderID, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		f := &model.File{}
		err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.ObjectKey, &f.Size, &f.MimeType,
			&f.SHA256Hash, &f.ParentFolderID, &f.IsDeleted, &f.DeletedAt,
			&f.Version, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		files = append(files, f)
	}
	return files, total, rows.Err()
}

func (r *FileRepository) Update(ctx context.Context, file *model.File) error {
	query := `
		UPDATE files SET name = $1, parent_folder_id = $2, version = version + 1, updated_at = NOW()
		WHERE id = $3 AND user_id = $4 AND version = $5 AND is_deleted = FALSE
		RETURNING version, updated_at`
	tag, err := r.pool.Exec(ctx, query, file.Name, file.ParentFolderID, file.ID, file.UserID, file.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrConflict
	}
	file.Version++
	file.UpdatedAt = time.Now()
	return nil
}

func (r *FileRepository) SoftDelete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE files SET is_deleted = TRUE, deleted_at = NOW(), version = version + 1, updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE`
	tag, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *FileRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM files WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *FileRepository) GetDeletedByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.File, error) {
	query := `
		SELECT id, user_id, name, object_key, size, mime_type, sha256_hash,
		       parent_folder_id, is_deleted, deleted_at, version, created_at, updated_at
		FROM files WHERE id = $1 AND user_id = $2 AND is_deleted = TRUE`
	file := &model.File{}
	err := r.pool.QueryRow(ctx, query, id, userID).
		Scan(&file.ID, &file.UserID, &file.Name, &file.ObjectKey, &file.Size, &file.MimeType,
			&file.SHA256Hash, &file.ParentFolderID, &file.IsDeleted, &file.DeletedAt,
			&file.Version, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return file, nil
}

func (r *FileRepository) ListDeleted(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*model.File, int, error) {
	countQuery := `SELECT COUNT(*) FROM files WHERE user_id = $1 AND is_deleted = TRUE`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, name, object_key, size, mime_type, sha256_hash,
		       parent_folder_id, is_deleted, deleted_at, version, created_at, updated_at
		FROM files
		WHERE user_id = $1 AND is_deleted = TRUE
		ORDER BY deleted_at DESC
		OFFSET $2 LIMIT $3`

	rows, err := r.pool.Query(ctx, query, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		f := &model.File{}
		err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.ObjectKey, &f.Size, &f.MimeType,
			&f.SHA256Hash, &f.ParentFolderID, &f.IsDeleted, &f.DeletedAt,
			&f.Version, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		files = append(files, f)
	}
	return files, total, rows.Err()
}

func (r *FileRepository) GetObjectKeysByFolderID(ctx context.Context, folderID uuid.UUID) ([]string, error) {
	query := `SELECT object_key FROM files WHERE parent_folder_id = $1 AND is_deleted = FALSE`
	rows, err := r.pool.Query(ctx, query, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (r *FileRepository) Restore(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE files SET is_deleted = FALSE, deleted_at = NULL, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND is_deleted = TRUE`
	tag, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
