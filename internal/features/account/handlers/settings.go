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
	out, err := h.LoadSettings.Execute(r.Context(), usecases.LoadSettingsInput{UserID: ctxkey.UserIDFrom(r.Context())})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, http.StatusOK, views.SettingsPage(out.Email, out.DiscogsUsername, i18n.CookieLang(r)))
}

func (h *Handler) LanguageSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch lang := r.FormValue("lang"); {
	case lang == "":
		i18n.ClearCookie(w)
	case i18n.IsSupported(lang):
		i18n.SetCookie(w, r, lang)
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/app/settings", http.StatusSeeOther)
}
