package handlers

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
)

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
