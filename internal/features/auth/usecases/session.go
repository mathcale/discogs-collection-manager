package usecases

import (
	"context"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

func createSession(ctx context.Context, sessions domain.SessionRepository, userID string, persistent bool) (string, time.Time, error) {
	session, err := domain.NewSession(userID, persistent)
	if err != nil {
		return "", time.Time{}, err
	}

	if err := sessions.Save(ctx, session); err != nil {
		return "", time.Time{}, err
	}

	return session.Token, session.ExpiresAt, nil
}
