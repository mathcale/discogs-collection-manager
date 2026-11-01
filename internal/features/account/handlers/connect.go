package handlers

import (
	"errors"
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/features/account/views"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
)

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	out, err := h.StartDiscogsConnect.Execute(r.Context(), usecases.StartDiscogsConnectInput{UserID: ctxkey.UserIDFrom(r.Context())})
	if err != nil {
		switch {
		case errors.Is(err, usecases.ErrRequestTokenFailed):
			ui.Render(w, r, http.StatusBadGateway, views.DiscogsConnectError(i18n.T(r.Context(), "Could not initiate Discogs connection. Please try again.")))
		case errors.Is(err, usecases.ErrAuthorizationURLFailed):
			ui.Render(w, r, http.StatusInternalServerError, views.DiscogsConnectError(i18n.T(r.Context(), "Could not build authorization URL.")))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, out.AuthorizationURL, http.StatusFound)
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	requestToken := r.URL.Query().Get("oauth_token")
	verifier := r.URL.Query().Get("oauth_verifier")

	if requestToken == "" || verifier == "" {
		ui.Render(w, r, http.StatusBadRequest, views.DiscogsConnectError(i18n.T(r.Context(), "Invalid callback parameters.")))
		return
	}

	_, err := h.CompleteDiscogsConnect.Execute(r.Context(), usecases.CompleteDiscogsConnectInput{
		UserID:       ctxkey.UserIDFrom(r.Context()),
		RequestToken: requestToken,
		Verifier:     verifier,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecases.ErrPendingRequestNotFound):
			ui.Render(w, r, http.StatusBadRequest, views.DiscogsConnectError(i18n.T(r.Context(), "OAuth session expired or not found. Please try connecting again.")))
		case errors.Is(err, usecases.ErrAccessTokenFailed):
			ui.Render(w, r, http.StatusBadGateway, views.DiscogsConnectError(i18n.T(r.Context(), "Could not connect your Discogs account. Please reconnect and try again.")))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/app/settings", http.StatusFound)
}

func (h *Handler) Disconnect(w http.ResponseWriter, r *http.Request) {
	_, err := h.DisconnectDiscogs.Execute(r.Context(), usecases.DisconnectDiscogsInput{UserID: ctxkey.UserIDFrom(r.Context())})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app/settings", http.StatusFound)
}
