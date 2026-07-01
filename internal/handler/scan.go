package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/mathcale/discogs-collection-manager/internal/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
	"github.com/mathcale/discogs-collection-manager/internal/repository"
	"github.com/mathcale/discogs-collection-manager/internal/templates"
)

type ScanHandler struct {
	Discogs     discogs.DiscogsClient
	OAuthTokens repository.OAuthTokenRepository
}

func NewScanHandler(d discogs.DiscogsClient, oauthTokens repository.OAuthTokenRepository) *ScanHandler {
	return &ScanHandler{Discogs: d, OAuthTokens: oauthTokens}
}

func (h *ScanHandler) ScanPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ScannerPage().Render(r.Context(), w)
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

	releases, err := h.Discogs.SearchByBarcode(r.Context(), barcode)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var rle *discogs.RateLimitError
		if errors.As(err, &rle) {
			w.WriteHeader(http.StatusTooManyRequests)
			templates.ScanErrorPartial("Discogs rate limit reached. Please wait a moment and try again.").Render(r.Context(), w)
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		templates.ScanErrorPartial("Could not search Discogs. Please try again.").Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(releases) == 0 {
		templates.ScanErrorPartial("No releases found for this barcode.").Render(r.Context(), w)
		return
	}
	if len(releases) > 1 {
		templates.ScanErrorPartial("Multiple releases found. Please search manually.").Render(r.Context(), w)
		return
	}

	templates.ScanConfirmPartial(releases[0]).Render(r.Context(), w)
}

func (h *ScanHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	releaseIDStr := r.FormValue("release_id")
	if releaseIDStr == "" {
		http.Error(w, "release_id is required", http.StatusBadRequest)
		return
	}

	releaseID, err := strconv.Atoi(releaseIDStr)
	if err != nil {
		http.Error(w, "invalid release_id", http.StatusBadRequest)
		return
	}

	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	token, err := h.OAuthTokens.FindByUserID(r.Context(), userID)
	if err != nil || token == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.Discogs.AddToCollection(r.Context(), token.DiscogsUsername, releaseID); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var rle *discogs.RateLimitError
		if errors.As(err, &rle) {
			w.WriteHeader(http.StatusTooManyRequests)
			templates.ScanErrorPartial("Discogs rate limit reached. Please wait a moment and try again.").Render(r.Context(), w)
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		templates.ScanErrorPartial("Could not add release to collection. Please try again.").Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ScanSuccessPartial().Render(r.Context(), w)
}

func (h *ScanHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app/scanner", http.StatusFound)
}
