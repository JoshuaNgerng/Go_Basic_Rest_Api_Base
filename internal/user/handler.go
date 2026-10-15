package user

import (
	"context"
	"errors"

	"authapi/api/user/userapi"
	"authapi/internal/auth"
	"authapi/internal/validate"
)

// Handler adapts the generated user API types to the user Service.
// It implements userapi.StrictServerInterface.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) GetMe(ctx context.Context, _ userapi.GetMeRequestObject) (userapi.GetMeResponseObject, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return userapi.GetMe401JSONResponse{Error: "unauthorized"}, nil
	}

	profile, err := h.svc.GetProfile(ctx, p.UserID)
	switch {
	case err == nil:
		return userapi.GetMe200JSONResponse{
			Email: profile.Email, Name: profile.Name,
			CreatedAt: profile.CreatedAt, UpdatedAt: profile.UpdatedAt,
		}, nil
	case errors.Is(err, ErrNotFound):
		return userapi.GetMe404JSONResponse{Error: ErrNotFound.Error()}, nil
	default:
		return nil, err
	}
}

func (h *Handler) UpdateMe(ctx context.Context, req userapi.UpdateMeRequestObject) (userapi.UpdateMeResponseObject, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return userapi.UpdateMe401JSONResponse{Error: "unauthorized"}, nil
	}

	profile, err := h.svc.UpdateProfile(ctx, p.UserID, req.Body.Email, req.Body.Name)

	var ve *validate.Error
	switch {
	case err == nil:
		return userapi.UpdateMe200JSONResponse{
			Email: profile.Email, Name: profile.Name,
			CreatedAt: profile.CreatedAt, UpdatedAt: profile.UpdatedAt,
		}, nil
	case errors.As(err, &ve):
		return userapi.UpdateMe400JSONResponse{Error: ve.Message}, nil
	case errors.Is(err, ErrNotFound):
		return userapi.UpdateMe404JSONResponse{Error: ErrNotFound.Error()}, nil
	case errors.Is(err, ErrEmailTaken):
		return userapi.UpdateMe409JSONResponse{Error: ErrEmailTaken.Error()}, nil
	default:
		return nil, err
	}
}

func (h *Handler) DeleteMe(ctx context.Context, _ userapi.DeleteMeRequestObject) (userapi.DeleteMeResponseObject, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return userapi.DeleteMe401JSONResponse{Error: "unauthorized"}, nil
	}

	id, err := h.svc.DeleteUser(ctx, p.UserID)
	switch {
	case err == nil:
		return userapi.DeleteMe200JSONResponse{DeletedId: id}, nil
	case errors.Is(err, ErrNotFound):
		return userapi.DeleteMe404JSONResponse{Error: ErrNotFound.Error()}, nil
	default:
		return nil, err
	}
}
