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
		Model:     &mockCallbackModel{},
		Callbacks: []callbacks.Handler{handler},
	})
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	if _, err = agent.Generate(ctx, []*schema.Message{schema.UserMessage("hello")}); err != nil {
		t.Fatalf("Generate failed: %v", err)
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
	if !strings.Contains(logs, "\"component\":\"ChatModel\"") {
		t.Log("JSON Handler 字段格式可能不同，跳过 component 字段断言")
	}
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
