package handlers

import (
	"errors"
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/views"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
)

func (h *Handler) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, http.StatusOK, views.ForgotPasswordPage(false))
}

func (h *Handler) ForgotPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if _, err := h.RequestPasswordReset.Execute(r.Context(), usecases.RequestPasswordResetInput{
		Email: r.FormValue("email"),
	}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, http.StatusOK, views.ForgotPasswordPage(true))
}

func (h *Handler) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	rawToken := r.URL.Query().Get("token")
	if rawToken == "" {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	if _, err := h.ValidateResetToken.Execute(r.Context(), usecases.ValidateResetTokenInput{Token: rawToken}); err != nil {
		if errors.Is(err, usecases.ErrResetTokenInvalid) {
			ui.Render(w, r, http.StatusOK, views.ResetPasswordPage("", i18n.T(r.Context(), "This reset link is invalid or has expired.")))
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, http.StatusOK, views.ResetPasswordPage(rawToken, ""))
}

func (h *Handler) ResetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	rawToken := r.FormValue("token")

	_, err := h.ResetPassword.Execute(r.Context(), usecases.ResetPasswordInput{
		Token:           rawToken,
		Password:        r.FormValue("password"),
		PasswordConfirm: r.FormValue("password_confirm"),
	})
	if err != nil {
		switch {
		case errors.Is(err, usecases.ErrPasswordMismatch):
			ui.Render(w, r, http.StatusUnprocessableEntity, views.ResetPasswordPage(rawToken, i18n.T(r.Context(), "Passwords do not match.")))
		case errors.Is(err, usecases.ErrResetTokenInvalid):
			ui.Render(w, r, http.StatusUnprocessableEntity, views.ResetPasswordPage("", i18n.T(r.Context(), "This reset link is invalid or has expired.")))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}
