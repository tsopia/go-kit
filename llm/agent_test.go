package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAgentOrchestrator_ToolSuccessAndErrorHealing(t *testing.T) {
	var requestBodies []openAIChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		requestBodies = append(requestBodies, body)

		// 第一次：模型要求调用工具
		if len(requestBodies) == 1 {
			resp := openAIChatResponse{
				ID:      "1",
				Model:   "gpt-4o",
				Created: time.Now().Unix(),
				Choices: []openAIChatResponseItem{
					{
						Index: 0,
						Message: openAIMessage{
							Role:    RoleAssistant,
							Content: "",
							ToolCalls: []openAIToolCall{
								{
									ID:   "call-success",
									Type: "function",
									Function: openAIToolFunctionCall{
										Name:      "sum",
										Arguments: `{"a":1,"b":2}`,
									},
								},
								{
									ID:   "call-error",
									Type: "function",
									Function: openAIToolFunctionCall{
										Name:      "fail",
										Arguments: `{}`,
									},
								},
							},
						},
						FinishReason: "tool_calls",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// 第二次：模型接收到工具结果，返回最终答案
		resp := openAIChatResponse{
			ID:      "2",
			Model:   "gpt-4o",
			Created: time.Now().Unix(),
			Choices: []openAIChatResponseItem{
				{
					Index: 0,
					Message: openAIMessage{
						Role:    RoleAssistant,
						Content: "sum=3, fail handled",
					},
					FinishReason: "stop",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient(context.Background(), Config{
		Provider: ProviderOpenAI,
		Model:    "gpt-4o",
		APIKey:   "k",
		BaseURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	orch := &AgentOrchestrator{
		Client: client,
		Tools: map[string]ToolExecutor{
			"sum": func(ctx context.Context, call ToolCall) (string, error) {
				return "3", nil
			},
			"fail": func(ctx context.Context, call ToolCall) (string, error) {
				return "", assertError("boom")
			},
		},
		MaxIterations: 3,
	}

	resp, err := orch.Run(context.Background(), []Message{
		{Role: RoleUser, Content: "calc"},
	}, WithTools([]ToolDefinition{
		{Name: "sum"},
		{Name: "fail"},
	}))
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if resp.Choices[0].Message.Content != "sum=3, fail handled" {
		t.Fatalf("unexpected final content: %s", resp.Choices[0].Message.Content)
	}

	// 校验模型第二次请求中包含工具响应（含错误信息）
	if len(requestBodies) < 2 {
		t.Fatalf("expected at least 2 requests")
	}
	toolReplyMessages := requestBodies[1].Messages
	foundSuccess, foundError := false, false
	for _, m := range toolReplyMessages {
		if m.Role == RoleTool && m.ToolCallID == "call-success" && m.Content == "3" {
			foundSuccess = true
		}
		if m.Role == RoleTool && m.ToolCallID == "call-error" && m.Content != "" {
			foundError = true
		}
	}
	if !foundSuccess || !foundError {
		t.Fatalf("tool feedback missing, success:%v error:%v", foundSuccess, foundError)
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
