package controllers

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/views"
)

func Landing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := views.LandingPage()
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render template", http.StatusInternalServerError)
		return
	}
}
