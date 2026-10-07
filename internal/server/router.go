package server

import (
	"log"
	"net/http"

	"authapi/internal/api"
)

func newRouter(handlers api.StrictServerInterface, authn Authenticator) http.Handler {
	mux := http.NewServeMux()

	// 1. wrap our typed handlers so they speak plain net/http
	strict := api.NewStrictHandlerWithOptions(handlers, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(w, http.StatusBadRequest, "invalid request body")
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			// never leak internal error details to clients
			log.Printf("handler error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		},
	})

	// 2. register every route declared in openapi.yaml on the mux
	api.HandlerWithOptions(strict, api.StdHTTPServerOptions{
		BaseRouter:  mux,
		Middlewares: []api.MiddlewareFunc{requireAuth(authn)},
	})

	// 3. routes that are not part of the API contract
	mux.HandleFunc("GET /openapi.yaml", serveSpec)
	mux.HandleFunc("GET /docs", serveSwaggerUI)

	return recoverer(logRequests(limitBody(mux)))
}
