package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/views"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
)

func (h *Handler) ScanPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, http.StatusOK, views.ScannerPage())
}

func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	barcode := r.FormValue("barcode")
	if barcode == "" {
		http.Error(w, "barcode is required", http.StatusBadRequest)
		return
	}

	out, err := h.LookupBarcode.Execute(r.Context(), usecases.LookupBarcodeInput{
		UserID:  ctxkey.UserIDFrom(r.Context()),
		Barcode: barcode,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecases.ErrDiscogsNotConnected):
			ui.Render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		case errors.Is(err, usecases.ErrSearchFailed):
			renderDiscogsError(w, r, err, i18n.T(r.Context(), "Could not search Discogs. Please try again."))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	if len(out.Releases) == 0 {
		ui.Render(w, r, http.StatusOK, views.ScanErrorPartial(i18n.T(r.Context(), "No releases found for this barcode.")))
		return
	}
	if len(out.Releases) > 1 {
		ui.Render(w, r, http.StatusOK, views.ScanConflictListPartial(out.Releases))
		return
	}

	ui.Render(w, r, http.StatusOK, views.ScanConfirmPartial(out.Releases[0]))
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	releaseIDStr := r.FormValue("release_id")
	if releaseIDStr == "" {
		ui.Render(w, r, http.StatusUnprocessableEntity, views.ScanErrorPartial(i18n.T(r.Context(), "Please select a release before confirming.")))
		return
	}

	releaseID, err := strconv.Atoi(releaseIDStr)
	if err != nil {
		http.Error(w, "invalid release_id", http.StatusBadRequest)
		return
	}

	out, err := h.AddToCollection.Execute(r.Context(), usecases.AddToCollectionInput{
		UserID:       ctxkey.UserIDFrom(r.Context()),
		ReleaseID:    releaseID,
		Acknowledged: r.FormValue("acknowledged") != "",
	})
	if err != nil {
		switch {
		case errors.Is(err, usecases.ErrDiscogsNotConnected):
			ui.Render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		case errors.Is(err, usecases.ErrCollectionCheckFailed):
			renderDiscogsError(w, r, err, i18n.T(r.Context(), "Could not check your collection. Please try again."))
		case errors.Is(err, usecases.ErrAddToCollectionFailed):
			renderDiscogsError(w, r, err, i18n.T(r.Context(), "Could not add release to collection. Please try again."))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	if out.Owned.Any() {
		ui.Render(w, r, http.StatusOK, views.ScanOwnedWarningPartial(releaseID, out.Owned))
		return
	}

	toast, _ := json.Marshal(map[string]any{"toast": map[string]string{
		"message": i18n.T(r.Context(), "Release added to your Discogs collection!"),
		"status":  "success",
	}})

	w.Header().Set("HX-Trigger", string(toast))
	w.Header().Set("HX-Trigger-After-Swap", `{"scan-success":true}`)
	ui.Render(w, r, http.StatusOK, views.ScanSuccessPartial())
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app/scanner", http.StatusFound)
}

func renderDiscogsError(w http.ResponseWriter, r *http.Request, err error, fallbackMessage string) {
	var rle *discogs.RateLimitError
	if errors.As(err, &rle) {
		ui.Render(w, r, http.StatusTooManyRequests, views.ScanErrorPartial(i18n.T(r.Context(), "Discogs rate limit reached. Please wait a moment and try again.")))
		return
	}

	var dae *discogs.DiscogsAuthError
	if errors.As(err, &dae) {
		ui.Render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		return
	}

	ui.Render(w, r, http.StatusBadGateway, views.ScanErrorPartial(fallbackMessage))
}
