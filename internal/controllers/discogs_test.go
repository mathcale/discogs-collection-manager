package controllers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"

	"github.com/dghubble/oauth1"

	"github.com/mathcale/discogs-collection-manager/internal/app/web"
	"github.com/mathcale/discogs-collection-manager/internal/controllers"
	"github.com/mathcale/discogs-collection-manager/internal/models"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
)

type DiscogsOAuthSuite struct {
	suite.Suite
	db          *sql.DB
	dbPath      string
	users       models.UserRepository
	sessions    models.SessionRepository
	oauthTokens models.OAuthTokenRepository
	router      http.Handler
	mockDiscogs *httptest.Server
	controller  *controllers.Controller
}

func TestDiscogsOAuthSuite(t *testing.T) {
	suite.Run(t, new(DiscogsOAuthSuite))
}

func (s *DiscogsOAuthSuite) SetupSuite() {
	f, err := os.CreateTemp("", "discogs-oauth-test-*.db")
	s.Require().NoError(err)
	s.dbPath = f.Name()
	f.Close()

	s.Require().NoError(migrate.Up(s.dbPath, "../../migrations"))

	db, err := sql.Open("sqlite3", s.dbPath)
	s.Require().NoError(err)
	s.db = db

	s.users = models.NewSQLiteUserRepository(db)
	s.sessions = models.NewSQLiteSessionRepository(db)
	s.oauthTokens = models.NewSQLiteOAuthTokenRepository(db)
}

func (s *DiscogsOAuthSuite) SetupTest() {
	_, err := s.db.Exec(`DELETE FROM oauth_tokens`)
	s.Require().NoError(err)
	_, err = s.db.Exec(`DELETE FROM sessions`)
	s.Require().NoError(err)
	_, err = s.db.Exec(`DELETE FROM users`)
	s.Require().NoError(err)

	s.mockDiscogs = httptest.NewServer(s.buildMockDiscogsRouter())

	oauthCfg := &oauth1.Config{
		ConsumerKey:    "test-key",
		ConsumerSecret: "test-secret",
		CallbackURL:    "http://localhost/discogs/callback",
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: s.mockDiscogs.URL + "/oauth/request_token",
			AuthorizeURL:    s.mockDiscogs.URL + "/oauth/authorize",
			AccessTokenURL:  s.mockDiscogs.URL + "/oauth/access_token",
		},
	}

	s.controller = &controllers.Controller{
		OAuthTokens:  s.oauthTokens,
		Sessions:     s.sessions,
		Users:        s.users,
		DiscogsOAuth: oauthCfg,
		IdentityURL:  s.mockDiscogs.URL + "/oauth/identity",
	}

	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(web.RequireAuth(s.sessions))
		r.Get("/app/settings", s.controller.SettingsPage)
		r.Get("/discogs/connect", s.controller.Connect)
		r.Get("/discogs/callback", s.controller.Callback)
		r.Post("/discogs/disconnect", s.controller.Disconnect)
	})

	s.router = r
}

func (s *DiscogsOAuthSuite) TearDownTest() {
	s.mockDiscogs.Close()
}

func (s *DiscogsOAuthSuite) TearDownSuite() {
	s.db.Close()
	os.Remove(s.dbPath)
}

func (s *DiscogsOAuthSuite) buildMockDiscogsRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/oauth/request_token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		fmt.Fprint(w, "oauth_token=mock-request-token&oauth_token_secret=mock-request-secret&oauth_callback_confirmed=true")
	})

	mux.HandleFunc("/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		fmt.Fprint(w, "oauth_token=mock-access-token&oauth_token_secret=mock-access-secret")
	})

	mux.HandleFunc("/oauth/identity", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       12345,
			"username": "testdiscogsuser",
		})
	})

	return mux
}

func (s *DiscogsOAuthSuite) TestConnect_RedirectsToDiscogsAuthorize() {
	user := s.createUser("connect@example.com")
	token := s.createSession(user.ID, time.Now().Add(6*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/discogs/connect", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)

	location := rr.Header().Get("Location")
	s.Contains(location, "/oauth/authorize")
	s.Contains(location, "oauth_token=mock-request-token")
}

func (s *DiscogsOAuthSuite) TestCallback_StoresAccessToken() {
	user := s.createUser("callback@example.com")
	sessionToken := s.createSession(user.ID, time.Now().Add(6*time.Hour))

	connectReq := httptest.NewRequest(http.MethodGet, "/discogs/connect", nil)
	connectReq.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	s.router.ServeHTTP(httptest.NewRecorder(), connectReq)

	callbackReq := httptest.NewRequest(http.MethodGet, "/discogs/callback?oauth_token=mock-request-token&oauth_verifier=mock-verifier", nil)
	callbackReq.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, callbackReq)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/app/settings", rr.Header().Get("Location"))

	stored, err := s.oauthTokens.FindByUserID(context.Background(), user.ID)
	s.Require().NoError(err)
	s.Require().NotNil(stored)
	s.Equal("mock-access-token", stored.AccessToken)
	s.Equal("mock-access-secret", stored.AccessTokenSecret)
	s.Equal("testdiscogsuser", stored.DiscogsUsername)
}

func (s *DiscogsOAuthSuite) TestCallback_FromDifferentUser_IsRejected() {
	attacker := s.createUser("attacker@example.com")
	attackerSession := s.createSession(attacker.ID, time.Now().Add(6*time.Hour))
	victim := s.createUser("victim@example.com")
	victimSession := s.createSession(victim.ID, time.Now().Add(6*time.Hour))

	connectReq := httptest.NewRequest(http.MethodGet, "/discogs/connect", nil)
	connectReq.AddCookie(&http.Cookie{Name: "session_token", Value: attackerSession})
	s.router.ServeHTTP(httptest.NewRecorder(), connectReq)

	callbackReq := httptest.NewRequest(http.MethodGet, "/discogs/callback?oauth_token=mock-request-token&oauth_verifier=mock-verifier", nil)
	callbackReq.AddCookie(&http.Cookie{Name: "session_token", Value: victimSession})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, callbackReq)

	s.Equal(http.StatusBadRequest, rr.Code)

	stored, err := s.oauthTokens.FindByUserID(context.Background(), victim.ID)
	s.Require().NoError(err)
	s.Nil(stored)
}

