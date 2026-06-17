package llm

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/tsopia/go-kit/utils"
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
				Execution:     ExecutionConfig{Mode: Conversation},
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{}},
			},
			want:   []string{`"event":"agent.start"`, `"event":"model.decision"`, `"event":"agent.end"`},
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
				Execution:     ExecutionConfig{Mode: Assistant},
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
				tt.cfg.Observability.StructuredLogs.Client = newJSONBufferLogClient(&buf)
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

func TestStructuredLogs_ModelDecision(t *testing.T) {
	tests := []struct {
		name string
		cfg  AgentConfig
		want []string
		forbid []string
	}{
		{
			name: "assistant_logs_plain_text_decision",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{{Role: schema.Assistant, Content: "plain answer"}},
				}},
				Execution:     ExecutionConfig{Mode: Assistant},
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{}},
			},
			want:   []string{`"event":"model.decision"`, `"configured_tool_choice":"allowed"`, `"tool_call_count":0`},
			forbid: []string{`"tool_choice":"allowed"`},
		},
		{
			name: "assistant_logs_tool_call_decision",
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
				Execution:     ExecutionConfig{Mode: Assistant},
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{}},
			},
			want:   []string{`"event":"model.decision"`, `"configured_tool_choice":"allowed"`, `"tool_call_count":1`, `"lookup_user"`},
			forbid: []string{`"tool_choice":"allowed"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if tt.cfg.Observability.StructuredLogs != nil {
				tt.cfg.Observability.StructuredLogs.Client = newJSONBufferLogClient(&buf)
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

func TestStructuredLogs_ToolLifecycle(t *testing.T) {
	tests := []struct {
		name    string
		cfg     AgentConfig
		want    []string
		wantErr string
	}{
		{
			name: "assistant_logs_tool_start_and_success",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{
						{
							Role: schema.Assistant,
							ToolCalls: []schema.ToolCall{
								{ID: "tc1", Function: schema.FunctionCall{Name: "lookup_user", Arguments: `{"name":"Alice"}`}},
							},
						},
						{Role: schema.Assistant, Content: "lookup done"},
					},
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
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{
					LogToolArguments: true,
					LogToolResults:   true,
				}},
			},
			want: []string{
				`"event":"tool.start"`,
				`"tool_name":"lookup_user"`,
				`"tool_call_id":"tc1"`,
				`"attempt":1`,
				`"arguments":"{\"name\":\"Alice\"}"`,
				`"event":"tool.success"`,
				`"result":"{\"name\":\"Alice\"}"`,
			},
		},
		{
			name: "extraction_logs_retryable_error_then_success_with_direct_return",
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
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{
					LogToolArguments: true,
					LogToolResults:   true,
				}},
			},
			want: []string{
				`"event":"tool.start"`,
				`"tool_name":"extract"`,
				`"attempt":1`,
				`"event":"tool.error"`,
				`"retryable":true`,
				`"terminal":false`,
				`"attempt":2`,
				`"event":"tool.success"`,
				`"direct_return":true`,
			},
		},
		{
			name: "extraction_logs_terminal_error_after_retry_exhausted",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{
						{
							Role: schema.Assistant,
							ToolCalls: []schema.ToolCall{
								{ID: "tc1", Function: schema.FunctionCall{Name: "extract", Arguments: `{"query":"bad-1"}`}},
							},
						},
						{
							Role: schema.Assistant,
							ToolCalls: []schema.ToolCall{
								{ID: "tc2", Function: schema.FunctionCall{Name: "extract", Arguments: `{"query":"bad-2"}`}},
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
						failCount: 2,
					},
				}},
				Execution: ExecutionConfig{
					Mode:       Extraction,
					MaxRetries: 2,
				},
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{
					LogToolArguments: true,
				}},
			},
			want: []string{
				`"event":"tool.error"`,
				`"retryable":false`,
				`"terminal":true`,
				`"attempt":2`,
			},
			wantErr: `extraction retries exhausted`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if tt.cfg.Observability.StructuredLogs != nil {
				tt.cfg.Observability.StructuredLogs.Client = newJSONBufferLogClient(&buf)
			}

			agent, err := NewAgent(context.Background(), tt.cfg)
			if err != nil {
				t.Fatalf("NewAgent failed: %v", err)
			}

			_, err = agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("hello")})
			if tt.wantErr == "" && err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			logs := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(logs, want) {
					t.Fatalf("missing log fragment %q\nlogs=%s", want, logs)
				}
			}
		})
	}
}

func TestStructuredLogs_AgentOutcome(t *testing.T) {
	tests := []struct {
		name string
		cfg  AgentConfig
		want []string
	}{
		{
			name: "conversation_logs_agent_start_and_end",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{{Role: schema.Assistant, Content: "plain answer"}},
				}},
				Execution:     ExecutionConfig{Mode: Conversation},
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{}},
			},
			want: []string{
				`"event":"agent.start"`,
				`"execution_mode":"conversation"`,
				`"tool_count":0`,
				`"direct_return_enabled":false`,
				`"event":"agent.end"`,
				`"status":"success"`,
			},
		},
		{
			name: "extraction_logs_direct_return_outcome",
			cfg: AgentConfig{
				Model: AgentModelConfig{Instance: &fakeToolCallingModel{
					responses: []*schema.Message{{
						Role: schema.Assistant,
						ToolCalls: []schema.ToolCall{
							{ID: "tc1", Function: schema.FunctionCall{Name: "extract", Arguments: `{"query":"good"}`}},
						},
					}},
				}},
				Tools: ToolsConfig{Invokable: []InvokableTool{
					&simpleTool{
						info: &schema.ToolInfo{
							Name: "extract",
							Desc: "extract",
							ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
								"query": {Type: schema.String, Desc: "query"},
							}),
						},
						ret: `{"result":"ok"}`,
					},
				}},
				Execution: ExecutionConfig{
					Mode:              Extraction,
					DirectReturnTools: map[string]struct{}{"extract": {}},
				},
				Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{}},
			},
			want: []string{
				`"event":"agent.start"`,
				`"execution_mode":"extraction"`,
				`"tool_count":1`,
				`"direct_return_enabled":true`,
				`"event":"agent.end"`,
				`"status":"success"`,
				`"direct_return":true`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if tt.cfg.Observability.StructuredLogs != nil {
				tt.cfg.Observability.StructuredLogs.Client = newJSONBufferLogClient(&buf)
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
		})
	}
}

func TestStructuredLogs_ContextAndInvocationID(t *testing.T) {
	client := &recordingLogClient{}

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: &fakeToolCallingModel{
			responses: []*schema.Message{
				{
					Role: schema.Assistant,
					ToolCalls: []schema.ToolCall{
						{ID: "tc1", Function: schema.FunctionCall{Name: "lookup_user", Arguments: `{"name":"Alice"}`}},
					},
				},
				{Role: schema.Assistant, Content: "lookup done"},
				{Role: schema.Assistant, Content: "plain answer"},
			},
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
		Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{
			Client: client,
		}},
	})
	if err != nil {
		t.Fatalf("NewAgent failed: %v", err)
	}

	ctx := utils.WithTraceAndRequestID(context.Background(), "trace-ctx", "req-ctx")

	if _, err := agent.Generate(ctx, []*schema.Message{schema.UserMessage("hello")}); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	firstEntries := client.snapshot()
	if len(firstEntries) == 0 {
		t.Fatal("expected structured logs")
	}

	firstInvocationID, ok := firstEntries[0].fields["invocation_id"].(string)
	if !ok || firstInvocationID == "" {
		t.Fatal("expected invocation_id field in structured logs")
	}

	for _, entry := range firstEntries {
		if entry.traceID != "trace-ctx" {
			t.Fatalf("expected trace id to propagate, got %q", entry.traceID)
		}
		if entry.requestID != "req-ctx" {
			t.Fatalf("expected request id to propagate, got %q", entry.requestID)
		}
		if entry.invocationID != firstInvocationID {
			t.Fatalf("expected invocation id from ctx to match fields, got ctx=%q field=%v", entry.invocationID, entry.fields["invocation_id"])
		}
		if entry.fields["invocation_id"] != firstInvocationID {
			t.Fatalf("expected same invocation id within one generate call, got %#v want %q", entry.fields["invocation_id"], firstInvocationID)
		}
	}

	if _, err := agent.Generate(ctx, []*schema.Message{schema.UserMessage("hello again")}); err != nil {
		t.Fatalf("second Generate failed: %v", err)
	}
	allEntries := client.snapshot()
	if len(allEntries) <= len(firstEntries) {
		t.Fatal("expected more log entries after second generate")
	}

	secondInvocationID, ok := allEntries[len(firstEntries)].fields["invocation_id"].(string)
	if !ok || secondInvocationID == "" {
		t.Fatal("expected second invocation_id field")
	}
	if secondInvocationID == firstInvocationID {
		t.Fatalf("expected unique invocation id per generate call, got same %q", secondInvocationID)
	}
}
