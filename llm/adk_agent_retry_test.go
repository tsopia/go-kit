package llm

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// retryCountModel 记录调用次数，前 failTimes 次返回 triggerErr，之后返回成功。
type retryCountModel struct {
	calls      atomic.Int32
	failTimes  int32
	triggerErr error
}

func (m *retryCountModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	n := m.calls.Add(1)
	if n <= m.failTimes {
		return nil, m.triggerErr
	}
	return &schema.Message{Role: schema.Assistant, Content: "ok"}, nil
}

func (m *retryCountModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not used")
}

func (m *retryCountModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func TestNewADKAgent_ModelRetry_TriggersOnTransientError(t *testing.T) {
	m := &retryCountModel{failTimes: 1, triggerErr: errors.New("http 429 Too Many Requests")}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model:     AgentModelConfig{Instance: m},
		Resilience: ResilienceConfig{ModelRetry: ModelRetryConfig{MaxRetries: 2}},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer agent.Close()

	_, err = agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got := m.calls.Load(); got != 2 {
		t.Errorf("model called %d times, want 2", got)
	}
}

func TestNewADKAgent_ModelRetry_NoRetryWhenZero(t *testing.T) {
	m := &retryCountModel{failTimes: 1, triggerErr: errors.New("http 429 Too Many Requests")}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: m},
		// Resilience not set → MaxRetries = 0 → no retry
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer agent.Close()

	_, err = agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if got := m.calls.Load(); got != 1 {
		t.Errorf("model called %d times, want 1", got)
	}
}

func TestNewADKAgent_ModelRetry_CustomIsRetryAble(t *testing.T) {
	sentinel := errors.New("custom-transient")
	m := &retryCountModel{failTimes: 1, triggerErr: sentinel}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: m},
		Resilience: ResilienceConfig{ModelRetry: ModelRetryConfig{
			MaxRetries:  2,
			IsRetryAble: func(err error) bool { return errors.Is(err, sentinel) },
		}},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer agent.Close()

	_, err = agent.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hi"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got := m.calls.Load(); got != 2 {
		t.Errorf("model called %d times, want 2", got)
	}
}
