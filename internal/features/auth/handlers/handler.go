package handlers

import (
	"net/http"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/usecases"
)

const (
	SessionCookieName      = "session_token"
	pendingLoginCookieName = "pending_login"
)

type Handler struct {
	RegisterUseCase             *usecases.Register
	LoginUseCase                *usecases.Login
	VerifyOTPUseCase            *usecases.VerifyOTP
	LogoutUseCase               *usecases.Logout
	RequestPasswordResetUseCase *usecases.RequestPasswordReset
	ValidateResetTokenUseCase   *usecases.ValidateResetToken
	ResetPasswordUseCase        *usecases.ResetPassword
	AuthenticateSessionUseCase  *usecases.AuthenticateSession

	SignupsEnabled bool
}

func startSession(
	w http.ResponseWriter,
	r *http.Request,
	token string,
	expiresAt time.Time,
	persistent bool,
) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	if persistent {
		cookie.Expires = expiresAt
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/app/scanner", http.StatusFound)
}
