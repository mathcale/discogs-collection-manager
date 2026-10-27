package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
)

type AuthenticateSessionInput struct {
	Token string
}

type AuthenticateSessionOutput struct {
	UserID string
}

var ErrSessionInvalid = errors.New("session invalid or expired")

type AuthenticateSession struct {
	Sessions models.SessionRepository
}

func (uc *AuthenticateSession) Execute(ctx context.Context, in AuthenticateSessionInput) (AuthenticateSessionOutput, error) {
	session, err := uc.Sessions.FindByToken(ctx, in.Token)
	if err != nil {
		return AuthenticateSessionOutput{}, err
	}
	if session == nil || session.IsExpired() {
		return AuthenticateSessionOutput{}, ErrSessionInvalid
	}

	return AuthenticateSessionOutput{UserID: session.UserID}, nil
}
