package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mathcale/discogs-collection-manager/internal/controllers"
	"github.com/mathcale/discogs-collection-manager/internal/core"
)

func NewRouter(k *core.Kernel) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(RequestLogger(k.Logger))
	r.Use(middleware.Recoverer)

	c := &controllers.Controller{
		Users:        k.Users,
		Sessions:     k.Sessions,
		OTPCodes:     k.OTPCodes,
		ResetTokens:  k.ResetTokens,
		OAuthTokens:  k.OAuthTokens,
		Mailer:       k.Mailer,
		DiscogsOAuth: k.DiscogsOAuth,
		BaseURL:      k.BaseURL,

		TwoStepAuthEnabled: k.TwoStepAuthEnabled,
		SignupsEnabled:     k.SignupsEnabled,
	}

	r.Group(k.Landing.Routes)

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
			r.Use(RequireAuth(k.Sessions))
			r.Post("/logout", c.Logout)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(RequireAuth(k.Sessions))
		r.Get("/app/settings", c.SettingsPage)
		k.Scanner.Routes(r)
		r.Get("/discogs/connect", c.Connect)
		r.Get("/discogs/callback", c.Callback)
		r.Post("/discogs/disconnect", c.Disconnect)
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
