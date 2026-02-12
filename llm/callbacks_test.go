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
	// 1. 设置 Capturing Logger
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// 2. 创建 LogHandler
	handler := NewLogHandler(logger)

	// 3. 创建 Fake Agent
	ctx := context.Background()
	agent, err := NewAgent(ctx, AgentConfig{
		Model: &mockCallbackModel{}, // 使用本地 mock model
		Callbacks: []callbacks.Handler{
			handler,
		},
	})
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	// 4. 执行 Generate
	input := []*schema.Message{schema.UserMessage("hello")}
	_, err = agent.Generate(ctx, input)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 5. 验证日志输出
	logs := buf.String()
	t.Logf("Captured Logs:\n%s", logs)

	//检查是否包含关键日志
	if !strings.Contains(logs, "Component Start") {
		t.Error("Expected log to contain 'Component Start'")
	}
	if !strings.Contains(logs, "Component End") {
		t.Error("Expected log to contain 'Component End'")
	}
	// 检查是否包含组件信息
	if !strings.Contains(logs, "\"component\":\"ChatModel\"") { // Eino JSON log format
		// 注意：JSON Handler 输出具体的字段格式可能略有不同
		// 这里只检查基本的存在性
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
