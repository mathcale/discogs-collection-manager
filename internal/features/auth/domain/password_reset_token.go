package domain

import (
	"context"
	"time"
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
