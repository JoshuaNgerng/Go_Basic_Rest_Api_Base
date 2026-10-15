package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"authapi/internal/db"
	"authapi/internal/validate"
)

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already in use")
)

// Profile is the user info we expose (deliberately without the primary key).
type Profile struct {
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
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
		CreatedAt: info.CreatedAt, UpdatedAt: info.UpdatedAt,
	}, nil
}

// UpdateProfile changes only the fields that are non-nil.
func (s *Service) UpdateProfile(ctx context.Context, userID int64, email, name *string) (Profile, error) {
	params := db.UpdateUserInfoByIdParams{ID: userID}

	if email != nil {
		e, err := validate.Email(*email)
		if err != nil {
			return Profile{}, err
		}
		params.Email = pgtype.Text{String: e, Valid: true}
	}
	if name != nil {
		n, err := validate.Name(*name)
		if err != nil {
			return Profile{}, err
		}
		params.Name = pgtype.Text{String: n, Valid: true}
	}

	info, err := s.q.UpdateUserInfoById(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return Profile{}, ErrEmailTaken
		}
		return Profile{}, fmt.Errorf("update user info: %w", err)
	}
	return Profile{
		Email: info.Email, Name: info.Name,
		CreatedAt: info.CreatedAt, UpdatedAt: info.UpdatedAt,
	}, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID int64) (int64, error) {
	id, err := s.q.DeleteUserById(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("delete user: %w", err)
	}
	return id, nil
}
