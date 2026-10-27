package controllers

import (
	"github.com/mathcale/discogs-collection-manager/internal/core/clients/mailer"
	accountmodels "github.com/mathcale/discogs-collection-manager/internal/features/account/models"
	authmodels "github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
)

type Controller struct {
	Users       authmodels.UserRepository
	Sessions    authmodels.SessionRepository
	OTPCodes    authmodels.OTPRepository
	ResetTokens authmodels.PasswordResetTokenRepository
	OAuthTokens accountmodels.OAuthTokenRepository
	Mailer      mailer.Mailer
	BaseURL     string

	TwoStepAuthEnabled bool
	SignupsEnabled     bool
}
