package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/domain"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/views"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
)

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	releaseIDStr := r.FormValue("release_id")

	if releaseIDStr == "" {
		ui.Render(w, r, http.StatusUnprocessableEntity, views.ScanErrorPartial(
			i18n.T(
				r.Context(),
				"Please select a release before confirming.",
			),
		))

		return
	}

	releaseID, err := strconv.Atoi(releaseIDStr)
	if err != nil {
		http.Error(w, "invalid release_id", http.StatusBadRequest)
		return
	}

	input := usecases.AddToCollectionInput{
		UserID:       ctxkey.UserIDFrom(r.Context()),
		ReleaseID:    releaseID,
		Acknowledged: r.FormValue("acknowledged") != "",
	}

	out, err := h.AddToCollectionUseCase.Execute(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDiscogsNotConnected):
			ui.Render(w, r, http.StatusUnauthorized, views.ScanReconnectPartial())
		case errors.Is(err, domain.ErrCollectionCheckFailed):
			renderDiscogsError(w, r, err, i18n.T(
				r.Context(),
				"Could not check your collection. Please try again.",
			))
		case errors.Is(err, domain.ErrAddToCollectionFailed):
			renderDiscogsError(w, r, err, i18n.T(
				r.Context(),
				"Could not add release to collection. Please try again.",
			))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}

		return
	}

	if out.Owned.Any() {
		ui.Render(w, r, http.StatusOK, views.ScanOwnedWarningPartial(releaseID, out.Owned))
		return
	}

	toast, _ := json.Marshal(map[string]any{
		"toast": map[string]string{
			"message": i18n.T(r.Context(), "Release added to your Discogs collection!"),
			"status":  "success",
		},
	})

	w.Header().Set("HX-Trigger", string(toast))
	w.Header().Set("HX-Trigger-After-Swap", `{"scan-success":true}`)

	ui.Render(w, r, http.StatusOK, views.ScanSuccessPartial())
}
