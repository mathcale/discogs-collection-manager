package handler_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"

	"github.com/mathcale/discogs-collection-manager/internal/handler"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
	"github.com/mathcale/discogs-collection-manager/internal/repository"
	"github.com/mathcale/discogs-collection-manager/internal/web"
)

type MockMailer struct {
	Sent []SentEmail
}

type SentEmail struct {
	To, Subject, Body string
}

func (m *MockMailer) Send(to, subject, body string) error {
	m.Sent = append(m.Sent, SentEmail{To: to, Subject: subject, Body: body})
	return nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

type AuthSuite struct {
	suite.Suite
	db         *sql.DB
	dbPath     string
	users      repository.UserRepository
	sessions   repository.SessionRepository
	otpCodes   repository.OTPRepository
	resetTokens repository.PasswordResetTokenRepository
	mailer     *MockMailer
	router     http.Handler
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthSuite))
}

func (s *AuthSuite) SetupSuite() {
	f, err := os.CreateTemp("", "auth-test-*.db")
	s.Require().NoError(err)
	s.dbPath = f.Name()
	f.Close()

	s.Require().NoError(migrate.Up(s.dbPath, "../../migrations"))

	db, err := sql.Open("sqlite3", s.dbPath)
	s.Require().NoError(err)
	s.db = db

	s.users = repository.NewSQLiteUserRepository(db)
	s.sessions = repository.NewSQLiteSessionRepository(db)
	s.otpCodes = repository.NewSQLiteOTPRepository(db)
	s.resetTokens = repository.NewSQLitePasswordResetTokenRepository(db)
}

