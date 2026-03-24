package llm

import "log/slog"

type structuredLogger struct {
	cfg *StructuredLogConfig
}

func newStructuredLogger(cfg *StructuredLogConfig) *structuredLogger {
	return &structuredLogger{cfg: cfg}
}

func (l *structuredLogger) enabled() bool {
	return l != nil && l.cfg != nil && l.cfg.Logger != nil
}

func (l *structuredLogger) log(event string, attrs ...any) {
	if !l.enabled() {
		return
	}
	fields := make([]any, 0, len(attrs)+1)
	fields = append(fields, slog.String("event", event))
	fields = append(fields, attrs...)
	l.cfg.Logger.Info(event, fields...)
}

func truncateField(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
