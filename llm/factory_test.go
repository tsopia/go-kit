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