func (s *AuthSuite) SetupTest() {
	_, err := s.db.Exec(`DELETE FROM otp_codes`)
	s.Require().NoError(err)
	_, err = s.db.Exec(`DELETE FROM password_reset_tokens`)
	s.Require().NoError(err)
	_, err = s.db.Exec(`DELETE FROM sessions`)
	s.Require().NoError(err)
	_, err = s.db.Exec(`DELETE FROM users`)
	s.Require().NoError(err)

	s.mailer = &MockMailer{}

	authHandler := handler.NewAuthHandler(s.users, s.sessions, s.otpCodes, s.resetTokens, s.mailer, "http://localhost")

	r := chi.NewRouter()
	r.Get("/register", authHandler.RegisterPage)
	r.Post("/register", authHandler.Register)
	r.Get("/login", authHandler.LoginPage)
	r.Post("/login", authHandler.Login)
	r.Get("/login/otp", authHandler.OTPPage)
	r.Post("/login/otp", authHandler.VerifyOTP)
	r.Get("/forgot-password", authHandler.ForgotPasswordPage)
	r.Post("/forgot-password", authHandler.ForgotPassword)
	r.Get("/reset-password", authHandler.ResetPasswordPage)
	r.Post("/reset-password", authHandler.ResetPassword)

	r.Group(func(r chi.Router) {
		r.Use(web.RequireAuth(s.sessions))
		r.Post("/logout", authHandler.Logout)
		r.Get("/app", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	s.router = r
}

func (s *AuthSuite) TearDownSuite() {
	s.db.Close()
	os.Remove(s.dbPath)
}

func (s *AuthSuite) TestRegisterPage_GET() {
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Register")
}

func (s *AuthSuite) TestRegister_POST_Success() {
	form := url.Values{"email": {"test@example.com"}, "password": {"password123"}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/login", rr.Header().Get("Location"))

	user, err := s.users.FindByEmail(req.Context(), "test@example.com")
	s.Require().NoError(err)
	s.Require().NotNil(user)
	s.NotEqual("password123", user.PasswordHash)
}

func (s *AuthSuite) TestRegister_POST_DuplicateEmail() {
	form := url.Values{"email": {"dup@example.com"}, "password": {"password123"}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	s.Equal(http.StatusFound, rr.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	s.Equal(http.StatusUnprocessableEntity, rr2.Code)
	s.Contains(rr2.Body.String(), "already registered")
}

func (s *AuthSuite) TestLoginPage_GET() {
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Log in")
	s.Contains(rr.Body.String(), "Forgot password")
}

func (s *AuthSuite) TestLogin_POST_ValidCredentials_RedirectsToOTP() {
	s.registerUser("login@example.com", "password123")

	form := url.Values{"email": {"login@example.com"}, "password": {"password123"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/login/otp", rr.Header().Get("Location"))

	s.Require().Len(s.mailer.Sent, 1)
	s.Equal("login@example.com", s.mailer.Sent[0].To)
	s.Contains(s.mailer.Sent[0].Subject, "login code")

	cookies := rr.Result().Cookies()
	var pendingCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "pending_login" {
			pendingCookie = c
		}
	}
	s.Require().NotNil(pendingCookie)
	s.Contains(pendingCookie.Value, ":0")
}

func (s *AuthSuite) TestLogin_POST_StayLoggedIn_PersistFlagInCookie() {
	s.registerUser("persist@example.com", "password123")

	form := url.Values{"email": {"persist@example.com"}, "password": {"password123"}, "stay_logged_in": {"1"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)

	cookies := rr.Result().Cookies()
	var pendingCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "pending_login" {
			pendingCookie = c
		}
	}
	s.Require().NotNil(pendingCookie)
	s.Contains(pendingCookie.Value, ":1")
}

func (s *AuthSuite) TestLogin_POST_InvalidCredentials() {
	s.registerUser("badpass@example.com", "correctpass")

	form := url.Values{"email": {"badpass@example.com"}, "password": {"wrongpass"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnauthorized, rr.Code)
	s.Contains(rr.Body.String(), "Invalid email or password")
	s.Empty(s.mailer.Sent)
}

func (s *AuthSuite) TestOTPPage_NoPendingCookie_RedirectsToLogin() {
	req := httptest.NewRequest(http.MethodGet, "/login/otp", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/login", rr.Header().Get("Location"))
}

func (s *AuthSuite) TestOTPPage_WithPendingCookie_ShowsForm() {
	req := httptest.NewRequest(http.MethodGet, "/login/otp", nil)
	req.AddCookie(&http.Cookie{Name: "pending_login", Value: "some-user-id:0"})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Verify")
}

func (s *AuthSuite) TestVerifyOTP_ValidCode_IssuesSession() {
	user := s.registerUser("otp@example.com", "password123")

	code := "123456"
	codeHash := hashToken(code)
	_, err := s.otpCodes.Create(context.Background(), user.ID, codeHash, time.Now().Add(10*time.Minute))
	s.Require().NoError(err)

	form := url.Values{"code": {code}}
	req := httptest.NewRequest(http.MethodPost, "/login/otp", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "pending_login", Value: user.ID + ":0"})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/app/settings", rr.Header().Get("Location"))

	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			sessionCookie = c
		}
	}
	s.Require().NotNil(sessionCookie)
	s.NotEmpty(sessionCookie.Value)
}

func (s *AuthSuite) TestVerifyOTP_ValidCode_StayLoggedIn_PersistentSession() {
	user := s.registerUser("otppersist@example.com", "password123")

	code := "654321"
	codeHash := hashToken(code)
	_, err := s.otpCodes.Create(context.Background(), user.ID, codeHash, time.Now().Add(10*time.Minute))
	s.Require().NoError(err)

	form := url.Values{"code": {code}}
	req := httptest.NewRequest(http.MethodPost, "/login/otp", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "pending_login", Value: user.ID + ":1"})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)

	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			sessionCookie = c
		}
	}
	s.Require().NotNil(sessionCookie)
	s.True(sessionCookie.Expires.After(time.Now().Add(24 * time.Hour)))
}

func (s *AuthSuite) TestVerifyOTP_InvalidCode_Returns422() {
	user := s.registerUser("otpwrong@example.com", "password123")

	code := "999999"
	codeHash := hashToken(code)
	_, err := s.otpCodes.Create(context.Background(), user.ID, codeHash, time.Now().Add(10*time.Minute))
	s.Require().NoError(err)

	form := url.Values{"code": {"000000"}}
	req := httptest.NewRequest(http.MethodPost, "/login/otp", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "pending_login", Value: user.ID + ":0"})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Contains(rr.Body.String(), "Invalid code")
}

func (s *AuthSuite) TestVerifyOTP_ExpiredCode_Returns422() {
	user := s.registerUser("otpexpired@example.com", "password123")

	code := "777777"
	codeHash := hashToken(code)
	_, err := s.otpCodes.Create(context.Background(), user.ID, codeHash, time.Now().Add(-1*time.Minute))
	s.Require().NoError(err)

	form := url.Values{"code": {code}}
	req := httptest.NewRequest(http.MethodPost, "/login/otp", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "pending_login", Value: user.ID + ":0"})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Contains(rr.Body.String(), "expired")
}

