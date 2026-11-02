package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type ValidateResetTokenInput struct {
	Token string
}

type ValidateResetTokenOutput struct{}

var ErrResetTokenInvalid = errors.New("reset token invalid or expired")

type ValidateResetToken struct {
	ResetTokens domain.PasswordResetTokenRepository
}

func (uc *ValidateResetToken) Execute(ctx context.Context, in ValidateResetTokenInput) (ValidateResetTokenOutput, error) {
	if _, err := findUsableResetToken(ctx, uc.ResetTokens, in.Token); err != nil {
		return ValidateResetTokenOutput{}, err
	}

	return ValidateResetTokenOutput{}, nil
}
