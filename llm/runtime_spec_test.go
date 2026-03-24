package llm

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestCompileRuntimeSpec(t *testing.T) {
	forcedTool := &simpleTool{
		info: &schema.ToolInfo{
			Name: "extract_order",
			Desc: "extract order",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"title": {Type: schema.String, Desc: "title"},
			}),
		},
		ret: `{"ok":true}`,
	}

	tests := []struct {
		name              string
		cfg               AgentConfig
		wantMode          ExecutionMode
		wantDisableTools  bool
		wantToolChoice    schema.ToolChoice
		wantRepairAttempt int
	}{
		{
			name: "conversation",
			cfg: AgentConfig{
				Execution: ExecutionConfig{Mode: Conversation},
			},
			wantMode:         Conversation,
			wantDisableTools: true,
			wantToolChoice:   schema.ToolChoiceForbidden,
		},
		{
			name: "assistant",
			cfg: AgentConfig{
				Tools:     ToolsConfig{Invokable: []InvokableTool{forcedTool}},
				Execution: ExecutionConfig{Mode: Assistant},
			},
			wantMode:         Assistant,
			wantDisableTools: false,
			wantToolChoice:   schema.ToolChoiceAllowed,
		},
		{
			name: "extraction",
			cfg: AgentConfig{
				Tools: ToolsConfig{Invokable: []InvokableTool{forcedTool}},
				Execution: ExecutionConfig{
					Mode:              Extraction,
					DirectReturnTools: map[string]struct{}{"extract_order": {}},
				},
			},
			wantMode:          Extraction,
			wantDisableTools:  false,
			wantToolChoice:    schema.ToolChoiceForced,
			wantRepairAttempt: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := compileRuntimeSpec(tt.cfg)
			if err != nil {
				t.Fatalf("compileRuntimeSpec failed: %v", err)
			}
			if spec.Execution.Mode != tt.wantMode {
				t.Fatalf("unexpected mode: got %q want %q", spec.Execution.Mode, tt.wantMode)
			}
			if spec.Execution.DisableTools != tt.wantDisableTools {
				t.Fatalf("unexpected disable tools flag: got %v want %v", spec.Execution.DisableTools, tt.wantDisableTools)
			}
			if spec.Execution.ToolChoice != tt.wantToolChoice {
				t.Fatalf("unexpected tool choice: got %q want %q", spec.Execution.ToolChoice, tt.wantToolChoice)
			}
			if spec.Execution.RepairMaxAttempts != tt.wantRepairAttempt {
				t.Fatalf("unexpected repair max attempts: got %d want %d", spec.Execution.RepairMaxAttempts, tt.wantRepairAttempt)
			}
		})
	}
}

func TestCompileRuntimeSpec_LegacyToolChoiceCompatibility(t *testing.T) {
	tests := []struct {
		name             string
		toolChoice       schema.ToolChoice
		wantMode         ExecutionMode
		wantDisableTools bool
		wantToolChoice   schema.ToolChoice
		wantRepairMax    int
	}{
		{
			name:             "forbidden_maps_to_conversation",
			toolChoice:       schema.ToolChoiceForbidden,
			wantMode:         Conversation,
			wantDisableTools: true,
			wantToolChoice:   schema.ToolChoiceForbidden,
		},
		{
			name:             "allowed_maps_to_assistant",
			toolChoice:       schema.ToolChoiceAllowed,
			wantMode:         Assistant,
			wantDisableTools: false,
			wantToolChoice:   schema.ToolChoiceAllowed,
		},
		{
			name:             "forced_maps_to_extraction",
			toolChoice:       schema.ToolChoiceForced,
			wantMode:         Extraction,
			wantDisableTools: false,
			wantToolChoice:   schema.ToolChoiceForced,
			wantRepairMax:    5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			choice := tt.toolChoice
			maxRetries := 0
			if choice == schema.ToolChoiceForced {
				maxRetries = 5
			}
			spec, err := compileRuntimeSpec(AgentConfig{
				Tools: ToolsConfig{Invokable: []InvokableTool{
					&simpleTool{
						info: &schema.ToolInfo{
							Name: "extract_order",
							Desc: "extract order",
							ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
								"title": {Type: schema.String, Desc: "title"},
							}),
						},
						ret: `{"ok":true}`,
					},
				}},
				Execution: ExecutionConfig{
					ToolChoice: &choice,
					MaxRetries: maxRetries,
				},
			})
			if err != nil {
				t.Fatalf("compileRuntimeSpec failed: %v", err)
			}
			if spec.Execution.Mode != tt.wantMode {
				t.Fatalf("unexpected mode: got %q want %q", spec.Execution.Mode, tt.wantMode)
			}
			if spec.Execution.DisableTools != tt.wantDisableTools {
				t.Fatalf("unexpected disable tools flag: got %v want %v", spec.Execution.DisableTools, tt.wantDisableTools)
			}
			if spec.Execution.ToolChoice != tt.wantToolChoice {
				t.Fatalf("unexpected tool choice: got %q want %q", spec.Execution.ToolChoice, tt.wantToolChoice)
			}
			if spec.Execution.RepairMaxAttempts != tt.wantRepairMax {
				t.Fatalf("unexpected repair max attempts: got %d want %d", spec.Execution.RepairMaxAttempts, tt.wantRepairMax)
			}
		})
	}
}

