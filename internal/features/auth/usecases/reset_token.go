package usecases

import (
	"context"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
)

func findUsableResetToken(ctx context.Context, resetTokens models.PasswordResetTokenRepository, rawToken string) (*models.PasswordResetToken, error) {
	record, err := resetTokens.FindByTokenHash(ctx, models.HashToken(rawToken))
	if err != nil {
		return nil, err
	}
	if record == nil || !record.IsUsable() {
		return nil, ErrResetTokenInvalid
	}

	return record, nil
}
