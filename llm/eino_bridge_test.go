package llm

import (
	"context"
	"testing"
)

func TestRegisterEinoProvider(t *testing.T) {
	RegisterEinoProvider("eino-stub", func(cfg Config) (EinoExecutor, error) {
		return einoExecutorFunc(func(ctx context.Context, messages []Message, options RequestOptions) (*ChatCompletion, error) {
			return &ChatCompletion{
				Model:    cfg.Model,
				Provider: Provider("eino-stub"),
				Choices: []CompletionChoice{{
					Message: Message{
						Role:    RoleAssistant,
						Content: messages[0].Content + " + echo",
					},
				}},
			}, nil
		}), nil
	})

	client, err := NewClient(context.Background(), Config{
		Provider: "eino-stub",
		Model:    "demo-model",
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Chat(context.Background(), []Message{{Role: RoleUser, Content: "ping"}})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}

	if resp.Model != "demo-model" || resp.Choices[0].Message.Content != "ping + echo" {
		t.Fatalf("unexpected eino result: %+v", resp)
	}
}

type einoExecutorFunc func(ctx context.Context, messages []Message, options RequestOptions) (*ChatCompletion, error)

func (f einoExecutorFunc) Chat(ctx context.Context, messages []Message, options RequestOptions) (*ChatCompletion, error) {
	return f(ctx, messages, options)
}
