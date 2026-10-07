package user

import (
	"context"
	"errors"

	"authapi/internal/api"
	"authapi/internal/auth"
)

// Handler adapts the generated API types to the user Service.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return api.GetMe401JSONResponse{Error: "unauthorized"}, nil
	}

	profile, err := h.svc.GetProfile(ctx, p.UserID)
	switch {
	case err == nil:
		return api.GetMe200JSONResponse{Email: profile.Email, Name: profile.Name, CreatedAt: profile.CreatedAt}, nil
	case errors.Is(err, ErrNotFound):
		return api.GetMe404JSONResponse{Error: ErrNotFound.Error()}, nil
	default:
		return nil, err
	}
}
