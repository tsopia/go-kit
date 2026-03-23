package httpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

// SSESender 是 SSE 事件发送接口。
type SSESender interface {
	Event(name string, data any) error
	Data(data any) error
	Comment(text string) error
}

// SSEHandlerFunc 是 SSE handler 的函数签名。
type SSEHandlerFunc func(ctx context.Context, send SSESender)

type sseSender struct {
	ginCtx *gin.Context
}

func (s *sseSender) Event(name string, data any) error {
	return s.writeEvent(name, data)
}

func (s *sseSender) Data(data any) error {
	return s.writeEvent("", data)
}

func (s *sseSender) Comment(text string) error {
	_, err := fmt.Fprintf(s.ginCtx.Writer, ": %s\n\n", text)
	if err != nil {
		return err
	}
	s.ginCtx.Writer.Flush()
	return nil
}

func (s *sseSender) writeEvent(name string, data any) error {
	var dataStr string
	switch d := data.(type) {
	case string:
		dataStr = d
	case []byte:
		dataStr = string(d)
	default:
		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal sse data: %w", err)
		}
		dataStr = string(b)
	}

	if name != "" {
		if _, err := fmt.Fprintf(s.ginCtx.Writer, "event: %s\n", name); err != nil {
			return err
		}
	}

	lines := splitLines(dataStr)
	for _, line := range lines {
		if _, err := fmt.Fprintf(s.ginCtx.Writer, "data: %s\n", line); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(s.ginCtx.Writer, "\n"); err != nil {
		return err
	}

	s.ginCtx.Writer.Flush()
	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
