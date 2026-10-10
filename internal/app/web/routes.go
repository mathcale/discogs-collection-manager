package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mathcale/discogs-collection-manager/internal/app"
	"github.com/mathcale/discogs-collection-manager/internal/controllers"
)

func NewRouter(a *app.App) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(RequestLogger(a.Logger))
	r.Use(middleware.Recoverer)

	c := &controllers.Controller{
		Users:        a.Users,
		Sessions:     a.Sessions,
		OTPCodes:     a.OTPCodes,
		ResetTokens:  a.ResetTokens,
		OAuthTokens:  a.OAuthTokens,
		Mailer:       a.Mailer,
		Discogs:      a.Discogs,
		DiscogsOAuth: a.DiscogsOAuth,
		BaseURL:      a.BaseURL,
	}

	r.Get("/", c.Landing)

	r.Route("/auth", func(r chi.Router) {
		r.Get("/register", c.RegisterPage)
		r.Post("/register", c.Register)
		r.Get("/login", c.LoginPage)
		r.Post("/login", c.Login)
		r.Get("/login/otp", c.OTPPage)
		r.Post("/login/otp", c.VerifyOTP)
		r.Get("/forgot-password", c.ForgotPasswordPage)
		r.Post("/forgot-password", c.ForgotPassword)
		r.Get("/reset-password", c.ResetPasswordPage)
		r.Post("/reset-password", c.ResetPassword)

		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(a.Sessions))
			r.Post("/logout", c.Logout)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(RequireAuth(a.Sessions))
		r.Get("/app/settings", c.SettingsPage)
		r.Get("/app/scanner", c.ScanPage)
		r.Post("/app/scan", c.Scan)
		r.Post("/app/scan/confirm", c.Confirm)
		r.Get("/app/scan/cancel", c.Cancel)
		r.Get("/discogs/connect", c.Connect)
		r.Get("/discogs/callback", c.Callback)
		r.Post("/discogs/disconnect", c.Disconnect)
	})

	r.Handle("/static/*", staticCache(http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))))

	return r
}

func staticCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=300")
		next.ServeHTTP(w, r)
	})
}
