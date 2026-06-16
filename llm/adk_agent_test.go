package llm

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// trackingModel 可注入行为的 model，用于测试并发和工具调用。
type trackingModel struct {
	generateFn func(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error)
	streamFn   func(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error)
}

func (m *trackingModel) Generate(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return m.generateFn(ctx, msgs, opts...)
}

func (m *trackingModel) Stream(ctx context.Context, msgs []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return m.streamFn(ctx, msgs, opts...)
}

func (m *trackingModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func TestNewADKAgent_MissingModel(t *testing.T) {
	_, err := NewADKAgent(context.Background(), AgentConfig{})
	if err == nil {
		t.Fatal("expected error when no model configured")
	}
}

func TestNewADKAgent_Conversation(t *testing.T) {
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{
			{Role: schema.Assistant, Content: "hello from adk"},
		},
	}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: fm},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer agent.Close()

	msg, err := agent.Generate(context.Background(), []*schema.Message{
		schema.UserMessage("hi"),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if msg.Content != "hello from adk" {
		t.Fatalf("expected 'hello from adk', got %q", msg.Content)
	}
	if fm.calls == 0 {
		t.Fatal("expected model.Generate to be called at least once")
	}
}

func TestADKAgent_Concurrency(t *testing.T) {
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
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model:       AgentModelConfig{Instance: tm},
		Concurrency: ConcurrencyConfig{MaxConcurrency: 2},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
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

func TestADKAgent_Concurrency_AcquireCancelledByContext(t *testing.T) {
	tm := &trackingModel{
		generateFn: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			time.Sleep(200 * time.Millisecond)
			return &schema.Message{Content: "ok"}, nil
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
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model:       AgentModelConfig{Instance: tm},
		Concurrency: ConcurrencyConfig{MaxConcurrency: 1},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer agent.Close()

	// 占满唯一的并发名额
	blockDone := make(chan struct{})
	go func() {
		defer close(blockDone)
		_, _ = agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("block")})
	}()

	// 等待占满
	time.Sleep(20 * time.Millisecond)

	// 第二个请求用短超时 ctx，应该在 acquire 阶段被取消
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = agent.Generate(ctx, []*schema.Message{schema.UserMessage("queued")})
	if err == nil {
		t.Fatal("expected error due to context cancellation while waiting for concurrency slot")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}

	<-blockDone
}
