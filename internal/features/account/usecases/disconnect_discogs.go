package usecases

import (
	"context"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
)

type DisconnectDiscogsInput struct {
	UserID string
}

type DisconnectDiscogsOutput struct{}

type DisconnectDiscogs struct {
	OAuthTokens domain.OAuthTokenRepository
}

func (uc *DisconnectDiscogs) Execute(ctx context.Context, in DisconnectDiscogsInput) (DisconnectDiscogsOutput, error) {
	if err := uc.OAuthTokens.DeleteByUserID(ctx, in.UserID); err != nil {
		return DisconnectDiscogsOutput{}, err
	}

	return DisconnectDiscogsOutput{}, nil
}
