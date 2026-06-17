package llm

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// ── ADK Extraction 端到端测试 ────────────────────────────────────────

func TestNewADKAgent_Extraction(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{
			// 第一次：返回 tool call（强制调用）
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{ID: "call_1", Function: schema.FunctionCall{Name: "extract", Arguments: `{}`}},
				},
			},
			// 第二次：工具执行成功后，返回最终文本
			{Role: schema.Assistant, Content: "extraction done"},
		},
	}
	et := &simpleTool{
		info: &schema.ToolInfo{Name: "extract", Desc: "extract tool"},
		ret:  `{"result":"ok"}`,
	}

	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model:     AgentModelConfig{Instance: fm},
		Tools:     ToolsConfig{Invokable: []InvokableTool{et}},
		Execution: ExecutionConfig{Mode: Extraction},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer agent.Close()

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		schema.UserMessage("extract data"),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if msg.Content != "extraction done" {
		t.Fatalf("expected 'extraction done', got %q", msg.Content)
	}
	if et.calls == 0 {
		t.Fatal("expected tool to be called at least once")
	}
	if fm.calls < 2 {
		t.Fatalf("expected model to be called at least twice, got %d", fm.calls)
	}
}

func TestNewADKAgent_Extraction_DirectReturn(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{ID: "call_1", Function: schema.FunctionCall{Name: "fetch", Arguments: `{}`}},
				},
			},
			{Role: schema.Assistant, Content: "should not reach here"},
		},
	}
	et := &simpleTool{
		info: &schema.ToolInfo{Name: "fetch", Desc: "fetch tool"},
		ret:  `{"data":"fetched"}`,
	}

	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model:     AgentModelConfig{Instance: fm},
		Tools:     ToolsConfig{Invokable: []InvokableTool{et}},
		Execution: ExecutionConfig{
			Mode:              Extraction,
			DirectReturnTools: map[string]struct{}{"fetch": {}},
		},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer agent.Close()

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		schema.UserMessage("fetch data"),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// DirectReturn 模式下，工具结果应直接返回
	if msg.Content != `{"data":"fetched"}` {
		t.Fatalf("expected direct return of tool result, got %q", msg.Content)
	}
	if et.calls == 0 {
		t.Fatal("expected tool to be called")
	}
}

// ── ADK Stream 测试 ──────────────────────────────────────────────────

func TestNewADKAgent_Stream(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{{Role: schema.Assistant, Content: "stream chunk"}},
	}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: fm},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer agent.Close()

	sr, err := agent.Stream(context.Background(), []*schema.Message{
		schema.UserMessage("hi"),
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer sr.Close()

	var content string
	for {
		msg, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		content += msg.Content
	}
	if content == "" {
		t.Fatal("expected non-empty stream content")
	}
}

func TestNewADKAgent_Stream_WithConcurrency(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{{Content: "ok"}},
	}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model:       AgentModelConfig{Instance: fm},
		Concurrency: ConcurrencyConfig{MaxConcurrency: 1},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer agent.Close()

	// 消费流后 guard 应释放，第二次调用不应阻塞
	sr1, err := agent.Stream(context.Background(), []*schema.Message{schema.UserMessage("a")})
	if err != nil {
		t.Fatalf("Stream 1: %v", err)
	}
	consumeStream(t, sr1)

	sr2, err := agent.Stream(context.Background(), []*schema.Message{schema.UserMessage("b")})
	if err != nil {
		t.Fatalf("Stream 2: %v", err)
	}
	consumeStream(t, sr2)
}

func consumeStream(t *testing.T, sr *schema.StreamReader[*schema.Message]) {
	t.Helper()
	defer sr.Close()
	for {
		_, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
	}
}

// ── ADK Observability 测试 ───────────────────────────────────────────

type captureLogClient struct {
	mu     sync.Mutex
	events []string
}

func (c *captureLogClient) Info(_ context.Context, _ string, fields ...any) {
	c.recordEvent(fields...)
}

func (c *captureLogClient) Error(_ context.Context, _ string, fields ...any) {
	c.recordEvent(fields...)
}

func (c *captureLogClient) recordEvent(fields ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 0; i+1 < len(fields); i += 2 {
		if key, ok := fields[i].(string); ok && key == "event" {
			if ev, ok := fields[i+1].(string); ok {
				c.events = append(c.events, ev)
			}
		}
	}
}

func (c *captureLogClient) hasEvent(event string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.events {
		if e == event {
			return true
		}
	}
	return false
}

func TestNewADKAgent_Observability_Logs(t *testing.T) {
	client := &captureLogClient{}
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{{Role: schema.Assistant, Content: "ok"}},
	}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: fm},
		Observability: ObservabilityConfig{
			StructuredLogs: &StructuredLogConfig{
				Client: client,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer agent.Close()

	_, err = agent.Generate(context.Background(), []*schema.Message{
		schema.UserMessage("hi"),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !client.hasEvent("agent.start") {
		t.Error("expected agent.start event in logs")
	}
	if !client.hasEvent("model.decision") {
		t.Error("expected model.decision event in logs")
	}
	if !client.hasEvent("agent.end") {
		t.Error("expected agent.end event in logs")
	}
}
