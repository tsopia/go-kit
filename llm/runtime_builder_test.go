package llm

import (
	"context"
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
