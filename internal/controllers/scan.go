package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/mathcale/discogs-collection-manager/internal/app/clients/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/models"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/views"
)

type ScanHandler struct {
	Discogs     discogs.DiscogsClient
	OAuthTokens models.OAuthTokenRepository
}

func NewScanHandler(d discogs.DiscogsClient, oauthTokens models.OAuthTokenRepository) *ScanHandler {
	return &ScanHandler{Discogs: d, OAuthTokens: oauthTokens}
}

func (h *ScanHandler) ScanPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.ScannerPage().Render(r.Context(), w)
}

func (h *ScanHandler) Scan(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	barcode := r.FormValue("barcode")
	if barcode == "" {
		http.Error(w, "barcode is required", http.StatusBadRequest)
		return
	}

	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	token, err := h.OAuthTokens.FindByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if token == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		views.ScanReconnectPartial().Render(r.Context(), w)
		return
	}

	releases, err := h.Discogs.SearchByBarcode(r.Context(), barcode)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var rle *discogs.RateLimitError
		if errors.As(err, &rle) {
			w.WriteHeader(http.StatusTooManyRequests)
			views.ScanErrorPartial("Discogs rate limit reached. Please wait a moment and try again.").Render(r.Context(), w)
			return
		}
		var dae *discogs.DiscogsAuthError
		if errors.As(err, &dae) {
			w.WriteHeader(http.StatusUnauthorized)
			views.ScanReconnectPartial().Render(r.Context(), w)
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		views.ScanErrorPartial("Could not search Discogs. Please try again.").Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(releases) == 0 {
		views.ScanErrorPartial("No releases found for this barcode.").Render(r.Context(), w)
		return
	}
	if len(releases) > 1 {
		views.ScanConflictListPartial(releases).Render(r.Context(), w)
		return
	}

	views.ScanConfirmPartial(releases[0]).Render(r.Context(), w)
}

func (h *ScanHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	releaseIDStr := r.FormValue("release_id")
	if releaseIDStr == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		views.ScanErrorPartial("Please select a release before confirming.").Render(r.Context(), w)
		return
	}

	releaseID, err := strconv.Atoi(releaseIDStr)
	if err != nil {
		http.Error(w, "invalid release_id", http.StatusBadRequest)
		return
	}

	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	token, err := h.OAuthTokens.FindByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if token == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		views.ScanReconnectPartial().Render(r.Context(), w)
		return
	}

	if err := h.Discogs.AddToCollection(r.Context(), token.DiscogsUsername, releaseID); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var rle *discogs.RateLimitError
		if errors.As(err, &rle) {
			w.WriteHeader(http.StatusTooManyRequests)
			views.ScanErrorPartial("Discogs rate limit reached. Please wait a moment and try again.").Render(r.Context(), w)
			return
		}
		var dae *discogs.DiscogsAuthError
		if errors.As(err, &dae) {
			w.WriteHeader(http.StatusUnauthorized)
			views.ScanReconnectPartial().Render(r.Context(), w)
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		views.ScanErrorPartial("Could not add release to collection. Please try again.").Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("HX-Trigger", "scan-success")
	views.ScanSuccessPartial().Render(r.Context(), w)
}

func (h *ScanHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app/scanner", http.StatusFound)
}
