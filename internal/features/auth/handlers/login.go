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

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, http.StatusOK, views.LoginPage("", "", h.SignupsEnabled))
}

func (h *Handler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)

		return
	}

	stayLoggedIn := r.FormValue("stay_logged_in") == "1"

	input := usecases.LoginInput{
		Email:        r.FormValue("email"),
		Password:     r.FormValue("password"),
		StayLoggedIn: stayLoggedIn,
	}

	out, err := h.LoginUseCase.Execute(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			ui.Render(w, r, http.StatusUnauthorized, views.LoginPage(
				i18n.T(r.Context(), "Invalid email or password."),
				r.FormValue("email"),
				h.SignupsEnabled,
			))

			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	if !out.OTPRequired {
		startSession(w, r, out.SessionToken, out.SessionExpiresAt, stayLoggedIn)

		return
	}

	persistentFlag := "0"

	if stayLoggedIn {
		persistentFlag = "1"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     pendingLoginCookieName,
		Value:    out.UserID + ":" + persistentFlag,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(out.OTPExpiresIn.Seconds()),
	})

	http.Redirect(w, r, "/auth/login/otp", http.StatusFound)
}
