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

type FolderRepository struct {
	pool *pgxpool.Pool
}

func NewFolderRepository(pool *pgxpool.Pool) *FolderRepository {
	return &FolderRepository{pool: pool}
}

func (r *FolderRepository) Insert(ctx context.Context, folder *model.Folder) error {
	query := `
		INSERT INTO folders (user_id, name, parent_folder_id)
		VALUES ($1, $2, $3)
		RETURNING id, version, created_at, updated_at`
	return r.pool.QueryRow(ctx, query, folder.UserID, folder.Name, folder.ParentFolderID).
		Scan(&folder.ID, &folder.Version, &folder.CreatedAt, &folder.UpdatedAt)
}

func (r *FolderRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.Folder, error) {
	query := `
		SELECT id, user_id, name, parent_folder_id, is_deleted, deleted_at, version, created_at, updated_at
		FROM folders WHERE id = $1 AND user_id = $2`
	folder := &model.Folder{}
	err := r.pool.QueryRow(ctx, query, id, userID).
		Scan(&folder.ID, &folder.UserID, &folder.Name, &folder.ParentFolderID,
			&folder.IsDeleted, &folder.DeletedAt, &folder.Version, &folder.CreatedAt, &folder.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return folder, nil
}

func (r *FolderRepository) ListByParent(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID) ([]*model.Folder, error) {
	query := `
		SELECT id, user_id, name, parent_folder_id, is_deleted, deleted_at, version, created_at, updated_at
		FROM folders
		WHERE user_id = $1 AND parent_folder_id IS NOT DISTINCT FROM $2 AND is_deleted = FALSE
		ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query, userID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*model.Folder
	for rows.Next() {
		f := &model.Folder{}
		err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.ParentFolderID,
			&f.IsDeleted, &f.DeletedAt, &f.Version, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func (r *FolderRepository) GetDeletedByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.Folder, error) {
	query := `
		SELECT id, user_id, name, parent_folder_id, is_deleted, deleted_at, version, created_at, updated_at
		FROM folders WHERE id = $1 AND user_id = $2 AND is_deleted = TRUE`
	folder := &model.Folder{}
	err := r.pool.QueryRow(ctx, query, id, userID).
		Scan(&folder.ID, &folder.UserID, &folder.Name, &folder.ParentFolderID,
			&folder.IsDeleted, &folder.DeletedAt, &folder.Version, &folder.CreatedAt, &folder.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return folder, nil
}

func (r *FolderRepository) ListDeleted(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*model.Folder, int, error) {
	countQuery := `SELECT COUNT(*) FROM folders WHERE user_id = $1 AND is_deleted = TRUE`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, name, parent_folder_id, is_deleted, deleted_at, version, created_at, updated_at
		FROM folders WHERE user_id = $1 AND is_deleted = TRUE
		ORDER BY deleted_at DESC
		OFFSET $2 LIMIT $3`

	rows, err := r.pool.Query(ctx, query, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var folders []*model.Folder
	for rows.Next() {
		f := &model.Folder{}
		err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.ParentFolderID,
			&f.IsDeleted, &f.DeletedAt, &f.Version, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		folders = append(folders, f)
	}
	return folders, total, rows.Err()
}

func (r *FolderRepository) Update(ctx context.Context, folder *model.Folder) error {
	query := `
		UPDATE folders SET name = $1, parent_folder_id = $2, version = version + 1, updated_at = NOW()
		WHERE id = $3 AND user_id = $4 AND version = $5 AND is_deleted = FALSE
		RETURNING version, updated_at`
	tag, err := r.pool.Exec(ctx, query, folder.Name, folder.ParentFolderID, folder.ID, folder.UserID, folder.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrConflict
	}
	folder.Version++
	folder.UpdatedAt = time.Now()
	return nil
}

func (r *FolderRepository) SoftDelete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE folders SET is_deleted = TRUE, deleted_at = NOW(), version = version + 1, updated_at = NOW()
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

func (r *FolderRepository) SoftDeleteCascade(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	now := time.Now()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		WITH RECURSIVE subfolders AS (
			SELECT id FROM folders WHERE id = $1 AND user_id = $2
			UNION ALL
			SELECT f.id FROM folders f JOIN subfolders s ON f.parent_folder_id = s.id
		)
		UPDATE folders SET is_deleted = TRUE, deleted_at = $3, version = version + 1, updated_at = $3
		WHERE id IN (SELECT id FROM subfolders)`, id, userID, now)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		WITH RECURSIVE subfolders AS (
			SELECT id FROM folders WHERE id = $1 AND user_id = $2
			UNION ALL
			SELECT f.id FROM folders f JOIN subfolders s ON f.parent_folder_id = s.id
		)
		UPDATE files SET is_deleted = TRUE, deleted_at = $3, version = version + 1, updated_at = $3
		WHERE parent_folder_id IN (SELECT id FROM subfolders)`, id, userID, now)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *FolderRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM folders WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *FolderRepository) Restore(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE folders SET is_deleted = FALSE, deleted_at = NULL, version = version + 1, updated_at = NOW()
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

type ancestorRow struct {
	ID             uuid.UUID
	Name           string
	ParentFolderID *uuid.UUID
}

func (r *FolderRepository) GetAncestors(ctx context.Context, folderID uuid.UUID, userID uuid.UUID) ([]ancestorRow, error) {
	query := `
		WITH RECURSIVE ancestors AS (
			SELECT id, name, parent_folder_id, 0 AS depth
			FROM folders WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
			UNION ALL
			SELECT f.id, f.name, f.parent_folder_id, a.depth + 1
			FROM folders f
			JOIN ancestors a ON f.id = a.parent_folder_id
			WHERE f.is_deleted = FALSE
		)
		SELECT id, name, parent_folder_id FROM ancestors ORDER BY depth DESC`
	rows, err := r.pool.Query(ctx, query, folderID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ancestorRow
	for rows.Next() {
		var row ancestorRow
		if err := rows.Scan(&row.ID, &row.Name, &row.ParentFolderID); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *FolderRepository) IsDescendant(ctx context.Context, folderID, ancestorID uuid.UUID, userID uuid.UUID) (bool, error) {
	query := `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_folder_id FROM folders WHERE id = $1 AND user_id = $3
			UNION ALL
			SELECT f.id, f.parent_folder_id FROM folders f
			JOIN ancestors a ON f.id = a.parent_folder_id
		)
		SELECT EXISTS(SELECT 1 FROM ancestors WHERE parent_folder_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, ancestorID, folderID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check descendant: %w", err)
	}
	return exists, nil
}
