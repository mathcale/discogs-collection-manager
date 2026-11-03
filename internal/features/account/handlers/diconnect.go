package handlers

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
)

func (h *Handler) Disconnect(w http.ResponseWriter, r *http.Request) {
	input := usecases.DisconnectDiscogsInput{
		UserID: ctxkey.UserIDFrom(r.Context()),
	}

	_, err := h.DisconnectDiscogsUseCase.Execute(r.Context(), input)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app/settings", http.StatusFound)
}
