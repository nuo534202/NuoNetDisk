package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nuonuo/nuonetdisk/internal/model"
)

type ShareRepository struct {
	pool *pgxpool.Pool
}

func NewShareRepository(pool *pgxpool.Pool) *ShareRepository {
	return &ShareRepository{pool: pool}
}

func (r *ShareRepository) Insert(ctx context.Context, link *model.ShareLink) error {
	query := `
		INSERT INTO share_links (user_id, resource_type, resource_id, token, permission, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query, link.UserID, link.ResourceType, link.ResourceID,
		link.Token, link.Permission, link.ExpiresAt).
		Scan(&link.ID, &link.CreatedAt)
}

func (r *ShareRepository) GetByToken(ctx context.Context, token string) (*model.ShareLink, error) {
	query := `
		SELECT id, user_id, resource_type, resource_id, token, permission, expires_at, is_revoked, created_at
		FROM share_links WHERE token = $1`
	link := &model.ShareLink{}
	err := r.pool.QueryRow(ctx, query, token).
		Scan(&link.ID, &link.UserID, &link.ResourceType, &link.ResourceID,
			&link.Token, &link.Permission, &link.ExpiresAt, &link.IsRevoked, &link.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return link, nil
}

func (r *ShareRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.ShareLink, error) {
	query := `
		SELECT id, user_id, resource_type, resource_id, token, permission, expires_at, is_revoked, created_at
		FROM share_links WHERE id = $1 AND user_id = $2`
	link := &model.ShareLink{}
	err := r.pool.QueryRow(ctx, query, id, userID).
		Scan(&link.ID, &link.UserID, &link.ResourceType, &link.ResourceID,
			&link.Token, &link.Permission, &link.ExpiresAt, &link.IsRevoked, &link.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return link, nil
}

func (r *ShareRepository) Revoke(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `UPDATE share_links SET is_revoked = TRUE WHERE id = $1 AND user_id = $2 AND is_revoked = FALSE`
	tag, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
