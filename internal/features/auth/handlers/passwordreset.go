package handlers

import (
	"errors"
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
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

	input := usecases.RequestPasswordResetInput{
		Email: r.FormValue("email"),
	}

	if _, err := h.RequestPasswordResetUseCase.Execute(r.Context(), input); err != nil {
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

	input := usecases.ValidateResetTokenInput{
		Token: rawToken,
	}

	if _, err := h.ValidateResetTokenUseCase.Execute(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrResetTokenInvalid) {
			ui.Render(w, r, http.StatusOK, views.ResetPasswordPage(
				"",
				i18n.T(r.Context(), "This reset link is invalid or has expired."),
			))

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

	input := usecases.ResetPasswordInput{
		Token:           rawToken,
		Password:        r.FormValue("password"),
		PasswordConfirm: r.FormValue("password_confirm"),
	}

	_, err := h.ResetPasswordUseCase.Execute(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPasswordMismatch):
			ui.Render(w, r, http.StatusUnprocessableEntity, views.ResetPasswordPage(
				rawToken,
				i18n.T(r.Context(), "Passwords do not match."),
			))
		case errors.Is(err, domain.ErrResetTokenInvalid):
			ui.Render(w, r, http.StatusUnprocessableEntity, views.ResetPasswordPage(
				"",
				i18n.T(r.Context(), "This reset link is invalid or has expired."),
			))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}

		return
	}

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}
