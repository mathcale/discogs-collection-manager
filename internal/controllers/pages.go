package controllers

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/views"
)

func (c *Controller) Landing(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, views.LandingPage(c.SignupsEnabled))
}
