package handlers

import "github.com/mathcale/discogs-collection-manager/internal/features/account/usecases"

type Handler struct {
	LoadSettings           *usecases.LoadSettings
	StartDiscogsConnect    *usecases.StartDiscogsConnect
	CompleteDiscogsConnect *usecases.CompleteDiscogsConnect
	DisconnectDiscogs      *usecases.DisconnectDiscogs
}
