package pgmq

import (
	"encoding/json"
	"errors"
	"log"
	"time"
)

// SimpleLogger 基础日志接口
type SimpleLogger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

type stdLogger struct{}

func (l stdLogger) Info(msg string, fields ...interface{})  { log.Printf("INFO %s %v", msg, fields) }
func (l stdLogger) Warn(msg string, fields ...interface{})  { log.Printf("WARN %s %v", msg, fields) }
func (l stdLogger) Error(msg string, fields ...interface{}) { log.Printf("ERROR %s %v", msg, fields) }

// Metrics 可插拔指标接口
type Metrics interface {
	IncProcessCount(queue string, status string)
	ObserveLatency(queue string, duration time.Duration)
}

type noopMetrics struct{}

func (m noopMetrics) IncProcessCount(queue string, status string)         {}
func (m noopMetrics) ObserveLatency(queue string, duration time.Duration) {}

var (
	ErrMissingDB        = errors.New("db 不能为空")
	ErrMissingClient    = errors.New("client 未初始化")
	ErrNoRows           = errors.New("pgmq: no rows in result set")
	ErrMissingQueue     = errors.New("queue 名称不能为空")
	ErrInvalidQueue     = errors.New("queue 名称不合法")
	ErrInvalidDelay     = errors.New("延迟时间不能为负")
	ErrInvalidQuantity  = errors.New("读取数量必须大于0")
	ErrInvalidConfig    = errors.New("配置无效")
	ErrExtensionMissing = errors.New("pgmq 扩展不存在")
	ErrDecodeMessage    = errors.New("消息解码失败")
)

// Message PGMQ 消息
type Message[T any] struct {
	MsgID      int64
	ReadCount  int64
	EnqueuedAt time.Time
	VT         time.Time
	Raw        json.RawMessage
	Headers    json.RawMessage
	Body       T
}
