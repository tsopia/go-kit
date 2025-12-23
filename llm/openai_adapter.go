package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tsopia/go-kit/httpclient"
)

const (
	openaiDefaultBaseURL = "https://api.openai.com"
)

type openAIAdapter struct {
	config Config
	client *httpclient.Client
}

func init() {
	RegisterProvider(ProviderOpenAI, newOpenAIAdapter)
}

func newOpenAIAdapter(_ context.Context, cfg Config) (Adapter, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai 需要提供 api_key")
	}

	baseURL := cfg.BaseURL
	if strings.TrimSpace(baseURL) == "" {
		baseURL = openaiDefaultBaseURL
	}

	client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
		BaseURL: baseURL,
		Timeout: cfg.timeoutOrDefault(),
		Headers: cfg.mergeHeaders(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", cfg.APIKey),
		}),
	})

	return &openAIAdapter{
		config: cfg,
		client: client,
	}, nil
}

func (a *openAIAdapter) Chat(ctx context.Context, req ChatRequest) (*ChatCompletion, error) {
	body := a.buildChatPayload(req)

	httpReq := a.client.NewRequest(http.MethodPost, a.chatPath()).
		Context(ctx).
		JSON(body.toMap())

	if len(req.Options.Headers) > 0 {
		httpReq.Headers(req.Options.Headers)
	}

	// 请求级别超时
	if req.Options.Timeout > 0 {
		httpReq.Timeout(req.Options.Timeout)
	}

	resp, err := httpReq.Do()
	if err != nil {
		return nil, fmt.Errorf("请求 openai 失败: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openai 响应异常(%d): %s", resp.StatusCode, resp.String())
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("解析 openai 响应失败: %w", err)
	}

	return parsed.toCompletion(req.Messages), nil
}

func (a *openAIAdapter) chatPath() string {
	isAzure := strings.Contains(strings.ToLower(a.config.BaseURL), "openai.azure.com")
	if !isAzure || a.config.Version == "" {
		return "/v1/chat/completions"
	}

	// azure openai 兼容路径: /openai/deployments/{model}/chat/completions?api-version=xxx
	version := url.QueryEscape(a.config.Version)
	return fmt.Sprintf("/openai/deployments/%s/chat/completions?api-version=%s",
		url.PathEscape(a.config.Model),
		version,
	)
}

func (a *openAIAdapter) buildChatPayload(req ChatRequest) openAIChatRequest {
	payload := openAIChatRequest{
		Model:    a.config.Model,
		Messages: convertMessages(req.Messages),
		Stream:   req.Options.Stream,
		User:     req.Options.User,
		Stop:     req.Options.Stop,
	}

	if req.Options.Temperature != nil {
		payload.Temperature = req.Options.Temperature
	}
	if req.Options.TopP != nil {
		payload.TopP = req.Options.TopP
	}
	if req.Options.MaxTokens != nil {
		payload.MaxTokens = req.Options.MaxTokens
	}

	// provider_raw 透传
	if len(req.Options.ProviderRaw) > 0 {
		for key, value := range req.Options.ProviderRaw {
			if payload.Extra == nil {
				payload.Extra = make(map[string]any)
			}
			payload.Extra[key] = value
		}
	}

	if len(req.Options.Tools) > 0 {
		payload.Tools = convertTools(req.Options.Tools)
	}
	if req.Options.ToolChoice != "" {
		payload.ToolChoice = req.Options.ToolChoice
	}

	return payload
}

type openAIChatRequest struct {
	Model       string          `json:"model,omitempty"`
	Messages    []openAIMessage `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	User        string          `json:"user,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Extra       map[string]any  `json:"-"`
	Tools       []openAITool    `json:"tools,omitempty"`
	ToolChoice  string          `json:"tool_choice,omitempty"`
}

type openAIChatResponse struct {
	ID      string                   `json:"id"`
	Model   string                   `json:"model"`
	Created int64                    `json:"created"`
	Choices []openAIChatResponseItem `json:"choices"`
	Usage   openAIUsage              `json:"usage"`
}

type openAIChatResponseItem struct {
	Index        int            `json:"index"`
	Message      openAIMessage  `json:"message"`
	FinishReason string         `json:"finish_reason"`
	ToolCalls    []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIMessage struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
	Name    string      `json:"name,omitempty"`
	ToolCallID string   `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

func convertMessages(messages []Message) []openAIMessage {
	result := make([]openAIMessage, 0, len(messages))
	for _, msg := range messages {
		result = append(result, openAIMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCalls:  convertToolCalls(msg.ToolCalls),
			ToolCallID: msg.ToolCallID,
		})
	}
	return result
}

func (req openAIChatRequest) toMap() map[string]any {
	payload := map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
	}

	if req.Temperature != nil {
		payload["temperature"] = req.Temperature
	}
	if req.TopP != nil {
		payload["top_p"] = req.TopP
	}
	if req.MaxTokens != nil {
		payload["max_tokens"] = req.MaxTokens
	}
	if req.User != "" {
		payload["user"] = req.User
	}
	if len(req.Stop) > 0 {
		payload["stop"] = req.Stop
	}
	if req.Stream {
		payload["stream"] = req.Stream
	}
	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
	}
	if req.ToolChoice != "" {
		payload["tool_choice"] = req.ToolChoice
	}
	for k, v := range req.Extra {
		payload[k] = v
	}

	return payload
}

func (resp openAIChatResponse) toCompletion(requestMessages []Message) *ChatCompletion {
	choices := make([]CompletionChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		message := Message{
			Role:       choice.Message.Role,
			Content:    choice.Message.Content,
			Name:       choice.Message.Name,
			ToolCalls:  convertOpenAIToolCalls(choice.Message.ToolCalls),
			ToolCallID: choice.Message.ToolCallID,
		}
		choices = append(choices, CompletionChoice{
			Index:        choice.Index,
			Message:      message,
			FinishReason: choice.FinishReason,
			ProviderMeta: nil,
		})
	}

	return &ChatCompletion{
		ID:       resp.ID,
		Model:    resp.Model,
		Provider: ProviderOpenAI,
		Created:  time.Unix(resp.Created, 0),
		Choices:  choices,
		Usage: CompletionUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Raw: map[string]any{
			"request_messages": requestMessages,
			"response":         resp,
		},
	}
}

type openAITool struct {
	Type     string                 `json:"type"`
	Function openAIToolFunctionSpec `json:"function"`
}

type openAIToolFunctionSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string                  `json:"id"`
	Type     string                  `json:"type"`
	Function openAIToolFunctionCall  `json:"function"`
}

type openAIToolFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func convertTools(tools []ToolDefinition) []openAITool {
	result := make([]openAITool, 0, len(tools))
	for _, tool := range tools {
		result = append(result, openAITool{
			Type: "function",
			Function: openAIToolFunctionSpec{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return result
}

func convertToolCalls(calls []ToolCall) []openAIToolCall {
	if len(calls) == 0 {
		return nil
	}
	result := make([]openAIToolCall, 0, len(calls))
	for _, call := range calls {
		result = append(result, openAIToolCall{
			ID:   call.ID,
			Type: call.Type,
			Function: openAIToolFunctionCall{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
		})
	}
	return result
}

func convertOpenAIToolCalls(calls []openAIToolCall) []ToolCall {
	if len(calls) == 0 {
		return nil
	}
	result := make([]ToolCall, 0, len(calls))
	for _, call := range calls {
		result = append(result, ToolCall{
			ID:   call.ID,
			Type: call.Type,
			Function: ToolFunctionCall{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
		})
	}
	return result
}
