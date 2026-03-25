package llm

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type blockingStreamModel struct {
	sent chan struct{}
}

func (m *blockingStreamModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return &schema.Message{Role: schema.Assistant, Content: "ok"}, nil
}

func (m *blockingStreamModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](0)
	go func() {
		defer sw.Close()
		sw.Send(&schema.Message{Role: schema.Assistant, Content: "chunk"}, nil)
		close(m.sent)
	}()
	return sr, nil
}

func (m *blockingStreamModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func TestObservedToolCallingModel_StreamBehavior(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "stream_does_not_consume_source_before_caller_reads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &blockingStreamModel{sent: make(chan struct{})}
			observed := newObservedToolCallingModel(model, newStructuredLogger(&StructuredLogConfig{Client: &recordingLogClient{}}), Assistant, schema.ToolChoiceAllowed)

			sr, err := observed.Stream(context.Background(), []*schema.Message{schema.UserMessage("hello")})
			if err != nil {
				t.Fatalf("Stream failed: %v", err)
			}

			select {
			case <-model.sent:
				t.Fatal("source stream was consumed before caller read from the returned stream")
			case <-time.After(50 * time.Millisecond):
			}

			msg, err := sr.Recv()
			if err != nil {
				t.Fatalf("Recv failed: %v", err)
			}
			if msg.Content != "chunk" {
				t.Fatalf("unexpected chunk content: %q", msg.Content)
			}

			select {
			case <-model.sent:
			case <-time.After(50 * time.Millisecond):
				t.Fatal("source stream was not consumed after caller read from the returned stream")
			}
		})
	}
}
