package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"authapi/api/auth/authapi"
	"authapi/api/user/userapi"
	"authapi/internal/auth"
)

// Authenticator is the only thing the middleware needs from the auth feature.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (auth.Principal, error)
}

// isProtected reports whether the matched operation declares `security` in its
// spec. Each generated wrapper puts its BearerAuthScopes key into the request
// context for such operations. Every domain has its own copy of the constant,
// so we accept any of them.
func isProtected(ctx context.Context) bool {
	return ctx.Value(authapi.BearerAuthScopes) != nil ||
		ctx.Value(userapi.BearerAuthScopes) != nil
}

// requireAuth enforces the `security: bearerAuth` declarations in the specs, so
// protected routes are defined in the YAML and not repeated in Go. It returns a
// plain func so it can be used as authapi.MiddlewareFunc and userapi.MiddlewareFunc.
func requireAuth(authn Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isProtected(r.Context()) {
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
