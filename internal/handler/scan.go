package handler

import (
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/templates"
)

type ScanHandler struct{}

func NewScanHandler() *ScanHandler {
	return &ScanHandler{}
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ScanResultPartial(barcode).Render(r.Context(), w)
}
