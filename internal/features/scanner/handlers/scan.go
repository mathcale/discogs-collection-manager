package handlers

import (
	"errors"
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/domain"
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

	input := usecases.LookupBarcodeInput{
		UserID:  ctxkey.UserIDFrom(r.Context()),
		Barcode: barcode,
	}

	out, err := h.LookupBarcodeUseCase.Execute(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDiscogsNotConnected):
			ui.Render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		case errors.Is(err, domain.ErrSearchFailed):
			renderDiscogsError(w, r, err, i18n.T(
				r.Context(),
				"Could not search Discogs. Please try again.",
			))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}

		return
	}

	if len(out.Releases) == 0 {
		ui.Render(w, r, http.StatusOK, views.ScanErrorPartial(
			i18n.T(r.Context(),
				"No releases found for this barcode.",
			),
		))

		return
	}

	if len(out.Releases) > 1 {
		ui.Render(w, r, http.StatusOK, views.ScanConflictListPartial(out.Releases))
		return
	}

	ui.Render(w, r, http.StatusOK, views.ScanConfirmPartial(out.Releases[0]))
}

func renderDiscogsError(w http.ResponseWriter, r *http.Request, err error, fallbackMessage string) {
	var rle *discogs.RateLimitError
	if errors.As(err, &rle) {
		ui.Render(w, r, http.StatusTooManyRequests, views.ScanErrorPartial(
			i18n.T(
				r.Context(),
				"Discogs rate limit reached. Please wait a moment and try again.",
			),
		))

		return
	}

	var dae *discogs.DiscogsAuthError
	if errors.As(err, &dae) {
		ui.Render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		return
	}

	ui.Render(w, r, http.StatusBadGateway, views.ScanErrorPartial(fallbackMessage))
}
