package web

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/mathcale/discogs-collection-manager/internal/models"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
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
			reqLog := log.With("method", r.Method, "url", r.URL.RequestURI(), "request_id", requestID)

			ctx := logger.WithCtx(r.Context(), reqLog)
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r.WithContext(ctx))

			reqLog.Info("request", "status", rw.status, "duration_ms", time.Since(start).Milliseconds())
		})
	}
}

func RequireAuth(sessions models.SessionRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_token")
			if err != nil {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			session, err := sessions.FindByToken(r.Context(), cookie.Value)
			if err != nil || session == nil || time.Now().After(session.ExpiresAt) {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			ctx := context.WithValue(r.Context(), ctxkey.UserID, session.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
