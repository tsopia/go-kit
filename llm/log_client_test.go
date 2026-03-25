package llm

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"github.com/tsopia/go-kit/kit"
	"github.com/tsopia/go-kit/utils"
)

var _ LogClient = (*kit.Logger)(nil)

type recordedLogEntry struct {
	level        string
	msg          string
	fields       map[string]any
	traceID      string
	requestID    string
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
		level:        level,
		msg:          msg,
		fields:       fieldsMap(fields...),
		traceID:      utils.TraceIDFromContext(ctx),
		requestID:    utils.RequestIDFromContext(ctx),
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

func TestTypedNilLogClientSafety(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "callback_handler_typed_nil_client_does_not_panic",
			run: func() (err error) {
				var client *kit.Logger
				handler := NewLogHandler(client)
				agent, err := NewAgent(context.Background(), AgentConfig{
					Model:         AgentModelConfig{Instance: &mockCallbackModel{}},
					Observability: ObservabilityConfig{Callbacks: []callbacks.Handler{handler}},
				})
				if err != nil {
					return err
				}
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panic: %v", r)
					}
				}()
				_, err = agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("hello")})
				return err
			},
		},
		{
			name: "structured_logs_typed_nil_client_does_not_panic",
			run: func() (err error) {
				var client *kit.Logger
				agent, err := NewAgent(context.Background(), AgentConfig{
					Model: AgentModelConfig{Instance: &fakeToolCallingModel{
						responses: []*schema.Message{{Role: schema.Assistant, Content: "plain answer"}},
					}},
					Execution: ExecutionConfig{Mode: Conversation},
					Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{
						Client: client,
					}},
				})
				if err != nil {
					return err
				}
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panic: %v", r)
					}
				}()
				_, err = agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("hello")})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestWithInvocationID_OverridesParentInvocationID(t *testing.T) {
	client := &recordingLogClient{}
	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: &fakeToolCallingModel{
			responses: []*schema.Message{{Role: schema.Assistant, Content: "plain answer"}},
		}},
		Execution: ExecutionConfig{Mode: Conversation},
		Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{
			Client: client,
		}},
	})
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	ctx := context.WithValue(context.Background(), ctxKeyInvocationID{}, "parent-invocation")
	if _, err := agent.Generate(ctx, []*schema.Message{schema.UserMessage("hello")}); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	entries := client.snapshot()
	if len(entries) == 0 {
		t.Fatal("expected log entries")
	}
	for _, entry := range entries {
		got, _ := entry.fields["invocation_id"].(string)
		if got == "" {
			t.Fatal("expected invocation_id field")
		}
		if got == "parent-invocation" {
			t.Fatalf("expected nested call to get a fresh invocation_id, got %q", got)
		}
	}
}
