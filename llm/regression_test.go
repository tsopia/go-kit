package llm

import (
	"context"
	"sync"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// F-6: react 路径（NewAgent）流式也应补记 model.decision（含 usage）。
// 此前只有 ADK 路径有对应测试。
func TestNewAgent_Stream_LogsDecisionWithUsage(t *testing.T) {
	client := &recordingLogClient{}
	fm := &fakeToolCallingModel{
		responses: []*schema.Message{{
			Role:    schema.Assistant,
			Content: "hi",
			ResponseMeta: &schema.ResponseMeta{
				FinishReason: "stop",
				Usage:        &schema.TokenUsage{PromptTokens: 3, CompletionTokens: 4, TotalTokens: 7},
			},
		}},
	}
	agent, err := NewAgent(context.Background(), AgentConfig{
		Model:         AgentModelConfig{Instance: fm},
		Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{Client: client}},
	})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	defer func() { _ = agent.Close() }()

	sr, err := agent.Stream(context.Background(), []*schema.Message{schema.UserMessage("hi")})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	drain(sr)

	entry, ok := findEntry(client.snapshot(), "model.decision")
	if !ok {
		t.Fatal("react Stream should log a model.decision entry")
	}
	if entry.fields["streaming"] != true {
		t.Errorf("model.decision should mark streaming=true, got %v", entry.fields["streaming"])
	}
	if entry.fields["total_tokens"] == nil {
		t.Errorf("model.decision should include total_tokens, fields: %v", entry.fields)
	}
}

// F-6: 并发调用 Generate 携带不同运行时 Option，应互不串扰且无数据竞争（-race）。
func TestADKAgent_RuntimeOptions_ConcurrentNoRace(t *testing.T) {
	var mu sync.Mutex
	seen := map[float32]bool{}
	tm := &trackingModel{
		generateFn: func(_ context.Context, _ []*schema.Message, opts ...model.Option) (*schema.Message, error) {
			o := model.GetCommonOptions(nil, opts...)
			if o.Temperature != nil {
				mu.Lock()
				seen[*o.Temperature] = true
				mu.Unlock()
			}
			return &schema.Message{Role: schema.Assistant, Content: "ok"}, nil
		},
	}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model: AgentModelConfig{Instance: tm},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer func() { _ = agent.Close() }()

	temps := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	var wg sync.WaitGroup
	for _, tp := range temps {
		wg.Add(1)
		go func(tp float32) {
			defer wg.Done()
			if _, err := agent.Generate(context.Background(),
				[]*schema.Message{schema.UserMessage("x")}, WithTemperature(tp)); err != nil {
				t.Errorf("Generate(temp=%v): %v", tp, err)
			}
		}(tp)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	for _, tp := range temps {
		if !seen[tp] {
			t.Errorf("temperature %v was not observed by the model (option lost/cross-talk)", tp)
		}
	}
}
