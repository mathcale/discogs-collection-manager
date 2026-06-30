package handler_test

import (
	"context"
	"database/sql"
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

type AuthSuite struct {
	suite.Suite
	db       *sql.DB
	dbPath   string
	users    repository.UserRepository
	sessions repository.SessionRepository
	router   http.Handler
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
}

func (s *AuthSuite) SetupTest() {
	_, err := s.db.Exec(`DELETE FROM sessions`)
	s.Require().NoError(err)
	_, err = s.db.Exec(`DELETE FROM users`)
	s.Require().NoError(err)

	authHandler := handler.NewAuthHandler(s.users, s.sessions)

	r := chi.NewRouter()
	r.Get("/register", authHandler.RegisterPage)
	r.Post("/register", authHandler.Register)
	r.Get("/login", authHandler.LoginPage)
	r.Post("/login", authHandler.Login)

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
}

func (s *AuthSuite) TestLogin_POST_Success() {
	s.registerUser("login@example.com", "password123")

	form := url.Values{"email": {"login@example.com"}, "password": {"password123"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/app", rr.Header().Get("Location"))

	cookies := rr.Result().Cookies()
	s.Require().NotEmpty(cookies)
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			sessionCookie = c
		}
	}
	s.Require().NotNil(sessionCookie)
	s.NotEmpty(sessionCookie.Value)
}

func (s *AuthSuite) TestLogin_POST_StayLoggedIn() {
	s.registerUser("persist@example.com", "password123")

	form := url.Values{"email": {"persist@example.com"}, "password": {"password123"}, "stay_logged_in": {"1"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func (s *AuthSuite) TestLogin_POST_InvalidCredentials() {
	s.registerUser("badpass@example.com", "correctpass")

	form := url.Values{"email": {"badpass@example.com"}, "password": {"wrongpass"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnauthorized, rr.Code)
	s.Contains(rr.Body.String(), "Invalid email or password")
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
