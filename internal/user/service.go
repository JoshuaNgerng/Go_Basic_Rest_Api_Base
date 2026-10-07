package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"authapi/internal/db"
)

var ErrNotFound = errors.New("user not found")

// Profile is the user info we expose (deliberately without the primary key).
type Profile struct {
	Email     string
	Name      string
	CreatedAt time.Time
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
	return Profile{Email: info.Email, Name: info.Name, CreatedAt: info.CreatedAt}, nil
}
