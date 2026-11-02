package usecases

import (
	"context"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type LogoutInput struct {
	SessionToken string
}

type LogoutOutput struct{}

type Logout struct {
	Sessions domain.SessionRepository
}

func (uc *Logout) Execute(ctx context.Context, in LogoutInput) (LogoutOutput, error) {
	return LogoutOutput{}, uc.Sessions.Delete(ctx, in.SessionToken)
}
