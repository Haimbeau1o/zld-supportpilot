package http

import (
	"context"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

type identityContextKey struct{}

func withIdentityContext(ctx context.Context, identityContext identity.IdentityContext) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identityContext)
}

func identityContextFromContext(ctx context.Context) (identity.IdentityContext, bool) {
	identityContext, ok := ctx.Value(identityContextKey{}).(identity.IdentityContext)
	return identityContext, ok
}
