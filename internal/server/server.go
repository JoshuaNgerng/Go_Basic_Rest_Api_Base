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

// Both feature handlers are called "Handler", and Go does not allow two
// embedded fields with the same name, so we give them local aliases.
type (
	authHandlers = auth.Handler
	userHandlers = user.Handler
)

// apiHandler implements api.StrictServerInterface by embedding the feature
// handlers: each one contributes the operations it implements.
type apiHandler struct {
	*authHandlers
	*userHandlers
}

type Server struct {
	httpServer *http.Server
}

func New(port string, authSvc *auth.Service, userSvc *user.Service) *Server {
	handlers := apiHandler{
		authHandlers: auth.NewHandler(authSvc),
		userHandlers: user.NewHandler(userSvc),
	}
	return &Server{
		httpServer: &http.Server{
			Addr:              ":" + port,
			Handler:           newRouter(handlers, authSvc),
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
