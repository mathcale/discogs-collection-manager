package handlers

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/landing/views"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
)

type Handler struct {
	SignupsEnabled bool
}

func (h *Handler) Landing(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, http.StatusOK, views.LandingPage(h.SignupsEnabled))
}
