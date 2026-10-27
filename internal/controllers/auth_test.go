package controllers_test

import (
	"context"
	"database/sql"
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

	"github.com/mathcale/discogs-collection-manager/internal/controllers"
	"github.com/mathcale/discogs-collection-manager/internal/core/web"
	"github.com/mathcale/discogs-collection-manager/internal/models"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
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

type AuthSuite struct {
	suite.Suite
	db          *sql.DB
	dbPath      string
	users       models.UserRepository
	sessions    models.SessionRepository
	otpCodes    models.OTPRepository
	resetTokens models.PasswordResetTokenRepository
	oauthTokens models.OAuthTokenRepository
	mailer      *MockMailer
	router      http.Handler
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

	s.users = models.NewSQLiteUserRepository(db)
	s.sessions = models.NewSQLiteSessionRepository(db)
	s.otpCodes = models.NewSQLiteOTPRepository(db)
	s.resetTokens = models.NewSQLitePasswordResetTokenRepository(db)
	s.oauthTokens = models.NewSQLiteOAuthTokenRepository(db)
}

func (s *AuthSuite) SetupTest() {
	_, err := s.db.Exec(`DELETE FROM otp_codes`)
	s.Require().NoError(err)
	_, err = s.db.Exec(`DELETE FROM password_reset_tokens`)
	s.Require().NoError(err)
	_, err = s.db.Exec(`DELETE FROM oauth_tokens`)
	s.Require().NoError(err)
	_, err = s.db.Exec(`DELETE FROM sessions`)
	s.Require().NoError(err)
	_, err = s.db.Exec(`DELETE FROM users`)
	s.Require().NoError(err)

	s.mailer = &MockMailer{}

	s.router = s.buildRouter(true, true)
}

func (s *AuthSuite) buildRouter(twoStepAuthEnabled, signupsEnabled bool) http.Handler {
	controller := &controllers.Controller{
		Users:       s.users,
		Sessions:    s.sessions,
		OTPCodes:    s.otpCodes,
		ResetTokens: s.resetTokens,
		OAuthTokens: s.oauthTokens,
		Mailer:      s.mailer,
		BaseURL:     "http://localhost",

		TwoStepAuthEnabled: twoStepAuthEnabled,
		SignupsEnabled:     signupsEnabled,
	}

	r := chi.NewRouter()
	r.Route("/auth", func(r chi.Router) {
		r.Get("/register", controller.RegisterPage)
		r.Post("/register", controller.Register)
		r.Get("/login", controller.LoginPage)
		r.Post("/login", controller.Login)
		r.Get("/login/otp", controller.OTPPage)
		r.Post("/login/otp", controller.VerifyOTP)
		r.Get("/forgot-password", controller.ForgotPasswordPage)
		r.Post("/forgot-password", controller.ForgotPassword)
		r.Get("/reset-password", controller.ResetPasswordPage)
		r.Post("/reset-password", controller.ResetPassword)

		r.Group(func(r chi.Router) {
			r.Use(web.RequireAuth(s.sessions))
			r.Post("/logout", controller.Logout)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(web.RequireAuth(s.sessions))
		r.Get("/app", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	return r
}

func (s *AuthSuite) TearDownSuite() {
	s.db.Close()
	os.Remove(s.dbPath)
}

func (s *AuthSuite) TestRegisterPage_GET() {
	req := httptest.NewRequest(http.MethodGet, "/auth/register", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Register")
}

func (s *AuthSuite) TestRegister_POST_Success() {
	form := url.Values{"email": {"test@example.com"}, "password": {"password123"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/auth/login", rr.Header().Get("Location"))

	user, err := s.users.FindByEmail(req.Context(), "test@example.com")
	s.Require().NoError(err)
	s.Require().NotNil(user)
	s.NotEqual("password123", user.PasswordHash)
}

func (s *AuthSuite) TestRegister_POST_DuplicateEmail() {
	form := url.Values{"email": {"dup@example.com"}, "password": {"password123"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	s.Equal(http.StatusFound, rr.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(form.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr2 := httptest.NewRecorder()
	s.router.ServeHTTP(rr2, req2)

	s.Equal(http.StatusUnprocessableEntity, rr2.Code)
	s.Contains(rr2.Body.String(), "already registered")
}

func (s *AuthSuite) TestLoginPage_GET() {
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Log in")
	s.Contains(rr.Body.String(), "Forgot password")
}

func (s *AuthSuite) TestLogin_POST_ValidCredentials_RedirectsToOTP() {
	s.registerUser("login@example.com", "password123")

	form := url.Values{"email": {"login@example.com"}, "password": {"password123"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/auth/login/otp", rr.Header().Get("Location"))

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
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
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
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnauthorized, rr.Code)
	s.Contains(rr.Body.String(), "Invalid email or password")
	s.Empty(s.mailer.Sent)
}

func (s *AuthSuite) TestOTPPage_NoPendingCookie_RedirectsToLogin() {
	req := httptest.NewRequest(http.MethodGet, "/auth/login/otp", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/auth/login", rr.Header().Get("Location"))
}

func (s *AuthSuite) TestOTPPage_WithPendingCookie_ShowsForm() {
	req := httptest.NewRequest(http.MethodGet, "/auth/login/otp", nil)
	req.AddCookie(&http.Cookie{Name: "pending_login", Value: "some-user-id:0"})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Verify")
}

func (s *AuthSuite) TestVerifyOTP_ValidCode_IssuesSession() {
	user := s.registerUser("otp@example.com", "password123")

	code := "123456"
	codeHash := models.HashToken(code)
	_, err := s.otpCodes.Create(context.Background(), user.ID, codeHash, time.Now().Add(10*time.Minute))
	s.Require().NoError(err)

	form := url.Values{"code": {code}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login/otp", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "pending_login", Value: user.ID + ":0"})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/app/scanner", rr.Header().Get("Location"))

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
	codeHash := models.HashToken(code)
	_, err := s.otpCodes.Create(context.Background(), user.ID, codeHash, time.Now().Add(10*time.Minute))
	s.Require().NoError(err)

	form := url.Values{"code": {code}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login/otp", strings.NewReader(form.Encode()))
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
	codeHash := models.HashToken(code)
	_, err := s.otpCodes.Create(context.Background(), user.ID, codeHash, time.Now().Add(10*time.Minute))
	s.Require().NoError(err)

	form := url.Values{"code": {"000000"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login/otp", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "pending_login", Value: user.ID + ":0"})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Contains(rr.Body.String(), "Invalid code")
}

func (s *AuthSuite) TestVerifyOTP_RepeatedWrongCodes_InvalidatesCode() {
	user := s.registerUser("otpbrute@example.com", "password123")

	code := "424242"
	_, err := s.otpCodes.Create(context.Background(), user.ID, models.HashToken(code), time.Now().Add(10*time.Minute))
	s.Require().NoError(err)

	submit := func(attempt string) *httptest.ResponseRecorder {
		form := url.Values{"code": {attempt}}
		req := httptest.NewRequest(http.MethodPost, "/auth/login/otp", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "pending_login", Value: user.ID + ":0"})
		rr := httptest.NewRecorder()
		s.router.ServeHTTP(rr, req)
		return rr
	}

	for range 5 {
		rr := submit("000000")
		s.Equal(http.StatusUnprocessableEntity, rr.Code)
	}

	rr := submit(code)
	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Nil(s.cookieNamed(rr, "session_token"))

	pending, err := s.otpCodes.FindPendingByUserID(context.Background(), user.ID)
	s.Require().NoError(err)
	s.Nil(pending)
}

func (s *AuthSuite) TestVerifyOTP_ExpiredCode_Returns422() {
	user := s.registerUser("otpexpired@example.com", "password123")

	code := "777777"
	codeHash := models.HashToken(code)
	_, err := s.otpCodes.Create(context.Background(), user.ID, codeHash, time.Now().Add(-1*time.Minute))
	s.Require().NoError(err)

	form := url.Values{"code": {code}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login/otp", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "pending_login", Value: user.ID + ":0"})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Contains(rr.Body.String(), "expired")
}

func (s *AuthSuite) TestVerifyOTP_NoPendingCookie_RedirectsToLogin() {
	form := url.Values{"code": {"123456"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login/otp", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/auth/login", rr.Header().Get("Location"))
}

func (s *AuthSuite) TestProtectedRoute_Unauthenticated() {
	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/auth/login", rr.Header().Get("Location"))
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
	s.Equal("/auth/login", rr.Header().Get("Location"))
}

func (s *AuthSuite) TestLogout() {
	user := s.registerUser("logout@example.com", "password123")
	token := s.createSession(user.ID, time.Now().Add(6*time.Hour), false)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/auth/login", rr.Header().Get("Location"))

	session, err := s.sessions.FindByToken(req.Context(), token)
	s.Require().NoError(err)
	s.Nil(session)
}

func (s *AuthSuite) TestForgotPasswordPage_GET() {
	req := httptest.NewRequest(http.MethodGet, "/auth/forgot-password", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Forgot password")
}

func (s *AuthSuite) TestForgotPassword_POST_KnownEmail_SendsEmail() {
	s.registerUser("reset@example.com", "password123")

	form := url.Values{"email": {"reset@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "you'll receive")
	s.Require().Len(s.mailer.Sent, 1)
	s.Equal("reset@example.com", s.mailer.Sent[0].To)
	s.Contains(s.mailer.Sent[0].Body, "auth/reset-password?token=")
}

func (s *AuthSuite) TestForgotPassword_POST_UnknownEmail_SilentSuccess() {
	form := url.Values{"email": {"nobody@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(form.Encode()))
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
	tokenHash := models.HashToken(rawToken)
	_, err := s.resetTokens.Create(context.Background(), user.ID, tokenHash, time.Now().Add(1*time.Hour))
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/auth/reset-password?token=%s", rawToken), nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Reset password")
	s.Contains(rr.Body.String(), rawToken)
}

func (s *AuthSuite) TestResetPasswordPage_InvalidToken_ShowsError() {
	req := httptest.NewRequest(http.MethodGet, "/auth/reset-password?token=nonexistent", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "invalid or has expired")
}

func (s *AuthSuite) TestResetPassword_POST_Success_UpdatesPasswordAndInvalidatesSessions() {
	user := s.registerUser("resetpost@example.com", "oldpassword")
	existingToken := s.createSession(user.ID, time.Now().Add(6*time.Hour), false)

	rawToken := "valid-reset-token-xyz"
	tokenHash := models.HashToken(rawToken)
	_, err := s.resetTokens.Create(context.Background(), user.ID, tokenHash, time.Now().Add(1*time.Hour))
	s.Require().NoError(err)

	form := url.Values{
		"token":            {rawToken},
		"password":         {"newpassword123"},
		"password_confirm": {"newpassword123"},
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/auth/login", rr.Header().Get("Location"))

	session, err := s.sessions.FindByToken(req.Context(), existingToken)
	s.Require().NoError(err)
	s.Nil(session)

	updated, err := s.users.FindByEmail(req.Context(), "resetpost@example.com")
	s.Require().NoError(err)
	s.NotEqual(user.PasswordHash, updated.PasswordHash)
}

func (s *AuthSuite) TestResetPassword_POST_Success_InvalidatesOtherUnusedTokens() {
	user := s.registerUser("resetother@example.com", "oldpassword")

	_, err := s.resetTokens.Create(context.Background(), user.ID, models.HashToken("first-token"), time.Now().Add(1*time.Hour))
	s.Require().NoError(err)
	_, err = s.resetTokens.Create(context.Background(), user.ID, models.HashToken("second-token"), time.Now().Add(1*time.Hour))
	s.Require().NoError(err)

	rr := s.postForm(s.router, "/auth/reset-password", url.Values{
		"token":            {"first-token"},
		"password":         {"newpassword123"},
		"password_confirm": {"newpassword123"},
	})
	s.Equal(http.StatusFound, rr.Code)

	rr = s.postForm(s.router, "/auth/reset-password", url.Values{
		"token":            {"second-token"},
		"password":         {"attackerpassword"},
		"password_confirm": {"attackerpassword"},
	})
	s.Equal(http.StatusUnprocessableEntity, rr.Code)
}

func (s *AuthSuite) TestForgotPassword_POST_InvalidatesPreviousToken() {
	user := s.registerUser("resetagain@example.com", "password123")

	_, err := s.resetTokens.Create(context.Background(), user.ID, models.HashToken("old-token"), time.Now().Add(1*time.Hour))
	s.Require().NoError(err)

	rr := s.postForm(s.router, "/auth/forgot-password", url.Values{"email": {"resetagain@example.com"}})
	s.Equal(http.StatusOK, rr.Code)

	rr = s.postForm(s.router, "/auth/reset-password", url.Values{
		"token":            {"old-token"},
		"password":         {"newpassword123"},
		"password_confirm": {"newpassword123"},
	})
	s.Equal(http.StatusUnprocessableEntity, rr.Code)
}

func (s *AuthSuite) TestRegister_POST_EmailCaseInsensitive_DuplicateRejected() {
	s.registerUser("case@example.com", "password123")

	rr := s.postForm(s.router, "/auth/register", url.Values{"email": {"  CASE@Example.com "}, "password": {"password123"}})

	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Contains(rr.Body.String(), "already registered")
}

func (s *AuthSuite) TestRegister_POST_StoresNormalizedEmail() {
	rr := s.postForm(s.router, "/auth/register", url.Values{"email": {" Mixed@Example.COM "}, "password": {"password123"}})
	s.Equal(http.StatusFound, rr.Code)

	user, err := s.users.FindByEmail(context.Background(), "mixed@example.com")
	s.Require().NoError(err)
	s.Require().NotNil(user)
	s.Equal("mixed@example.com", user.Email)
}

func (s *AuthSuite) TestLogin_POST_EmailCaseInsensitive() {
	s.registerUser("logincase@example.com", "password123")
	router := s.buildRouter(false, true)

	rr := s.postForm(router, "/auth/login", url.Values{"email": {"LoginCase@Example.com"}, "password": {"password123"}})

	s.Equal(http.StatusFound, rr.Code)
	s.NotNil(s.cookieNamed(rr, "session_token"))
}

func (s *AuthSuite) TestForgotPassword_POST_EmailCaseInsensitive() {
	s.registerUser("forgotcase@example.com", "password123")

	rr := s.postForm(s.router, "/auth/forgot-password", url.Values{"email": {"ForgotCase@Example.com"}})

	s.Equal(http.StatusOK, rr.Code)
	s.Require().Len(s.mailer.Sent, 1)
	s.Equal("forgotcase@example.com", s.mailer.Sent[0].To)
}

func (s *AuthSuite) TestResetPassword_POST_Success_DeletesOAuthToken() {
	user := s.registerUser("resetoauth@example.com", "oldpassword")
	_, err := s.oauthTokens.Upsert(context.Background(), user.ID, "access", "secret", "attacker")
	s.Require().NoError(err)

	_, err = s.resetTokens.Create(context.Background(), user.ID, models.HashToken("oauth-reset-token"), time.Now().Add(1*time.Hour))
	s.Require().NoError(err)

	rr := s.postForm(s.router, "/auth/reset-password", url.Values{
		"token":            {"oauth-reset-token"},
		"password":         {"newpassword123"},
		"password_confirm": {"newpassword123"},
	})
	s.Equal(http.StatusFound, rr.Code)

	token, err := s.oauthTokens.FindByUserID(context.Background(), user.ID)
	s.Require().NoError(err)
	s.Nil(token)
}

func (s *AuthSuite) TestResetPassword_POST_UsedToken_Rejected() {
	user := s.registerUser("resetused@example.com", "password123")

	rawToken := "used-reset-token"
	tokenHash := models.HashToken(rawToken)
	record, err := s.resetTokens.Create(context.Background(), user.ID, tokenHash, time.Now().Add(1*time.Hour))
	s.Require().NoError(err)
	s.Require().NoError(s.resetTokens.MarkUsed(context.Background(), record.ID))

	form := url.Values{
		"token":            {rawToken},
		"password":         {"newpassword"},
		"password_confirm": {"newpassword"},
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Contains(rr.Body.String(), "invalid or has expired")
}

func (s *AuthSuite) TestResetPassword_POST_PasswordMismatch_Returns422() {
	user := s.registerUser("resetmismatch@example.com", "password123")

	rawToken := "mismatch-token"
	tokenHash := models.HashToken(rawToken)
	_, err := s.resetTokens.Create(context.Background(), user.ID, tokenHash, time.Now().Add(1*time.Hour))
	s.Require().NoError(err)

	form := url.Values{
		"token":            {rawToken},
		"password":         {"newpassword"},
		"password_confirm": {"differentpassword"},
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Contains(rr.Body.String(), "do not match")
}

func (s *AuthSuite) postForm(router http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func (s *AuthSuite) cookieNamed(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (s *AuthSuite) TestLogin_TwoStepDisabled_StartsSessionDirectly() {
	s.registerUser("direct@example.com", "password123")
	router := s.buildRouter(false, true)

	rr := s.postForm(router, "/auth/login", url.Values{"email": {"direct@example.com"}, "password": {"password123"}})

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/app/scanner", rr.Header().Get("Location"))
	session := s.cookieNamed(rr, "session_token")
	s.Require().NotNil(session)
	s.True(session.Expires.IsZero())
	s.Nil(s.cookieNamed(rr, "pending_login"))
	s.Empty(s.mailer.Sent)

	var otpCount int
	s.Require().NoError(s.db.QueryRow(`SELECT COUNT(*) FROM otp_codes`).Scan(&otpCount))
	s.Zero(otpCount)
}

func (s *AuthSuite) TestLogin_TwoStepDisabled_StayLoggedIn_SetsLongExpiry() {
	s.registerUser("longlived@example.com", "password123")
	router := s.buildRouter(false, true)

	rr := s.postForm(router, "/auth/login", url.Values{"email": {"longlived@example.com"}, "password": {"password123"}, "stay_logged_in": {"1"}})

	session := s.cookieNamed(rr, "session_token")
	s.Require().NotNil(session)
	s.WithinDuration(time.Now().Add(30*24*time.Hour), session.Expires, time.Minute)
}

func (s *AuthSuite) TestLogin_TwoStepDisabled_InvalidPassword() {
	s.registerUser("nope@example.com", "password123")
	router := s.buildRouter(false, true)

	rr := s.postForm(router, "/auth/login", url.Values{"email": {"nope@example.com"}, "password": {"wrong"}})

	s.Equal(http.StatusUnauthorized, rr.Code)
	s.Nil(s.cookieNamed(rr, "session_token"))
}

func (s *AuthSuite) TestOTPPage_TwoStepDisabled_RedirectsToLogin() {
	router := s.buildRouter(false, true)

	req := httptest.NewRequest(http.MethodGet, "/auth/login/otp", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/auth/login", rr.Header().Get("Location"))
}

func (s *AuthSuite) TestRegister_SignupsDisabled_ReturnsNotFound() {
	router := s.buildRouter(true, false)

	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, httptest.NewRequest(http.MethodGet, "/auth/register", nil))
	s.Equal(http.StatusNotFound, getRR.Code)

	postRR := s.postForm(router, "/auth/register", url.Values{"email": {"new@example.com"}, "password": {"password123"}})
	s.Equal(http.StatusNotFound, postRR.Code)

	user, err := s.users.FindByEmail(context.Background(), "new@example.com")
	s.Require().NoError(err)
	s.Nil(user)
}

func (s *AuthSuite) TestLoginPage_RegisterLinkFollowsSignupsFlag() {
	enabled := httptest.NewRecorder()
	s.router.ServeHTTP(enabled, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	s.Contains(enabled.Body.String(), "/auth/register")

	disabled := httptest.NewRecorder()
	s.buildRouter(true, false).ServeHTTP(disabled, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	s.NotContains(disabled.Body.String(), "/auth/register")
}

func (s *AuthSuite) TestLanding_RegisterLinkFollowsSignupsFlag() {
	enabled := httptest.NewRecorder()
	(&controllers.Controller{SignupsEnabled: true}).Landing(enabled, httptest.NewRequest(http.MethodGet, "/", nil))
	s.Contains(enabled.Body.String(), "/auth/register")

	disabled := httptest.NewRecorder()
	(&controllers.Controller{SignupsEnabled: false}).Landing(disabled, httptest.NewRequest(http.MethodGet, "/", nil))
	s.NotContains(disabled.Body.String(), "/auth/register")
}

func (s *AuthSuite) registerUser(email, password string) *models.User {
	form := url.Values{"email": {email}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(form.Encode()))
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
