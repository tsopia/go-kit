package llm

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// TestOptimization_RealAgent verifies that the robust Agent implementation
// correctly cancels the context after a successful tool call when ToolReturnDirectly is enabled,
// preventing the model from being called a second time for summary generation.
func TestOptimization_RealAgent(t *testing.T) {
	mock := &mockCountModel{}
	forced := schema.ToolChoiceForced

	// Define a simple tool
	tool := &testTool{
		name: "test_tool",
		fn: func(ctx context.Context, args string) (string, error) {
			return `{"status": "success"}`, nil
		},
	}

	// Create Agent with ToolReturnDirectly enabled
	agent, err := NewAgent(context.Background(), AgentConfig{
		Model:              mock,
		InvokableTools:     []InvokableTool{tool},
		ToolChoice:         &forced,
		ToolReturnDirectly: map[string]struct{}{"test_tool": {}},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Execute
	resp, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "run test"},
	})
	if err != nil {
		t.Fatalf("Agent.Generate failed: %v", err)
	}

	// Verify Result
	if resp.Content != `{"status": "success"}` {
		t.Errorf("Expected result '{\"status\": \"success\"}', got '%s'", resp.Content)
	}

	// Verify Model Calls
	// Should be 1 (Tool Call) -> Tool Exec -> Cancel -> Done.
	// If 2, then optimization failed (Summary generation started).
	fmt.Printf("Model Calls: %d\n", mock.calls)
	if mock.calls != 1 {
		t.Errorf("Optimization failed! Model was called %d times (expected 1)", mock.calls)
	} else {
		fmt.Println("Optimization SUCCESS: Model was called only once.")
	}
}

// --- Mocks ---

type testTool struct {
	name string
	fn   func(context.Context, string) (string, error)
}

func (t *testTool) Info() *schema.ToolInfo {
	return &schema.ToolInfo{Name: t.name, Desc: "test tool", ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{})}
}
func (t *testTool) Invoke(ctx context.Context, args string) (string, error) {
	return t.fn(ctx, args)
}

type mockCountModel struct {
	calls int
}

func (m *mockCountModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// Check context first (Eino behavior)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.calls++

	// First call: Return tool call
	if m.calls == 1 {
		return &schema.Message{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{{
				Function: schema.FunctionCall{
					Name:      "test_tool",
					Arguments: "{}",
				},
			}},
		}, nil
	}

	// Second call (Summary): Return summary
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "Done.",
	}, nil
}
func (m *mockCountModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}
func (m *mockCountModel) BindTools(tools []*schema.ToolInfo) error { return nil }
func (m *mockCountModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}
