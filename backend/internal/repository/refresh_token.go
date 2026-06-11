package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nuonuo/nuonetdisk/internal/model"
)

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) Insert(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query, token.UserID, token.TokenHash, token.ExpiresAt).
		Scan(&token.ID, &token.CreatedAt)
}

func (r *RefreshTokenRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, is_revoked, created_at
		FROM refresh_tokens WHERE id = $1`
	token := &model.RefreshToken{}
	err := r.pool.QueryRow(ctx, query, id).
		Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.IsRevoked, &token.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return token, nil
}

func (r *RefreshTokenRepository) MarkRevoked(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE refresh_tokens SET is_revoked = TRUE WHERE id = $1 AND is_revoked = FALSE`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET is_revoked = TRUE WHERE user_id = $1 AND is_revoked = FALSE`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}
