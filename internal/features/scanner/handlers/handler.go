package handlers

import "github.com/mathcale/discogs-collection-manager/internal/features/scanner/usecases"

type Handler struct {
	LookupBarcode   *usecases.LookupBarcode
	AddToCollection *usecases.AddToCollection
}
