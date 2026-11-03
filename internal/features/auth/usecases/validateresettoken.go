package usecases

import (
	"context"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type (
	ValidateResetTokenInput struct {
		Token string
	}

	ValidateResetTokenOutput struct{}

	ValidateResetToken struct {
		ResetTokensRepo domain.PasswordResetTokenRepository
	}
)

func (uc *ValidateResetToken) Execute(
	ctx context.Context,
	in ValidateResetTokenInput,
) (ValidateResetTokenOutput, error) {
	if _, err := findUsableResetToken(ctx, uc.ResetTokensRepo, in.Token); err != nil {
		return ValidateResetTokenOutput{}, err
	}

	return ValidateResetTokenOutput{}, nil
}
