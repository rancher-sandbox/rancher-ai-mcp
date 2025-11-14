package middleware

import (
	"context"
)

// contextKey is a custom type for context keys to avoid collisions with
// other packages that might use string keys. Using a struct pointer ensures
// uniqueness since each instance has a unique memory address.
type contextKey struct{ name string }

var (
	// tokenCtxKey is the context key for storing the JWT bearer token.
	// It's unexported to prevent external packages from accessing it directly.
	tokenCtxKey = &contextKey{"token"}
)

// Token context helpers.

// WithToken sets the token into the context.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenCtxKey, token)
}

// Token gets the token from the context.
//
// Returns empty string if no token is found.
func Token(ctx context.Context) string {
	token, ok := ctx.Value(tokenCtxKey).(string)
	if ok {
		return token
	}

	return ""
}
