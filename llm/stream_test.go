package llm

import (
	"context"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type fakeStreamModel struct {
	cfg       ModelConfig
	streamMsg []*schema.Message
}

func (f *fakeStreamModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return f, nil
}
func (f *fakeStreamModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return nil, ErrStreamNotSupported
}
func (f *fakeStreamModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return newSliceStream(f.streamMsg), nil
}
func (f *fakeStreamModel) BindTools(_ []*schema.ToolInfo) error { return nil }
func (f *fakeStreamModel) GetModelConfig() ModelConfig          { return f.cfg }

func TestConcatStreamContent(t *testing.T) {
	msgs := []*schema.Message{
		{Content: "Hello "},
		{Content: "World"},
		{Content: "!"},
	}
	stream := newSliceStream(msgs)
	result, err := ConcatStreamContent(stream)
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello World!" {
		t.Fatalf("unexpected content: %q", result)
	}
}

func TestConcatStreamContentEmpty(t *testing.T) {
	result, err := ConcatStreamContent(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Fatalf("expected empty, got: %q", result)
	}
}

func TestRunToolCallLoopStreamNoToolCall(t *testing.T) {
	m := &fakeStreamModel{
		cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_OPTIONAL}},
		streamMsg: []*schema.Message{
			{Content: "part 1 "},
			{Content: "part 2"},
		},
	}
	msgs := []*schema.Message{{Role: schema.User, Content: "test"}}
	stream, err := RunToolCallLoopStream(context.Background(), m, msgs, nil, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ConcatStreamContent(stream)
	if err != nil {
		t.Fatal(err)
	}
	if result != "part 1 part 2" {
		t.Fatalf("unexpected content: %q", result)
	}
}

func TestRunToolCallLoopStreamToolCallError(t *testing.T) {
	m := &fakeStreamModel{
		cfg: ModelConfig{ToolCallPolicy: ToolCallPolicy{Mode: TOOL_OPTIONAL}},
		streamMsg: []*schema.Message{
			{ToolCalls: []schema.ToolCall{{ID: "tc1", Function: schema.FunctionCall{Name: "foo", Arguments: `{}`}}}},
		},
	}
	msgs := []*schema.Message{{Role: schema.User, Content: "test"}}
	_, err := RunToolCallLoopStream(context.Background(), m, msgs, nil, RunOptions{})
	if err == nil {
		t.Fatal("expected error for tool call in stream")
	}
}

func TestNewSliceStream(t *testing.T) {
	msgs := []*schema.Message{{Content: "a"}, {Content: "b"}}
	s := newSliceStream(msgs)
	defer s.Close()

	m1, err := s.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if m1.Content != "a" {
		t.Fatalf("unexpected: %q", m1.Content)
	}

	m2, err := s.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if m2.Content != "b" {
		t.Fatalf("unexpected: %q", m2.Content)
	}

	_, err = s.Recv()
	if err != io.EOF {
		t.Fatalf("expected EOF, got: %v", err)
	}
}
