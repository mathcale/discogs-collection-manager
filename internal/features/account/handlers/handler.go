package handlers

import "github.com/mathcale/discogs-collection-manager/internal/features/account/usecases"

type Handler struct {
	LoadSettingsUseCase           *usecases.LoadSettings
	StartDiscogsConnectUseCase    *usecases.StartDiscogsConnect
	CompleteDiscogsConnectUseCase *usecases.CompleteDiscogsConnect
	DisconnectDiscogsUseCase      *usecases.DisconnectDiscogs
}
