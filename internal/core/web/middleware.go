package web

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/logger"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func RequestLogger(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			requestID := middleware.GetReqID(r.Context())
			reqLog := log.With("method", r.Method, "url", r.URL.Path, "request_id", requestID)

			ctx := logger.WithCtx(r.Context(), reqLog)
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r.WithContext(ctx))

			reqLog.Info("request", "status", rw.status, "duration_ms", time.Since(start).Milliseconds())
		})
	}
}
