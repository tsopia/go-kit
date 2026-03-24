package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// ── 用户配置区 ──────────────────────────────────────────────────────────

var IntegrationTestConfig = ModelConfig{
	Protocol: OPENAI_COMPAT,
	Model:    "gpt-5.2",
	APIKey:   "sk-XZ9c3ZXxxxxxx", // 填入真实 API Key 以运行集成测试
	BaseURL:  "https://xxxxxx",
}

func ensureConfig(t *testing.T) {
	// 跳过需要真实 API 调用的集成测试
	t.Skip("Skipping integration test: requires real API endpoint")
}

// ── Mock 工具定义 ───────────────────────────────────────────────────────

type StockTool struct {
	called int
}

func (s *StockTool) Info() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "get_stock_price",
		Desc: "Get current stock price for a given symbol.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"symbol": {
				Type:     schema.String,
				Desc:     "The stock symbol, e.g. AAPL, GOOGL",
				Required: true,
			},
		}),
	}
}

func (s *StockTool) Invoke(_ context.Context, argsStr string) (string, error) {
	s.called++
	fmt.Printf("[Tool] get_stock_price called with: %s\n", argsStr)
	return `{"symbol": "AAPL", "price": 150.25, "currency": "USD"}`, nil
}

// ── 集成测试 ──────────────────────────────────────────────────────────

func TestReal_Agent_ToolCall(t *testing.T) {
	ensureConfig(t)

	tool := &StockTool{}

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model:  AgentModelConfig{Config: IntegrationTestConfig},
		Tools:  ToolsConfig{Invokable: []InvokableTool{tool}},
		Prompt: PromptConfig{System: "你是一个股票助手。用户会询问股票信息，请使用工具查询后回复。"},
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("\n=== TestReal_Agent_ToolCall ===")
	msg, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "查一下 AAPL 的股价"},
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Response: %s\n", msg.Content)
	if tool.called == 0 {
		t.Error("expected tool to be called")
	}
	if msg.Content == "" {
		t.Error("expected non-empty response")
	}
}

func TestReal_Agent_SimpleChat(t *testing.T) {
	ensureConfig(t)

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Config: IntegrationTestConfig},
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("\n=== TestReal_Agent_SimpleChat ===")
	msg, err := agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "用一句话介绍 Go 语言"},
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Response: %s\n", msg.Content)
	if msg.Content == "" {
		t.Error("expected non-empty response")
	}
}

func TestReal_Agent_Stream(t *testing.T) {
	ensureConfig(t)

	agent, err := NewAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Config: IntegrationTestConfig},
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("\n=== TestReal_Agent_Stream ===")
	stream, err := agent.Stream(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "写一首五言绝句"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	var b strings.Builder
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatal(err)
		}
		b.WriteString(chunk.Content)
	}

	result := b.String()
	fmt.Printf("Stream Output: %s\n", result)
	if result == "" {
		t.Error("expected non-empty stream output")
	}
}