func TestCompileRuntimeSpec_RejectsInvalidExecutionConfig(t *testing.T) {
	extractTool := &simpleTool{
		info: &schema.ToolInfo{
			Name: "extract_order",
			Desc: "extract order",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"title": {Type: schema.String, Desc: "title"},
			}),
		},
		ret: `{"ok":true}`,
	}

	tests := []struct {
		name    string
		cfg     AgentConfig
		wantErr string
	}{
		{
			name: "conversation_rejects_tools",
			cfg: AgentConfig{
				Tools:     ToolsConfig{Invokable: []InvokableTool{extractTool}},
				Execution: ExecutionConfig{Mode: Conversation},
			},
			wantErr: "conversation mode does not allow tools",
		},
		{
			name: "conversation_rejects_retries",
			cfg: AgentConfig{
				Execution: ExecutionConfig{
					Mode:       Conversation,
					MaxRetries: 1,
				},
			},
			wantErr: "conversation mode does not allow max retries",
		},
		{
			name: "conversation_rejects_direct_return",
			cfg: AgentConfig{
				Execution: ExecutionConfig{
					Mode:              Conversation,
					DirectReturnTools: map[string]struct{}{"extract_order": {}},
				},
			},
			wantErr: "conversation mode does not allow direct return tools",
		},
		{
			name: "assistant_rejects_retries",
			cfg: AgentConfig{
				Tools: ToolsConfig{Invokable: []InvokableTool{extractTool}},
				Execution: ExecutionConfig{
					Mode:       Assistant,
					MaxRetries: 1,
				},
			},
			wantErr: "assistant mode does not allow max retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compileRuntimeSpec(tt.cfg)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("unexpected error: got %q want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCompileRuntimeSpec_ModeOverridesLegacyToolChoice(t *testing.T) {
	forced := schema.ToolChoiceForced

	spec, err := compileRuntimeSpec(AgentConfig{
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
		Execution: ExecutionConfig{
			Mode:       Assistant,
			ToolChoice: &forced,
		},
	})
	if err != nil {
		t.Fatalf("compileRuntimeSpec failed: %v", err)
	}
	if spec.Execution.Mode != Assistant {
		t.Fatalf("unexpected mode: got %q want %q", spec.Execution.Mode, Assistant)
	}
	if spec.Execution.ToolChoice != schema.ToolChoiceAllowed {
		t.Fatalf("unexpected tool choice: got %q want %q", spec.Execution.ToolChoice, schema.ToolChoiceAllowed)
	}
}

func TestCompileRuntimeSpec_MCPServersEnableTools(t *testing.T) {
	tests := []struct {
		name             string
		cfg              AgentConfig
		wantMode         ExecutionMode
		wantDisableTools bool
	}{
		{
			name: "default_mode_with_mcp_servers_uses_assistant",
			cfg: AgentConfig{
				Tools: ToolsConfig{
					MCPServers: []MCPConfig{{Name: "mcp_tool", Protocol: MCPProtocolSSE, BaseURL: "http://example.com"}},
				},
			},
			wantMode:         Assistant,
			wantDisableTools: false,
		},
		{
			name: "legacy_allowed_with_only_mcp_servers_keeps_tools_enabled",
			cfg: func() AgentConfig {
				allowed := schema.ToolChoiceAllowed
				return AgentConfig{
					Tools: ToolsConfig{
						MCPServers: []MCPConfig{{Name: "mcp_tool", Protocol: MCPProtocolSSE, BaseURL: "http://example.com"}},
					},
					Execution: ExecutionConfig{ToolChoice: &allowed},
				}
			}(),
			wantMode:         Assistant,
			wantDisableTools: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := compileRuntimeSpec(tt.cfg)
			if err != nil {
				t.Fatalf("compileRuntimeSpec failed: %v", err)
			}
			if spec.Execution.Mode != tt.wantMode {
				t.Fatalf("unexpected mode: got %q want %q", spec.Execution.Mode, tt.wantMode)
			}
			if spec.Execution.DisableTools != tt.wantDisableTools {
				t.Fatalf("unexpected disable tools flag: got %v want %v", spec.Execution.DisableTools, tt.wantDisableTools)
			}
		})
	}
}
