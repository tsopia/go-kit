package llm

import (
	"context"
	"testing"
)

func TestNewModelValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ModelConfig
		wantErr string
	}{
		{
			name:    "missing model",
			cfg:     ModelConfig{Protocol: OPENAI, APIKey: "key"},
			wantErr: "model is required",
		},
		{
			name:    "compat missing base url",
			cfg:     ModelConfig{Protocol: OPENAI_COMPAT, APIKey: "key", Model: "gpt-4"},
			wantErr: "base url is required",
		},
		{
			name:    "missing api key non-ollama",
			cfg:     ModelConfig{Protocol: DEEPSEEK, Model: "deepseek-chat"},
			wantErr: "api key is required",
		},
		{
			name:    "unsupported protocol",
			cfg:     ModelConfig{Protocol: "FOO", APIKey: "key", Model: "m"},
			wantErr: "unsupported protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewModel(context.Background(), tt.cfg)
			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantErr)
			}
			if !containsStr(err.Error(), tt.wantErr) {
				t.Fatalf("expected error %q to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestOllamaSkipsAPIKeyCheck(t *testing.T) {
	_, err := NewModel(context.Background(), ModelConfig{Protocol: OLLAMA, Model: "llama3"})
	// Ollama 可以不传 APIKey，但可能因为连不上 host 而失败
	// 我们只验证不会因 "api key is required" 而失败
	if err != nil && containsStr(err.Error(), "api key is required") {
		t.Fatalf("ollama should not require api key: %v", err)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && findStr(s, sub)
}

func findStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
