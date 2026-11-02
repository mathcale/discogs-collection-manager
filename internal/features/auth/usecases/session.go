package usecases

import (
	"context"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

func createSession(ctx context.Context, sessions domain.SessionRepository, userID string, persistent bool) (string, time.Time, error) {
	token, err := domain.GenerateToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := time.Now().Add(domain.SessionExpiry(persistent))

	if _, err := sessions.Create(ctx, userID, token, expiresAt, persistent); err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}
