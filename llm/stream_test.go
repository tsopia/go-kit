package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/tsopia/go-kit/llm/model"
	"github.com/tsopia/go-kit/llm/tool"
)

type fakeStreamModel struct {
	fakeModel
	streams [][]model.ChatMessage
	streamI int
}

func (f *fakeStreamModel) WithTools(_ ...tool.InvokableTool) (model.ToolCallingChatModel, error) {
	return f, nil
}

func (f *fakeStreamModel) GenerateStream(_ context.Context, _ []model.ChatMessage) (model.ChatMessageStream, error) {
	if f.streamI >= len(f.streams) {
		return nil, io.EOF
	}
	stream := model.NewSliceMessageStream(f.streams[f.streamI])
	f.streamI++
	return stream, nil
}

func TestConcatStreamContent(t *testing.T) {
	stream := model.NewSliceMessageStream([]model.ChatMessage{{Content: "hello"}, {Content: " "}, {Content: "world"}})
	got, err := ConcatStreamContent(stream)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestRunToolCallLoopStream(t *testing.T) {
	m := &fakeStreamModel{
		fakeModel: fakeModel{cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_OPTIONAL}}},
		streams:   [][]model.ChatMessage{{{Content: "你好"}, {Content: "，世界"}}},
	}

	stream, err := RunToolCallLoopStream(context.Background(), m, nil, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ConcatStreamContent(stream)
	if err != nil {
		t.Fatal(err)
	}
	if got != "你好，世界" {
		t.Fatalf("unexpected stream result: %q", got)
	}
}

func TestRunToolCallLoopStreamWithToolCallReturnsError(t *testing.T) {
	m := &fakeStreamModel{
		fakeModel: fakeModel{cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_OPTIONAL}}},
		streams:   [][]model.ChatMessage{{{ToolCalls: []model.ToolCall{{Name: "sum", Arguments: json.RawMessage(`{"a":1}`)}}}}},
	}

	_, err := RunToolCallLoopStream(context.Background(), m, nil, RunOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunToolCallLoopStreamNeedsStreamingModel(t *testing.T) {
	m := &fakeModel{cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_OPTIONAL}}}
	_, err := RunToolCallLoopStream(context.Background(), m, []tool.InvokableTool{}, RunOptions{})
	if !errors.Is(err, ErrStreamNotSupported) {
		t.Fatalf("expected ErrStreamNotSupported, got %v", err)
	}
}
