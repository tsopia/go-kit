package llm

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func streamWithUsage() *schema.StreamReader[*schema.Message] {
	sr, sw := schema.Pipe[*schema.Message](2)
	go func() {
		defer sw.Close()
		sw.Send(&schema.Message{
			Role:    schema.Assistant,
			Content: "hello world",
			ResponseMeta: &schema.ResponseMeta{
				FinishReason: "stop",
				Usage:        &schema.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
			},
		}, nil)
	}()
	return sr
}

func drain(sr *schema.StreamReader[*schema.Message]) {
	defer sr.Close()
	for {
		if _, err := sr.Recv(); err != nil {
			return
		}
	}
}

func findEntry(entries []recordedLogEntry, msg string) (recordedLogEntry, bool) {
	for _, e := range entries {
		if e.msg == msg {
			return e, true
		}
	}
	return recordedLogEntry{}, false
}

// O-008/O-009: 流式下应补记 model.decision，且包含 usage。
func TestADKAgent_Stream_LogsDecisionWithUsage(t *testing.T) {
	client := &recordingLogClient{}
	tm := &trackingModel{
		streamFn: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
			return streamWithUsage(), nil
		},
	}
	agent, err := NewADKAgent(context.Background(), AgentConfig{
		Model:         AgentModelConfig{Instance: tm},
		Observability: ObservabilityConfig{StructuredLogs: &StructuredLogConfig{Client: client}},
	})
	if err != nil {
		t.Fatalf("NewADKAgent: %v", err)
	}
	defer func() { _ = agent.Close() }()

	sr, err := agent.Stream(context.Background(), []*schema.Message{schema.UserMessage("hi")})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	drain(sr)

	entry, ok := findEntry(client.snapshot(), "model.decision")
	if !ok {
		t.Fatal("expected a model.decision log entry in streaming mode")
	}
	if entry.fields["streaming"] != true {
		t.Errorf("model.decision should mark streaming=true, got %v", entry.fields["streaming"])
	}
	if entry.fields["total_tokens"] == nil {
		t.Errorf("model.decision should include total_tokens, got fields: %v", entry.fields)
	}
}

// observeStreamDecision 在 logs 关闭时应原样返回，不引入额外 goroutine/pipe。
func TestObserveStreamDecision_DisabledPassthrough(t *testing.T) {
	sr := streamWithUsage()
	out := observeStreamDecision(context.Background(), sr, newStructuredLogger(nil), Assistant, schema.ToolChoiceAllowed)
	if out != sr {
		t.Fatal("with logging disabled, observeStreamDecision must return the original stream")
	}
}

// 流内容应被完整转发给调用方（不被聚合吞掉）。
func TestObserveStreamDecision_ForwardsChunks(t *testing.T) {
	client := &recordingLogClient{}
	logs := newStructuredLogger(&StructuredLogConfig{Client: client})
	out := observeStreamDecision(context.Background(), streamWithUsage(), logs, Assistant, schema.ToolChoiceAllowed)

	var got string
	for {
		msg, err := out.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("recv: %v", err)
		}
		got += msg.Content
	}
	out.Close()
	if got != "hello world" {
		t.Fatalf("forwarded content = %q, want 'hello world'", got)
	}
}
