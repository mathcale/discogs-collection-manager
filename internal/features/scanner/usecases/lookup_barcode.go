package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	accountdomain "github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
)

type LookupBarcodeInput struct {
	UserID  string
	Barcode string
}

type LookupBarcodeOutput struct {
	Releases []discogs.Release
}

var (
	ErrDiscogsNotConnected = errors.New("discogs account not connected")
	ErrSearchFailed        = errors.New("discogs search failed")
)

type oauthTokenFinder interface {
	FindByUserID(ctx context.Context, userID string) (*accountdomain.OAuthToken, error)
}

type LookupBarcode struct {
	Discogs     discogs.DiscogsClient
	OAuthTokens oauthTokenFinder
}

func (uc *LookupBarcode) Execute(ctx context.Context, in LookupBarcodeInput) (LookupBarcodeOutput, error) {
	_, err := uc.OAuthTokens.FindByUserID(ctx, in.UserID)
	if errors.Is(err, accountdomain.ErrNotFound) {
		return LookupBarcodeOutput{}, ErrDiscogsNotConnected
	}
	if err != nil {
		return LookupBarcodeOutput{}, err
	}

	releases, err := uc.Discogs.SearchByBarcode(ctx, in.Barcode)
	if err != nil {
		return LookupBarcodeOutput{}, fmt.Errorf("%w: %w", ErrSearchFailed, err)
	}

	return LookupBarcodeOutput{Releases: releases}, nil
}
