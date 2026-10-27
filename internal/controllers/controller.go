package controllers

import (
	"net/http"
	"sync"

	"github.com/a-h/templ"
	"github.com/dghubble/oauth1"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/core/clients/mailer"
	accountmodels "github.com/mathcale/discogs-collection-manager/internal/features/account/models"
	authmodels "github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/ctxkey"
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

func render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	component.Render(r.Context(), w)
}

func currentUserID(r *http.Request) string {
	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	return userID
}
