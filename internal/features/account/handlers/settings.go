package handlers

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/features/account/views"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
)

func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	out, err := h.LoadSettings.Execute(r.Context(), usecases.LoadSettingsInput{UserID: ctxkey.UserIDFrom(r.Context())})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, http.StatusOK, views.SettingsPage(out.Email, out.DiscogsUsername))
}
