package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/views"
)

func (c *Controller) ScanPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, views.ScannerPage())
}

func (c *Controller) Scan(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	barcode := r.FormValue("barcode")
	if barcode == "" {
		http.Error(w, "barcode is required", http.StatusBadRequest)
		return
	}

	userID := currentUserID(r)
	token, err := c.OAuthTokens.FindByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if token == nil {
		render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		return
	}

	releases, err := c.Discogs.SearchByBarcode(r.Context(), barcode)
	if err != nil {
		renderDiscogsError(w, r, err, "Could not search Discogs. Please try again.")
		return
	}

	if len(releases) == 0 {
		render(w, r, http.StatusOK, views.ScanErrorPartial("No releases found for this barcode."))
		return
	}
	if len(releases) > 1 {
		render(w, r, http.StatusOK, views.ScanConflictListPartial(releases))
		return
	}

	render(w, r, http.StatusOK, views.ScanConfirmPartial(releases[0]))
}

func (c *Controller) Confirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	releaseIDStr := r.FormValue("release_id")
	if releaseIDStr == "" {
		render(w, r, http.StatusUnprocessableEntity, views.ScanErrorPartial("Please select a release before confirming."))
		return
	}

	releaseID, err := strconv.Atoi(releaseIDStr)
	if err != nil {
		http.Error(w, "invalid release_id", http.StatusBadRequest)
		return
	}

	userID := currentUserID(r)
	token, err := c.OAuthTokens.FindByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if token == nil {
		render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		return
	}

	if r.FormValue("acknowledged") == "" {
		release, err := c.Discogs.GetRelease(r.Context(), releaseID)
		if err != nil {
			renderDiscogsError(w, r, err, "Could not check your collection. Please try again.")
			return
		}

		owned, err := c.Discogs.ListCollection(r.Context(), token.DiscogsUsername)
		if err != nil {
			renderDiscogsError(w, r, err, "Could not check your collection. Please try again.")
			return
		}

		if matches := discogs.MatchOwned(release, owned); matches.Any() {
			render(w, r, http.StatusOK, views.ScanOwnedWarningPartial(releaseID, matches))
			return
		}
	}

	if err := c.Discogs.AddToCollection(r.Context(), token.DiscogsUsername, releaseID); err != nil {
		renderDiscogsError(w, r, err, "Could not add release to collection. Please try again.")
		return
	}

	w.Header().Set("HX-Trigger", "scan-success")
	render(w, r, http.StatusOK, views.ScanSuccessPartial())
}

func (c *Controller) Cancel(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app/scanner", http.StatusFound)
}

func renderDiscogsError(w http.ResponseWriter, r *http.Request, err error, fallbackMessage string) {
	var rle *discogs.RateLimitError
	if errors.As(err, &rle) {
		render(w, r, http.StatusTooManyRequests, views.ScanErrorPartial("Discogs rate limit reached. Please wait a moment and try again."))
		return
	}

	var dae *discogs.DiscogsAuthError
	if errors.As(err, &dae) {
		render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		return
	}

	render(w, r, http.StatusBadGateway, views.ScanErrorPartial(fallbackMessage))
}
