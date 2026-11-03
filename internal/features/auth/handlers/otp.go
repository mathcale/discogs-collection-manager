package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/views"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
)

func (h *Handler) OTPPage(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(pendingLoginCookieName); err != nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	ui.Render(w, r, http.StatusOK, views.OTPPage(""))
}

func (h *Handler) VerifyOTPSubmit(w http.ResponseWriter, r *http.Request) {
	pendingCookie, err := r.Cookie(pendingLoginCookieName)
	if err != nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	parts := strings.SplitN(pendingCookie.Value, ":", 2)
	if len(parts) != 2 {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	userID := parts[0]
	stayLoggedIn := parts[1] == "1"

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	input := usecases.VerifyOTPInput{
		UserID:       userID,
		Code:         r.FormValue("code"),
		StayLoggedIn: stayLoggedIn,
	}

	out, err := h.VerifyOTPUseCase.Execute(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOTPNotFound):
			ui.Render(w, r, http.StatusUnprocessableEntity, views.OTPPage(
				i18n.T(
					r.Context(),
					"Invalid or expired code. Please log in again.",
				),
			))
		case errors.Is(err, domain.ErrOTPExpired):
			ui.Render(w, r, http.StatusUnprocessableEntity, views.OTPPage(
				i18n.T(
					r.Context(),
					"Code has expired. Please log in again.",
				),
			))
		case errors.Is(err, domain.ErrOTPMismatch):
			ui.Render(w, r, http.StatusUnprocessableEntity, views.OTPPage(
				i18n.T(
					r.Context(),
					"Invalid code. Please try again.",
				),
			))
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}

		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     pendingLoginCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	startSession(w, r, out.SessionToken, out.SessionExpiresAt, stayLoggedIn)
}
