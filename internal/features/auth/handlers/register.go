package handlers

import (
	"errors"
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/views"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
)

func (h *Handler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if !h.SignupsEnabled {
		http.NotFound(w, r)
		return
	}

	ui.Render(w, r, http.StatusOK, views.RegisterPage(""))
}

func (h *Handler) RegisterSubmit(w http.ResponseWriter, r *http.Request) {
	if !h.SignupsEnabled {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	_, err := h.Register.Execute(r.Context(), usecases.RegisterInput{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	})
	if err != nil {
		if errors.Is(err, usecases.ErrEmailTaken) {
			ui.Render(w, r, http.StatusUnprocessableEntity, views.RegisterPage(i18n.T(r.Context(), "This email is already registered.")))
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}
