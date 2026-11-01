package web_test

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/core"
	"github.com/mathcale/discogs-collection-manager/internal/core/web"
	"github.com/mathcale/discogs-collection-manager/internal/features/account"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth"
	"github.com/mathcale/discogs-collection-manager/internal/features/landing"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/logger"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
)

func TestRouter_CrossSitePostToLogin_Rejected(t *testing.T) {
	router := web.NewRouter(&core.Kernel{Logger: logger.NewWithWriter("production", io.Discard), Landing: landing.New(landing.Deps{}), Auth: auth.New(auth.Deps{}), Account: account.New(account.Deps{}), Scanner: scanner.New(scanner.Deps{})})

	form := url.Values{"email": {"victim@example.com"}, "password": {"attacker-password"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRouter_MismatchedOriginPostToLogin_Rejected(t *testing.T) {
	router := web.NewRouter(&core.Kernel{Logger: logger.NewWithWriter("production", io.Discard), Landing: landing.New(landing.Deps{}), Auth: auth.New(auth.Deps{}), Account: account.New(account.Deps{}), Scanner: scanner.New(scanner.Deps{})})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRouter_LandingWithPortugueseBrowser_RendersPortuguese(t *testing.T) {
	router := web.NewRouter(&core.Kernel{Logger: logger.NewWithWriter("production", io.Discard), Landing: landing.New(landing.Deps{}), Auth: auth.New(auth.Deps{}), Account: account.New(account.Deps{}), Scanner: scanner.New(scanner.Deps{})})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "pt-BR")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `lang="pt-BR"`)
	assert.Contains(t, rec.Body.String(), "Conecte sua conta do Discogs para começar a escanear sua coleção de vinis.")
}

func TestRouter_LandingWithEnglishCookie_OverridesPortugueseBrowser(t *testing.T) {
	router := web.NewRouter(&core.Kernel{Logger: logger.NewWithWriter("production", io.Discard), Landing: landing.New(landing.Deps{}), Auth: auth.New(auth.Deps{}), Account: account.New(account.Deps{}), Scanner: scanner.New(scanner.Deps{})})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "pt-BR")
	req.AddCookie(&http.Cookie{Name: "lang", Value: "en"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Contains(t, rec.Body.String(), `lang="en"`)
	assert.Contains(t, rec.Body.String(), "Connect your Discogs account to start scanning your vinyl collection.")
}

func TestRouter_FailedLoginWithPortugueseBrowser_RendersPortugueseError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "router.db")
	require.NoError(t, migrate.Up(dbPath, "../../../migrations"))
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	router := web.NewRouter(&core.Kernel{Logger: logger.NewWithWriter("production", io.Discard), Landing: landing.New(landing.Deps{}), Auth: auth.New(auth.Deps{DB: db}), Account: account.New(account.Deps{}), Scanner: scanner.New(scanner.Deps{})})

	form := url.Values{"email": {"nobody@example.com"}, "password": {"wrong-password"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "pt-BR")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "E-mail ou senha inválidos.")
}
