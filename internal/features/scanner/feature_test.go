package scanner_test

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

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	accountdomain "github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
	accountstore "github.com/mathcale/discogs-collection-manager/internal/features/account/store"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth"
	authdomain "github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
	authstore "github.com/mathcale/discogs-collection-manager/internal/features/auth/store"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
)

type mockDiscogsClient struct {
	searchReturns         []discogs.Release
	searchErr             error
	addToCollectionCalled bool
	addToCollectionErr    error
	releaseDetail         discogs.ReleaseDetail
	owned                 []discogs.OwnedRelease
}

func (m *mockDiscogsClient) SearchByBarcode(_ context.Context, _ string) ([]discogs.Release, error) {
	return m.searchReturns, m.searchErr
}

func (m *mockDiscogsClient) GetRelease(_ context.Context, _ int) (discogs.ReleaseDetail, error) {
	return m.releaseDetail, nil
}

func (m *mockDiscogsClient) ListCollection(_ context.Context, _ string) ([]discogs.OwnedRelease, error) {
	return m.owned, nil
}

func (m *mockDiscogsClient) AddToCollection(_ context.Context, _ string, _ int) error {
	m.addToCollectionCalled = true
	return m.addToCollectionErr
}

type ScanSuite struct {
	suite.Suite
	db          *sql.DB
	dbPath      string
	sessions    authdomain.SessionRepository
	oauthTokens accountdomain.OAuthTokenRepository
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

	s.Require().NoError(migrate.Up(s.dbPath, "../../../migrations"))

	db, err := sql.Open("sqlite3", s.dbPath)
	s.Require().NoError(err)
	s.db = db

	s.sessions = authstore.NewSessionRepository(db)
	s.oauthTokens = accountstore.NewOAuthTokenRepository(db)
}

func (s *ScanSuite) SetupTest() {
	_, err := s.db.Exec(`DELETE FROM sessions; DELETE FROM oauth_tokens`)
	s.Require().NoError(err)

	s.mockDiscogs = &mockDiscogsClient{}

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.New(auth.Deps{DB: s.db}).RequireAuth)
		scanner.New(scanner.Deps{DB: s.db, DiscogsClient: s.mockDiscogs}).Routes(r)
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
	s.Equal("/auth/login", rr.Header().Get("Location"))
}

