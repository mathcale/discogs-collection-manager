package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
	"github.com/mathcale/discogs-collection-manager/internal/views"
)

func (c *Controller) ScanPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, http.StatusOK, views.ScannerPage())
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

	userID := ctxkey.UserIDFrom(r.Context())
	token, err := c.OAuthTokens.FindByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if token == nil {
		ui.Render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		return
	}

	releases, err := c.Discogs.SearchByBarcode(r.Context(), barcode)
	if err != nil {
		renderDiscogsError(w, r, err, "Could not search Discogs. Please try again.")
		return
	}

	if len(releases) == 0 {
		ui.Render(w, r, http.StatusOK, views.ScanErrorPartial("No releases found for this barcode."))
		return
	}
	if len(releases) > 1 {
		ui.Render(w, r, http.StatusOK, views.ScanConflictListPartial(releases))
		return
	}

	ui.Render(w, r, http.StatusOK, views.ScanConfirmPartial(releases[0]))
}

func (c *Controller) Confirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	releaseIDStr := r.FormValue("release_id")
	if releaseIDStr == "" {
		ui.Render(w, r, http.StatusUnprocessableEntity, views.ScanErrorPartial("Please select a release before confirming."))
		return
	}

	releaseID, err := strconv.Atoi(releaseIDStr)
	if err != nil {
		http.Error(w, "invalid release_id", http.StatusBadRequest)
		return
	}

	userID := ctxkey.UserIDFrom(r.Context())
	token, err := c.OAuthTokens.FindByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if token == nil {
		ui.Render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
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
			ui.Render(w, r, http.StatusOK, views.ScanOwnedWarningPartial(releaseID, matches))
			return
		}
	}

	if err := c.Discogs.AddToCollection(r.Context(), token.DiscogsUsername, releaseID); err != nil {
		renderDiscogsError(w, r, err, "Could not add release to collection. Please try again.")
		return
	}

	w.Header().Set("HX-Trigger", "scan-success")
	ui.Render(w, r, http.StatusOK, views.ScanSuccessPartial())
}

func (c *Controller) Cancel(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app/scanner", http.StatusFound)
}

func renderDiscogsError(w http.ResponseWriter, r *http.Request, err error, fallbackMessage string) {
	var rle *discogs.RateLimitError
	if errors.As(err, &rle) {
		ui.Render(w, r, http.StatusTooManyRequests, views.ScanErrorPartial("Discogs rate limit reached. Please wait a moment and try again."))
		return
	}

	var dae *discogs.DiscogsAuthError
	if errors.As(err, &dae) {
		ui.Render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		return
	}

	ui.Render(w, r, http.StatusBadGateway, views.ScanErrorPartial(fallbackMessage))
}
