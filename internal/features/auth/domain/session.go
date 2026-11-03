package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	shortSessionExpiry = 6 * time.Hour
	longSessionExpiry  = 30 * 24 * time.Hour
)

type (
	SessionRepository interface {
		FindByToken(ctx context.Context, token string) (*Session, error)
		Save(ctx context.Context, session *Session) error
		Delete(ctx context.Context, token string) error
		DeleteAllForUser(ctx context.Context, userID string) error
	}

	Session struct {
		ID         string
		UserID     string
		Token      string
		ExpiresAt  time.Time
		Persistent bool
		CreatedAt  time.Time
	}
)

func NewSession(userID string, persistent bool) (*Session, error) {
	token, err := GenerateToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &Session{
		ID:         uuid.NewString(),
		UserID:     userID,
		Token:      token,
		ExpiresAt:  now.Add(sessionExpiry(persistent)),
		Persistent: persistent,
		CreatedAt:  now,
	}, nil
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func sessionExpiry(persistent bool) time.Duration {
	if persistent {
		return longSessionExpiry
	}

	return shortSessionExpiry
}
