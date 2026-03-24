package llm

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestOptimization_RealAgent(t *testing.T) {
	tests := []struct {
		name               string
		toolReturnDirectly map[string]struct{}
		wantContent        string
		wantCalls          int
	}{
		{
			name:               "direct_return_skips_summary",
			toolReturnDirectly: map[string]struct{}{"test_tool": {}},
			wantContent:        `{"status": "success"}`,
			wantCalls:          1,
		},
		{
			name:        "fallback_summary_after_tool_call",
			wantContent: "Done.",
			wantCalls:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCountModel{}
			forced := schema.ToolChoiceForced
			tool := &testTool{
				name: "test_tool",
				fn: func(ctx context.Context, args string) (string, error) {
					return `{"status": "success"}`, nil
				},
			}

			agent, err := NewAgent(context.Background(), AgentConfig{
				Model: AgentModelConfig{Instance: mock},
				Tools: ToolsConfig{Invokable: []InvokableTool{tool}},
				Execution: ExecutionConfig{
					ToolChoice:        &forced,
					DirectReturnTools: tt.toolReturnDirectly,
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			resp, err := agent.Generate(context.Background(), []*schema.Message{
				{Role: schema.User, Content: "run test"},
			})
			if err != nil {
				t.Fatalf("Agent.Generate failed: %v", err)
			}
			if resp.Content != tt.wantContent {
				t.Fatalf("unexpected content: got %q want %q", resp.Content, tt.wantContent)
			}
			if mock.calls != tt.wantCalls {
				t.Fatalf("unexpected model call count: got %d want %d", mock.calls, tt.wantCalls)
			}
		})
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
