package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type passwordResetTokenRepository struct {
	db *sql.DB
}

func NewPasswordResetTokenRepository(db *sql.DB) domain.PasswordResetTokenRepository {
	return &passwordResetTokenRepository{
		db: db,
	}
}

func (r *passwordResetTokenRepository) Save(
	ctx context.Context,
	token *domain.PasswordResetToken,
) error {
	var usedAt any
	if token.UsedAt != nil {
		usedAt = token.UsedAt.UTC().Format(time.DateTime)
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, used_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET used_at = excluded.used_at`,
		token.ID, token.UserID, token.TokenHash,
		token.ExpiresAt.UTC().Format(time.DateTime), usedAt, token.CreatedAt.UTC().Format(time.DateTime),
	)

	return err
}

func (r *passwordResetTokenRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*domain.PasswordResetToken, error) {
	var t domain.PasswordResetToken
	var usedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, token_hash, expires_at, used_at, created_at
		 FROM password_reset_tokens
		 WHERE token_hash = ?`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &usedAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}

	return &t, nil
}

func (r *passwordResetTokenRepository) MarkAllUsedForUser(
	ctx context.Context,
	userID string,
) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = ? WHERE user_id = ? AND used_at IS NULL`,
		time.Now().UTC().Format(time.DateTime), userID,
	)

	return err
}
