package usecases

import (
	"context"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/models"
)

type DisconnectDiscogsInput struct {
	UserID string
}

type DisconnectDiscogsOutput struct{}

type DisconnectDiscogs struct {
	OAuthTokens models.OAuthTokenRepository
}

func (uc *DisconnectDiscogs) Execute(ctx context.Context, in DisconnectDiscogsInput) (DisconnectDiscogsOutput, error) {
	if err := uc.OAuthTokens.DeleteByUserID(ctx, in.UserID); err != nil {
		return DisconnectDiscogsOutput{}, err
	}

	return DisconnectDiscogsOutput{}, nil
}
