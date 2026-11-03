package handlers

import (
	"errors"
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
	"github.com/mathcale/discogs-collection-manager/internal/features/account/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/features/account/views"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
)

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	input := usecases.StartDiscogsConnectInput{
		UserID: ctxkey.UserIDFrom(r.Context()),
	}

	out, err := h.StartDiscogsConnectUseCase.Execute(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRequestTokenFailed):
			ui.Render(w, r, http.StatusBadGateway, views.DiscogsConnectError(
				i18n.T(
					r.Context(),
					"Could not initiate Discogs connection. Please try again.",
				),
			))
		case errors.Is(err, domain.ErrAuthorizationURLFailed):
			ui.Render(w, r, http.StatusInternalServerError, views.DiscogsConnectError(
				i18n.T(
					r.Context(),
					"Could not build authorization URL.",
				),
			))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}

		return
	}

	http.Redirect(w, r, out.AuthorizationURL, http.StatusFound)
}
