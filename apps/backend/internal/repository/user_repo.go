package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, display_name, bio, age, gender, country)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`, user.Username, user.Email, user.PasswordHash, user.DisplayName, user.Bio, user.Age, user.Gender, user.Country).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return mapUserWriteError(err)
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.getUser(ctx, `SELECT id, username, email, password_hash, display_name, bio, age, gender, country, created_at FROM users WHERE email = $1`, email)
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return r.getUser(ctx, `SELECT id, username, email, password_hash, display_name, bio, age, gender, country, created_at FROM users WHERE username = $1`, username)
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return r.getUser(ctx, `SELECT id, username, email, password_hash, display_name, bio, age, gender, country, created_at FROM users WHERE id = $1`, id)
}

func (r *UserRepository) CreateSession(ctx context.Context, session *model.Session) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`, session.UserID, session.TokenHash, session.ExpiresAt).Scan(&session.ID)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *UserRepository) GetUserByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*model.User, error) {
	user, err := r.getUser(ctx, `
		SELECT u.id, u.username, u.email, u.password_hash, u.display_name, u.bio, u.age, u.gender, u.country, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > $2
	`, tokenHash, now)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *UserRepository) getUser(ctx context.Context, query string, args ...any) (*model.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func scanUser(row rowScanner) (*model.User, error) {
	var user model.User
	if err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Bio,
		&user.Age,
		&user.Gender,
		&user.Country,
		&user.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &user, nil
}

func mapUserWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if strings.Contains(pgErr.ConstraintName, "username") {
			return model.Conflict("username is already taken")
		}
		if strings.Contains(pgErr.ConstraintName, "email") {
			return model.Conflict("email is already in use")
		}
		return model.Conflict("account already exists")
	}
	return fmt.Errorf("insert user: %w", err)
}
