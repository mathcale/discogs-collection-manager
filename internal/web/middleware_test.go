package web_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/logger"
	"github.com/mathcale/discogs-collection-manager/internal/web"
)

func TestRequestLogger_LogsStructuredFields(t *testing.T) {
	var buf bytes.Buffer
	log := logger.NewWithWriter("production", &buf)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(web.RequestLogger(log))
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "GET", result["method"])
	assert.Equal(t, "/test", result["url"])
	assert.EqualValues(t, http.StatusOK, result["status"])
	assert.NotEmpty(t, result["request_id"])
	assert.NotNil(t, result["duration_ms"])
}

func TestRequestLogger_LogsStatusCode(t *testing.T) {
	var buf bytes.Buffer
	log := logger.NewWithWriter("production", &buf)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(web.RequestLogger(log))
	r.Get("/not-found", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.EqualValues(t, http.StatusNotFound, result["status"])
}

func TestRequestLogger_InjectsLoggerIntoContext(t *testing.T) {
	var buf bytes.Buffer
	log := logger.NewWithWriter("production", &buf)

	var handlerCalled bool
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(web.RequestLogger(log))
	r.Get("/ctx", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		ctxLog := logger.FromCtx(r.Context())
		ctxLog.Info("from handler")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ctx", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)

	var lines []map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry map[string]any
		require.NoError(t, json.Unmarshal(line, &entry))
		lines = append(lines, entry)
	}

	require.Len(t, lines, 2)

	handlerLog := lines[0]
	assert.Equal(t, "from handler", handlerLog["message"])
	assert.Equal(t, "GET", handlerLog["method"])
	assert.Equal(t, "/ctx", handlerLog["url"])

	requestLog := lines[1]
	assert.Equal(t, "request", requestLog["message"])
}
