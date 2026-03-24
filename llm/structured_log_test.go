package llm

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestStructuredLogs_Contract(t *testing.T) {
	tests := []struct {
		name   string
		cfg    AgentConfig
		want   []string
		forbid []string
	}{
		{
			name: "conversation_logs_agent_and_model_decision",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{{Role: schema.Assistant, Content: "plain answer"}},
				}},
				Execution: ExecutionConfig{Mode: Conversation},
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{}},
			},
			want: []string{`"event":"agent.start"`, `"event":"model.decision"`, `"event":"agent.end"`},
			forbid: []string{`"event":"tool.start"`, `"event":"tool.success"`, `"event":"tool.error"`},
		},
		{
			name: "assistant_logs_tool_lifecycle",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{{
						Role: schema.Assistant,
						ToolCalls: []schema.ToolCall{
							{ID: "tc1", Function: schema.FunctionCall{Name: "lookup_user", Arguments: `{"name":"Alice"}`}},
						},
					}},
				}},
				Tools: ToolsConfig{Invokable: []InvokableTool{
					&simpleTool{
						info: &schema.ToolInfo{
							Name: "lookup_user",
							Desc: "lookup user",
							ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
								"name": {Type: schema.String, Desc: "name"},
							}),
						},
						ret: `{"name":"Alice"}`,
					},
				}},
				Execution: ExecutionConfig{Mode: Assistant},
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{}},
			},
			want: []string{`"event":"tool.start"`, `"event":"tool.success"`},
		},
		{
			name: "extraction_logs_retry_and_direct_return",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{
						{
							Role: schema.Assistant,
							ToolCalls: []schema.ToolCall{
								{ID: "tc1", Function: schema.FunctionCall{Name: "extract", Arguments: `{"query":"bad"}`}},
							},
						},
						{
							Role: schema.Assistant,
							ToolCalls: []schema.ToolCall{
								{ID: "tc2", Function: schema.FunctionCall{Name: "extract", Arguments: `{"query":"good"}`}},
							},
						},
					},
				}},
				Tools: ToolsConfig{Invokable: []InvokableTool{
					&failingTool{
						info: &schema.ToolInfo{
							Name: "extract",
							Desc: "extract",
							ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
								"query": {Type: schema.String, Desc: "query"},
							}),
						},
						failCount: 1,
					},
				}},
				Execution: ExecutionConfig{
					Mode:              Extraction,
					MaxRetries:        2,
					DirectReturnTools: map[string]struct{}{"extract": {}},
				},
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{}},
			},
			want: []string{`"event":"tool.error"`, `"retryable":true`, `"event":"tool.success"`, `"direct_return":true`, `"event":"agent.end"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if tt.cfg.Observability.StructuredLogs != nil {
				tt.cfg.Observability.StructuredLogs.Logger = slog.New(slog.NewJSONHandler(&buf, nil))
			}

			agent, err := NewAgent(context.Background(), tt.cfg)
			if err != nil {
				t.Fatalf("NewAgent failed: %v", err)
			}

			_, err = agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("hello")})
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			logs := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(logs, want) {
					t.Fatalf("missing log fragment %q\nlogs=%s", want, logs)
				}
			}
			for _, forbid := range tt.forbid {
				if strings.Contains(logs, forbid) {
					t.Fatalf("unexpected log fragment %q\nlogs=%s", forbid, logs)
				}
			}
		})
	}
}
