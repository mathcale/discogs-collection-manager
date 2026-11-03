package usecases

import (
	"context"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
)

type (
	DisconnectDiscogsInput struct {
		UserID string
	}

	DisconnectDiscogsOutput struct{}

	DisconnectDiscogs struct {
		OAuthTokensRepo domain.OAuthTokenRepository
	}
)

func (uc *DisconnectDiscogs) Execute(
	ctx context.Context,
	in DisconnectDiscogsInput,
) (DisconnectDiscogsOutput, error) {
	if err := uc.OAuthTokensRepo.DeleteByUserID(ctx, in.UserID); err != nil {
		return DisconnectDiscogsOutput{}, err
	}

	return DisconnectDiscogsOutput{}, nil
}
