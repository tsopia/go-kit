package llm

import "testing"

func TestNewToolCallingModelValidation(t *testing.T) {
	_, err := NewToolCallingModel(ModelConfig{Protocol: OPENAI_COMPAT, Model: "m"})
	if err == nil {
		t.Fatal("expected api key error")
	}
	_, err = NewToolCallingModel(ModelConfig{Protocol: ModelProtocol("X"), APIKey: "k", Model: "m"})
	if err == nil {
		t.Fatal("expected protocol error")
	}
}

func TestModelConfigNormalizedBaseURLForKnownOpenAICompatVendors(t *testing.T) {
	qwenCfg := (ModelConfig{Protocol: OPENAI_COMPAT, Model: "qwen-plus"}).normalized()
	if qwenCfg.BaseURL != "https://dashscope.aliyuncs.com/compatible-mode/v1" {
		t.Fatalf("unexpected qwen base url: %s", qwenCfg.BaseURL)
	}

	deepseekCfg := (ModelConfig{Protocol: OPENAI_COMPAT, Model: "deepseek-chat"}).normalized()
	if deepseekCfg.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("unexpected deepseek base url: %s", deepseekCfg.BaseURL)
	}

	genericCfg := (ModelConfig{Protocol: OPENAI_COMPAT, Model: "gpt-4o-mini"}).normalized()
	if genericCfg.BaseURL != "" {
		t.Fatalf("unexpected generic base url: %s", genericCfg.BaseURL)
	}
}
