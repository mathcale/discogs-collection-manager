package handlers

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/features/account/views"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
)

func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	input := usecases.LoadSettingsInput{
		UserID: ctxkey.UserIDFrom(r.Context()),
	}

	out, err := h.LoadSettingsUseCase.Execute(r.Context(), input)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	view := views.SettingsPage(out.Email, out.DiscogsUsername, i18n.CookieLang(r))

	ui.Render(w, r, http.StatusOK, view)
}
