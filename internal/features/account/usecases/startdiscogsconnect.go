package usecases

import (
	"context"
	"fmt"

	"github.com/dghubble/oauth1"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
)

type (
	StartDiscogsConnectInput struct {
		UserID string
	}

	StartDiscogsConnectOutput struct {
		AuthorizationURL string
	}

	StartDiscogsConnect struct {
		OAuth   *oauth1.Config
		Pending *PendingOAuthRequests
	}
)

func (uc *StartDiscogsConnect) Execute(
	_ context.Context,
	in StartDiscogsConnectInput,
) (StartDiscogsConnectOutput, error) {
	requestToken, requestSecret, err := uc.OAuth.RequestToken()
	if err != nil {
		return StartDiscogsConnectOutput{}, fmt.Errorf("%w: %w", domain.ErrRequestTokenFailed, err)
	}

	uc.Pending.Store(requestToken, requestSecret, in.UserID)

	authURL, err := uc.OAuth.AuthorizationURL(requestToken)
	if err != nil {
		uc.Pending.Delete(requestToken)

		return StartDiscogsConnectOutput{}, fmt.Errorf("%w: %w", domain.ErrAuthorizationURLFailed, err)
	}

	return StartDiscogsConnectOutput{
		AuthorizationURL: authURL.String(),
	}, nil
}
