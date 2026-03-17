package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

func TestMiddlewarePropagatesSpanContext(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Middleware(Config{
		TracerName:     "test-httpserver",
		TracerProvider: fakeTracerProvider{},
	}))
	engine.GET("/users/:id", func(c *gin.Context) {
		span := oteltrace.SpanFromContext(c.Request.Context())
		if !span.SpanContext().IsValid() {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusNoContent)
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
	}
}

type fakeTracerProvider struct {
	embedded.TracerProvider
}

func (fakeTracerProvider) Tracer(string, ...oteltrace.TracerOption) oteltrace.Tracer {
	return fakeTracer{}
}

type fakeTracer struct {
	embedded.Tracer
}

func (fakeTracer) Start(ctx context.Context, _ string, _ ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	spanContext := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    [16]byte{1},
		SpanID:     [8]byte{1},
		TraceFlags: oteltrace.FlagsSampled,
	})
	span := fakeSpan{spanContext: spanContext}

	return oteltrace.ContextWithSpan(ctx, span), span
}

type fakeSpan struct {
	embedded.Span
	spanContext oteltrace.SpanContext
}

func (s fakeSpan) End(...oteltrace.SpanEndOption) {}

func (s fakeSpan) AddEvent(string, ...oteltrace.EventOption) {}

func (s fakeSpan) AddLink(oteltrace.Link) {}

func (s fakeSpan) IsRecording() bool { return true }

func (s fakeSpan) RecordError(error, ...oteltrace.EventOption) {}

func (s fakeSpan) SpanContext() oteltrace.SpanContext { return s.spanContext }

func (s fakeSpan) SetStatus(codes.Code, string) {}

func (s fakeSpan) SetName(string) {}

func (s fakeSpan) SetAttributes(...attribute.KeyValue) {}

func (s fakeSpan) TracerProvider() oteltrace.TracerProvider { return fakeTracerProvider{} }
