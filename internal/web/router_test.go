package web_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/kernel"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/logger"
	"github.com/mathcale/discogs-collection-manager/internal/web"
)

func TestRouter_CrossSitePostToLogin_Rejected(t *testing.T) {
	router := web.NewRouter(&kernel.Kernel{Logger: logger.NewWithWriter("production", io.Discard)})

	form := url.Values{"email": {"victim@example.com"}, "password": {"attacker-password"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRouter_MismatchedOriginPostToLogin_Rejected(t *testing.T) {
	router := web.NewRouter(&kernel.Kernel{Logger: logger.NewWithWriter("production", io.Discard)})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
