package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nuonuo/nuonetdisk/internal/model"
)

type UserFilter struct {
	Q            string
	EmailFilter  string
	NameFilter   string
	RoleFilter   string // "admin" or "user"
	GenderFilter string
	DateFrom     string // RFC 3339 / ISO date
	DateTo       string // RFC 3339 / ISO date
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (email, password_hash, display_name, is_admin)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, query, user.Email, user.PasswordHash, user.DisplayName, user.IsAdmin).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (r *UserRepository) ListAll(ctx context.Context, offset, limit int, filter UserFilter) ([]*model.User, int, error) {
	conditions := []string{}
	args := []interface{}{}
	argIdx := 1

	if filter.Q != "" {
		like := "%" + filter.Q + "%"
		conditions = append(conditions, fmt.Sprintf("(email ILIKE $%d OR display_name ILIKE $%d)", argIdx, argIdx))
		args = append(args, like)
		argIdx++
	}
	if filter.EmailFilter != "" {
		conditions = append(conditions, fmt.Sprintf("email ILIKE $%d", argIdx))
		args = append(args, "%"+filter.EmailFilter+"%")
		argIdx++
	}
	if filter.NameFilter != "" {
		conditions = append(conditions, fmt.Sprintf("display_name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.NameFilter+"%")
		argIdx++
	}
	if filter.RoleFilter == "admin" {
		conditions = append(conditions, fmt.Sprintf("is_admin = $%d", argIdx))
		args = append(args, true)
		argIdx++
	} else if filter.RoleFilter == "user" {
		conditions = append(conditions, fmt.Sprintf("is_admin = $%d", argIdx))
		args = append(args, false)
		argIdx++
	}
	if filter.GenderFilter == "unspecified" {
		conditions = append(conditions, "(gender = '' OR gender IS NULL)")
	} else if filter.GenderFilter != "" {
		conditions = append(conditions, fmt.Sprintf("gender = $%d", argIdx))
		args = append(args, filter.GenderFilter)
		argIdx++
	}
	// DateFrom is treated as an inclusive lower bound on created_at.
	if filter.DateFrom != "" {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	// DateTo is an ISO date (YYYY-MM-DD); make it inclusive of the whole day.
	if filter.DateTo != "" {
		conditions = append(conditions, fmt.Sprintf("created_at < ($%d::date + interval '1 day')", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM users" + whereClause
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.User{}, 0, nil
	}

	query := `SELECT id, email, password_hash, display_name, avatar_url, bio, gender, is_admin, created_at, updated_at
		FROM users` + whereClause + ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL, &u.Bio, &u.Gender, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

func (r *UserRepository) SetAdmin(ctx context.Context, userID uuid.UUID, isAdmin bool) error {
	_, err := r.pool.Exec(ctx, "UPDATE users SET is_admin = $2, updated_at = NOW() WHERE id = $1", userID, isAdmin)
	return err
}

func (r *UserRepository) Ping(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, "SELECT 1")
	return err
}

func scanUser(row pgx.Row) (*model.User, error) {
	user := &model.User{}
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.AvatarURL, &user.Bio, &user.Gender, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

const userColumns = "id, email, password_hash, display_name, avatar_url, bio, gender, is_admin, created_at, updated_at"

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := "SELECT " + userColumns + " FROM users WHERE email = $1"
	return scanUser(r.pool.QueryRow(ctx, query, email))
}

func (r *UserRepository) GetByID(ctx context.Context, id interface{}) (*model.User, error) {
	query := "SELECT " + userColumns + " FROM users WHERE id = $1"
	return scanUser(r.pool.QueryRow(ctx, query, id))
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, hash string) error {
	_, err := r.pool.Exec(ctx, "UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1", userID, hash)
	return err
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users SET display_name = $2, avatar_url = $3, bio = $4, gender = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`
	return r.pool.QueryRow(ctx, query, user.ID, user.DisplayName, user.AvatarURL, user.Bio, user.Gender).
		Scan(&user.UpdatedAt)
}
