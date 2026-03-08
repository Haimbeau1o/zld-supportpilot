package http

import (
	"context"

	"github.com/Haimbeau1o/zld-supportpilot/internal/module/identity"
)

type identityContextKey struct{}
type requestIDContextKey struct{}

func withIdentityContext(ctx context.Context, identityContext identity.IdentityContext) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identityContext)
}

func identityContextFromContext(ctx context.Context) (identity.IdentityContext, bool) {
	identityContext, ok := ctx.Value(identityContextKey{}).(identity.IdentityContext)
	return identityContext, ok
}

func withRequestIDContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func requestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	return requestID, ok
}
