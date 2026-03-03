package kit

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// Logger 日志记录器
type Logger struct {
	opts       Options
	logger     *slog.Logger
	level      *slog.LevelVar
	levelValue Level // 内部存储解析后的 Level
	extractor  *contextExtractor
	tracer     *stackTracer
	webhookMgr *webhookManager
	mu         sync.RWMutex
}

// New 创建新的日志记录器
func New(opts Options) *Logger {
	opts.ensureDefaults()

	parsedLevel := parseLevel(opts.Level)
	level := &slog.LevelVar{}
	level.Set(parsedLevel.slogLevel())

	l := &Logger{
		opts:       opts,
		level:      level,
		levelValue: parsedLevel,
		extractor:  newContextExtractor(*opts.ContextKeys),
		tracer:     newStackTracer(opts.StackTrace),
		webhookMgr: newWebhookManager(opts.Webhooks),
	}

	l.buildLogger()
	return l
}

// buildLogger 构建内部 slog logger
func (l *Logger) buildLogger() {
	handlerOpts := &slog.HandlerOptions{
		Level:     l.level,
		AddSource: l.opts.AddCaller,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 自定义时间格式
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(l.opts.TimeFormat))
			}
			return a
		},
	}

	var handler slog.Handler
	switch l.opts.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(l.opts.Output, handlerOpts)
	case FormatText:
		handler = slog.NewTextHandler(l.opts.Output, handlerOpts)
	default:
		handler = slog.NewJSONHandler(l.opts.Output, handlerOpts)
	}

	l.logger = slog.New(&kitHandler{
		Handler:      handler,
		logger:       l,
		addCaller:    l.opts.AddCaller,
		callerSkip:   3, // kitHandler.Handle -> logger.log -> logger.logWithLevel -> Debug/Info/...
	})
}

// log 执行日志记录
func (l *Logger) log(ctx context.Context, level Level, msg string, attrs ...slog.Attr) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 检查级别
	if !l.enabled(level) {
		return
	}

	// 提取上下文信息
	ctxFields := l.extractor.extract(ctx)

	// 构建属性列表
	allAttrs := make([]slog.Attr, 0, len(attrs)+len(ctxFields)+2)

	// 添加上下文字段
	if traceID := ctxFields["trace_id"]; traceID != "" {
		allAttrs = append(allAttrs, slog.String("trace_id", traceID))
	}
	if requestID := ctxFields["request_id"]; requestID != "" {
		allAttrs = append(allAttrs, slog.String("request_id", requestID))
	}
	if userID := ctxFields["user_id"]; userID != "" {
		allAttrs = append(allAttrs, slog.String("user_id", userID))
	}

	// 添加自定义字段
	for k, v := range ctxFields {
		if k != "trace_id" && k != "request_id" && k != "user_id" {
			allAttrs = append(allAttrs, slog.String(k, v))
		}
	}

	// 添加用户传入的 attrs
	allAttrs = append(allAttrs, attrs...)

	// 处理堆栈跟踪
	var stackStr string
	if l.tracer.shouldCaptureStack(level) {
		stack := l.tracer.getStack()
		if len(stack) > 0 {
			stackStr = strings.Join(stack, "\n")
			allAttrs = append(allAttrs, slog.String("stack_trace", stackStr))
		}
	}

	// 获取调用者信息
	caller := l.tracer.getCaller(3)
	if caller != "" {
		allAttrs = append(allAttrs, slog.String("caller", caller))
	}

	// 构建 fields map 用于 webhook
	fields := make(map[string]interface{}, len(attrs))
	for _, attr := range attrs {
		fields[attr.Key] = attr.Value.Any()
	}

	// 记录日志
	l.logger.LogAttrs(ctx, level.slogLevel(), msg, allAttrs...)

	// 触发 webhook
	record := LogRecord{
		Level:      level,
		Message:    msg,
		Time:       time.Now(),
		TraceID:    ctxFields["trace_id"],
		RequestID:  ctxFields["request_id"],
		UserID:     ctxFields["user_id"],
		Caller:     caller,
		StackTrace: stackStr,
		Fields:     fields,
	}
	l.webhookMgr.send(ctx, record)

	// Fatal 和 Panic 特殊处理
	switch level {
	case FatalLevel:
		// 确保日志写入
		l.Sync()
		os.Exit(1)
	case PanicLevel:
		panic(msg)
	}
}

// enabled 检查日志级别是否启用
func (l *Logger) enabled(level Level) bool {
	return level >= l.currentLevel()
}

// currentLevel 获取当前日志级别
func (l *Logger) currentLevel() Level {
	return l.levelValue
}

// parseFields 解析字段为 slog.Attr
func parseFields(fields []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(fields)/2)
	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, slog.Any(key, fields[i+1]))
	}
	return attrs
}

// Debug 输出调试日志
func (l *Logger) Debug(ctx context.Context, msg string, fields ...any) {
	l.log(ctx, DebugLevel, msg, parseFields(fields)...)
}

// Info 输出信息日志
func (l *Logger) Info(ctx context.Context, msg string, fields ...any) {
	l.log(ctx, InfoLevel, msg, parseFields(fields)...)
}

// Warn 输出警告日志
func (l *Logger) Warn(ctx context.Context, msg string, fields ...any) {
	l.log(ctx, WarnLevel, msg, parseFields(fields)...)
}

// Error 输出错误日志
func (l *Logger) Error(ctx context.Context, msg string, fields ...any) {
	l.log(ctx, ErrorLevel, msg, parseFields(fields)...)
}

