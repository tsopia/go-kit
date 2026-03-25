package llm

import (
	"context"
	"unicode/utf8"
)

type structuredLogger struct {
	cfg *StructuredLogConfig
}

func newStructuredLogger(cfg *StructuredLogConfig) *structuredLogger {
	return &structuredLogger{cfg: cfg}
}

func (l *structuredLogger) enabled() bool {
	return l != nil && l.cfg != nil && l.cfg.Client != nil
}

func (l *structuredLogger) logInfo(ctx context.Context, event string, attrs ...any) {
	if !l.enabled() {
		return
	}
	fields := make([]any, 0, len(attrs)+1)
	fields = append(fields, "event", event)
	fields = append(fields, attrs...)
	fields = appendInvocationIDField(ctx, fields)
	l.cfg.Client.Info(ctx, event, fields...)
}

func (l *structuredLogger) logError(ctx context.Context, event string, attrs ...any) {
	if !l.enabled() {
		return
	}
	fields := make([]any, 0, len(attrs)+1)
	fields = append(fields, "event", event)
	fields = append(fields, attrs...)
	fields = appendInvocationIDField(ctx, fields)
	l.cfg.Client.Error(ctx, event, fields...)
}

func truncateField(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	for !utf8.ValidString(value[:limit]) && limit > 0 {
		limit--
	}
	if limit <= 0 {
		return ""
	}
	return value[:limit]
}
