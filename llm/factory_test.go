package llm

import (
	"context"
	"fmt"
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
			wantErr: "unsupported model protocol",
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

func TestNewModelFactory_AfterUpgrade(t *testing.T) {
	tests := []struct {
		name string
		cfg  ModelConfig
	}{
		{
			name: "openai",
			cfg: ModelConfig{
				Protocol: OPENAI,
				Model:    "gpt-4o-mini",
				APIKey:   "test-key",
			},
		},
		{
			name: "qwen",
			cfg: ModelConfig{
				Protocol: QWEN,
				Model:    "qwen-plus",
				APIKey:   "test-key",
			},
		},
		{
			name: "gemini",
			cfg: ModelConfig{
				Protocol: GEMINI,
				Model:    "gemini-2.0-flash",
				APIKey:   "test-key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewModel(context.Background(), tt.cfg)
			if err != nil {
				t.Fatalf("NewModel failed: %v", err)
			}
			if model == nil {
				t.Fatal("expected non-nil model")
			}
		})
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

func TestThinkingConfig_Mapping(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ModelConfig
		wantErr bool
	}{
		{
			name: "claude_thinking_enabled",
			cfg: ModelConfig{
				Protocol: CLAUDE,
				Model:    "claude-sonnet-4-20250514",
				APIKey:   "test-key",
				Thinking: &ThinkingConfig{Enable: true, BudgetTokens: 10000},
			},
		},
		{
			name: "claude_thinking_disabled",
			cfg: ModelConfig{
				Protocol: CLAUDE,
				Model:    "claude-sonnet-4-20250514",
				APIKey:   "test-key",
				Thinking: &ThinkingConfig{Enable: false},
			},
		},
		{
			name: "openai_thinking_via_extra_fields",
			cfg: ModelConfig{
				Protocol: OPENAI,
				Model:    "gpt-4o-mini",
				APIKey:   "test-key",
				Thinking: &ThinkingConfig{Enable: true},
			},
		},
		{
			name: "kimi_thinking_disabled",
			cfg: ModelConfig{
				Protocol: KIMI,
				Model:    "kimi-k2.6",
				APIKey:   "test-key",
				Thinking: &ThinkingConfig{Enable: false},
			},
		},
		{
			name: "qwen_thinking_enabled",
			cfg: ModelConfig{
				Protocol: QWEN,
				Model:    "qwen3.5-plus",
				APIKey:   "test-key",
				Thinking: &ThinkingConfig{Enable: true},
			},
		},
		{
			name: "deepseek_thinking_enabled",
			cfg: ModelConfig{
				Protocol: DEEPSEEK,
				Model:    "deepseek-reasoner",
				APIKey:   "test-key",
				Thinking: &ThinkingConfig{Enable: true},
			},
		},
		{
			name: "gemini_thinking_with_budget",
			cfg: ModelConfig{
				Protocol: GEMINI,
				Model:    "gemini-2.5-flash",
				APIKey:   "test-key",
				Thinking: &ThinkingConfig{Enable: true, BudgetTokens: 8192},
			},
		},
		{
			name: "ollama_thinking_enabled",
			cfg: ModelConfig{
				Protocol: OLLAMA,
				Model:    "qwen3",
				Thinking: &ThinkingConfig{Enable: true},
			},
		},
		{
			name: "ark_thinking_enabled",
			cfg: ModelConfig{
				Protocol: ARK,
				Model:    "doubao-1.5-pro",
				APIKey:   "test-key",
				Thinking: &ThinkingConfig{Enable: true},
			},
		},
		{
			name: "openai_nil_thinking_no_extra_fields",
			cfg: ModelConfig{
				Protocol: OPENAI,
				Model:    "gpt-4o-mini",
				APIKey:   "test-key",
			},
		},
		{
			name: "qianfan_no_thinking_support_ignored",
			cfg: ModelConfig{
				Protocol: QIANFAN,
				Model:    "ernie-4.0",
				Thinking: &ThinkingConfig{Enable: true},
			},
		},
		{
			name: "openai_extra_fields_only",
			cfg: ModelConfig{
				Protocol:    OPENAI,
				Model:       "gpt-4o-mini",
				APIKey:      "test-key",
				ExtraFields: map[string]any{"custom_param": "value"},
			},
		},
		{
			name: "openai_thinking_plus_extra_fields",
			cfg: ModelConfig{
				Protocol:    OPENAI,
				Model:       "gpt-4o-mini",
				APIKey:      "test-key",
				Thinking:    &ThinkingConfig{Enable: true},
				ExtraFields: map[string]any{"custom_param": "value"},
			},
		},
		{
			name: "openai_extra_fields_override_thinking",
			cfg: ModelConfig{
				Protocol:    OPENAI,
				Model:       "gpt-4o-mini",
				APIKey:      "test-key",
				Thinking:    &ThinkingConfig{Enable: true},
				ExtraFields: map[string]any{"thinking": map[string]any{"type": "custom"}},
			},
		},
		{
			name: "claude_extra_fields_passthrough",
			cfg: ModelConfig{
				Protocol:    CLAUDE,
				Model:       "claude-sonnet-4-20250514",
				APIKey:      "test-key",
				Thinking:    &ThinkingConfig{Enable: true, BudgetTokens: 5000},
				ExtraFields: map[string]any{"future_param": 42},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewModel(context.Background(), tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMergeOpenAIExtraFields(t *testing.T) {
	tests := []struct {
		name     string
		thinking *ThinkingConfig
		extra    map[string]any
		want     map[string]any
	}{
		{
			name:     "nil_both",
			thinking: nil,
			extra:    nil,
			want:     map[string]any{},
		},
		{
			name:     "thinking_enabled_no_extra",
			thinking: &ThinkingConfig{Enable: true},
			extra:    nil,
			want: map[string]any{
				"thinking": map[string]any{"type": "enabled"},
			},
		},
		{
			name:     "thinking_disabled_with_extra",
			thinking: &ThinkingConfig{Enable: false},
			extra:    map[string]any{"foo": "bar"},
			want: map[string]any{
				"foo":      "bar",
				"thinking": map[string]any{"type": "disabled"},
			},
		},
		{
			name:     "extra_overrides_thinking",
			thinking: &ThinkingConfig{Enable: true},
			extra:    map[string]any{"thinking": map[string]any{"type": "custom"}},
			want: map[string]any{
				"thinking": map[string]any{"type": "custom"},
			},
		},
		{
			name:     "only_extra_no_thinking",
			thinking: nil,
			extra:    map[string]any{"key": "val"},
			want: map[string]any{
				"key": "val",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeOpenAIExtraFields(tt.thinking, tt.extra)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d fields, want %d fields", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				gotV, ok := got[k]
				if !ok {
					t.Fatalf("missing key %q", k)
				}
				if fmt.Sprintf("%v", gotV) != fmt.Sprintf("%v", v) {
					t.Fatalf("key %q: got %v, want %v", k, gotV, v)
				}
			}
		})
	}
}
