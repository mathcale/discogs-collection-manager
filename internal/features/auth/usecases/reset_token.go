package usecases

import (
	"context"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

func findUsableResetToken(ctx context.Context, resetTokens domain.PasswordResetTokenRepository, rawToken string) (*domain.PasswordResetToken, error) {
	record, err := resetTokens.FindByTokenHash(ctx, domain.HashToken(rawToken))
	if err != nil {
		return nil, err
	}
	if record == nil || !record.IsUsable() {
		return nil, ErrResetTokenInvalid
	}

	return record, nil
}
