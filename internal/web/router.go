package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mathcale/discogs-collection-manager/internal/handler"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/kernel"
)

func NewRouter(k *kernel.Kernel) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	authHandler := handler.NewAuthHandler(k.Users, k.Sessions)
	discogsOAuthHandler := handler.NewDiscogsOAuthHandler(k.OAuthTokens, k.Sessions, k.Users, k.DiscogsOAuth)
	scanHandler := handler.NewScanHandler()

	r.Get("/", handler.Landing)
	r.Get("/register", authHandler.RegisterPage)
	r.Post("/register", authHandler.Register)
	r.Get("/login", authHandler.LoginPage)
	r.Post("/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(RequireAuth(k.Sessions))
		r.Post("/logout", authHandler.Logout)
		r.Get("/app/settings", discogsOAuthHandler.SettingsPage)
		r.Get("/app/scanner", scanHandler.ScanPage)
		r.Post("/app/scan", scanHandler.Scan)
		r.Get("/auth/discogs/connect", discogsOAuthHandler.Connect)
		r.Get("/auth/discogs/callback", discogsOAuthHandler.Callback)
		r.Post("/auth/discogs/disconnect", discogsOAuthHandler.Disconnect)
	})

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	return r
}
