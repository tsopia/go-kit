package llm

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// flagMiddleware 记录 BeforeAgent 是否被触发，用于验证用户自定义 middleware 接入。
type flagMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	fired *atomic.Int32
	order *[]string
	tag   string
}

func (m *flagMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	m.fired.Add(1)
	if m.order != nil {
		*m.order = append(*m.order, m.tag)
	}
	return ctx, runCtx, nil
}

func TestNewADKAgent_CustomMiddleware_Fires(t *testing.T) {
	var fired atomic.Int32
	mw := &flagMiddleware{fired: &fired}
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{{Role: schema.Assistant, Content: "ok"}},
	}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model:       AgentModelConfig{Instance: fm},
		Middlewares: []adk.ChatModelAgentMiddleware{mw},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer func() { _ = agent.Close() }()

	if _, err := agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if fired.Load() == 0 {
		t.Fatal("custom middleware BeforeAgent was not invoked")
	}
}

func TestNewADKAgent_CustomMiddleware_OrderBeforePackage(t *testing.T) {
	var fired atomic.Int32
	var order []string
	mw1 := &flagMiddleware{fired: &fired, order: &order, tag: "user1"}
	mw2 := &flagMiddleware{fired: &fired, order: &order, tag: "user2"}
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{{Role: schema.Assistant, Content: "ok"}},
	}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model:       AgentModelConfig{Instance: fm},
		Middlewares: []adk.ChatModelAgentMiddleware{mw1, mw2},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer func() { _ = agent.Close() }()

	if _, err := agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(order) < 2 || order[0] != "user1" || order[1] != "user2" {
		t.Fatalf("user middlewares should fire in registration order, got %v", order)
	}
}

func TestNewADKAgent_NoMiddleware_NoRegression(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{{Role: schema.Assistant, Content: "plain"}},
	}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: fm},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer func() { _ = agent.Close() }()

	msg, err := agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if msg.Content != "plain" {
		t.Fatalf("expected 'plain', got %q", msg.Content)
	}
}
