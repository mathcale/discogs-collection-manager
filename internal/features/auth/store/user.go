package store

import (
	"context"
	"database/sql"
	"time"

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

func (r *userRepository) Save(ctx context.Context, user *domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			email = excluded.email,
			password_hash = excluded.password_hash,
			created_at = excluded.created_at`,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt.UTC().Format(time.DateTime),
	)

	return err
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
