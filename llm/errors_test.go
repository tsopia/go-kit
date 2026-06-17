package llm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// TestSentinelErrors_ProgrammaticCheck 验证关键错误路径可被 errors.Is 精确判断。
func TestSentinelErrors_ProgrammaticCheck(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		buildErr func() error
		target   error
	}{
		{
			name:     "missing model",
			buildErr: func() error { _, err := NewModel(ctx, ModelConfig{}); return err },
			target:   ErrMissingModel,
		},
		{
			name: "missing base url for compat",
			buildErr: func() error {
				_, err := NewModel(ctx, ModelConfig{Protocol: OPENAI_COMPAT, Model: "m", APIKey: "k"})
				return err
			},
			target: ErrMissingBaseURL,
		},
		{
			name: "missing api key",
			buildErr: func() error {
				_, err := NewModel(ctx, ModelConfig{Protocol: OPENAI, Model: "m"})
				return err
			},
			target: ErrMissingAPIKey,
		},
		{
			name: "unsupported protocol",
			buildErr: func() error {
				_, err := NewModel(ctx, ModelConfig{Protocol: "NOPE", Model: "m", APIKey: "k"})
				return err
			},
			target: ErrUnsupportedProtocol,
		},
		{
			name: "invalid config: conversation with tools",
			buildErr: func() error {
				_, err := compileRuntimeSpec(AgentConfig{
					Execution: ExecutionConfig{Mode: Conversation},
					Tools:     ToolsConfig{Standard: []tool.BaseTool{nil}},
				})
				return err
			},
			target: ErrInvalidConfig,
		},
		{
			name: "invalid config: unknown tool choice",
			buildErr: func() error {
				bad := schema.ToolChoice("weird")
				_, err := compileRuntimeSpec(AgentConfig{
					Execution: ExecutionConfig{ToolChoice: &bad},
				})
				return err
			},
			target: ErrInvalidConfig,
		},
		{
			name: "unknown mcp protocol",
			buildErr: func() error {
				_, _, err := NewMCPTools(ctx, MCPConfig{Protocol: "carrier-pigeon"})
				return err
			},
			target: ErrUnknownMCPProtocol,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.buildErr()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tt.target) {
				t.Errorf("errors.Is(err, %v) = false, want true; err=%v", tt.target, err)
			}
		})
	}
}

// TestSentinelErrors_PreserveContext 验证包装后仍保留具体上下文信息。
func TestSentinelErrors_PreserveContext(t *testing.T) {
	_, err := NewModel(context.Background(), ModelConfig{Protocol: "NOPE", Model: "m", APIKey: "k"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnsupportedProtocol) {
		t.Fatalf("want ErrUnsupportedProtocol, got %v", err)
	}
	// 错误文本应仍包含具体 protocol 值，方便排查。
	if got := err.Error(); !strings.Contains(got, "NOPE") {
		t.Errorf("error should contain protocol value NOPE, got: %q", got)
	}
}
