package controllers

import (
	"sync"

	"github.com/dghubble/oauth1"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/core/clients/mailer"
	accountmodels "github.com/mathcale/discogs-collection-manager/internal/features/account/models"
	authmodels "github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
)

type Controller struct {
	Users         authmodels.UserRepository
	Sessions      authmodels.SessionRepository
	OTPCodes      authmodels.OTPRepository
	ResetTokens   authmodels.PasswordResetTokenRepository
	OAuthTokens   accountmodels.OAuthTokenRepository
	Mailer        mailer.Mailer
	Discogs       discogs.DiscogsClient
	DiscogsOAuth  *oauth1.Config
	BaseURL       string
	IdentityURL   string
	requestTokens sync.Map

	TwoStepAuthEnabled bool
	SignupsEnabled     bool
}
