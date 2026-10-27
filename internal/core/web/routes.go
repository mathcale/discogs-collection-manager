package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mathcale/discogs-collection-manager/internal/core"
)

func NewRouter(k *core.Kernel) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(RequestLogger(k.Logger))
	r.Use(middleware.Recoverer)

	r.Group(func(r chi.Router) {
		r.Use(k.Auth.RedirectIfAuthenticated)
		k.Landing.Routes(r)
		k.Auth.Routes(r)
	})
	r.Group(func(r chi.Router) {
		r.Use(k.Auth.RequireAuth)
		k.Account.Routes(r)
		k.Scanner.Routes(r)
	})

	r.Handle("/static/*", staticCache(http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))))

	return http.NewCrossOriginProtection().Handler(r)
}

func staticCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=300")
		next.ServeHTTP(w, r)
	})
}
