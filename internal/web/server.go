package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/kernel"
)

const shutdownTimeout = 30 * time.Second

type Server struct {
	http   *http.Server
	kernel *kernel.Kernel
}

func NewServer(k *kernel.Kernel) *Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Server{
		http: &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: NewRouter(k),
		},
		kernel: k,
	}
}

func (s *Server) Run() error {
	go func() {
		s.kernel.Logger.Info("server listening", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			s.kernel.Logger.Error("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.kernel.Logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced shutdown: %w", err)
	}

	s.kernel.Logger.Info("server stopped")
	return nil
}
