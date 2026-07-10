package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string) error
}

type SQLiteUserRepository struct {
	db *sql.DB
}

func NewSQLiteUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{
		db: db,
	}
}

func (r *SQLiteUserRepository) Create(
	ctx context.Context,
	email, passwordHash string,
) (*User, error) {
	id := uuid.NewString()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		id, email, passwordHash, now.Format(time.DateTime),
	)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
	}, nil
}

func (r *SQLiteUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	var createdAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &u, nil
}

func (r *SQLiteUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	var u User
	var createdAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &u, nil
}

func (r *SQLiteUserRepository) UpdatePassword(
	ctx context.Context,
	id, passwordHash string,
) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ? WHERE id = ?`,
		passwordHash, id,
	)

	return err
}
