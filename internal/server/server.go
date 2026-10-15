package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"authapi/internal/auth"
	"authapi/internal/user"
)

type Server struct {
	httpServer *http.Server
}

// New builds the handlers from the services and mounts them on one router.
// Each domain has its own generated interface, so there is nothing to combine:
// auth.Handler serves the auth routes and user.Handler serves the user routes.
func New(port string, authSvc *auth.Service, userSvc *user.Service) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              ":" + port,
			Handler:           newRouter(auth.NewHandler(authSvc), user.NewHandler(userSvc), authSvc),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Run blocks until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s  (docs: /docs)", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
