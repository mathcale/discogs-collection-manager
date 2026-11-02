package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	accountdomain "github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
)

type AddToCollectionInput struct {
	UserID       string
	ReleaseID    int
	Acknowledged bool
}

type AddToCollectionOutput struct {
	Owned discogs.OwnedMatches
}

var (
	ErrCollectionCheckFailed = errors.New("collection check failed")
	ErrAddToCollectionFailed = errors.New("add to collection failed")
)

type AddToCollection struct {
	Discogs     discogs.DiscogsClient
	OAuthTokens oauthTokenFinder
}

func (uc *AddToCollection) Execute(ctx context.Context, in AddToCollectionInput) (AddToCollectionOutput, error) {
	token, err := uc.OAuthTokens.FindByUserID(ctx, in.UserID)
	if errors.Is(err, accountdomain.ErrNotFound) {
		return AddToCollectionOutput{}, ErrDiscogsNotConnected
	}
	if err != nil {
		return AddToCollectionOutput{}, err
	}

	if !in.Acknowledged {
		release, err := uc.Discogs.GetRelease(ctx, in.ReleaseID)
		if err != nil {
			return AddToCollectionOutput{}, fmt.Errorf("%w: %w", ErrCollectionCheckFailed, err)
		}

		owned, err := uc.Discogs.ListCollection(ctx, token.DiscogsUsername)
		if err != nil {
			return AddToCollectionOutput{}, fmt.Errorf("%w: %w", ErrCollectionCheckFailed, err)
		}

		if matches := discogs.MatchOwned(release, owned); matches.Any() {
			return AddToCollectionOutput{Owned: matches}, nil
		}
	}

	if err := uc.Discogs.AddToCollection(ctx, token.DiscogsUsername, in.ReleaseID); err != nil {
		return AddToCollectionOutput{}, fmt.Errorf("%w: %w", ErrAddToCollectionFailed, err)
	}

	return AddToCollectionOutput{}, nil
}
