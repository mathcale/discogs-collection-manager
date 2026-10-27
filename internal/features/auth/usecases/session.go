package usecases

import (
	"context"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
)

func createSession(ctx context.Context, sessions models.SessionRepository, userID string, persistent bool) (string, time.Time, error) {
	token, err := models.GenerateToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := time.Now().Add(models.SessionExpiry(persistent))

	if _, err := sessions.Create(ctx, userID, token, expiresAt, persistent); err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}
