package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// ── 环境变量读取 ─────────────────────────────────────────────────────────

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// ── DeepSeek 流式 inspect ───────────────────────────────────────────────

func TestInspectDeepSeekStream(t *testing.T) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	model := envOrDefault("DEEPSEEK_MODEL", "deepseek-reasoner")
	if apiKey == "" {
		t.Skip("DEEPSEEK_API_KEY required")
	}

	ctx := context.Background()
	cfg := ModelConfig{
		Protocol: DEEPSEEK,
		APIKey:   apiKey,
		Model:    model,
		Thinking: &ThinkingConfig{Enable: true},
	}

	agent, err := NewAgent(ctx, AgentConfig{
		Model:     AgentModelConfig{Config: cfg},
		Execution: ExecutionConfig{Mode: Conversation},
	})
	if err != nil {
		t.Fatalf("new agent: %v", err)
	}
	defer agent.Close()

	messages := []*schema.Message{
		schema.UserMessage("你好，请简单介绍一下自己。用一句话。"),
	}

	fmt.Println("\n========== DeepSeek Stream Inspect ==========")
	fmt.Printf("Model: %s\n", model)
	fmt.Printf("Thinking: enabled\n\n")

	stream, err := agent.Stream(ctx, messages)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer stream.Close()

	var fullContent strings.Builder
	chunkIdx := 0

	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("\n--- Stream EOF ---")
				break
			}
			t.Fatalf("recv: %v", err)
		}

		fmt.Printf("\n[Chunk %d]\n", chunkIdx)
		fmt.Printf("  Role:         %q\n", msg.Role)
		fmt.Printf("  Content:      %q\n", msg.Content)
		fmt.Printf("  ContentLen:   %d\n", len(msg.Content))
		fmt.Printf("  ToolCalls:    %d\n", len(msg.ToolCalls))
		if len(msg.ToolCalls) > 0 {
			for i, tc := range msg.ToolCalls {
				fmt.Printf("    [%d] Name=%s ID=%s Args=%s\n", i, tc.Function.Name, tc.ID, tc.Function.Arguments)
			}
		}
		if msg.ResponseMeta != nil {
			fmt.Printf("  FinishReason: %q\n", msg.ResponseMeta.FinishReason)
			if msg.ResponseMeta.Usage != nil {
				fmt.Printf("  Usage:        %+v\n", msg.ResponseMeta.Usage)
				if msg.ResponseMeta.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
					fmt.Printf("  ReasoningTok: %d\n", msg.ResponseMeta.Usage.CompletionTokensDetails.ReasoningTokens)
				}
			}
		}

		fullContent.WriteString(msg.Content)
		chunkIdx++
	}

	fmt.Println("\n========== Full Aggregated Content ==========")
	fmt.Println(fullContent.String())
	fmt.Println("==============================================")
}

// ── Qwen 流式 inspect ───────────────────────────────────────────────────

func TestInspectQwenStream(t *testing.T) {
	apiKey := os.Getenv("QWEN_API_KEY")
	model := envOrDefault("QWEN_MODEL", "qwen3-235b-a22b")
	if apiKey == "" {
		t.Skip("QWEN_API_KEY required")
	}

	ctx := context.Background()
	cfg := ModelConfig{
		Protocol: QWEN,
		APIKey:   apiKey,
		Model:    model,
		Thinking: &ThinkingConfig{Enable: true},
	}

	agent, err := NewAgent(ctx, AgentConfig{
		Model:     AgentModelConfig{Config: cfg},
		Execution: ExecutionConfig{Mode: Conversation},
	})
	if err != nil {
		t.Fatalf("new agent: %v", err)
	}
	defer agent.Close()

	messages := []*schema.Message{
		schema.UserMessage("你好，请简单介绍一下自己。用一句话。"),
	}

	fmt.Println("\n========== Qwen Stream Inspect ==========")
	fmt.Printf("Model: %s\n", model)
	fmt.Printf("Thinking: enabled\n\n")

	stream, err := agent.Stream(ctx, messages)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer stream.Close()

	var fullContent strings.Builder
	chunkIdx := 0

	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("\n--- Stream EOF ---")
				break
			}
			t.Fatalf("recv: %v", err)
		}

		fmt.Printf("\n[Chunk %d]\n", chunkIdx)
		fmt.Printf("  Role:         %q\n", msg.Role)
		fmt.Printf("  Content:      %q\n", msg.Content)
		fmt.Printf("  ContentLen:   %d\n", len(msg.Content))
		fmt.Printf("  ToolCalls:    %d\n", len(msg.ToolCalls))
		if len(msg.ToolCalls) > 0 {
			for i, tc := range msg.ToolCalls {
				fmt.Printf("    [%d] Name=%s ID=%s Args=%s\n", i, tc.Function.Name, tc.ID, tc.Function.Arguments)
			}
		}
		if msg.ResponseMeta != nil {
			fmt.Printf("  FinishReason: %q\n", msg.ResponseMeta.FinishReason)
			if msg.ResponseMeta.Usage != nil {
				fmt.Printf("  Usage:        %+v\n", msg.ResponseMeta.Usage)
				if msg.ResponseMeta.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
					fmt.Printf("  ReasoningTok: %d\n", msg.ResponseMeta.Usage.CompletionTokensDetails.ReasoningTokens)
				}
			}
		}

		fullContent.WriteString(msg.Content)
		chunkIdx++
	}

	fmt.Println("\n========== Full Aggregated Content ==========")
	fmt.Println(fullContent.String())
	fmt.Println("==============================================")
}
