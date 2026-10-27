package scanner

import (
	"database/sql"

	"github.com/go-chi/chi/v5"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	accountmodels "github.com/mathcale/discogs-collection-manager/internal/features/account/models"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/handlers"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner/usecases"
)

type Deps struct {
	DB      *sql.DB
	Discogs discogs.DiscogsClient
}

type Module struct {
	handler *handlers.Handler
}

func New(d Deps) *Module {
	oauthTokens := accountmodels.NewOAuthTokenRepository(d.DB)

	return &Module{handler: &handlers.Handler{
		LookupBarcode:   &usecases.LookupBarcode{Discogs: d.Discogs, OAuthTokens: oauthTokens},
		AddToCollection: &usecases.AddToCollection{Discogs: d.Discogs, OAuthTokens: oauthTokens},
	}}
}

func (m *Module) Routes(r chi.Router) {
	r.Get("/app/scanner", m.handler.ScanPage)
	r.Post("/app/scan", m.handler.Scan)
	r.Post("/app/scan/confirm", m.handler.Confirm)
	r.Get("/app/scan/cancel", m.handler.Cancel)
}
