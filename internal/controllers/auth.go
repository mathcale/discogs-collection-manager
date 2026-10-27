package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	authmodels "github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
	"github.com/mathcale/discogs-collection-manager/internal/ui"
	"github.com/mathcale/discogs-collection-manager/internal/views"
)

const (
	SessionCookieName      = "session_token"
	pendingLoginCookieName = "pending_login"
)

func (c *Controller) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if !c.SignupsEnabled {
		http.NotFound(w, r)
		return
	}

	ui.Render(w, r, http.StatusOK, views.RegisterPage(""))
}

func (c *Controller) Register(w http.ResponseWriter, r *http.Request) {
	if !c.SignupsEnabled {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := authmodels.NormalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")

	existing, err := c.Users.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		ui.Render(w, r, http.StatusUnprocessableEntity, views.RegisterPage("This email is already registered."))
		return
	}

	hash, err := authmodels.HashPassword(password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := c.Users.Create(r.Context(), email, hash); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

func (c *Controller) LoginPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, http.StatusOK, views.LoginPage("", c.SignupsEnabled))
}

func (c *Controller) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := authmodels.NormalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")
	stayLoggedIn := r.FormValue("stay_logged_in") == "1"

	user, err := c.Users.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user == nil || !user.CheckPassword(password) {
		ui.Render(w, r, http.StatusUnauthorized, views.LoginPage("Invalid email or password.", c.SignupsEnabled))
		return
	}

	if !c.TwoStepAuthEnabled {
		c.startSession(w, r, user.ID, stayLoggedIn)
		return
	}

	code, err := authmodels.GenerateOTPCode()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	codeHash := authmodels.HashToken(code)
	if _, err := c.OTPCodes.Create(r.Context(), user.ID, codeHash, time.Now().Add(authmodels.OTPExpiry)); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := c.Mailer.Send(user.Email, "Your login code", fmt.Sprintf("Your verification code is: <strong>%s</strong>. It expires in 10 minutes.", code)); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	persistentFlag := "0"
	if stayLoggedIn {
		persistentFlag = "1"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     pendingLoginCookieName,
		Value:    user.ID + ":" + persistentFlag,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(authmodels.OTPExpiry.Seconds()),
	})

	http.Redirect(w, r, "/auth/login/otp", http.StatusFound)
}

func (c *Controller) OTPPage(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(pendingLoginCookieName); err != nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	ui.Render(w, r, http.StatusOK, views.OTPPage(""))
}

func (c *Controller) VerifyOTP(w http.ResponseWriter, r *http.Request) {
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

	code := r.FormValue("code")

	otp, err := c.OTPCodes.FindPendingByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if otp == nil {
		ui.Render(w, r, http.StatusUnprocessableEntity, views.OTPPage("Invalid or expired code. Please log in again."))
		return
	}

	if otp.IsExpired() {
		ui.Render(w, r, http.StatusUnprocessableEntity, views.OTPPage("Code has expired. Please log in again."))
		return
	}

	if !otp.Matches(code) {
		if err := c.OTPCodes.RecordFailedAttempt(r.Context(), otp.ID, authmodels.MaxOTPFailedAttempts); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		ui.Render(w, r, http.StatusUnprocessableEntity, views.OTPPage("Invalid code. Please try again."))
		return
	}

	if err := c.OTPCodes.MarkUsed(r.Context(), otp.ID); err != nil {
		if errors.Is(err, authmodels.ErrOTPNotPending) {
			ui.Render(w, r, http.StatusUnprocessableEntity, views.OTPPage("Invalid or expired code. Please log in again."))
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     pendingLoginCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	c.startSession(w, r, userID, stayLoggedIn)
}

func (c *Controller) startSession(w http.ResponseWriter, r *http.Request, userID string, stayLoggedIn bool) {
	token, err := authmodels.GenerateToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(authmodels.SessionExpiry(stayLoggedIn))

	if _, err := c.Sessions.Create(r.Context(), userID, token, expiresAt, stayLoggedIn); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if stayLoggedIn {
		cookie.Expires = expiresAt
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/app/scanner", http.StatusFound)
}

func (c *Controller) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		c.Sessions.Delete(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

func (c *Controller) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, http.StatusOK, views.ForgotPasswordPage(false))
}

func (c *Controller) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := authmodels.NormalizeEmail(r.FormValue("email"))

	user, err := c.Users.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if user != nil {
		rawToken, err := authmodels.GenerateToken()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if err := c.ResetTokens.MarkAllUsedForUser(r.Context(), user.ID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		tokenHash := authmodels.HashToken(rawToken)
		if _, err := c.ResetTokens.Create(r.Context(), user.ID, tokenHash, time.Now().Add(authmodels.ResetTokenExpiry)); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resetLink := c.BaseURL + "/auth/reset-password?token=" + rawToken
		body := fmt.Sprintf("Click the link below to reset your password:<br><a href=\"%s\">%s</a><br>This link expires in 1 hour.", resetLink, resetLink)
		if err := c.Mailer.Send(user.Email, "Reset your password", body); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	ui.Render(w, r, http.StatusOK, views.ForgotPasswordPage(true))
}

func (c *Controller) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	rawToken := r.URL.Query().Get("token")
	if rawToken == "" {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	tokenHash := authmodels.HashToken(rawToken)
	record, err := c.ResetTokens.FindByTokenHash(r.Context(), tokenHash)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if record == nil || !record.IsUsable() {
		ui.Render(w, r, http.StatusOK, views.ResetPasswordPage("", "This reset link is invalid or has expired."))
		return
	}

	ui.Render(w, r, http.StatusOK, views.ResetPasswordPage(rawToken, ""))
}

func (c *Controller) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	rawToken := r.FormValue("token")
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	if password != passwordConfirm {
		ui.Render(w, r, http.StatusUnprocessableEntity, views.ResetPasswordPage(rawToken, "Passwords do not match."))
		return
	}

	tokenHash := authmodels.HashToken(rawToken)
	record, err := c.ResetTokens.FindByTokenHash(r.Context(), tokenHash)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if record == nil || !record.IsUsable() {
		ui.Render(w, r, http.StatusUnprocessableEntity, views.ResetPasswordPage("", "This reset link is invalid or has expired."))
		return
	}

	hash, err := authmodels.HashPassword(password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := c.Users.UpdatePassword(r.Context(), record.UserID, hash); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := c.ResetTokens.MarkAllUsedForUser(r.Context(), record.UserID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := c.Sessions.DeleteAllForUser(r.Context(), record.UserID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := c.OAuthTokens.DeleteByUserID(r.Context(), record.UserID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}