func (s *AuthSuite) TestVerifyOTP_NoPendingCookie_RedirectsToLogin() {
	form := url.Values{"code": {"123456"}}
	req := httptest.NewRequest(http.MethodPost, "/login/otp", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/login", rr.Header().Get("Location"))
}

func (s *AuthSuite) TestProtectedRoute_Unauthenticated() {
	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/login", rr.Header().Get("Location"))
}

func (s *AuthSuite) TestProtectedRoute_WithValidSession() {
	user := s.registerUser("authed@example.com", "password123")
	token := s.createSession(user.ID, time.Now().Add(6*time.Hour), false)

	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
}

func (s *AuthSuite) TestProtectedRoute_ExpiredSession() {
	user := s.registerUser("expired@example.com", "password123")
	token := s.createSession(user.ID, time.Now().Add(-1*time.Hour), false)

	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/login", rr.Header().Get("Location"))
}

func (s *AuthSuite) TestLogout() {
	user := s.registerUser("logout@example.com", "password123")
	token := s.createSession(user.ID, time.Now().Add(6*time.Hour), false)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/login", rr.Header().Get("Location"))

	session, err := s.sessions.FindByToken(req.Context(), token)
	s.Require().NoError(err)
	s.Nil(session)
}

func (s *AuthSuite) TestForgotPasswordPage_GET() {
	req := httptest.NewRequest(http.MethodGet, "/forgot-password", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Forgot password")
}

func (s *AuthSuite) TestForgotPassword_POST_KnownEmail_SendsEmail() {
	s.registerUser("reset@example.com", "password123")

	form := url.Values{"email": {"reset@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "you'll receive")
	s.Require().Len(s.mailer.Sent, 1)
	s.Equal("reset@example.com", s.mailer.Sent[0].To)
	s.Contains(s.mailer.Sent[0].Body, "reset-password?token=")
}

func (s *AuthSuite) TestForgotPassword_POST_UnknownEmail_SilentSuccess() {
	form := url.Values{"email": {"nobody@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "you'll receive")
	s.Empty(s.mailer.Sent)
}

func (s *AuthSuite) TestResetPasswordPage_ValidToken_ShowsForm() {
	user := s.registerUser("resetpage@example.com", "password123")

	rawToken := "test-reset-token-abc123"
	tokenHash := hashToken(rawToken)
	_, err := s.resetTokens.Create(context.Background(), user.ID, tokenHash, time.Now().Add(1*time.Hour))
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/reset-password?token=%s", rawToken), nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Reset password")
	s.Contains(rr.Body.String(), rawToken)
}

func (s *AuthSuite) TestResetPasswordPage_InvalidToken_ShowsError() {
	req := httptest.NewRequest(http.MethodGet, "/reset-password?token=nonexistent", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "invalid or has expired")
}

func (s *AuthSuite) TestResetPassword_POST_Success_UpdatesPasswordAndInvalidatesSessions() {
	user := s.registerUser("resetpost@example.com", "oldpassword")
	existingToken := s.createSession(user.ID, time.Now().Add(6*time.Hour), false)

	rawToken := "valid-reset-token-xyz"
	tokenHash := hashToken(rawToken)
	_, err := s.resetTokens.Create(context.Background(), user.ID, tokenHash, time.Now().Add(1*time.Hour))
	s.Require().NoError(err)

	form := url.Values{
		"token":            {rawToken},
		"password":         {"newpassword123"},
		"password_confirm": {"newpassword123"},
	}
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/login", rr.Header().Get("Location"))

	session, err := s.sessions.FindByToken(req.Context(), existingToken)
	s.Require().NoError(err)
	s.Nil(session)

	updated, err := s.users.FindByEmail(req.Context(), "resetpost@example.com")
	s.Require().NoError(err)
	s.NotEqual(user.PasswordHash, updated.PasswordHash)
}

func (s *AuthSuite) TestResetPassword_POST_UsedToken_Rejected() {
	user := s.registerUser("resetused@example.com", "password123")

	rawToken := "used-reset-token"
	tokenHash := hashToken(rawToken)
	record, err := s.resetTokens.Create(context.Background(), user.ID, tokenHash, time.Now().Add(1*time.Hour))
	s.Require().NoError(err)
	s.Require().NoError(s.resetTokens.MarkUsed(context.Background(), record.ID))

	form := url.Values{
		"token":            {rawToken},
		"password":         {"newpassword"},
		"password_confirm": {"newpassword"},
	}
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Contains(rr.Body.String(), "invalid or has expired")
}

func (s *AuthSuite) TestResetPassword_POST_PasswordMismatch_Returns422() {
	user := s.registerUser("resetmismatch@example.com", "password123")

	rawToken := "mismatch-token"
	tokenHash := hashToken(rawToken)
	_, err := s.resetTokens.Create(context.Background(), user.ID, tokenHash, time.Now().Add(1*time.Hour))
	s.Require().NoError(err)

	form := url.Values{
		"token":            {rawToken},
		"password":         {"newpassword"},
		"password_confirm": {"differentpassword"},
	}
	req := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Contains(rr.Body.String(), "do not match")
}

func (s *AuthSuite) registerUser(email, password string) *repository.User {
	form := url.Values{"email": {email}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	s.Require().Equal(http.StatusFound, rr.Code)

	user, err := s.users.FindByEmail(req.Context(), email)
	s.Require().NoError(err)
	s.Require().NotNil(user)
	return user
}

func (s *AuthSuite) createSession(userID string, expiresAt time.Time, persistent bool) string {
	token := "test-token-" + userID
	_, err := s.sessions.Create(context.Background(), userID, token, expiresAt, persistent)
	s.Require().NoError(err)
	return token
}
