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

	r.Use(middleware.RequestID)
	r.Use(RequestLogger(k.Logger))
	r.Use(middleware.Recoverer)

	authHandler := handler.NewAuthHandler(k.Users, k.Sessions, k.OTPCodes, k.ResetTokens, k.Mailer, k.BaseURL, k.TwoStepAuthEnabled, k.SignupsEnabled)
	discogsOAuthHandler := handler.NewDiscogsOAuthHandler(k.OAuthTokens, k.Sessions, k.Users, k.DiscogsOAuth)
	scanHandler := handler.NewScanHandler(k.Discogs, k.OAuthTokens)

	r.Get("/", handler.Landing(k.SignupsEnabled))

	r.Route("/auth", func(r chi.Router) {
		r.Get("/register", authHandler.RegisterPage)
		r.Post("/register", authHandler.Register)
		r.Get("/login", authHandler.LoginPage)
		r.Post("/login", authHandler.Login)
		r.Get("/login/otp", authHandler.OTPPage)
		r.Post("/login/otp", authHandler.VerifyOTP)
		r.Get("/forgot-password", authHandler.ForgotPasswordPage)
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Get("/reset-password", authHandler.ResetPasswordPage)
		r.Post("/reset-password", authHandler.ResetPassword)

		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(k.Sessions))
			r.Post("/logout", authHandler.Logout)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(RequireAuth(k.Sessions))
		r.Get("/app/settings", discogsOAuthHandler.SettingsPage)
		r.Get("/app/scanner", scanHandler.ScanPage)
		r.Post("/app/scan", scanHandler.Scan)
		r.Post("/app/scan/confirm", scanHandler.Confirm)
		r.Get("/app/scan/cancel", scanHandler.Cancel)
		r.Get("/discogs/connect", discogsOAuthHandler.Connect)
		r.Get("/discogs/callback", discogsOAuthHandler.Callback)
		r.Post("/discogs/disconnect", discogsOAuthHandler.Disconnect)
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
