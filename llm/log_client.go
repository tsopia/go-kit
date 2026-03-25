package llm

import (
	"context"

	"github.com/tsopia/go-kit/utils"
)

type LogClient interface {
	Info(ctx context.Context, msg string, fields ...any)
	Error(ctx context.Context, msg string, fields ...any)
}

type ctxKeyInvocationID struct{}

func withInvocationID(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if invocationIDFromContext(ctx) != "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyInvocationID{}, utils.GenerateID())
}

func invocationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	invocationID, _ := ctx.Value(ctxKeyInvocationID{}).(string)
	return invocationID
}

func appendInvocationIDField(ctx context.Context, fields []any) []any {
	if invocationID := invocationIDFromContext(ctx); invocationID != "" {
		fields = append(fields, "invocation_id", invocationID)
	}
	return fields
}
