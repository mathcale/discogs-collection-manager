package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mathcale/discogs-collection-manager/internal/kernel"
	"github.com/mathcale/discogs-collection-manager/internal/router"
)

func main() {
	k, err := kernel.New()
	if err != nil {
		log.Fatalf("failed to initialise kernel: %v", err)
	}
	defer k.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("server listening on %s", addr)

	if err := http.ListenAndServe(addr, router.New(k)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
