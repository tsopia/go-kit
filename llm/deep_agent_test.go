package llm

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestNewDeepAgent_MissingModel(t *testing.T) {
	_, err := NewDeepAgent(context.Background(), DeepAgentConfig{
		Model: AgentModelConfig{Config: ModelConfig{}},
	})
	if err == nil {
		t.Fatal("expected error when no model configured")
	}
}

func TestNewDeepAgent_Basic(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{
			{Role: schema.Assistant, Content: "deep agent response"},
		},
	}
	agent, err := NewDeepAgent(context.Background(), DeepAgentConfig{
		Model: AgentModelConfig{Instance: fm},
		Name:  "test-deep",
	})
	if err != nil {
		t.Fatalf("NewDeepAgent: %v", err)
	}
	defer agent.Close()

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		schema.UserMessage("hello"),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if msg.Content != "deep agent response" {
		t.Fatalf("expected 'deep agent response', got %q", msg.Content)
	}
}

func TestNewDeepAgent_WithInMemoryBackend(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{
			{Role: schema.Assistant, Content: "ok"},
		},
	}
	agent, err := NewDeepAgent(context.Background(), DeepAgentConfig{
		Model:   AgentModelConfig{Instance: fm},
		Name:    "test-deep-fs",
		Backend: filesystem.NewInMemoryBackend(),
	})
	if err != nil {
		t.Fatalf("NewDeepAgent: %v", err)
	}
	defer agent.Close()

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		schema.UserMessage("list files"),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
}

func TestNewDeepAgent_Concurrency(t *testing.T) {
	var current, maxSeen atomic.Int32
	tm := &trackingModel{
		generateFn: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			cur := current.Add(1)
			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			current.Add(-1)
			return &schema.Message{Role: schema.Assistant, Content: "ok"}, nil
		},
		streamFn: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
			sr, sw := schema.Pipe[*schema.Message](1)
			go func() {
				defer sw.Close()
				sw.Send(&schema.Message{Content: "ok"}, nil)
			}()
			return sr, nil
		},
	}
	agent, err := NewDeepAgent(context.Background(), DeepAgentConfig{
		Model:       AgentModelConfig{Instance: tm},
		Concurrency: ConcurrencyConfig{MaxConcurrency: 2},
	})
	if err != nil {
		t.Fatalf("NewDeepAgent: %v", err)
	}
	defer agent.Close()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("x")})
			if err != nil {
				t.Errorf("Generate error: %v", err)
			}
		}()
	}
	wg.Wait()

	if max := maxSeen.Load(); max > 2 {
		t.Fatalf("max concurrent %d exceeded limit 2", max)
	}
}
