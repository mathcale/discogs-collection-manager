package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

func findUsableResetToken(ctx context.Context, resetTokens domain.PasswordResetTokenRepository, rawToken string) (*domain.PasswordResetToken, error) {
	record, err := resetTokens.FindByTokenHash(ctx, domain.HashToken(rawToken))
	if errors.Is(err, domain.ErrNotFound) {
		return nil, ErrResetTokenInvalid
	}
	if err != nil {
		return nil, err
	}
	if !record.IsUsable() {
		return nil, ErrResetTokenInvalid
	}

	return record, nil
}
