package llm

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/tsopia/go-kit/utils"
)

func TestLogHandler_Integration(t *testing.T) {
	client := &recordingLogClient{}
	handler := NewLogHandler(client)

	ctx := utils.WithTraceAndRequestID(context.Background(), "trace-123", "req-456")
	agent, err := NewAgent(ctx, AgentConfig{
		Model:         AgentModelConfig{Instance: &mockCallbackModel{}},
		Observability: ObservabilityConfig{Callbacks: []callbacks.Handler{handler}},
	})
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	resp, err := agent.Generate(ctx, []*schema.Message{schema.UserMessage("hello")})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.Content != "mock response" {
		t.Fatalf("unexpected response: %q", resp.Content)
	}

	entries := client.snapshot()
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 log entries, got %d", len(entries))
	}

	if entries[0].msg != "Component Start" {
		t.Fatalf("unexpected first message: %q", entries[0].msg)
	}
	last := entries[len(entries)-1]
	if last.msg != "Component End" {
		t.Fatalf("unexpected last message: %q", last.msg)
	}

	for _, entry := range entries {
		if entry.traceID != "trace-123" {
			t.Fatalf("expected trace id to propagate, got %q", entry.traceID)
		}
		if entry.requestID != "req-456" {
			t.Fatalf("expected request id to propagate, got %q", entry.requestID)
		}
		if entry.invocationID == "" {
			t.Fatal("expected invocation id in callback logs")
		}
		if _, ok := entry.fields["event"]; ok {
			t.Fatal("did not expect structured event fields in NewLogHandler output")
		}
	}
}

func TestStreamingAndObservabilityConfig(t *testing.T) {
	t.Run("observability_callbacks", func(t *testing.T) {
		var starts, ends int
		handler := callbacks.NewHandlerBuilder().
			OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
				starts++
				return ctx
			}).
			OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
				ends++
				return ctx
			}).
			Build()

		agent, err := NewAgent(context.Background(), AgentConfig{
			Model:         AgentModelConfig{Instance: &mockCallbackModel{}},
			Observability: ObservabilityConfig{Callbacks: []callbacks.Handler{handler}},
		})
		if err != nil {
			t.Fatalf("NewAgent failed: %v", err)
		}

		resp, err := agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("hello")})
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		if resp.Content != "mock response" {
			t.Fatalf("unexpected response: %q", resp.Content)
		}
		if starts == 0 || ends == 0 {
			t.Fatalf("expected callbacks to run, got starts=%d ends=%d", starts, ends)
		}
	})

	t.Run("streaming_tool_call_checker", func(t *testing.T) {
		var checkerCalled bool
		agent, err := NewAgent(context.Background(), AgentConfig{
			Model: AgentModelConfig{Instance: &fakeToolCallingModel{
				responses: []*schema.Message{{Role: schema.Assistant, Content: "stream reply"}},
			}},
			Streaming: StreamingConfig{
				ToolCallChecker: func(_ context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
					checkerCalled = true
					defer sr.Close()
					msg, err := sr.Recv()
					if err != nil {
						return false, err
					}
					return len(msg.ToolCalls) > 0, nil
				},
			},
		})
		if err != nil {
			t.Fatalf("NewAgent failed: %v", err)
		}

		stream, err := agent.Stream(context.Background(), []*schema.Message{schema.UserMessage("hello")})
		if err != nil {
			t.Fatalf("Stream failed: %v", err)
		}
		defer stream.Close()

		msg, err := stream.Recv()
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}
		if msg.Content != "stream reply" {
			t.Fatalf("unexpected stream content: %q", msg.Content)
		}
		if !checkerCalled {
			t.Fatal("expected streaming tool call checker to run")
		}
	})
}

// 模拟的 Mock Model，复用 optimization_test.go 中的定义
// 但 optimization_test.go 中的 mock 是不公开的，所以这里可能需要重新定义一个简单的
type mockCallbackModel struct{}

func (m *mockCallbackModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage("mock response", nil), nil
}
func (m *mockCallbackModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}
func (m *mockCallbackModel) BindTools(tools []*schema.ToolInfo) error { return nil }
func (m *mockCallbackModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}
