package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func TestBuildPromptAndTools(t *testing.T) {
	t.Helper()

	originalLoader := mcpToolLoader
	defer func() {
		mcpToolLoader = originalLoader
	}()

	mcpCleanupCalled := false
	mcpToolLoader = func(_ context.Context, cfg MCPConfig) ([]tool.BaseTool, func() error, error) {
		loaded := &ToolAdapter{inner: &simpleTool{
			info: &schema.ToolInfo{Name: cfg.Name, Desc: "mcp"},
			ret:  `{"source":"mcp"}`,
		}}
		return []tool.BaseTool{loaded}, func() error {
			mcpCleanupCalled = true
			return nil
		}, nil
	}

	spec, err := compileRuntimeSpec(AgentConfig{
		Prompt: PromptConfig{
			System: "system prompt",
			PrepareMessages: func(_ context.Context, input []*schema.Message) []*schema.Message {
				return append(input, schema.UserMessage("prepared"))
			},
			RewriteHistory: func(_ context.Context, input []*schema.Message) []*schema.Message {
				return input[:1]
			},
		},
		Tools: ToolsConfig{
			Standard: []tool.BaseTool{
				NewToolAdapter(&simpleTool{info: &schema.ToolInfo{Name: "standard", Desc: "standard"}, ret: `{"source":"standard"}`}),
			},
			Invokable: []InvokableTool{
				&simpleTool{info: &schema.ToolInfo{Name: "invokable", Desc: "invokable"}, ret: `{"source":"invokable"}`},
			},
			MCPServers: []MCPConfig{{Name: "mcp_tool", Protocol: MCPProtocolSSE, BaseURL: "http://example.com"}},
		},
	})
	if err != nil {
		t.Fatalf("compileRuntimeSpec failed: %v", err)
	}

	built, err := buildPromptAndTools(context.Background(), spec)
	if err != nil {
		t.Fatalf("buildPromptAndTools failed: %v", err)
	}
	defer func() {
		if err := built.Cleanup(); err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
	}()

	gotMessages := built.MessageModifier(context.Background(), []*schema.Message{schema.UserMessage("hello")})
	if len(gotMessages) != 3 {
		t.Fatalf("unexpected message count: got %d want 3", len(gotMessages))
	}
	if gotMessages[0].Role != schema.System || gotMessages[0].Content != "system prompt" {
		t.Fatalf("unexpected system message: %#v", gotMessages[0])
	}
	if gotMessages[2].Content != "prepared" {
		t.Fatalf("unexpected prepared message: %#v", gotMessages[2])
	}

	rewritten := built.MessageRewriter(context.Background(), []*schema.Message{
		schema.UserMessage("hello"),
		schema.AssistantMessage("world", nil),
	})
	if len(rewritten) != 1 || rewritten[0].Content != "hello" {
		t.Fatalf("unexpected rewritten history: %#v", rewritten)
	}

	if len(built.Tools) != 3 {
		t.Fatalf("unexpected tool count: got %d want 3", len(built.Tools))
	}
	if err := built.Cleanup(); err != nil {
		t.Fatalf("second cleanup failed: %v", err)
	}
	if !mcpCleanupCalled {
		t.Fatal("expected MCP cleanup to be called")
	}
}

func TestBuildPromptAndTools_ErrorPathCleanup(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		cfg          AgentConfig
		loader       func(*int) func(context.Context, MCPConfig) ([]tool.BaseTool, func() error, error)
		wantErr      string
		wantCleanups int
	}{
		{
			name: "load_error_cleans_previous_mcp_clients",
			cfg: AgentConfig{
				Tools: ToolsConfig{
					MCPServers: []MCPConfig{
						{Name: "first", Protocol: MCPProtocolSSE, BaseURL: "http://example.com/1"},
						{Name: "second", Protocol: MCPProtocolSSE, BaseURL: "http://example.com/2"},
					},
				},
				Execution: ExecutionConfig{Mode: Assistant},
			},
			loader: func(cleanups *int) func(context.Context, MCPConfig) ([]tool.BaseTool, func() error, error) {
				call := 0
				return func(_ context.Context, cfg MCPConfig) ([]tool.BaseTool, func() error, error) {
					call++
					if call == 2 {
						return nil, nil, errors.New("boom")
					}
					loaded := &ToolAdapter{inner: &simpleTool{
						info: &schema.ToolInfo{Name: cfg.Name, Desc: "mcp"},
						ret:  `{"source":"mcp"}`,
					}}
					return []tool.BaseTool{loaded}, func() error {
						*cleanups = *cleanups + 1
						return nil
					}, nil
				}
			},
			wantErr:      "load MCP tools: boom",
			wantCleanups: 1,
		},
		{
			name: "validation_error_cleans_loaded_mcp_clients",
			cfg: AgentConfig{
				Tools: ToolsConfig{
					MCPServers: []MCPConfig{{Name: "mcp_tool", Protocol: MCPProtocolSSE, BaseURL: "http://example.com"}},
				},
				Execution: ExecutionConfig{
					Mode:              Assistant,
					DirectReturnTools: map[string]struct{}{"missing_tool": {}},
				},
			},
			loader: func(cleanups *int) func(context.Context, MCPConfig) ([]tool.BaseTool, func() error, error) {
				return func(_ context.Context, cfg MCPConfig) ([]tool.BaseTool, func() error, error) {
					loaded := &ToolAdapter{inner: &simpleTool{
						info: &schema.ToolInfo{Name: cfg.Name, Desc: "mcp"},
						ret:  `{"source":"mcp"}`,
					}}
					return []tool.BaseTool{loaded}, func() error {
						*cleanups = *cleanups + 1
						return nil
					}, nil
				}
			},
			wantErr:      "validate direct return tools: direct return tool not found: missing_tool",
			wantCleanups: 1,
		},
	}

	originalLoader := mcpToolLoader
	defer func() {
		mcpToolLoader = originalLoader
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupCalls := 0
			mcpToolLoader = tt.loader(&cleanupCalls)

			spec, err := compileRuntimeSpec(tt.cfg)
			if err != nil {
				t.Fatalf("compileRuntimeSpec failed: %v", err)
			}

			_, err = buildPromptAndTools(context.Background(), spec)
			if err == nil {
				t.Fatal("expected buildPromptAndTools to fail")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("unexpected error: got %q want %q", err.Error(), tt.wantErr)
			}
			if cleanupCalls != tt.wantCleanups {
				t.Fatalf("unexpected cleanup call count: got %d want %d", cleanupCalls, tt.wantCleanups)
			}
		})
	}
}
