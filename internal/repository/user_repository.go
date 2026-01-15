package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/aminshahid573/taskmanager/internal/domain"
	"github.com/google/uuid"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Name,
		user.CreatedAt, user.UpdatedAt,
	)

	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, email_verified, email_verified_at, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	var user domain.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.EmailVerified, &user.EmailVerifiedAt,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewAppError(domain.ErrCodeUserNotFound, "User not found", 404)
		}
		return nil, domain.ErrDatabaseError.WithError(err)
	}

	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, email_verified, email_verified_at, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	var user domain.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.EmailVerified, &user.EmailVerifiedAt,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewAppError(domain.ErrCodeUserNotFound, "User not found", 404)
		}
		return nil, domain.ErrDatabaseError.WithError(err)
	}

	return &user, nil
}

func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, domain.ErrDatabaseError.WithError(err)
	}

	return exists, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET name = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	user.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query, user.Name, user.UpdatedAt, user.ID)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	if rows == 0 {
		return domain.NewAppError(domain.ErrCodeUserNotFound, "User not found", 404)
	}

	return nil
}

func (r *UserRepository) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET email_verified = true, email_verified_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, now, userID)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	if rows == 0 {
		return domain.NewAppError(domain.ErrCodeUserNotFound, "User not found", 404)
	}

	return nil
}
