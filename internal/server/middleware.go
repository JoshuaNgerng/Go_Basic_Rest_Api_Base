package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"authapi/internal/api"
	"authapi/internal/auth"
)

// Authenticator is the only thing the middleware needs from the auth feature.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (auth.Principal, error)
}

// requireAuth enforces the `security: bearerAuth` declarations in the spec.
// The generated wrapper puts api.BearerAuthScopes into the request context for
// every operation that declares it, so protected routes are defined in the
// YAML, not repeated in Go.
func requireAuth(authn Authenticator) api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Context().Value(api.BearerAuthScopes) == nil {
				next.ServeHTTP(w, r) // public operation
				return
			}

			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			p, err := authn.Authenticate(r.Context(), token)
			switch {
			case errors.Is(err, auth.ErrInvalidToken):
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			case errors.Is(err, auth.ErrTokenRevoked):
				writeError(w, http.StatusUnauthorized, "token has been revoked")
				return
			case err != nil:
				log.Printf("authenticate: %v", err)
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.ContextWithPrincipal(r.Context(), p)))
		})
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v", rec)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}
