package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"authapi/internal/db"
)

var ErrNotFound = errors.New("user not found")

// Profile is the user info we expose (deliberately without the primary key).
type Profile struct {
	Email     string
	Name      string
	CreatedAt time.Time
	UpdateAt  time.Time
}

type Service struct{ q *db.Queries }

func NewService(q *db.Queries) *Service { return &Service{q: q} }

func (s *Service) GetProfile(ctx context.Context, userID int64) (Profile, error) {
	info, err := s.q.GetUserInfo(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("get user info: %w", err)
	}
	return Profile{
		Email: info.Email, Name: info.Name,
		CreatedAt: info.CreatedAt, UpdateAt: info.UpdatedAt,
	}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, email, name *string) (Profile, error) {
	params := db.UpdateUserInfoByIdParams{
		ID: userID,
	}
	if email != nil {
		params.Email = pgtype.Text{
			String: *email,
			Valid:  true,
		}
	}

	if name != nil {
		params.Name = pgtype.Text{
			String: *name,
			Valid:  true,
		}
	}
	info, err := s.q.UpdateUserInfoById(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("update user info: %w", err)
	}
	return Profile{Email: info.Email, Name: info.Name, CreatedAt: info.CreatedAt}, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID int64) (int64, error) {
	id, err := s.q.DeleteUserById(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("get user info: %w", err)
	}
	return id, nil
}
