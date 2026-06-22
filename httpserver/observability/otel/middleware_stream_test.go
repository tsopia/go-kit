package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

type recordingTracerProvider struct {
	embedded.TracerProvider
	span *recordingSpan
}

func (p recordingTracerProvider) Tracer(string, ...oteltrace.TracerOption) oteltrace.Tracer {
	return recordingTracer{span: p.span}
}

type recordingTracer struct {
	embedded.Tracer
	span *recordingSpan
}

func (t recordingTracer) Start(ctx context.Context, _ string, _ ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return oteltrace.ContextWithSpan(ctx, t.span), t.span
}

type recordingSpan struct {
	embedded.Span
	attrs      []attribute.KeyValue
	statusCode codes.Code
}

func (s *recordingSpan) End(...oteltrace.SpanEndOption)              {}
func (s *recordingSpan) AddEvent(string, ...oteltrace.EventOption)   {}
func (s *recordingSpan) AddLink(oteltrace.Link)                      {}
func (s *recordingSpan) IsRecording() bool                          { return true }
func (s *recordingSpan) RecordError(error, ...oteltrace.EventOption) {}
func (s *recordingSpan) SpanContext() oteltrace.SpanContext {
	return oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    [16]byte{1},
		SpanID:     [8]byte{1},
		TraceFlags: oteltrace.FlagsSampled,
	})
}
func (s *recordingSpan) SetStatus(code codes.Code, _ string)         { s.statusCode = code }
func (s *recordingSpan) SetName(string)                              {}
func (s *recordingSpan) SetAttributes(kv ...attribute.KeyValue)      { s.attrs = append(s.attrs, kv...) }
func (s *recordingSpan) TracerProvider() oteltrace.TracerProvider {
	return recordingTracerProvider{span: s}
}

func TestMiddleware_StreamingSpanAttribute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	span := &recordingSpan{}
	engine := gin.New()
	engine.Use(Middleware(Config{TracerProvider: recordingTracerProvider{span: span}}))
	engine.GET("/stream", func(c *gin.Context) {
		c.Set(utils.StreamingKey, "sse")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stream", nil))

	var hasAttr bool
	for _, attr := range span.attrs {
		if string(attr.Key) == "stream.transport" && attr.Value.AsString() == "sse" {
			hasAttr = true
		}
	}
	if !hasAttr {
		t.Errorf("span missing stream.transport=sse attribute: %+v", span.attrs)
	}
	if span.statusCode == codes.Error {
		t.Error("streaming span should not be marked error for 200")
	}
}
