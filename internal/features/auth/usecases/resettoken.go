package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

func findUsableResetToken(
	ctx context.Context,
	resetTokens domain.PasswordResetTokenRepository,
	rawToken string,
) (*domain.PasswordResetToken, error) {
	record, err := resetTokens.FindByTokenHash(ctx, domain.HashToken(rawToken))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrResetTokenInvalid
		}

		return nil, err
	}

	if !record.IsUsable() {
		return nil, domain.ErrResetTokenInvalid
	}

	return record, nil
}
