package controllers

import (
	"cmp"
	"encoding/json"
	"net/http"

	"github.com/dghubble/oauth1"

	"github.com/mathcale/discogs-collection-manager/internal/views"
)

const defaultDiscogsIdentityURL = "https://api.discogs.com/oauth/identity"

func (c *Controller) SettingsPage(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)

	user, err := c.Users.FindByID(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := c.OAuthTokens.FindByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	discogsUsername := ""
	if token != nil {
		discogsUsername = token.DiscogsUsername
	}

	render(w, r, http.StatusOK, views.SettingsPage(user.Email, discogsUsername))
}

func (c *Controller) Connect(w http.ResponseWriter, r *http.Request) {
	requestToken, requestSecret, err := c.DiscogsOAuth.RequestToken()
	if err != nil {
		render(w, r, http.StatusBadGateway, views.DiscogsConnectError("Could not initiate Discogs connection. Please try again."))
		return
	}

	c.requestTokens.Store(requestToken, requestSecret)

	authURL, err := c.DiscogsOAuth.AuthorizationURL(requestToken)
	if err != nil {
		c.requestTokens.Delete(requestToken)
		render(w, r, http.StatusInternalServerError, views.DiscogsConnectError("Could not build authorization URL."))
		return
	}

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (c *Controller) Callback(w http.ResponseWriter, r *http.Request) {
	requestToken := r.URL.Query().Get("oauth_token")
	verifier := r.URL.Query().Get("oauth_verifier")

	if requestToken == "" || verifier == "" {
		render(w, r, http.StatusBadRequest, views.DiscogsConnectError("Invalid callback parameters."))
		return
	}

	requestSecretVal, ok := c.requestTokens.Load(requestToken)
	if !ok {
		render(w, r, http.StatusBadRequest, views.DiscogsConnectError("OAuth session expired or not found. Please try connecting again."))
		return
	}
	requestSecret, _ := requestSecretVal.(string)
	c.requestTokens.Delete(requestToken)

	accessToken, accessSecret, err := c.DiscogsOAuth.AccessToken(requestToken, requestSecret, verifier)
	if err != nil {
		render(w, r, http.StatusBadGateway, views.DiscogsConnectError("Could not connect your Discogs account. Please reconnect and try again."))
		return
	}

	discogsUsername := c.fetchDiscogsUsername(r, accessToken, accessSecret)

	userID := currentUserID(r)
	if _, err := c.OAuthTokens.Upsert(r.Context(), userID, accessToken, accessSecret, discogsUsername); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app/settings", http.StatusFound)
}

func (c *Controller) Disconnect(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)

	if err := c.OAuthTokens.DeleteByUserID(r.Context(), userID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app/settings", http.StatusFound)
}

func (c *Controller) fetchDiscogsUsername(r *http.Request, accessToken, accessSecret string) string {
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := c.DiscogsOAuth.Client(r.Context(), token)

	resp, err := httpClient.Get(cmp.Or(c.IdentityURL, defaultDiscogsIdentityURL))
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