func (s *DiscogsOAuthSuite) TestDisconnect_RemovesToken() {
	user := s.createUser("disconnect@example.com")
	sessionToken := s.createSession(user.ID, time.Now().Add(6*time.Hour))

	_, err := s.oauthTokens.Upsert(context.Background(), user.ID, "tok", "sec", "discogsman")
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodPost, "/discogs/disconnect", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/app/settings", rr.Header().Get("Location"))

	stored, err := s.oauthTokens.FindByUserID(context.Background(), user.ID)
	s.Require().NoError(err)
	s.Nil(stored)
}

func (s *DiscogsOAuthSuite) TestCallback_401FromDiscogs_ShowsReconnectPrompt() {
	s.mockDiscogs.Close()

	unauthorizedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/request_token" {
			w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
			fmt.Fprint(w, "oauth_token=bad-request-token&oauth_token_secret=bad-secret&oauth_callback_confirmed=true")
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid consumer.")
	}))
	defer unauthorizedServer.Close()

	oauthCfg := &oauth1.Config{
		ConsumerKey:    "bad-key",
		ConsumerSecret: "bad-secret",
		CallbackURL:    "http://localhost/discogs/callback",
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: unauthorizedServer.URL + "/oauth/request_token",
			AuthorizeURL:    unauthorizedServer.URL + "/oauth/authorize",
			AccessTokenURL:  unauthorizedServer.URL + "/oauth/access_token",
		},
	}
	s.controller.DiscogsOAuth = oauthCfg

	user := s.createUser("unauthorized@example.com")
	sessionToken := s.createSession(user.ID, time.Now().Add(6*time.Hour))

	connectReq := httptest.NewRequest(http.MethodGet, "/discogs/connect", nil)
	connectReq.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	s.router.ServeHTTP(httptest.NewRecorder(), connectReq)

	callbackReq := httptest.NewRequest(http.MethodGet, "/discogs/callback?oauth_token=bad-request-token&oauth_verifier=bad-verifier", nil)
	callbackReq.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, callbackReq)

	s.NotEqual(http.StatusInternalServerError, rr.Code)
	body := rr.Body.String()
	s.Contains(body, "reconnect")
}

func (s *DiscogsOAuthSuite) TestSettingsPage_ShowsConnectedStatus() {
	user := s.createUser("settings@example.com")
	sessionToken := s.createSession(user.ID, time.Now().Add(6*time.Hour))
	_, err := s.oauthTokens.Upsert(context.Background(), user.ID, "tok", "sec", "myDiscogsUser")
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/app/settings", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "myDiscogsUser")
	s.Contains(rr.Body.String(), "Connected as")
}

func (s *DiscogsOAuthSuite) TestSettingsPage_ShowsDisconnectedStatus() {
	user := s.createUser("notoken@example.com")
	sessionToken := s.createSession(user.ID, time.Now().Add(6*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/app/settings", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Not connected")
}

func (s *DiscogsOAuthSuite) TestSettingsPage_ShowsNavigation() {
	user := s.createUser("nav@example.com")
	sessionToken := s.createSession(user.ID, time.Now().Add(6*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/app/settings", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	body := rr.Body.String()
	s.Contains(body, "/app/settings")
	s.Contains(body, "/app/scanner")
}

func (s *DiscogsOAuthSuite) createUser(email string) *models.User {
	form := url.Values{"email": {email}, "password": {"password123"}}
	user, err := s.users.Create(context.Background(), email, "hashed")
	s.Require().NoError(err)
	_ = form
	return user
}

func (s *DiscogsOAuthSuite) createSession(userID string, expiresAt time.Time) string {
	token := "discogs-test-token-" + userID
	_, err := s.sessions.Create(context.Background(), userID, token, expiresAt, false)
	s.Require().NoError(err)
	return token
}
