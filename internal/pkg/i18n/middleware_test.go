package i18n_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
)

func resolveLang(acceptLanguage, cookie string) (string, http.Header) {
	var resolved string
	handler := i18n.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resolved = i18n.Lang(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: i18n.CookieName, Value: cookie})
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return resolved, rec.Header()
}

func TestMiddleware_ResolvesLanguage(t *testing.T) {
	tests := []struct {
		name           string
		acceptLanguage string
		cookie         string
		want           string
	}{
		{"no header", "", "", "en"},
		{"exact pt-BR", "pt-BR", "", "pt-BR"},
		{"bare pt", "pt", "", "pt-BR"},
		{"european portuguese", "pt-PT,pt;q=0.9", "", "pt-BR"},
		{"unsupported language", "fr-FR,fr;q=0.9", "", "en"},
		{"q-values pick the preferred supported language", "de;q=0.5, pt-BR;q=0.8", "", "pt-BR"},
		{"cookie beats header", "pt-BR", "en", "en"},
		{"unsupported cookie falls through to header", "pt-BR", "xx", "pt-BR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := resolveLang(tt.acceptLanguage, tt.cookie)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMiddleware_SetsVaryHeader(t *testing.T) {
	_, header := resolveLang("pt-BR", "")

	assert.Equal(t, "Accept-Language, Cookie", header.Get("Vary"))
}
