package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAIAdapterChat(t *testing.T) {
	var captured openAIChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("missing authorization header, got: %s", got)
		}

		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		resp := openAIChatResponse{
			ID:      "chatcmpl-1",
			Model:   "gpt-4o",
			Created: time.Now().Unix(),
			Choices: []openAIChatResponseItem{
				{
					Index: 0,
					Message: openAIMessage{
						Role:    RoleAssistant,
						Content: "hi",
					},
					FinishReason: "stop",
				},
			},
			Usage: openAIUsage{
				PromptTokens:     3,
				CompletionTokens: 7,
				TotalTokens:      10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := Config{
		Provider: ProviderOpenAI,
		Model:    "gpt-4o",
		BaseURL:  server.URL,
		APIKey:   "test-key",
		Options: RequestOptions{
			Temperature: floatPtr(0.1),
		},
	}

	client, err := NewClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	completion, err := client.Chat(
		context.Background(),
		[]Message{{Role: RoleUser, Content: "hello"}},
		WithMaxTokens(16),
	)
	if err != nil {
		t.Fatalf("chat: %v", err)
	}

	if completion.Model != "gpt-4o" || len(completion.Choices) != 1 {
		t.Fatalf("unexpected completion: %+v", completion)
	}

	if captured.MaxTokens == nil || *captured.MaxTokens != 16 {
		t.Fatalf("max_tokens not merged: %+v", captured.MaxTokens)
	}
	if captured.Temperature == nil || *captured.Temperature != 0.1 {
		t.Fatalf("default temperature not applied: %+v", captured.Temperature)
	}
	if captured.Messages[0].Content != "hello" || captured.Messages[0].Role != RoleUser {
		t.Fatalf("unexpected message payload: %+v", captured.Messages[0])
	}
}

func TestOptionMerge(t *testing.T) {
	merged := mergeOptions(RequestOptions{
		Headers: map[string]string{"X-Test": "base"},
		Stop:    []string{"stop"},
	}, WithHeaders(map[string]string{"X-Test": "override", "X-Another": "y"}), WithStop("a", "b"))

	if merged.Headers["X-Test"] != "override" || merged.Headers["X-Another"] != "y" {
		t.Fatalf("headers not merged: %+v", merged.Headers)
	}
	if len(merged.Stop) != 2 || merged.Stop[0] != "a" {
		t.Fatalf("stop not overridden: %+v", merged.Stop)
	}
}

func TestStreamNotSupported(t *testing.T) {
	cfg := Config{
		Provider: ProviderOpenAI,
		Model:    "gpt-4o-mini",
		BaseURL:  "http://example.com",
		APIKey:   "k",
	}
	client := &Client{
		provider:       cfg.Provider,
		adapter:        &openAIAdapter{config: cfg, client: nil},
		defaultOptions: RequestOptions{},
	}

	_, _, err := client.Stream(context.Background(), []Message{{Role: RoleUser, Content: "hi"}})
	if err == nil || err != ErrStreamNotSupported {
		t.Fatalf("expected ErrStreamNotSupported, got %v", err)
	}
}

func TestUnregisteredProvider(t *testing.T) {
	_, err := NewClient(context.Background(), Config{
		Provider: "unknown",
		Model:    "test",
	})
	if err == nil {
		t.Fatalf("expected error for unknown provider")
	}
}

func floatPtr(v float64) *float64 { return &v }
