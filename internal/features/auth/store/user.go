package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Create(
	ctx context.Context,
	email, passwordHash string,
) (*domain.User, error) {
	id := uuid.NewString()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		id, email, passwordHash, now.Format(time.DateTime),
	)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
	}, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	var createdAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &u, nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var u domain.User
	var createdAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &u, nil
}

func (r *userRepository) UpdatePassword(
	ctx context.Context,
	id, passwordHash string,
) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ? WHERE id = ?`,
		passwordHash, id,
	)

	return err
}
