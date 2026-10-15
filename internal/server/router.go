package server

import (
	"net/http"

	"authapi/api/auth/authapi"
	"authapi/api/user/userapi"
)

func newRouter(authH authapi.StrictServerInterface, userH userapi.StrictServerInterface, authn Authenticator) http.Handler {
	mux := http.NewServeMux()
	authMW := requireAuth(authn)

	// Each domain is wired the same way, onto the SAME mux:
	//   1. wrap the typed handler so it speaks plain net/http
	//   2. register that domain's routes (from its own openapi.yaml) on the mux
	authapi.HandlerWithOptions(
		authapi.NewStrictHandlerWithOptions(authH, nil, authapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  badRequest,
			ResponseErrorHandlerFunc: internalError,
		}),
		authapi.StdHTTPServerOptions{
			BaseRouter:  mux,
			Middlewares: []authapi.MiddlewareFunc{authMW},
		},
	)
	userapi.HandlerWithOptions(
		userapi.NewStrictHandlerWithOptions(userH, nil, userapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  badRequest,
			ResponseErrorHandlerFunc: internalError,
		}),
		userapi.StdHTTPServerOptions{
			BaseRouter:  mux,
			Middlewares: []userapi.MiddlewareFunc{authMW},
		},
	)

	// routes that are not part of any API contract
	mux.Handle("GET /openapi/", serveSpecs())
	mux.HandleFunc("GET /docs", serveSwaggerUI)

	return recoverer(logRequests(limitBody(mux)))
}
