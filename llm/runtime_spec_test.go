package llm

import (
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
				Tools:     ToolsConfig{Invokable: []InvokableTool{forcedTool}},
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
