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

	"github.com/mathcale/discogs-collection-manager/internal/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/handler"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
	"github.com/mathcale/discogs-collection-manager/internal/repository"
	"github.com/mathcale/discogs-collection-manager/internal/web"
)

type mockDiscogsClient struct {
	searchReturns         []discogs.Release
	searchErr             error
	addToCollectionCalled bool
	addToCollectionErr    error
}

func (m *mockDiscogsClient) SearchByBarcode(_ context.Context, _ string) ([]discogs.Release, error) {
	return m.searchReturns, m.searchErr
}

func (m *mockDiscogsClient) AddToCollection(_ context.Context, _ string, _ int) error {
	m.addToCollectionCalled = true
	return m.addToCollectionErr
}

type ScanSuite struct {
	suite.Suite
	db          *sql.DB
	dbPath      string
	sessions    repository.SessionRepository
	oauthTokens repository.OAuthTokenRepository
	router      http.Handler
	mockDiscogs *mockDiscogsClient
}

func TestScanSuite(t *testing.T) {
	suite.Run(t, new(ScanSuite))
}

func (s *ScanSuite) SetupSuite() {
	f, err := os.CreateTemp("", "scan-test-*.db")
	s.Require().NoError(err)
	s.dbPath = f.Name()
	f.Close()

	s.Require().NoError(migrate.Up(s.dbPath, "../../migrations"))

	db, err := sql.Open("sqlite3", s.dbPath)
	s.Require().NoError(err)
	s.db = db

	s.sessions = repository.NewSQLiteSessionRepository(db)
	s.oauthTokens = repository.NewSQLiteOAuthTokenRepository(db)
}

func (s *ScanSuite) SetupTest() {
	_, err := s.db.Exec(`DELETE FROM sessions; DELETE FROM oauth_tokens`)
	s.Require().NoError(err)

	s.mockDiscogs = &mockDiscogsClient{}

	scanHandler := handler.NewScanHandler(s.mockDiscogs, s.oauthTokens)

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(web.RequireAuth(s.sessions))
		r.Get("/app/scanner", scanHandler.ScanPage)
		r.Post("/app/scan", scanHandler.Scan)
		r.Post("/app/scan/confirm", scanHandler.Confirm)
		r.Get("/app/scan/cancel", scanHandler.Cancel)
	})

	s.router = r
}

func (s *ScanSuite) TearDownSuite() {
	s.db.Close()
	os.Remove(s.dbPath)
}

func (s *ScanSuite) TestScanPage_RequiresAuth() {
	req := httptest.NewRequest(http.MethodGet, "/app/scanner", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/login", rr.Header().Get("Location"))
}

func (s *ScanSuite) TestScanEndpoint_RequiresAuth() {
	form := url.Values{"barcode": {"1234567890"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/login", rr.Header().Get("Location"))
}

func (s *ScanSuite) TestScan_POST_MissingBarcode() {
	token := s.createSession("user-missing-barcode", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodPost, "/app/scan", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusBadRequest, rr.Code)
}

func (s *ScanSuite) TestScan_POST_SingleMatch_RendersConfirmCard() {
	token := s.createSession("user-single-match", time.Now().Add(time.Hour))
	s.mockDiscogs.searchReturns = []discogs.Release{
		{ID: 12345, Title: "Artist - Album", Country: "US", Year: "1991", Label: "Some Label", Thumb: "", URI: "https://www.discogs.com/release/12345"},
	}

	form := url.Values{"barcode": {"1234567890"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "Artist - Album")
}

func (s *ScanSuite) TestScan_POST_RateLimit_RendersErrorPartial() {
	token := s.createSession("user-rate-limit", time.Now().Add(time.Hour))
	s.mockDiscogs.searchErr = &discogs.RateLimitError{}

	form := url.Values{"barcode": {"1234567890"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusTooManyRequests, rr.Code)
	s.Contains(rr.Body.String(), "rate limit")
}

func (s *ScanSuite) TestScanConfirm_POST_CallsAddToCollection() {
	token := s.createSession("user-confirm", time.Now().Add(time.Hour))
	s.createOAuthToken("user-confirm", "discogs-user")

	form := url.Values{"release_id": {"12345"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan/confirm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.True(s.mockDiscogs.addToCollectionCalled)
	s.Contains(rr.Body.String(), "collection")
}

func (s *ScanSuite) TestScanCancel_GET_DoesNotCallAddToCollection() {
	token := s.createSession("user-cancel", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/app/scan/cancel", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.False(s.mockDiscogs.addToCollectionCalled)
	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/app/scanner", rr.Header().Get("Location"))
}

func (s *ScanSuite) createSession(userID string, expiresAt time.Time) string {
	token := "scan-test-token-" + userID
	_, err := s.sessions.Create(context.Background(), userID, token, expiresAt, false)
	s.Require().NoError(err)
	return token
}

func (s *ScanSuite) createOAuthToken(userID, discogsUsername string) {
	_, err := s.oauthTokens.Upsert(context.Background(), userID, "access-token", "access-secret", discogsUsername)
	s.Require().NoError(err)
}
