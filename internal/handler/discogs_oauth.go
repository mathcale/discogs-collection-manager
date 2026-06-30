package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/dghubble/oauth1"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/repository"
	"github.com/mathcale/discogs-collection-manager/internal/templates"
)

const defaultDiscogsIdentityURL = "https://api.discogs.com/oauth/identity"

type DiscogsOAuthHandler struct {
	OAuthTokens   repository.OAuthTokenRepository
	Sessions      repository.SessionRepository
	Users         repository.UserRepository
	OAuthConfig   *oauth1.Config
	IdentityURL   string
	requestTokens sync.Map
}

func NewDiscogsOAuthHandler(
	oauthTokens repository.OAuthTokenRepository,
	sessions repository.SessionRepository,
	users repository.UserRepository,
	oauthConfig *oauth1.Config,
) *DiscogsOAuthHandler {
	return &DiscogsOAuthHandler{
		OAuthTokens: oauthTokens,
		Sessions:    sessions,
		Users:       users,
		OAuthConfig: oauthConfig,
		IdentityURL: defaultDiscogsIdentityURL,
	}
}

func (h *DiscogsOAuthHandler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxkey.UserID).(string)

	user, err := h.Users.FindByID(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := h.OAuthTokens.FindByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	discogsUsername := ""
	if token != nil {
		discogsUsername = token.DiscogsUsername
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.SettingsPage(user.Email, discogsUsername).Render(r.Context(), w)
}

func (h *DiscogsOAuthHandler) Connect(w http.ResponseWriter, r *http.Request) {
	requestToken, requestSecret, err := h.OAuthConfig.RequestToken()
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		templates.DiscogsConnectError("Could not initiate Discogs connection. Please try again.").Render(r.Context(), w)
		return
	}

	h.requestTokens.Store(requestToken, requestSecret)

	authURL, err := h.OAuthConfig.AuthorizationURL(requestToken)
	if err != nil {
		h.requestTokens.Delete(requestToken)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		templates.DiscogsConnectError("Could not build authorization URL.").Render(r.Context(), w)
		return
	}

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (h *DiscogsOAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	requestToken := r.URL.Query().Get("oauth_token")
	verifier := r.URL.Query().Get("oauth_verifier")

	if requestToken == "" || verifier == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		templates.DiscogsConnectError("Invalid callback parameters.").Render(r.Context(), w)
		return
	}

	requestSecretVal, ok := h.requestTokens.Load(requestToken)
	if !ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		templates.DiscogsConnectError("OAuth session expired or not found. Please try connecting again.").Render(r.Context(), w)
		return
	}
	requestSecret, _ := requestSecretVal.(string)
	h.requestTokens.Delete(requestToken)

	accessToken, accessSecret, err := h.OAuthConfig.AccessToken(requestToken, requestSecret, verifier)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		templates.DiscogsConnectError("Could not connect your Discogs account. Please reconnect and try again.").Render(r.Context(), w)
		return
	}

	discogsUsername := h.fetchDiscogsUsername(r, accessToken, accessSecret)

	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	if _, err := h.OAuthTokens.Upsert(r.Context(), userID, accessToken, accessSecret, discogsUsername); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app/settings", http.StatusFound)
}

func (h *DiscogsOAuthHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxkey.UserID).(string)

	if err := h.OAuthTokens.DeleteByUserID(r.Context(), userID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app/settings", http.StatusFound)
}

func (h *DiscogsOAuthHandler) fetchDiscogsUsername(r *http.Request, accessToken, accessSecret string) string {
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := h.OAuthConfig.Client(r.Context(), token)

	resp, err := httpClient.Get(h.IdentityURL)
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
