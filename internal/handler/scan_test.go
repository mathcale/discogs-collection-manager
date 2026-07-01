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

type ScanSuite struct {
	suite.Suite
	db       *sql.DB
	dbPath   string
	sessions repository.SessionRepository
	router   http.Handler
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
}

func (s *ScanSuite) SetupTest() {
	_, err := s.db.Exec(`DELETE FROM sessions`)
	s.Require().NoError(err)

	scanHandler := handler.NewScanHandler()

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(web.RequireAuth(s.sessions))
		r.Get("/app/scanner", scanHandler.ScanPage)
		r.Post("/app/scan", scanHandler.Scan)
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

func (s *ScanSuite) TestScan_POST_ValidBarcode() {
	token := s.createSession("user-1", time.Now().Add(time.Hour))

	form := url.Values{"barcode": {"1234567890"}}
	req := httptest.NewRequest(http.MethodPost, "/app/scan", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Contains(rr.Body.String(), "1234567890")
}

func (s *ScanSuite) TestScan_POST_MissingBarcode() {
	token := s.createSession("user-2", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodPost, "/app/scan", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusBadRequest, rr.Code)
}

func (s *ScanSuite) createSession(userID string, expiresAt time.Time) string {
	token := "scan-test-token-" + userID
	_, err := s.sessions.Create(context.Background(), userID, token, expiresAt, false)
	s.Require().NoError(err)
	return token
}
