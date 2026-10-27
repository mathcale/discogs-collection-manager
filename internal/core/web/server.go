package web

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/core"
)

const shutdownTimeout = 30 * time.Second

type Server struct {
	http   *http.Server
	kernel *core.Kernel
}

func NewServer(k *core.Kernel) *Server {
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
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	tls := certFile != "" && keyFile != ""
	scheme := "http"

	if tls {
		scheme = "https"
	}

	go func() {
		if os.Getenv("ENVIRONMENT") == "development" {
			if ip := localIP(); ip != "" {
				port := s.http.Addr[1:]

				s.kernel.Logger.Info(
					"server listening",
					"addr", s.http.Addr,
					"network", fmt.Sprintf("%s://%s:%s", scheme, ip, port),
				)
			} else {
				s.kernel.Logger.Info("server listening", "addr", s.http.Addr)
			}
		} else {
			s.kernel.Logger.Info("server listening", "addr", s.http.Addr)
		}

		var err error
		if tls {
			err = s.http.ListenAndServeTLS(certFile, keyFile)
		} else {
			err = s.http.ListenAndServe()
		}

		if !errors.Is(err, http.ErrServerClosed) {
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

func localIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}

	return ""
}
