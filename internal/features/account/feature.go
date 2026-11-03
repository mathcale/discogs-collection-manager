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

type (
	Deps struct {
		DB                 *sql.DB
		DiscogsOAuthConfig *oauth1.Config
		IdentityURL        string
	}

	Module struct {
		handler *handlers.Handler
	}
)

func New(d Deps) *Module {
	oauthTokensRepo := store.NewOAuthTokenRepository(d.DB)
	usersRepo := authstore.NewUserRepository(d.DB)

	pendingOAuthReqsUseCase := &usecases.PendingOAuthRequests{}

	return &Module{
		handler: &handlers.Handler{
			LoadSettingsUseCase: &usecases.LoadSettings{
				UsersRepo:       usersRepo,
				OAuthTokensRepo: oauthTokensRepo,
			},
			StartDiscogsConnectUseCase: &usecases.StartDiscogsConnect{
				OAuth:   d.DiscogsOAuthConfig,
				Pending: pendingOAuthReqsUseCase,
			},
			CompleteDiscogsConnectUseCase: &usecases.CompleteDiscogsConnect{
				OAuth:           d.DiscogsOAuthConfig,
				Pending:         pendingOAuthReqsUseCase,
				OAuthTokensRepo: oauthTokensRepo,
				IdentityURL:     d.IdentityURL,
			},
			DisconnectDiscogsUseCase: &usecases.DisconnectDiscogs{
				OAuthTokensRepo: oauthTokensRepo,
			},
		}}
}

func (m *Module) Routes(r chi.Router) {
	r.Get("/app/settings", m.handler.SettingsPage)
	r.Post("/app/settings/language", m.handler.LanguageSubmit)
	r.Get("/discogs/connect", m.handler.Connect)
	r.Get("/discogs/callback", m.handler.Callback)
	r.Post("/discogs/disconnect", m.handler.Disconnect)
}
