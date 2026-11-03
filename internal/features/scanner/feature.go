package scanner

import (
	"database/sql"

	"github.com/go-chi/chi/v5"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	accountstore "github.com/mathcale/discogs-collection-manager/internal/features/account/store"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/handlers"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/usecases"
)

type (
	Deps struct {
		DB            *sql.DB
		DiscogsClient discogs.DiscogsClient
	}

	Module struct {
		handler *handlers.Handler
	}
)

func New(d Deps) *Module {
	oauthTokensRepo := accountstore.NewOAuthTokenRepository(d.DB)

	return &Module{
		handler: &handlers.Handler{
			LookupBarcodeUseCase: &usecases.LookupBarcode{
				DiscogsClient:   d.DiscogsClient,
				OAuthTokensRepo: oauthTokensRepo,
			},
			AddToCollectionUseCase: &usecases.AddToCollection{
				DiscogsClient:   d.DiscogsClient,
				OAuthTokensRepo: oauthTokensRepo,
			},
		}}
}

func (m *Module) Routes(r chi.Router) {
	r.Get("/app/scanner", m.handler.ScanPage)
	r.Post("/app/scan", m.handler.Scan)
	r.Post("/app/scan/confirm", m.handler.Confirm)
	r.Get("/app/scan/cancel", m.handler.Cancel)
}
