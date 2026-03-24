package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SSEOption 是 SSE 路由的配置选项。
type SSEOption interface {
	apply(*sseConfig)
}

// sseConfig 是 SSE 内部配置。
type sseConfig struct {
	heartbeatInterval time.Duration
}

type sseOptionFunc func(*sseConfig)

func (f sseOptionFunc) apply(cfg *sseConfig) {
	f(cfg)
}

// WithHeartbeat 启用 SSE 心跳保活。
// interval 是心跳间隔，建议 30s-60s（小于 Nginx 默认 60s 超时）。
// 心跳格式为: `: ping 2026-03-23T10:15:30Z\n\n`
func WithHeartbeat(interval time.Duration) SSEOption {
	return sseOptionFunc(func(cfg *sseConfig) {
		cfg.heartbeatInterval = interval
	})
}

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
	config *sseConfig
	mu     sync.Mutex // 保护 Writer 并发访问
}

func (s *sseSender) Event(name string, data any) error {
	return s.writeEvent(name, data)
}

func (s *sseSender) Data(data any) error {
	return s.writeEvent("", data)
}

func (s *sseSender) Comment(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := fmt.Fprintf(s.ginCtx.Writer, ": %s\n\n", text)
	if err != nil {
		return err
	}
	s.ginCtx.Writer.Flush()
	return nil
}

func (s *sseSender) writeEvent(name string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	if len(lines) == 0 {
		if _, err := fmt.Fprint(s.ginCtx.Writer, "data: \n"); err != nil {
			return err
		}
	}
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

// runHeartbeat 启动心跳 goroutine，返回带取消的 context 和 done channel。
// 如果未配置心跳间隔，直接返回原 context 和已关闭的 done channel。
func (s *sseSender) runHeartbeat(ctx context.Context) (context.Context, <-chan struct{}) {
	if s.config == nil || s.config.heartbeatInterval <= 0 {
		return ctx, closedChan()
	}

	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer cancel()
		ticker := time.NewTicker(s.config.heartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				timestamp := t.Format(time.RFC3339)
				_ = s.Comment(fmt.Sprintf("ping %s", timestamp))
			}
		}
	}()
	return ctx, done
}

// closedChan 返回一个已关闭的 channel
func closedChan() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// logDisconnect 在连接断开时打印日志。
func (s *sseSender) logDisconnect(ctx context.Context) {
	if err := ctx.Err(); err != nil {
		slog.Info("sse client disconnected",
			"path", s.ginCtx.Request.URL.Path,
			"error", err,
		)
	}
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
