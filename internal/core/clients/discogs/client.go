package discogs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dghubble/oauth1"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
)

const (
	defaultSearchURL = "https://api.discogs.com/database/search"
	defaultAPIBase   = "https://api.discogs.com"
	userAgent        = "DiscogsCollectionManager/1.0"
)

type (
	DiscogsClient interface {
		SearchByBarcode(ctx context.Context, barcode string) ([]Release, error)
		GetRelease(ctx context.Context, releaseID int) (ReleaseDetail, error)
		ListCollection(ctx context.Context, username string) ([]OwnedRelease, error)
		AddToCollection(ctx context.Context, username string, releaseID int) error
	}

	HTTPDiscogsClient struct {
		oauthConfig *oauth1.Config
		oauthTokens domain.OAuthTokenRepository
		searchURL   string
		apiBase     string
	}
)

func NewHTTPDiscogsClient(
	cfg *oauth1.Config,
	oauthTokens domain.OAuthTokenRepository,
) *HTTPDiscogsClient {
	return &HTTPDiscogsClient{
		oauthConfig: cfg,
		oauthTokens: oauthTokens,
		searchURL:   defaultSearchURL,
		apiBase:     defaultAPIBase,
	}
}

func (c *HTTPDiscogsClient) SearchByBarcode(
	ctx context.Context,
	barcode string,
) ([]Release, error) {
	httpClient, err := c.httpClientForUser(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s?barcode=%s", c.searchURL, barcode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &RateLimitError{}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &DiscogsAuthError{}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discogs search failed with status %d", resp.StatusCode)
	}

	var result struct {
		Results []Release `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Results, nil
}

func (c *HTTPDiscogsClient) AddToCollection(
	ctx context.Context,
	username string,
	releaseID int,
) error {
	httpClient, err := c.httpClientForUser(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"%s/users/%s/collection/folders/1/releases/%d",
		c.apiBase, username, releaseID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return &DiscogsAuthError{}
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("discogs add to collection failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPDiscogsClient) GetRelease(ctx context.Context, releaseID int) (ReleaseDetail, error) {
	var detail ReleaseDetail

	err := c.getJSON(ctx, fmt.Sprintf("%s/releases/%d", c.apiBase, releaseID), &detail)

	return detail, err
}

func (c *HTTPDiscogsClient) ListCollection(
	ctx context.Context,
	username string,
) ([]OwnedRelease, error) {
	var owned []OwnedRelease

	for page := 1; ; page++ {
		var result struct {
			Pagination struct {
				Pages int `json:"pages"`
			} `json:"pagination"`
			Releases []struct {
				BasicInformation OwnedRelease `json:"basic_information"`
			} `json:"releases"`
		}

		url := fmt.Sprintf(
			"%s/users/%s/collection/folders/0/releases?per_page=100&page=%d",
			c.apiBase, username, page,
		)

		if err := c.getJSON(ctx, url, &result); err != nil {
			return nil, err
		}

		for _, r := range result.Releases {
			owned = append(owned, r.BasicInformation)
		}

		if page >= result.Pagination.Pages {
			return owned, nil
		}
	}
}

func (c *HTTPDiscogsClient) httpClientForUser(ctx context.Context) (*http.Client, error) {
	userID := ctxkey.UserIDFrom(ctx)

	token, err := c.oauthTokens.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no discogs token for user")
	}

	oauthToken := oauth1.NewToken(token.AccessToken, token.AccessTokenSecret)

	return c.oauthConfig.Client(ctx, oauthToken), nil
}

func (c *HTTPDiscogsClient) getJSON(ctx context.Context, url string, target any) error {
	httpClient, err := c.httpClientForUser(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return &DiscogsAuthError{}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discogs request %s failed with status %d", url, resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
