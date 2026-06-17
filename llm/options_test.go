package llm

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestADKAgent_RuntimeOptions_AppliedToModel(t *testing.T) {
	var got *model.Options
	tm := &trackingModel{
		generateFn: func(_ context.Context, _ []*schema.Message, opts ...model.Option) (*schema.Message, error) {
			got = model.GetCommonOptions(nil, opts...)
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

	_, err = agent.Generate(context.Background(),
		[]*schema.Message{schema.UserMessage("hi")},
		WithTemperature(0.1), WithMaxTokens(256), WithTopP(0.9),
	)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got == nil {
		t.Fatal("model did not receive options")
	}
	if got.Temperature == nil || *got.Temperature != 0.1 {
		t.Errorf("temperature = %v, want 0.1", got.Temperature)
	}
	if got.MaxTokens == nil || *got.MaxTokens != 256 {
		t.Errorf("maxTokens = %v, want 256", got.MaxTokens)
	}
	if got.TopP == nil || *got.TopP != 0.9 {
		t.Errorf("topP = %v, want 0.9", got.TopP)
	}
}

func TestADKAgent_RuntimeOptions_NoneIsBackwardCompatible(t *testing.T) {
	var got *model.Options
	tm := &trackingModel{
		generateFn: func(_ context.Context, _ []*schema.Message, opts ...model.Option) (*schema.Message, error) {
			got = model.GetCommonOptions(nil, opts...)
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

	if _, err := agent.Generate(context.Background(), []*schema.Message{schema.UserMessage("hi")}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got != nil && got.Temperature != nil {
		t.Errorf("no options should be applied, got temperature %v", *got.Temperature)
	}
}

func TestGenerateConfig_BuildsModelOptions(t *testing.T) {
	c := buildGenerateConfig([]GenerateOption{WithTemperature(0.5)})
	if len(c.modelOpts) != 1 {
		t.Fatalf("expected 1 model option, got %d", len(c.modelOpts))
	}
	if len(c.runOptions()) != 1 {
		t.Fatalf("expected 1 run option")
	}
	empty := buildGenerateConfig(nil)
	if empty.runOptions() != nil {
		t.Errorf("no options should yield nil run options")
	}
}
