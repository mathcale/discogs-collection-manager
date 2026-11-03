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
	LookupBarcodeInput struct {
		UserID  string
		Barcode string
	}

	LookupBarcodeOutput struct {
		Releases []discogs.Release
	}

	oauthTokenFinder interface {
		FindByUserID(ctx context.Context, userID string) (*accountdomain.OAuthToken, error)
	}

	LookupBarcode struct {
		DiscogsClient   discogs.DiscogsClient
		OAuthTokensRepo oauthTokenFinder
	}
)

func (uc *LookupBarcode) Execute(
	ctx context.Context,
	in LookupBarcodeInput,
) (LookupBarcodeOutput, error) {
	_, err := uc.OAuthTokensRepo.FindByUserID(ctx, in.UserID)
	if err != nil {
		if errors.Is(err, accountdomain.ErrNotFound) {
			return LookupBarcodeOutput{}, domain.ErrDiscogsNotConnected
		}

		return LookupBarcodeOutput{}, err
	}

	releases, err := uc.DiscogsClient.SearchByBarcode(ctx, in.Barcode)
	if err != nil {
		return LookupBarcodeOutput{}, fmt.Errorf("%w: %w", domain.ErrSearchFailed, err)
	}

	return LookupBarcodeOutput{
		Releases: releases,
	}, nil
}
