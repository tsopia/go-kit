package llm

import "testing"

func TestNewToolCallingModelValidation(t *testing.T) {
	_, err := NewToolCallingModel(ModelConfig{Protocol: OPENAI_COMPAT, Model: "m"})
	if err == nil {
		t.Fatal("expected base url error")
	}
	_, err = NewToolCallingModel(ModelConfig{Protocol: OPENAI_COMPAT, Model: "m", BaseURL: "https://x"})
	if err == nil {
		t.Fatal("expected api key error")
	}
	_, err = NewToolCallingModel(ModelConfig{Protocol: ModelProtocol("X"), APIKey: "k", Model: "m"})
	if err == nil {
		t.Fatal("expected protocol error")
	}
}

func TestNewToolCallingModelSupportsEinoExtProtocols(t *testing.T) {
	protocols := []ModelProtocol{ARK, DEEPSEEK, ARKBOT, CLAUDE, GEMINI, OLLAMA, OPENAI, QIANFAN, QWEN, OPENAI_COMPAT, CLAUDE_COMPAT}
	for _, p := range protocols {
		cfg := ModelConfig{Protocol: p, Model: "m", APIKey: "k", BaseURL: "https://x"}
		if p == OLLAMA {
			cfg.APIKey = ""
		}
		if p != OPENAI_COMPAT && p != CLAUDE_COMPAT {
			cfg.BaseURL = ""
		}
		if _, err := NewToolCallingModel(cfg); err != nil {
			t.Fatalf("protocol %s should be supported, got err=%v", p, err)
		}
	}
}

func TestCompatProtocolRequiresBaseURL(t *testing.T) {
	_, err := NewToolCallingModel(ModelConfig{Protocol: OPENAI_COMPAT, APIKey: "k", Model: "m"})
	if err == nil {
		t.Fatal("expected base url error for openai compat")
	}
	_, err = NewToolCallingModel(ModelConfig{Protocol: CLAUDE_COMPAT, APIKey: "k", Model: "m"})
	if err == nil {
		t.Fatal("expected base url error for claude compat")
	}
}
