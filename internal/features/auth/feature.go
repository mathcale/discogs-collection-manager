package auth

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/mailer"
	accountmodels "github.com/mathcale/discogs-collection-manager/internal/features/account/models"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/handlers"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/usecases"
)

type Deps struct {
	DB                 *sql.DB
	Mailer             mailer.Mailer
	BaseURL            string
	TwoStepAuthEnabled bool
	SignupsEnabled     bool
}

type Module struct {
	handler *handlers.Handler
}

func New(d Deps) *Module {
	users := models.NewUserRepository(d.DB)
	sessions := models.NewSessionRepository(d.DB)
	otpCodes := models.NewOTPRepository(d.DB)
	resetTokens := models.NewPasswordResetTokenRepository(d.DB)
	oauthTokens := accountmodels.NewOAuthTokenRepository(d.DB)

	return &Module{handler: &handlers.Handler{
		Register:             &usecases.Register{Users: users},
		Login:                &usecases.Login{Users: users, Sessions: sessions, OTPCodes: otpCodes, Mailer: d.Mailer, TwoStepAuthEnabled: d.TwoStepAuthEnabled},
		VerifyOTP:            &usecases.VerifyOTP{OTPCodes: otpCodes, Sessions: sessions},
		Logout:               &usecases.Logout{Sessions: sessions},
		RequestPasswordReset: &usecases.RequestPasswordReset{Users: users, ResetTokens: resetTokens, Mailer: d.Mailer, BaseURL: d.BaseURL},
		ValidateResetToken:   &usecases.ValidateResetToken{ResetTokens: resetTokens},
		ResetPassword:        &usecases.ResetPassword{Users: users, ResetTokens: resetTokens, Sessions: sessions, OAuthTokens: oauthTokens},
		AuthenticateSession:  &usecases.AuthenticateSession{Sessions: sessions},
		SignupsEnabled:       d.SignupsEnabled,
	}}
}

func (m *Module) Routes(r chi.Router) {
	r.Get("/auth/register", m.handler.RegisterPage)
	r.Post("/auth/register", m.handler.RegisterSubmit)
	r.Get("/auth/login", m.handler.LoginPage)
	r.Post("/auth/login", m.handler.LoginSubmit)
	r.Get("/auth/login/otp", m.handler.OTPPage)
	r.Post("/auth/login/otp", m.handler.VerifyOTPSubmit)
	r.Get("/auth/forgot-password", m.handler.ForgotPasswordPage)
	r.Post("/auth/forgot-password", m.handler.ForgotPasswordSubmit)
	r.Get("/auth/reset-password", m.handler.ResetPasswordPage)
	r.Post("/auth/reset-password", m.handler.ResetPasswordSubmit)

	r.Group(func(r chi.Router) {
		r.Use(m.handler.RequireAuth)
		r.Post("/auth/logout", m.handler.LogoutSubmit)
	})
}

func (m *Module) RequireAuth(next http.Handler) http.Handler {
	return m.handler.RequireAuth(next)
}
