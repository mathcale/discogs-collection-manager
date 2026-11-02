package domain

import (
	"context"
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

func NewPasswordResetToken(userID, tokenHash string) *PasswordResetToken {
	now := time.Now().UTC()

	return &PasswordResetToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(ResetTokenExpiry),
		CreatedAt: now,
	}
}

func (t *PasswordResetToken) IsUsable() bool {
	return t.UsedAt == nil && !time.Now().After(t.ExpiresAt)
}

type PasswordResetTokenRepository interface {
	Save(ctx context.Context, token *PasswordResetToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkAllUsedForUser(ctx context.Context, userID string) error
}
