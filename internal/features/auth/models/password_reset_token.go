package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const ResetTokenExpiry = time.Hour

type PasswordResetToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (t *PasswordResetToken) IsUsable() bool {
	return t.UsedAt == nil && !time.Now().After(t.ExpiresAt)
}

type PasswordResetTokenRepository interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*PasswordResetToken, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkUsed(ctx context.Context, id string) error
	MarkAllUsedForUser(ctx context.Context, userID string) error
}

type passwordResetTokenRepository struct {
	db *sql.DB
}

func NewPasswordResetTokenRepository(db *sql.DB) PasswordResetTokenRepository {
	return &passwordResetTokenRepository{
		db: db,
	}
}

func (r *passwordResetTokenRepository) Create(
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

func (r *passwordResetTokenRepository) FindByTokenHash(
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

func (r *passwordResetTokenRepository) MarkUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.DateTime), id,
	)

	return err
}

func (r *passwordResetTokenRepository) MarkAllUsedForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = ? WHERE user_id = ? AND used_at IS NULL`,
		time.Now().UTC().Format(time.DateTime), userID,
	)

	return err
}
