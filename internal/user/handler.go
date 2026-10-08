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
		return api.GetMe200JSONResponse{
			Email: profile.Email, Name: profile.Name,
			CreatedAt: profile.CreatedAt, UpdateAt: profile.UpdateAt,
		}, nil
	case errors.Is(err, ErrNotFound):
		return api.GetMe404JSONResponse{Error: ErrNotFound.Error()}, nil
	default:
		return nil, err
	}
}

func (h *Handler) UpdateMe(ctx context.Context, req api.UpdateMeRequestObject) (api.UpdateMeResponseObject, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return api.UpdateMe401JSONResponse{Error: "unauthorized"}, nil
	}

	profile, err := h.svc.UpdateProfile(ctx, p.UserID, req.Body.Email, req.Body.Name)
	switch {
	case err == nil:
		return api.UpdateMe200JSONResponse{
			Email: profile.Email, Name: profile.Name,
			CreatedAt: profile.CreatedAt, UpdateAt: profile.UpdateAt,
		}, nil
	case errors.Is(err, ErrNotFound):
		return api.UpdateMe404JSONResponse{Error: ErrNotFound.Error()}, nil
	default:
		return nil, err
	}
}

func (h *Handler) DeleteMe(ctx context.Context, _ api.DeleteMeRequestObject) (api.DeleteMeResponseObject, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return api.DeleteMe404JSONResponse{Error: "unauthorized"}, nil
	}

	id64, err := h.svc.DeleteUser(ctx, p.UserID)
	id := int(id64)
	switch {
	case err == nil:
		return api.DeleteMe200JSONResponse{DeletedId: &id}, nil
	case errors.Is(err, ErrNotFound):
		return api.DeleteMe404JSONResponse{Error: ErrNotFound.Error()}, nil
	default:
		return nil, err
	}
}
