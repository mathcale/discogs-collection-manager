package main

import (
	"log"

	"github.com/mathcale/discogs-collection-manager/internal/core"
	"github.com/mathcale/discogs-collection-manager/internal/core/web"
)

func main() {
	k, err := core.New()
	if err != nil {
		log.Fatalf("failed to initialise app: %v", err)
	}

	defer k.Close()

	if err := web.NewServer(k).Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
