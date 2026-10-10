package main

import (
	"log"

	"github.com/mathcale/discogs-collection-manager/internal/app"
	"github.com/mathcale/discogs-collection-manager/internal/app/web"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatalf("failed to initialise app: %v", err)
	}

	defer a.Close()

	if err := web.NewServer(a).Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
