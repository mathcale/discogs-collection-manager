package domain

import (
	"context"
	"time"
)

const (
	shortSessionExpiry = 6 * time.Hour
	longSessionExpiry  = 30 * 24 * time.Hour
)

func SessionExpiry(persistent bool) time.Duration {
	if persistent {
		return longSessionExpiry
	}
	return shortSessionExpiry
}

type Session struct {
	ID         string
	UserID     string
	Token      string
	ExpiresAt  time.Time
	Persistent bool
	CreatedAt  time.Time
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

type SessionRepository interface {
	Create(ctx context.Context, userID, token string, expiresAt time.Time, persistent bool) (*Session, error)
	FindByToken(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
	DeleteAllForUser(ctx context.Context, userID string) error
}
