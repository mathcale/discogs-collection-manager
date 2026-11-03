package handlers

import "net/http"

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app/scanner", http.StatusFound)
}
