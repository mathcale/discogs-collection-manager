package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/dghubble/oauth1"
)

type StartDiscogsConnectInput struct {
	UserID string
}

type StartDiscogsConnectOutput struct {
	AuthorizationURL string
}

var (
	ErrRequestTokenFailed     = errors.New("discogs request token failed")
	ErrAuthorizationURLFailed = errors.New("discogs authorization url failed")
)

type StartDiscogsConnect struct {
	OAuth   *oauth1.Config
	Pending *PendingOAuthRequests
}

func (uc *StartDiscogsConnect) Execute(_ context.Context, in StartDiscogsConnectInput) (StartDiscogsConnectOutput, error) {
	requestToken, requestSecret, err := uc.OAuth.RequestToken()
	if err != nil {
		return StartDiscogsConnectOutput{}, fmt.Errorf("%w: %w", ErrRequestTokenFailed, err)
	}

	uc.Pending.Store(requestToken, requestSecret, in.UserID)

	authURL, err := uc.OAuth.AuthorizationURL(requestToken)
	if err != nil {
		uc.Pending.Delete(requestToken)
		return StartDiscogsConnectOutput{}, fmt.Errorf("%w: %w", ErrAuthorizationURLFailed, err)
	}

	return StartDiscogsConnectOutput{AuthorizationURL: authURL.String()}, nil
}
