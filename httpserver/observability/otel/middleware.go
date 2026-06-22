package otel

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Config 描述 tracing 中间件配置。
type Config struct {
	TracerProvider  oteltrace.TracerProvider
	TracerName      string
	Propagator      propagation.TextMapPropagator
	SpanNameBuilder func(*gin.Context) string
}

// Middleware 创建 tracing 中间件。
func Middleware(config Config) gin.HandlerFunc {
	provider := config.TracerProvider
	if provider == nil {
		provider = otel.GetTracerProvider()
	}

	tracerName := config.TracerName
	if tracerName == "" {
		tracerName = "github.com/tsopia/go-kit/httpserver/observability/otel"
	}

	propagator := config.Propagator
	if propagator == nil {
		propagator = otel.GetTextMapPropagator()
	}

	tracer := provider.Tracer(tracerName)

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = propagator.Extract(ctx, propagation.HeaderCarrier(c.Request.Header))

		spanName := requestSpanName(c, config.SpanNameBuilder)
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if transport := c.GetString(utils.StreamingKey); transport != "" {
			span.SetAttributes(attribute.String("stream.transport", transport))
			if len(c.Errors) > 0 {
				lastErr := c.Errors.Last().Err
				span.RecordError(lastErr)
				span.SetStatus(codes.Error, lastErr.Error())
			}
			return
		}

		if len(c.Errors) > 0 {
			lastErr := c.Errors.Last().Err
			span.RecordError(lastErr)
			span.SetStatus(codes.Error, lastErr.Error())
			return
		}

		if c.Writer.Status() >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(c.Writer.Status()))
		}
	}
}

func requestSpanName(c *gin.Context, custom func(*gin.Context) string) string {
	if custom != nil {
		return custom(c)
	}

	if fullPath := c.FullPath(); fullPath != "" {
		return c.Request.Method + " " + fullPath
	}

	return c.Request.Method + " " + c.Request.URL.Path
}
