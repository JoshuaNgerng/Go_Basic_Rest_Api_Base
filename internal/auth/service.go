package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"authapi/internal/db"
	"authapi/internal/validate"
)

// Domain errors. The HTTP layer decides which status code each one maps to.
var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrIncorrectPassword  = errors.New("current password is incorrect")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenRevoked       = errors.New("token has been revoked")
)

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
	email, err := validate.Email(email)
	if err != nil {
		return err
	}
	name, err = validate.Name(name)
	if err != nil {
		return err
	}
	if err := validate.Password(password); err != nil {
		return err
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

// ChangePassword replaces the caller's password. It requires the current
// password, and afterwards every token issued before now is rejected by
// Authenticate (see password_changed_at), so the user must log in again.
func (s *Service) ChangePassword(ctx context.Context, userID int64, current, next string) error {
	if err := validate.Password(next); err != nil {
		return err
	}

	hash, err := s.q.GetUserPasswordHashById(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidToken // the user no longer exists
	}
	if err != nil {
		return fmt.Errorf("get password hash: %w", err)
	}

	ok, err := VerifyPassword(current, hash)
	if err != nil || !ok {
		return ErrIncorrectPassword
	}

	newHash, err := HashPassword(next, s.iterations)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	err = s.q.UpdateUserPasswordById(ctx, db.UpdateUserPasswordByIdParams{ID: userID, PasswordHash: newHash})
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

// Authenticate verifies a raw JWT, checks it has not been revoked, and checks
// it was issued after the user's last password change. It is used by the HTTP
// middleware.
func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	claims, err := s.jwt.Parse(token)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || claims.IssuedAt == nil {
		return Principal{}, ErrInvalidToken
	}

	revoked, err := s.q.IsTokenRevoked(ctx, claims.ID)
	if err != nil {
		return Principal{}, fmt.Errorf("check revoked: %w", err)
	}
	if revoked {
		return Principal{}, ErrTokenRevoked
	}

	changedAt, err := s.q.GetPasswordChangedAt(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrInvalidToken // the user was deleted
	}
	if err != nil {
		return Principal{}, fmt.Errorf("get password_changed_at: %w", err)
	}
	// JWT timestamps have one-second precision, so compare at that precision.
	if claims.IssuedAt.Time.Before(changedAt.Truncate(time.Second)) {
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
