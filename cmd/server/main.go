package main

import (
	"log"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/kernel"
	"github.com/mathcale/discogs-collection-manager/internal/web"
)

func main() {
	k, err := kernel.New()
	if err != nil {
		log.Fatalf("failed to initialise kernel: %v", err)
	}

	defer k.Close()

	if err := web.NewServer(k).Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
