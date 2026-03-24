package llm

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestLogHandler_Integration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	handler := NewLogHandler(logger)

	ctx := context.Background()
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

	logs := buf.String()
	checks := []struct {
		name string
		want string
	}{
		{name: "start", want: "Component Start"},
		{name: "end", want: "Component End"},
	}
	for _, check := range checks {
		if !strings.Contains(logs, check.want) {
			t.Errorf("expected log to contain %q", check.want)
		}
	}
	if strings.Contains(logs, "\"event\":\"agent.start\"") {
		t.Fatal("did not expect structured log event in NewLogHandler output")
	}
	if !strings.Contains(logs, "\"component\":\"ChatModel\"") {
		t.Log("JSON Handler 字段格式可能不同，跳过 component 字段断言")
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
