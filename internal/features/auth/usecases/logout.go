package usecases

import (
	"context"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type (
	LogoutInput struct {
		SessionToken string
	}

	LogoutOutput struct{}

	Logout struct {
		SessionsRepo domain.SessionRepository
	}
)

func (uc *Logout) Execute(ctx context.Context, in LogoutInput) (LogoutOutput, error) {
	return LogoutOutput{}, uc.SessionsRepo.Delete(ctx, in.SessionToken)
}