// Fatal 输出致命错误日志并退出程序
func (l *Logger) Fatal(ctx context.Context, msg string, fields ...any) {
	l.log(ctx, FatalLevel, msg, parseFields(fields)...)
}

// Panic 输出 panic 日志并触发 panic
func (l *Logger) Panic(ctx context.Context, msg string, fields ...any) {
	l.log(ctx, PanicLevel, msg, parseFields(fields)...)
}

// Debugf 输出格式化调试日志
func (l *Logger) Debugf(ctx context.Context, format string, args ...any) {
	if !l.enabled(DebugLevel) {
		return
	}
	l.log(ctx, DebugLevel, fmt.Sprintf(format, args...))
}

// Infof 输出格式化信息日志
func (l *Logger) Infof(ctx context.Context, format string, args ...any) {
	if !l.enabled(InfoLevel) {
		return
	}
	l.log(ctx, InfoLevel, fmt.Sprintf(format, args...))
}

// Warnf 输出格式化警告日志
func (l *Logger) Warnf(ctx context.Context, format string, args ...any) {
	if !l.enabled(WarnLevel) {
		return
	}
	l.log(ctx, WarnLevel, fmt.Sprintf(format, args...))
}

// Errorf 输出格式化错误日志
func (l *Logger) Errorf(ctx context.Context, format string, args ...any) {
	if !l.enabled(ErrorLevel) {
		return
	}
	l.log(ctx, ErrorLevel, fmt.Sprintf(format, args...))
}

// Fatalf 输出格式化致命错误日志并退出程序
func (l *Logger) Fatalf(ctx context.Context, format string, args ...any) {
	if !l.enabled(FatalLevel) {
		return
	}
	l.log(ctx, FatalLevel, fmt.Sprintf(format, args...))
}

// Panicf 输出格式化 panic 日志并触发 panic
func (l *Logger) Panicf(ctx context.Context, format string, args ...any) {
	l.log(ctx, PanicLevel, fmt.Sprintf(format, args...))
}

// SetLevel 设置日志级别
// level 可以是: "debug", "info", "warn", "error", "fatal", "panic"
func (l *Logger) SetLevel(level string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	parsedLevel := parseLevel(level)
	l.levelValue = parsedLevel
	l.level.Set(parsedLevel.slogLevel())
}

// Sync 同步日志缓冲区
func (l *Logger) Sync() error {
	return nil
}

// WithCtx 返回绑定到指定上下文的 Logger
// 用于链式调用场景
func (l *Logger) WithCtx(ctx context.Context) *contextLogger {
	return &contextLogger{
		logger: l,
		ctx:    ctx,
	}
}

// kitHandler 自定义 slog Handler
// 用于在日志记录时添加额外信息
type kitHandler struct {
	slog.Handler
	logger     *Logger
	addCaller  bool
	callerSkip int
}

// contextLogger 绑定到特定上下文的日志记录器
type contextLogger struct {
	logger *Logger
	ctx    context.Context
}

// Debug 使用绑定的上下文输出调试日志
func (c *contextLogger) Debug(msg string, fields ...any) {
	c.logger.Debug(c.ctx, msg, fields...)
}

// Info 使用绑定的上下文输出信息日志
func (c *contextLogger) Info(msg string, fields ...any) {
	c.logger.Info(c.ctx, msg, fields...)
}

// Warn 使用绑定的上下文输出警告日志
func (c *contextLogger) Warn(msg string, fields ...any) {
	c.logger.Warn(c.ctx, msg, fields...)
}

// Error 使用绑定的上下文输出错误日志
func (c *contextLogger) Error(msg string, fields ...any) {
	c.logger.Error(c.ctx, msg, fields...)
}

// Fatal 使用绑定的上下文输出致命错误日志
func (c *contextLogger) Fatal(msg string, fields ...any) {
	c.logger.Fatal(c.ctx, msg, fields...)
}

// Panic 使用绑定的上下文输出 panic 日志
func (c *contextLogger) Panic(msg string, fields ...any) {
	c.logger.Panic(c.ctx, msg, fields...)
}

// Debugf 使用绑定的上下文输出格式化调试日志
func (c *contextLogger) Debugf(format string, args ...any) {
	c.logger.Debugf(c.ctx, format, args...)
}

// Infof 使用绑定的上下文输出格式化信息日志
func (c *contextLogger) Infof(format string, args ...any) {
	c.logger.Infof(c.ctx, format, args...)
}

// Warnf 使用绑定的上下文输出格式化警告日志
func (c *contextLogger) Warnf(format string, args ...any) {
	c.logger.Warnf(c.ctx, format, args...)
}

// Errorf 使用绑定的上下文输出格式化错误日志
func (c *contextLogger) Errorf(format string, args ...any) {
	c.logger.Errorf(c.ctx, format, args...)
}

// Fatalf 使用绑定的上下文输出格式化致命错误日志
func (c *contextLogger) Fatalf(format string, args ...any) {
	c.logger.Fatalf(c.ctx, format, args...)
}

// Panicf 使用绑定的上下文输出格式化 panic 日志
func (c *contextLogger) Panicf(format string, args ...any) {
	c.logger.Panicf(c.ctx, format, args...)
}

// ensure io.Writer interface for compatibility
var _ io.Writer = (*Logger)(nil)

// Write implements io.Writer for compatibility
func (l *Logger) Write(p []byte) (n int, err error) {
	l.Info(context.Background(), string(p))
	return len(p), nil
}
