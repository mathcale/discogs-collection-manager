package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type (
	AuthenticateSessionInput struct {
		Token string
	}

	AuthenticateSessionOutput struct {
		UserID string
	}

	AuthenticateSession struct {
		SessionsRepo domain.SessionRepository
	}
)

func (uc *AuthenticateSession) Execute(
	ctx context.Context,
	in AuthenticateSessionInput,
) (AuthenticateSessionOutput, error) {
	session, err := uc.SessionsRepo.FindByToken(ctx, in.Token)
	if errors.Is(err, domain.ErrNotFound) {
		return AuthenticateSessionOutput{}, domain.ErrSessionInvalid
	}

	if err != nil {
		return AuthenticateSessionOutput{}, err
	}

	if session.IsExpired() {
		return AuthenticateSessionOutput{}, domain.ErrSessionInvalid
	}

	return AuthenticateSessionOutput{
		UserID: session.UserID,
	}, nil
}
