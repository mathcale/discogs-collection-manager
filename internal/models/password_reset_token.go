package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type PasswordResetToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type PasswordResetTokenRepository interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*PasswordResetToken, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkUsed(ctx context.Context, id string) error
}

type SQLitePasswordResetTokenRepository struct {
	db *sql.DB
}

func NewSQLitePasswordResetTokenRepository(db *sql.DB) *SQLitePasswordResetTokenRepository {
	return &SQLitePasswordResetTokenRepository{
		db: db,
	}
}

func (r *SQLitePasswordResetTokenRepository) Create(
	ctx context.Context,
	userID, tokenHash string,
	expiresAt time.Time,
) (*PasswordResetToken, error) {
	id := uuid.NewString()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, tokenHash, expiresAt.UTC().Format(time.DateTime), now.Format(time.DateTime),
	)
	if err != nil {
		return nil, err
	}

	return &PasswordResetToken{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

func (r *SQLitePasswordResetTokenRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*PasswordResetToken, error) {
	var t PasswordResetToken
	var usedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, token_hash, expires_at, used_at, created_at
		 FROM password_reset_tokens
		 WHERE token_hash = ?`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &usedAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}

	return &t, nil
}

func (r *SQLitePasswordResetTokenRepository) MarkUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.DateTime), id,
	)

	return err
}
