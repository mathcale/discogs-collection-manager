package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mathcale/discogs-collection-manager/internal/handler"
	"github.com/mathcale/discogs-collection-manager/internal/kernel"
)

// New builds and returns the application router with all routes registered.
func New(k *kernel.Kernel) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", handler.Landing)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	return r
}
