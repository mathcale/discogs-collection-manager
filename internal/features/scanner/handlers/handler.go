package handlers

import "github.com/mathcale/discogs-collection-manager/internal/features/scanner/usecases"

type Handler struct {
	LookupBarcodeUseCase   *usecases.LookupBarcode
	AddToCollectionUseCase *usecases.AddToCollection
}
