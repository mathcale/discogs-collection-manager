package landing

import (
	"github.com/go-chi/chi/v5"

	"github.com/mathcale/discogs-collection-manager/internal/features/landing/handlers"
)

type Deps struct {
	SignupsEnabled bool
}

type Module struct {
	handler *handlers.Handler
}

func New(d Deps) *Module {
	return &Module{handler: &handlers.Handler{SignupsEnabled: d.SignupsEnabled}}
}

func (m *Module) Routes(r chi.Router) {
	r.Get("/", m.handler.Landing)
}
