package llm

import (
	"context"
	"reflect"

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

func isNilLogClient(client LogClient) bool {
	if client == nil {
		return true
	}
	value := reflect.ValueOf(client)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
