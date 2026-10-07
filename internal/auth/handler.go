package auth

import (
	"context"
	"errors"
	"time"

	"authapi/internal/api"
)

// Handler adapts the generated API types to the auth Service.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(ctx context.Context, req api.RegisterRequestObject) (api.RegisterResponseObject, error) {
	err := h.svc.Register(ctx, req.Body.Email, req.Body.Name, req.Body.Password)

	var ve *ValidationError
	switch {
	case err == nil:
		return api.Register201JSONResponse{Message: "registered"}, nil
	case errors.As(err, &ve):
		return api.Register400JSONResponse{Error: ve.Message}, nil
	case errors.Is(err, ErrEmailTaken):
		return api.Register409JSONResponse{Error: ErrEmailTaken.Error()}, nil
	default:
		return nil, err
	}
}

func (h *Handler) Login(ctx context.Context, req api.LoginRequestObject) (api.LoginResponseObject, error) {
	tok, err := h.svc.Login(ctx, req.Body.Email, req.Body.Password)
	switch {
	case err == nil:
		return api.Login200JSONResponse{
			AccessToken: tok.Value,
			TokenType:   "Bearer",
			ExpiresIn:   int(time.Until(tok.ExpiresAt).Seconds()),
		}, nil
	case errors.Is(err, ErrInvalidCredentials):
		return api.Login401JSONResponse{Error: ErrInvalidCredentials.Error()}, nil
	default:
		return nil, err
	}
}

func (h *Handler) Logout(ctx context.Context, _ api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		return api.Logout401JSONResponse{Error: "unauthorized"}, nil
	}
	if err := h.svc.Logout(ctx, p); err != nil {
		return nil, err
	}
	return api.Logout200JSONResponse{Message: "logged out"}, nil
}
