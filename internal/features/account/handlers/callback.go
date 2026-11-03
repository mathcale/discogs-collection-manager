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

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	requestToken := r.URL.Query().Get("oauth_token")
	verifier := r.URL.Query().Get("oauth_verifier")

	if requestToken == "" || verifier == "" {
		ui.Render(w, r, http.StatusBadRequest, views.DiscogsConnectError(
			i18n.T(
				r.Context(),
				"Invalid callback parameters.",
			),
		))

		return
	}

	input := usecases.CompleteDiscogsConnectInput{
		UserID:       ctxkey.UserIDFrom(r.Context()),
		RequestToken: requestToken,
		Verifier:     verifier,
	}

	_, err := h.CompleteDiscogsConnectUseCase.Execute(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, usecases.ErrPendingRequestNotFound):
			ui.Render(w, r, http.StatusBadRequest, views.DiscogsConnectError(
				i18n.T(
					r.Context(),
					"OAuth session expired or not found. Please try connecting again.",
				),
			))
		case errors.Is(err, usecases.ErrAccessTokenFailed):
			ui.Render(w, r, http.StatusBadGateway, views.DiscogsConnectError(
				i18n.T(
					r.Context(),
					"Could not connect your Discogs account. Please reconnect and try again.",
				),
			))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}

		return
	}

	http.Redirect(w, r, "/app/settings", http.StatusFound)
}
