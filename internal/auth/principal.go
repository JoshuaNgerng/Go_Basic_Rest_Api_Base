package auth

import (
	"context"
	"time"
)

// Principal is the authenticated caller, derived from a verified JWT.
type Principal struct {
	UserID    int64
	TokenID   string
	ExpiresAt time.Time
}

type principalKey struct{}

func ContextWithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
