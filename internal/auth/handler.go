package auth

import (
	"context"
	"errors"
	"time"

	"authapi/api/auth/authapi"
	"authapi/internal/validate"
)

// Handler adapts the generated auth API types to the auth Service.
// It implements authapi.StrictServerInterface.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(ctx context.Context, req authapi.RegisterRequestObject) (authapi.RegisterResponseObject, error) {
	err := h.svc.Register(ctx, req.Body.Email, req.Body.Name, req.Body.Password)

	var ve *validate.Error
	switch {
	case err == nil:
		return authapi.Register201JSONResponse{Message: "registered"}, nil
	case errors.As(err, &ve):
		return authapi.Register400JSONResponse{Error: ve.Message}, nil
	case errors.Is(err, ErrEmailTaken):
		return authapi.Register409JSONResponse{Error: ErrEmailTaken.Error()}, nil
	default:
		return nil, err
	}
}

func (h *Handler) Login(ctx context.Context, req authapi.LoginRequestObject) (authapi.LoginResponseObject, error) {
	tok, err := h.svc.Login(ctx, req.Body.Email, req.Body.Password)
	switch {
	case err == nil:
		return authapi.Login200JSONResponse{
			AccessToken: tok.Value,
			TokenType:   "Bearer",
			ExpiresIn:   int(time.Until(tok.ExpiresAt).Seconds()),
		}, nil
	case errors.Is(err, ErrInvalidCredentials):
		return authapi.Login401JSONResponse{Error: ErrInvalidCredentials.Error()}, nil
	default:
		return nil, err
	}
}

func (h *Handler) Logout(ctx context.Context, _ authapi.LogoutRequestObject) (authapi.LogoutResponseObject, error) {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		return authapi.Logout401JSONResponse{Error: "unauthorized"}, nil
	}
	if err := h.svc.Logout(ctx, p); err != nil {
		return nil, err
	}
	return authapi.Logout200JSONResponse{Message: "logged out"}, nil
}

func (h *Handler) ChangePassword(ctx context.Context, req authapi.ChangePasswordRequestObject) (authapi.ChangePasswordResponseObject, error) {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		return authapi.ChangePassword401JSONResponse{Error: "unauthorized"}, nil
	}

	err := h.svc.ChangePassword(ctx, p.UserID, req.Body.CurrentPassword, req.Body.NewPassword)

	var ve *validate.Error
	switch {
	case err == nil:
		return authapi.ChangePassword200JSONResponse{Message: "password changed, please log in again"}, nil
	case errors.As(err, &ve):
		return authapi.ChangePassword400JSONResponse{Error: ve.Message}, nil
	case errors.Is(err, ErrIncorrectPassword):
		return authapi.ChangePassword403JSONResponse{Error: ErrIncorrectPassword.Error()}, nil
	case errors.Is(err, ErrInvalidToken):
		return authapi.ChangePassword401JSONResponse{Error: ErrInvalidToken.Error()}, nil
	default:
		return nil, err
	}
}
