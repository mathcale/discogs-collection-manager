package account_test

import (
	"context"
	"database/sql"
	"encoding/json"
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

	"github.com/dghubble/oauth1"

	"github.com/mathcale/discogs-collection-manager/internal/features/account"
	accountdomain "github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
	accountstore "github.com/mathcale/discogs-collection-manager/internal/features/account/store"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth"
	authdomain "github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
	authstore "github.com/mathcale/discogs-collection-manager/internal/features/auth/store"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
)

type DiscogsOAuthSuite struct {
	suite.Suite
	db          *sql.DB
	dbPath      string
	users       authdomain.UserRepository
	sessions    authdomain.SessionRepository
	oauthTokens accountdomain.OAuthTokenRepository
	router      http.Handler
	mockDiscogs *httptest.Server
}

func TestDiscogsOAuthSuite(t *testing.T) {
	suite.Run(t, new(DiscogsOAuthSuite))
}

func (s *DiscogsOAuthSuite) SetupSuite() {
	f, err := os.CreateTemp("", "discogs-oauth-test-*.db")
	s.Require().NoError(err)
	s.dbPath = f.Name()
	f.Close()

	s.Require().NoError(migrate.Up(s.dbPath, "../../../migrations"))

	db, err := sql.Open("sqlite3", s.dbPath)
	s.Require().NoError(err)
	s.db = db

	s.users = authstore.NewUserRepository(db)
	s.sessions = authstore.NewSessionRepository(db)
	s.oauthTokens = accountstore.NewOAuthTokenRepository(db)
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

	s.router = s.buildRouter(oauthCfg)
}

func (s *DiscogsOAuthSuite) buildRouter(oauthCfg *oauth1.Config) http.Handler {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(auth.New(auth.Deps{DB: s.db}).RequireAuth)
		account.New(account.Deps{
			DB:                 s.db,
			DiscogsOAuthConfig: oauthCfg,
			IdentityURL:        s.mockDiscogs.URL + "/oauth/identity",
		}).Routes(r)
	})

	return r
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

func (s *DiscogsOAuthSuite) TestCallback_TwiceForSameUser_LeavesOneConnectionWithLatestCredentials() {
	user := s.createUser("twice@example.com")
	sessionToken := s.createSession(user.ID, time.Now().Add(6*time.Hour))
	s.Require().NoError(s.saveOAuthToken(user.ID, "stale-access", "stale-secret", "staleuser"))

	connectReq := httptest.NewRequest(http.MethodGet, "/discogs/connect", nil)
	connectReq.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	s.router.ServeHTTP(httptest.NewRecorder(), connectReq)

	callbackReq := httptest.NewRequest(http.MethodGet, "/discogs/callback?oauth_token=mock-request-token&oauth_verifier=mock-verifier", nil)
	callbackReq.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, callbackReq)
	s.Equal(http.StatusFound, rr.Code)

	var count int
	s.Require().NoError(s.db.QueryRow(`SELECT COUNT(*) FROM oauth_tokens WHERE user_id = ?`, user.ID).Scan(&count))
	s.Equal(1, count)

	stored, err := s.oauthTokens.FindByUserID(context.Background(), user.ID)
	s.Require().NoError(err)
	s.Equal("mock-access-token", stored.AccessToken)
	s.Equal("mock-access-secret", stored.AccessTokenSecret)
	s.Equal("testdiscogsuser", stored.DiscogsUsername)
}

func (s *DiscogsOAuthSuite) saveOAuthToken(userID, accessToken, accessSecret, username string) error {
	return s.oauthTokens.Save(context.Background(), accountdomain.NewOAuthToken(userID, accessToken, accessSecret, username))
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
	s.Require().ErrorIs(err, accountdomain.ErrNotFound)
	s.Nil(stored)
}

func (s *DiscogsOAuthSuite) TestDisconnect_RemovesToken() {
	user := s.createUser("disconnect@example.com")
	sessionToken := s.createSession(user.ID, time.Now().Add(6*time.Hour))

	err := s.saveOAuthToken(user.ID, "tok", "sec", "discogsman")
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodPost, "/discogs/disconnect", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/app/settings", rr.Header().Get("Location"))

	stored, err := s.oauthTokens.FindByUserID(context.Background(), user.ID)
	s.Require().ErrorIs(err, accountdomain.ErrNotFound)
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
	s.router = s.buildRouter(oauthCfg)

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
	err := s.saveOAuthToken(user.ID, "tok", "sec", "myDiscogsUser")
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

func (s *DiscogsOAuthSuite) postLanguage(value string) *httptest.ResponseRecorder {
	user := s.createUser("lang@example.com")
	sessionToken := s.createSession(user.ID, time.Now().Add(6*time.Hour))

	form := url.Values{"lang": {value}}
	req := httptest.NewRequest(http.MethodPost, "/app/settings/language", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)
	return rr
}

func (s *DiscogsOAuthSuite) langCookie(rr *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == "lang" {
			return c
		}
	}
	return nil
}

func (s *DiscogsOAuthSuite) TestLanguage_SupportedValue_SetsCookieAndRedirects() {
	rr := s.postLanguage("pt-BR")

	s.Equal(http.StatusSeeOther, rr.Code)
	s.Equal("/app/settings", rr.Header().Get("Location"))
	cookie := s.langCookie(rr)
	s.Require().NotNil(cookie)
	s.Equal("pt-BR", cookie.Value)
	s.Greater(cookie.MaxAge, 0)
}

func (s *DiscogsOAuthSuite) TestLanguage_SystemDefault_ClearsCookie() {
	rr := s.postLanguage("")

	s.Equal(http.StatusSeeOther, rr.Code)
	cookie := s.langCookie(rr)
	s.Require().NotNil(cookie)
	s.Less(cookie.MaxAge, 0)
}

func (s *DiscogsOAuthSuite) TestLanguage_UnsupportedValue_IsRejected() {
	rr := s.postLanguage("xx")

	s.Equal(http.StatusBadRequest, rr.Code)
	s.Nil(s.langCookie(rr))
}

func (s *DiscogsOAuthSuite) TestSettingsPage_PreselectsLanguageFromCookie() {
	user := s.createUser("preselect@example.com")
	sessionToken := s.createSession(user.ID, time.Now().Add(6*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/app/settings", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	req.AddCookie(&http.Cookie{Name: "lang", Value: "pt-BR"})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), `name="lang" value="pt-BR"`)
}

func (s *DiscogsOAuthSuite) createUser(email string) *authdomain.User {
	user, err := authdomain.NewUser(email, "password123")
	s.Require().NoError(err)
	s.Require().NoError(s.users.Save(context.Background(), user))
	return user
}

func (s *DiscogsOAuthSuite) createSession(userID string, expiresAt time.Time) string {
	token := "discogs-test-token-" + userID
	err := s.sessions.Save(context.Background(), &authdomain.Session{ID: "session-" + token, UserID: userID, Token: token, ExpiresAt: expiresAt, Persistent: false, CreatedAt: time.Now().UTC()})
	s.Require().NoError(err)
	return token
}
