package web

import (
	"net/http"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/repository"
)

type contextKey string

const ContextKeyUserID contextKey = "userID"

func RequireAuth(sessions repository.SessionRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_token")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			session, err := sessions.FindByToken(r.Context(), cookie.Value)
			if err != nil || session == nil || time.Now().After(session.ExpiresAt) {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			ctx := r.Context()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
