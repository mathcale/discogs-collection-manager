package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	accountdomain "github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/domain"
)

type (
	AddToCollectionInput struct {
		UserID       string
		ReleaseID    int
		Acknowledged bool
	}

	AddToCollectionOutput struct {
		Owned discogs.OwnedMatches
	}

	AddToCollection struct {
		DiscogsClient   discogs.DiscogsClient
		OAuthTokensRepo oauthTokenFinder
	}
)

func (uc *AddToCollection) Execute(
	ctx context.Context,
	in AddToCollectionInput,
) (AddToCollectionOutput, error) {
	token, err := uc.OAuthTokensRepo.FindByUserID(ctx, in.UserID)
	if err != nil {
		if errors.Is(err, accountdomain.ErrNotFound) {
			return AddToCollectionOutput{}, domain.ErrDiscogsNotConnected
		}

		return AddToCollectionOutput{}, err
	}

	if !in.Acknowledged {
		release, err := uc.DiscogsClient.GetRelease(ctx, in.ReleaseID)
		if err != nil {
			return AddToCollectionOutput{}, fmt.Errorf("%w: %w", domain.ErrCollectionCheckFailed, err)
		}

		owned, err := uc.DiscogsClient.ListCollection(ctx, token.DiscogsUsername)
		if err != nil {
			return AddToCollectionOutput{}, fmt.Errorf("%w: %w", domain.ErrCollectionCheckFailed, err)
		}

		if matches := discogs.MatchOwned(release, owned); matches.Any() {
			return AddToCollectionOutput{
				Owned: matches,
			}, nil
		}
	}

	if err := uc.DiscogsClient.AddToCollection(
		ctx,
		token.DiscogsUsername,
		in.ReleaseID,
	); err != nil {
		return AddToCollectionOutput{}, fmt.Errorf("%w: %w", domain.ErrAddToCollectionFailed, err)
	}

	return AddToCollectionOutput{}, nil
}
