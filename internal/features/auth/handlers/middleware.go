package handlers

import (
	"context"
	"net/http"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/usecases"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
)

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		input := usecases.AuthenticateSessionInput{
			Token: cookie.Value,
		}

		out, err := h.AuthenticateSessionUseCase.Execute(r.Context(), input)
		if err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		ctx := context.WithValue(r.Context(), ctxkey.UserID, out.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) RedirectIfAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		input := usecases.AuthenticateSessionInput{
			Token: cookie.Value,
		}

		if _, err := h.AuthenticateSessionUseCase.Execute(r.Context(), input); err != nil {
			next.ServeHTTP(w, r)
			return
		}

		http.Redirect(w, r, "/app/scanner", http.StatusFound)
	})
}
