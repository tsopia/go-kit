package llm

import (
	"bytes"
	"context"
	"log/slog"
	"sync"

	"github.com/tsopia/go-kit/kit"
	"github.com/tsopia/go-kit/utils"
)

var _ LogClient = (*kit.Logger)(nil)

type recordedLogEntry struct {
	level       string
	msg         string
	fields      map[string]any
	traceID     string
	requestID   string
	invocationID string
}

type recordingLogClient struct {
	mu      sync.Mutex
	entries []recordedLogEntry
}

func (c *recordingLogClient) Info(ctx context.Context, msg string, fields ...any) {
	c.record("info", ctx, msg, fields...)
}

func (c *recordingLogClient) Error(ctx context.Context, msg string, fields ...any) {
	c.record("error", ctx, msg, fields...)
}

func (c *recordingLogClient) record(level string, ctx context.Context, msg string, fields ...any) {
	entry := recordedLogEntry{
		level:       level,
		msg:         msg,
		fields:      fieldsMap(fields...),
		traceID:     utils.TraceIDFromContext(ctx),
		requestID:   utils.RequestIDFromContext(ctx),
		invocationID: invocationIDFromContext(ctx),
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = append(c.entries, entry)
}

func (c *recordingLogClient) snapshot() []recordedLogEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]recordedLogEntry, len(c.entries))
	copy(out, c.entries)
	return out
}

type jsonBufferLogClient struct {
	logger *slog.Logger
}

func newJSONBufferLogClient(buf *bytes.Buffer) LogClient {
	return &jsonBufferLogClient{
		logger: slog.New(slog.NewJSONHandler(buf, nil)),
	}
}

func (c *jsonBufferLogClient) Info(ctx context.Context, msg string, fields ...any) {
	c.logger.Log(ctx, slog.LevelInfo, msg, fields...)
}

func (c *jsonBufferLogClient) Error(ctx context.Context, msg string, fields ...any) {
	c.logger.Log(ctx, slog.LevelError, msg, fields...)
}

func fieldsMap(fields ...any) map[string]any {
	m := make(map[string]any, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		m[key] = fields[i+1]
	}
	return m
}
