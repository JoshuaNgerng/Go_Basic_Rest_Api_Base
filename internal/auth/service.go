package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"authapi/internal/db"
)

// Domain errors. The HTTP layer decides which status code each one maps to.
var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenRevoked       = errors.New("token has been revoked")
)

// ValidationError means the caller sent unacceptable input.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// Token is what a successful login produces.
type Token struct {
	Value     string
	ExpiresAt time.Time
}

type Service struct {
	q          *db.Queries
	jwt        *JWTManager
	iterations int
	dummyHash  string // used to equalize timing when the email doesn't exist
}

func NewService(q *db.Queries, jwt *JWTManager, iterations int) (*Service, error) {
	dummy, err := HashPassword("dummy-password", iterations)
	if err != nil {
		return nil, err
	}
	return &Service{q: q, jwt: jwt, iterations: iterations, dummyHash: dummy}, nil
}

func (s *Service) Register(ctx context.Context, email, name, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)

	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return &ValidationError{"invalid email"}
	}
	if name == "" {
		return &ValidationError{"name is required"}
	}
	if len(password) < 8 || len(password) > 128 {
		return &ValidationError{"password must be 8-128 characters"}
	}

	hash, err := HashPassword(password, s.iterations)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = s.q.CreateUser(ctx, db.CreateUserParams{Email: email, Name: name, PasswordHash: hash})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return ErrEmailTaken
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *Service) Login(ctx context.Context, email, password string) (Token, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		_, _ = VerifyPassword(password, s.dummyHash) // burn the same CPU time
		return Token{}, ErrInvalidCredentials
	}
	if err != nil {
		return Token{}, fmt.Errorf("get user: %w", err)
	}

	ok, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !ok {
		return Token{}, ErrInvalidCredentials
	}

	value, exp, err := s.jwt.Issue(user.ID)
	if err != nil {
		return Token{}, fmt.Errorf("issue token: %w", err)
	}
	return Token{Value: value, ExpiresAt: exp}, nil
}

// Logout revokes the token the caller authenticated with.
func (s *Service) Logout(ctx context.Context, p Principal) error {
	err := s.q.RevokeToken(ctx, db.RevokeTokenParams{Jti: p.TokenID, ExpiresAt: p.ExpiresAt})
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	return nil
}

// Authenticate verifies a raw JWT and checks it has not been revoked.
// It is used by the HTTP middleware.
func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	claims, err := s.jwt.Parse(token)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}

	revoked, err := s.q.IsTokenRevoked(ctx, claims.ID)
	if err != nil {
		return Principal{}, fmt.Errorf("check revoked: %w", err)
	}
	if revoked {
		return Principal{}, ErrTokenRevoked
	}

	p := Principal{UserID: userID, TokenID: claims.ID}
	if claims.ExpiresAt != nil {
		p.ExpiresAt = claims.ExpiresAt.Time
	}
	return p, nil
}

// PurgeExpiredTokens removes revoked-token rows that no longer matter.
func (s *Service) PurgeExpiredTokens(ctx context.Context) error {
	return s.q.DeleteExpiredRevokedTokens(ctx)
}
