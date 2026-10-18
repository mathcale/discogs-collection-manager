package handler

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/templates"
)

func Landing(signupsEnabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		component := templates.LandingPage(signupsEnabled)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "failed to render template", http.StatusInternalServerError)
			return
		}
	}
}
