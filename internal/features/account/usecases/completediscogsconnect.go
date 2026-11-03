package usecases

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dghubble/oauth1"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
)

const defaultDiscogsIdentityURL = "https://api.discogs.com/oauth/identity"

var (
	ErrPendingRequestNotFound = errors.New("pending oauth request not found")
	ErrAccessTokenFailed      = errors.New("discogs access token failed")
)

type (
	CompleteDiscogsConnectInput struct {
		UserID       string
		RequestToken string
		Verifier     string
	}

	CompleteDiscogsConnectOutput struct{}

	CompleteDiscogsConnect struct {
		OAuth           *oauth1.Config
		Pending         *PendingOAuthRequests
		OAuthTokensRepo domain.OAuthTokenRepository
		IdentityURL     string
	}
)

func (uc *CompleteDiscogsConnect) Execute(
	ctx context.Context,
	in CompleteDiscogsConnectInput,
) (CompleteDiscogsConnectOutput, error) {
	secret, pendingUserID, ok := uc.Pending.Take(in.RequestToken)
	if !ok || pendingUserID != in.UserID {
		return CompleteDiscogsConnectOutput{}, ErrPendingRequestNotFound
	}

	accessToken, accessSecret, err := uc.OAuth.AccessToken(in.RequestToken, secret, in.Verifier)
	if err != nil {
		return CompleteDiscogsConnectOutput{}, fmt.Errorf("%w: %w", ErrAccessTokenFailed, err)
	}

	discogsUsername := uc.fetchDiscogsUsername(ctx, accessToken, accessSecret)

	token := domain.NewOAuthToken(in.UserID, accessToken, accessSecret, discogsUsername)

	if err := uc.OAuthTokensRepo.Save(ctx, token); err != nil {
		return CompleteDiscogsConnectOutput{}, err
	}

	return CompleteDiscogsConnectOutput{}, nil
}

func (uc *CompleteDiscogsConnect) fetchDiscogsUsername(
	ctx context.Context,
	accessToken, accessSecret string,
) string {
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := uc.OAuth.Client(ctx, token)

	resp, err := httpClient.Get(cmp.Or(uc.IdentityURL, defaultDiscogsIdentityURL))
	if err != nil {
		return ""
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var identity struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&identity); err != nil {
		return ""
	}

	return identity.Username
}
