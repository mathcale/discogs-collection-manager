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

	"github.com/mathcale/discogs-collection-manager/internal/app/clients/mailer"
	"github.com/mathcale/discogs-collection-manager/internal/models"
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

type AuthHandler struct {
	Users       models.UserRepository
	Sessions    models.SessionRepository
	OTPCodes    models.OTPRepository
	ResetTokens models.PasswordResetTokenRepository
	Mailer      mailer.Mailer
	BaseURL     string
}

func NewAuthHandler(
	users models.UserRepository,
	sessions models.SessionRepository,
	otpCodes models.OTPRepository,
	resetTokens models.PasswordResetTokenRepository,
	m mailer.Mailer,
	baseURL string,
) *AuthHandler {
	return &AuthHandler{
		Users:       users,
		Sessions:    sessions,
		OTPCodes:    otpCodes,
		ResetTokens: resetTokens,
		Mailer:      m,
		BaseURL:     baseURL,
	}
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.RegisterPage("").Render(r.Context(), w)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	existing, err := h.Users.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		views.RegisterPage("This email is already registered.").Render(r.Context(), w)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := h.Users.Create(r.Context(), email, string(hash)); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.LoginPage("").Render(r.Context(), w)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	stayLoggedIn := r.FormValue("stay_logged_in") == "1"

	user, err := h.Users.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		views.LoginPage("Invalid email or password.").Render(r.Context(), w)
		return
	}

	code, err := generateOTPCode()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	codeHash := hashToken(code)
	if _, err := h.OTPCodes.Create(r.Context(), user.ID, codeHash, time.Now().Add(otpExpiry)); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.Mailer.Send(user.Email, "Your login code", fmt.Sprintf("Your verification code is: <strong>%s</strong>. It expires in 10 minutes.", code)); err != nil {
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

func (h *AuthHandler) OTPPage(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(pendingLoginCookieName); err != nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.OTPPage("").Render(r.Context(), w)
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
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

	otp, err := h.OTPCodes.FindPendingByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if otp == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		views.OTPPage("Invalid or expired code. Please log in again.").Render(r.Context(), w)
		return
	}

	if time.Now().After(otp.ExpiresAt) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		views.OTPPage("Code has expired. Please log in again.").Render(r.Context(), w)
		return
	}

	if hashToken(code) != otp.CodeHash {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		views.OTPPage("Invalid code. Please try again.").Render(r.Context(), w)
		return
	}

	if err := h.OTPCodes.MarkUsed(r.Context(), otp.ID); err != nil {
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

	if _, err := h.Sessions.Create(r.Context(), userID, token, expiresAt, stayLoggedIn); err != nil {
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

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		h.Sessions.Delete(r.Context(), cookie.Value)
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

func (h *AuthHandler) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.ForgotPasswordPage(false).Render(r.Context(), w)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")

	user, err := h.Users.FindByEmail(r.Context(), email)
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
		if _, err := h.ResetTokens.Create(r.Context(), user.ID, tokenHash, time.Now().Add(resetTokenExpiry)); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resetLink := h.BaseURL + "/auth/reset-password?token=" + rawToken
		body := fmt.Sprintf("Click the link below to reset your password:<br><a href=\"%s\">%s</a><br>This link expires in 1 hour.", resetLink, resetLink)
		if err := h.Mailer.Send(user.Email, "Reset your password", body); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.ForgotPasswordPage(true).Render(r.Context(), w)
}

func (h *AuthHandler) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	rawToken := r.URL.Query().Get("token")
	if rawToken == "" {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	tokenHash := hashToken(rawToken)
	record, err := h.ResetTokens.FindByTokenHash(r.Context(), tokenHash)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if record == nil || record.UsedAt != nil || time.Now().After(record.ExpiresAt) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		views.ResetPasswordPage("", "This reset link is invalid or has expired.").Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.ResetPasswordPage(rawToken, "").Render(r.Context(), w)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	rawToken := r.FormValue("token")
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	if password != passwordConfirm {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		views.ResetPasswordPage(rawToken, "Passwords do not match.").Render(r.Context(), w)
		return
	}

	tokenHash := hashToken(rawToken)
	record, err := h.ResetTokens.FindByTokenHash(r.Context(), tokenHash)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if record == nil || record.UsedAt != nil || time.Now().After(record.ExpiresAt) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		views.ResetPasswordPage("", "This reset link is invalid or has expired.").Render(r.Context(), w)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.Users.UpdatePassword(r.Context(), record.UserID, string(hash)); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.ResetTokens.MarkUsed(r.Context(), record.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.Sessions.DeleteAllForUser(r.Context(), record.UserID); err != nil {
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
