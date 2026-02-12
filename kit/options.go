// Package kit 提供基于标准库 slog 的日志封装，支持上下文提取、webhook 通知和堆栈跟踪。
package kit

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Level 日志级别
type Level int

const (
	// DebugLevel 调试级别
	DebugLevel Level = iota - 1
	// InfoLevel 信息级别
	InfoLevel
	// WarnLevel 警告级别
	WarnLevel
	// ErrorLevel 错误级别
	ErrorLevel
	// FatalLevel 致命错误级别（会调用 os.Exit(1)）
	FatalLevel
	// PanicLevel Panic 级别（会调用 panic）
	PanicLevel
)

// String 返回日志级别字符串
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	case PanicLevel:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel 从字符串解析日志级别
// 输入字符串会转成大写后匹配，如 "info", "INFO", "Info" 都能正确解析
func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN", "WARNING":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	case "FATAL":
		return FatalLevel
	case "PANIC":
		return PanicLevel
	default:
		return InfoLevel
	}
}

// slogLevel 转换为标准库 slog 的级别
func (l Level) slogLevel() slog.Level {
	switch l {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel, FatalLevel, PanicLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Format 日志格式类型
type Format string

const (
	// FormatJSON JSON 格式输出
	FormatJSON Format = "json"
	// FormatText 文本格式输出
	FormatText Format = "text"
)

// String 返回格式字符串
func (f Format) String() string {
	return string(f)
}

// ContextKeys 定义要从 context 中提取的 keys
type ContextKeys struct {
	// Trace trace ID 可能的 key 列表，按优先级查找
	Trace []string
	// Request request ID 可能的 key 列表，按优先级查找
	Request []string
	// User user ID 可能的 key 列表，按优先级查找
	User []string
	// Custom 自定义字段映射，key 为日志字段名，value 为可能的 context key 列表
	// 例如：{"session_id": ["session_id", "sid", "x-session-id"]}
	Custom map[string][]string
}

// defaultContextKeys 全局默认 context keys
var defaultContextKeys = ContextKeys{
	Trace:   []string{"trace_id", "x-trace-id", "X-Trace-Id", "traceId", "uber-trace-id"},
	Request: []string{"request_id", "x-request-id", "X-Request-Id", "requestId", "x-requestid"},
	User:    []string{"user_id", "userId", "x-user-id"},
}

// SetDefaultContextKeys 设置全局默认 context keys
func SetDefaultContextKeys(keys ContextKeys) {
	defaultContextKeys = keys
}

// getDefaultContextKeys 获取全局默认 context keys（返回副本）
func getDefaultContextKeys() ContextKeys {
	// 复制一份，防止外部修改
	result := ContextKeys{
		Trace:   append([]string(nil), defaultContextKeys.Trace...),
		Request: append([]string(nil), defaultContextKeys.Request...),
		User:    append([]string(nil), defaultContextKeys.User...),
	}
	if defaultContextKeys.Custom != nil {
		result.Custom = make(map[string][]string, len(defaultContextKeys.Custom))
		for k, v := range defaultContextKeys.Custom {
			result.Custom[k] = append([]string(nil), v...)
		}
	}
	return result
}

// StackTraceConfig 堆栈跟踪配置
type StackTraceConfig struct {
	// Enabled 是否启用堆栈跟踪
	Enabled bool
	// Level 触发堆栈的最小级别，默认 Error
	Level Level
	// Depth 堆栈深度，默认 32
	Depth int
	// SkipRuntime 是否跳过 runtime 帧，默认 true
	SkipRuntime bool
}

// defaultStackTraceConfig 默认堆栈跟踪配置
var defaultStackTraceConfig = StackTraceConfig{
	Enabled:     true,
	Level:       ErrorLevel,
	Depth:       32,
	SkipRuntime: true,
}

// Options 日志配置选项
type Options struct {
	// Level 日志级别，默认 "info"
	// 支持: "debug", "info", "warn", "error", "fatal", "panic"
	Level string

	// Format 输出格式，默认 FormatJSON
	Format Format

	// Output 输出目标，默认 os.Stdout
	// 可以通过 io.MultiWriter(os.Stdout, file) 同时输出到多个目标
	Output io.Writer

	// AddCaller 是否添加调用者信息（文件:行号），默认 true
	AddCaller bool

	// StackTrace 堆栈跟踪配置
	StackTrace StackTraceConfig

	// ContextKeys 上下文提取配置，如果不设置使用全局默认值
	ContextKeys *ContextKeys

	// Webhooks webhook 配置列表，错误级别自动触发
	Webhooks []*WebhookConfig

	// TimeFormat 时间格式，默认 time.RFC3339
	TimeFormat string

	// Color 是否启用颜色输出（仅文本格式），默认 false
	Color bool
}

// LogRecord 日志记录信息，用于 webhook
type LogRecord struct {
	Level      Level
	Message    string
	Time       time.Time
	TraceID    string
	RequestID  string
	UserID     string
	Caller     string
	StackTrace string
	Fields     map[string]interface{}
}

// WebhookBuildPayload 自定义 payload 构建函数类型
type WebhookBuildPayload func(ctx context.Context, record LogRecord) map[string]interface{}

// WebhookFilter 日志过滤函数类型
type WebhookFilter func(ctx context.Context, record LogRecord) bool

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	// Name webhook 名称，用于标识
	Name string

	// URL 接收地址（必填）
	URL string

	// Method 请求方法，默认 POST
	Method string

	// Headers 额外请求头
	Headers map[string]string

	// BuildPayload 自定义 payload 构建函数
	// 如果为空，使用默认 payload 格式
	BuildPayload WebhookBuildPayload

	// Filter 过滤函数，只有返回 true 才发送
	// 如果为空，默认只发送 Error 及以上级别
	Filter WebhookFilter

	// Timeout 请求超时，默认 5 秒
	Timeout time.Duration
}

// parseLevel 解析字符串日志级别为 Level 类型
// 有效值: DEBUG/INFO/WARN/WARNING/ERROR/FATAL/PANIC（大小写不敏感）
// 空字符串返回 InfoLevel（无警告）
// 无效值打印警告到 stderr 并返回 InfoLevel
func parseLevel(s string) Level {
	if s == "" {
		return InfoLevel
	}
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN", "WARNING":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	case "FATAL":
		return FatalLevel
	case "PANIC":
		return PanicLevel
	default:
		fmt.Fprintf(os.Stderr, "[kit] invalid log level %q, using default INFO\n", s)
		return InfoLevel
	}
}

// ensureDefaults 确保配置有默认值
func (o *Options) ensureDefaults() {
	if o.Level == "" {
		o.Level = "info"
	}
	if o.Format == "" {
		o.Format = FormatJSON
	}
	if o.Output == nil {
		o.Output = os.Stdout
	}
	if o.TimeFormat == "" {
		o.TimeFormat = time.RFC3339
	}
	if o.ContextKeys == nil {
		keys := getDefaultContextKeys()
		o.ContextKeys = &keys
	}
	if !o.StackTrace.Enabled && o.StackTrace.Level == 0 && o.StackTrace.Depth == 0 {
		o.StackTrace = defaultStackTraceConfig
	}
	if o.StackTrace.Depth == 0 {
		o.StackTrace.Depth = 32
	}
	if o.StackTrace.Level == 0 {
		o.StackTrace.Level = ErrorLevel
	}
}
