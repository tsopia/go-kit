package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tsopia/go-kit/llm"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置环境变量 OPENAI_API_KEY")
	}

	client, err := llm.NewClient(ctx, llm.Config{
		Provider: llm.ProviderOpenAI,
		Model:    "gpt-4o-mini",
		APIKey:   apiKey,
		Options: llm.RequestOptions{
			Temperature: llm.FloatPtr(0.3),
		},
	})
	if err != nil {
		log.Fatalf("初始化 LLM 客户端失败: %v", err)
	}

	resp, err := client.Chat(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "用一句话介绍 go-kit 工具库"},
	}, llm.WithMaxTokens(64))
	if err != nil {
		log.Fatalf("请求模型失败: %v", err)
	}

	if len(resp.Choices) > 0 {
		fmt.Println("模型回复:", resp.Choices[0].Message.Content)
	} else {
		fmt.Println("未收到模型回复")
	}
}
