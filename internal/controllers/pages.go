package controllers

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/ui"
	"github.com/mathcale/discogs-collection-manager/internal/views"
)

func (c *Controller) Landing(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, http.StatusOK, views.LandingPage(c.SignupsEnabled))
}