func (s *ScanSuite) TestScanEndpoint_RequiresAuth() {
	form := url.Values{"barcode": {"1234567890"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusFound, rr.Code)
	s.Equal("/auth/login", rr.Header().Get("Location"))
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
	s.createOAuthToken("user-single-match", "discogs-user")
	s.mockDiscogs.searchReturns = []discogs.Release{
		{ID: 12345, Title: "Artist - Album", Country: "US", Year: "1991", Labels: []string{"Some Label"}, Thumb: "", Path: "/release/12345"},
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
	s.createOAuthToken("user-rate-limit", "discogs-user")
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
	s.JSONEq(
		`{"toast":{"message":"Release added to your Discogs collection!","status":"success"}}`,
		rr.Header().Get("HX-Trigger"),
	)
	s.JSONEq(`{"scan-success":true}`, rr.Header().Get("HX-Trigger-After-Swap"))
}

func (s *ScanSuite) postConfirm(userID string, form url.Values) *httptest.ResponseRecorder {
	token := s.createSession(userID, time.Now().Add(time.Hour))
	s.createOAuthToken(userID, "discogs-user")

	req := httptest.NewRequest(http.MethodPost, "/app/scan/confirm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	return rr
}

func (s *ScanSuite) TestScanConfirm_POST_ExactReleaseOwned_WarnsWithoutAdding() {
	s.mockDiscogs.releaseDetail = discogs.ReleaseDetail{ID: 12345, Title: "Album", Artists: []discogs.Artist{{Name: "Artist"}}}
	s.mockDiscogs.owned = []discogs.OwnedRelease{
		{ID: 12345, Title: "Album", Artists: []discogs.Artist{{Name: "Artist"}}},
	}

	rr := s.postConfirm("user-exact-owned", url.Values{"release_id": {"12345"}})

	s.Equal(http.StatusOK, rr.Code)
	s.False(s.mockDiscogs.addToCollectionCalled)
	s.Contains(rr.Body.String(), "Already in your collection")
	s.Contains(rr.Body.String(), `name="acknowledged"`)
}

func (s *ScanSuite) TestScanConfirm_POST_OtherEditionOwned_WarnsWithoutAdding() {
	s.mockDiscogs.releaseDetail = discogs.ReleaseDetail{ID: 12345, Title: "Album", Artists: []discogs.Artist{{Name: "Artist"}}}
	s.mockDiscogs.owned = []discogs.OwnedRelease{
		{ID: 99999, Title: "album", Year: 2010, Artists: []discogs.Artist{{Name: "ARTIST"}}},
	}

	rr := s.postConfirm("user-similar-owned", url.Values{"release_id": {"12345"}})

	s.Equal(http.StatusOK, rr.Code)
	s.False(s.mockDiscogs.addToCollectionCalled)
	s.Contains(rr.Body.String(), "You own another edition")
	s.NotContains(rr.Body.String(), "Already in your collection")
}

func (s *ScanSuite) TestScanConfirm_POST_UnrelatedReleaseOwned_AddsDirectly() {
	s.mockDiscogs.releaseDetail = discogs.ReleaseDetail{ID: 12345, Title: "Album", Artists: []discogs.Artist{{Name: "Artist"}}}
	s.mockDiscogs.owned = []discogs.OwnedRelease{
		{ID: 99999, Title: "Other Album", Artists: []discogs.Artist{{Name: "Artist"}}},
		{ID: 88888, Title: "Album", Artists: []discogs.Artist{{Name: "Other Artist"}}},
	}

	rr := s.postConfirm("user-unrelated-owned", url.Values{"release_id": {"12345"}})

	s.Equal(http.StatusOK, rr.Code)
	s.True(s.mockDiscogs.addToCollectionCalled)
}

func (s *ScanSuite) TestScanConfirm_POST_Acknowledged_AddsDespiteOwnedRelease() {
	s.mockDiscogs.releaseDetail = discogs.ReleaseDetail{ID: 12345, Title: "Album", Artists: []discogs.Artist{{Name: "Artist"}}}
	s.mockDiscogs.owned = []discogs.OwnedRelease{
		{ID: 12345, Title: "Album", Artists: []discogs.Artist{{Name: "Artist"}}},
	}

	rr := s.postConfirm("user-acknowledged", url.Values{"release_id": {"12345"}, "acknowledged": {"1"}})

	s.Equal(http.StatusOK, rr.Code)
	s.True(s.mockDiscogs.addToCollectionCalled)
}

func (s *ScanSuite) TestScan_POST_MultipleMatches_RendersConflictList() {
	token := s.createSession("user-multi-match", time.Now().Add(time.Hour))
	s.createOAuthToken("user-multi-match", "discogs-user")
	s.mockDiscogs.searchReturns = []discogs.Release{
		{ID: 11111, Title: "Artist - Album (US)", Country: "US", Year: "1991", Labels: []string{"Label A"}, Thumb: "", Path: "/release/11111"},
		{ID: 22222, Title: "Artist - Album (UK)", Country: "UK", Year: "1992", Labels: []string{"Label B"}, Thumb: "", Path: "/release/22222"},
	}

	form := url.Values{"barcode": {"1234567890"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	body := rr.Body.String()
	s.Contains(body, "Artist - Album (US)")
	s.Contains(body, "Artist - Album (UK)")
	s.Contains(body, "11111")
	s.Contains(body, "22222")
}

func (s *ScanSuite) TestScanConfirm_POST_NoSelection_RendersErrorPartial() {
	token := s.createSession("user-no-selection", time.Now().Add(time.Hour))
	s.createOAuthToken("user-no-selection", "discogs-user")

	req := httptest.NewRequest(http.MethodPost, "/app/scan/confirm", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.False(s.mockDiscogs.addToCollectionCalled)
	s.Contains(rr.Body.String(), "select")
}

func (s *ScanSuite) TestScan_POST_DiscogsAuthError_RendersReconnectPartial() {
	token := s.createSession("user-auth-error-search", time.Now().Add(time.Hour))
	s.createOAuthToken("user-auth-error-search", "discogs-user")
	s.mockDiscogs.searchErr = &discogs.DiscogsAuthError{}

	form := url.Values{"barcode": {"1234567890"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnauthorized, rr.Code)
	s.Contains(rr.Body.String(), "reconnect")
	s.Contains(rr.Body.String(), "/discogs/connect")
}

func (s *ScanSuite) TestScanConfirm_POST_DiscogsAuthError_RendersReconnectPartial() {
	token := s.createSession("user-auth-error-confirm", time.Now().Add(time.Hour))
	s.createOAuthToken("user-auth-error-confirm", "discogs-user")
	s.mockDiscogs.addToCollectionErr = &discogs.DiscogsAuthError{}

	form := url.Values{"release_id": {"12345"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan/confirm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnauthorized, rr.Code)
	s.Contains(rr.Body.String(), "reconnect")
	s.Contains(rr.Body.String(), "/discogs/connect")
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

func (s *ScanSuite) TestScan_POST_NoDiscogsToken_RendersReconnectPartial() {
	token := s.createSession("user-no-oauth-scan", time.Now().Add(time.Hour))

	form := url.Values{"barcode": {"1234567890"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnauthorized, rr.Code)
	s.Contains(rr.Body.String(), "reconnect")
	s.Contains(rr.Body.String(), "/discogs/connect")
	s.False(s.mockDiscogs.addToCollectionCalled)
}

func (s *ScanSuite) TestScanConfirm_POST_NoDiscogsToken_RendersReconnectPartial() {
	token := s.createSession("user-no-oauth-confirm", time.Now().Add(time.Hour))

	form := url.Values{"release_id": {"12345"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan/confirm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusUnauthorized, rr.Code)
	s.Contains(rr.Body.String(), "reconnect")
	s.Contains(rr.Body.String(), "/discogs/connect")
	s.False(s.mockDiscogs.addToCollectionCalled)
}

func (s *ScanSuite) TestScanPage_ShowsNavigation() {
	token := s.createSession("user-nav", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/app/scanner", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	body := rr.Body.String()
	s.Contains(body, "/app/settings")
	s.Contains(body, "/app/scanner")
}

func (s *ScanSuite) createSession(userID string, expiresAt time.Time) string {
	token := "scan-test-token-" + userID
	err := s.sessions.Save(context.Background(), &authdomain.Session{ID: "session-" + token, UserID: userID, Token: token, ExpiresAt: expiresAt, Persistent: false, CreatedAt: time.Now().UTC()})
	s.Require().NoError(err)
	return token
}

func (s *ScanSuite) createOAuthToken(userID, discogsUsername string) {
	err := s.oauthTokens.Save(context.Background(), accountdomain.NewOAuthToken(userID, "access-token", "access-secret", discogsUsername))
	s.Require().NoError(err)
}
