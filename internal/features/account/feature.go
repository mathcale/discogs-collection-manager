package account

import (
	"database/sql"

	"github.com/dghubble/oauth1"
	"github.com/go-chi/chi/v5"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/handlers"
	"github.com/mathcale/discogs-collection-manager/internal/features/account/store"
	"github.com/mathcale/discogs-collection-manager/internal/features/account/usecases"
	authstore "github.com/mathcale/discogs-collection-manager/internal/features/auth/store"
)

type Deps struct {
	DB           *sql.DB
	DiscogsOAuth *oauth1.Config
	IdentityURL  string
}

type Module struct {
	handler *handlers.Handler
}

func New(d Deps) *Module {
	oauthTokens := store.NewOAuthTokenRepository(d.DB)
	users := authstore.NewUserRepository(d.DB)
	pending := &usecases.PendingOAuthRequests{}

	return &Module{handler: &handlers.Handler{
		LoadSettings:           &usecases.LoadSettings{Users: users, OAuthTokens: oauthTokens},
		StartDiscogsConnect:    &usecases.StartDiscogsConnect{OAuth: d.DiscogsOAuth, Pending: pending},
		CompleteDiscogsConnect: &usecases.CompleteDiscogsConnect{OAuth: d.DiscogsOAuth, Pending: pending, OAuthTokens: oauthTokens, IdentityURL: d.IdentityURL},
		DisconnectDiscogs:      &usecases.DisconnectDiscogs{OAuthTokens: oauthTokens},
	}}
}

func (m *Module) Routes(r chi.Router) {
	r.Get("/app/settings", m.handler.SettingsPage)
	r.Post("/app/settings/language", m.handler.LanguageSubmit)
	r.Get("/discogs/connect", m.handler.Connect)
	r.Get("/discogs/callback", m.handler.Callback)
	r.Post("/discogs/disconnect", m.handler.Disconnect)
}
