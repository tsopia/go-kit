package llm

import (
	"context"
	"log/slog"
	"unicode/utf8"
)

type structuredLogger struct {
	cfg *StructuredLogConfig
}

func newStructuredLogger(cfg *StructuredLogConfig) *structuredLogger {
	return &structuredLogger{cfg: cfg}
}

func (l *structuredLogger) enabled() bool {
	return l != nil && l.cfg != nil && l.cfg.Logger != nil
}

func (l *structuredLogger) log(level slog.Level, event string, attrs ...any) {
	if !l.enabled() {
		return
	}
	fields := make([]any, 0, len(attrs)+1)
	fields = append(fields, slog.String("event", event))
	fields = append(fields, attrs...)
	l.cfg.Logger.Log(context.Background(), level, event, fields...)
}

func (l *structuredLogger) logInfo(event string, attrs ...any) {
	l.log(slog.LevelInfo, event, attrs...)
}

func (l *structuredLogger) logError(event string, attrs ...any) {
	l.log(slog.LevelError, event, attrs...)
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
