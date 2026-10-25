package discogs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dghubble/oauth1"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/repository"
)

const (
	defaultSearchURL = "https://api.discogs.com/database/search"
	defaultAPIBase   = "https://api.discogs.com"
	userAgent        = "DiscogsCollectionManager/1.0"
)

type Format struct {
	Name         string   `json:"name"`
	Text         string   `json:"text"`
	Descriptions []string `json:"descriptions"`
}

type Release struct {
	ID      int      `json:"id"`
	Title   string   `json:"title"`
	Country string   `json:"country"`
	Year    string   `json:"year"`
	Labels  []string `json:"label"`
	Catno   string   `json:"catno"`
	Formats []Format `json:"formats"`
	Thumb   string   `json:"thumb"`
	Path    string   `json:"uri"`
}

func (r Release) Label() string {
	if len(r.Labels) == 0 {
		return ""
	}
	return r.Labels[0]
}

func (r Release) FormatSummary() string {
	var parts []string
	for _, f := range r.Formats {
		parts = append(parts, f.Name)
		parts = append(parts, f.Descriptions...)
	}
	return strings.Join(parts, ", ")
}

func (r Release) Variant() string {
	var texts []string
	for _, f := range r.Formats {
		if f.Text != "" {
			texts = append(texts, f.Text)
		}
	}
	return strings.Join(texts, ", ")
}

func (r Release) URL() string {
	return "https://www.discogs.com" + r.Path
}

type RateLimitError struct{}

func (e *RateLimitError) Error() string {
	return "discogs rate limit exceeded"
}

type DiscogsAuthError struct{}

func (e *DiscogsAuthError) Error() string {
	return "discogs token revoked or unauthorised"
}

type DiscogsClient interface {
	SearchByBarcode(ctx context.Context, barcode string) ([]Release, error)
	AddToCollection(ctx context.Context, username string, releaseID int) error
}

type HTTPDiscogsClient struct {
	oauthConfig *oauth1.Config
	oauthTokens repository.OAuthTokenRepository
	searchURL   string
	apiBase     string
}

func NewHTTPDiscogsClient(cfg *oauth1.Config, oauthTokens repository.OAuthTokenRepository) *HTTPDiscogsClient {
	return &HTTPDiscogsClient{
		oauthConfig: cfg,
		oauthTokens: oauthTokens,
		searchURL:   defaultSearchURL,
		apiBase:     defaultAPIBase,
	}
}

func (c *HTTPDiscogsClient) httpClientForUser(ctx context.Context) (*http.Client, error) {
	userID, _ := ctx.Value(ctxkey.UserID).(string)
	token, err := c.oauthTokens.FindByUserID(ctx, userID)
	if err != nil || token == nil {
		return nil, fmt.Errorf("no discogs token for user")
	}
	oauthToken := oauth1.NewToken(token.AccessToken, token.AccessTokenSecret)
	return c.oauthConfig.Client(ctx, oauthToken), nil
}

func (c *HTTPDiscogsClient) SearchByBarcode(ctx context.Context, barcode string) ([]Release, error) {
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

func (c *HTTPDiscogsClient) AddToCollection(ctx context.Context, username string, releaseID int) error {
	httpClient, err := c.httpClientForUser(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/users/%s/collection/folders/1/releases/%d", c.apiBase, username, releaseID)
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
