package controllers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mathcale/discogs-collection-manager/internal/views"
)

const (
	sessionCookieName      = "session_token"
	pendingLoginCookieName = "pending_login"
	shortSessionExpiry     = 6 * time.Hour
	longSessionExpiry      = 30 * 24 * time.Hour
	otpExpiry              = 10 * time.Minute
	resetTokenExpiry       = 1 * time.Hour
)

func (c *Controller) RegisterPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, views.RegisterPage(""))
}

func (c *Controller) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	existing, err := c.Users.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		render(w, r, http.StatusUnprocessableEntity, views.RegisterPage("This email is already registered."))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := c.Users.Create(r.Context(), email, string(hash)); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

func (c *Controller) LoginPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, views.LoginPage(""))
}

func (c *Controller) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	stayLoggedIn := r.FormValue("stay_logged_in") == "1"

	user, err := c.Users.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		render(w, r, http.StatusUnauthorized, views.LoginPage("Invalid email or password."))
		return
	}

	code, err := generateOTPCode()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	codeHash := hashToken(code)
	if _, err := c.OTPCodes.Create(r.Context(), user.ID, codeHash, time.Now().Add(otpExpiry)); err != nil {
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
		MaxAge:   int(otpExpiry.Seconds()),
	})

	http.Redirect(w, r, "/auth/login/otp", http.StatusFound)
}

func (c *Controller) OTPPage(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(pendingLoginCookieName); err != nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	render(w, r, http.StatusOK, views.OTPPage(""))
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
		render(w, r, http.StatusUnprocessableEntity, views.OTPPage("Invalid or expired code. Please log in again."))
		return
	}

	if time.Now().After(otp.ExpiresAt) {
		render(w, r, http.StatusUnprocessableEntity, views.OTPPage("Code has expired. Please log in again."))
		return
	}

	if hashToken(code) != otp.CodeHash {
		render(w, r, http.StatusUnprocessableEntity, views.OTPPage("Invalid code. Please try again."))
		return
	}

	if err := c.OTPCodes.MarkUsed(r.Context(), otp.ID); err != nil {
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

	token, err := generateToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	expiry := shortSessionExpiry
	if stayLoggedIn {
		expiry = longSessionExpiry
	}
	expiresAt := time.Now().Add(expiry)

	if _, err := c.Sessions.Create(r.Context(), userID, token, expiresAt, stayLoggedIn); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     sessionCookieName,
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
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		c.Sessions.Delete(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

func (c *Controller) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, views.ForgotPasswordPage(false))
}

func (c *Controller) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")

	user, err := c.Users.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if user != nil {
		rawToken, err := generateToken()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		tokenHash := hashToken(rawToken)
		if _, err := c.ResetTokens.Create(r.Context(), user.ID, tokenHash, time.Now().Add(resetTokenExpiry)); err != nil {
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

	render(w, r, http.StatusOK, views.ForgotPasswordPage(true))
}

func (c *Controller) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	rawToken := r.URL.Query().Get("token")
	if rawToken == "" {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	tokenHash := hashToken(rawToken)
	record, err := c.ResetTokens.FindByTokenHash(r.Context(), tokenHash)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if record == nil || record.UsedAt != nil || time.Now().After(record.ExpiresAt) {
		render(w, r, http.StatusOK, views.ResetPasswordPage("", "This reset link is invalid or has expired."))
		return
	}

	render(w, r, http.StatusOK, views.ResetPasswordPage(rawToken, ""))
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
		render(w, r, http.StatusUnprocessableEntity, views.ResetPasswordPage(rawToken, "Passwords do not match."))
		return
	}

	tokenHash := hashToken(rawToken)
	record, err := c.ResetTokens.FindByTokenHash(r.Context(), tokenHash)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if record == nil || record.UsedAt != nil || time.Now().After(record.ExpiresAt) {
		render(w, r, http.StatusUnprocessableEntity, views.ResetPasswordPage("", "This reset link is invalid or has expired."))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := c.Users.UpdatePassword(r.Context(), record.UserID, string(hash)); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := c.ResetTokens.MarkUsed(r.Context(), record.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := c.Sessions.DeleteAllForUser(r.Context(), record.UserID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
