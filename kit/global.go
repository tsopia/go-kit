package kit

import (
	"context"
)

// std 全局默认日志记录器
var std = New(Options{})

// Init 使用指定选项初始化全局日志记录器
func Init(opts Options) {
	std = New(opts)
}

// SetLevel 设置全局日志级别
// level 可以是: "debug", "info", "warn", "error", "fatal", "panic"
func SetLevel(level string) {
	std.SetLevel(level)
}

// AddWebhook 添加 webhook 到全局日志记录器
func AddWebhook(config *WebhookConfig) {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.opts.Webhooks = append(std.opts.Webhooks, config)
	std.webhookMgr = newWebhookManager(std.opts.Webhooks)
}

// WithCtx 返回绑定到指定上下文的全局日志记录器
func WithCtx(ctx context.Context) *contextLogger {
	return std.WithCtx(ctx)
}

// Debug 输出调试日志到全局记录器
func Debug(ctx context.Context, msg string, fields ...any) {
	std.Debug(ctx, msg, fields...)
}

// Info 输出信息日志到全局记录器
func Info(ctx context.Context, msg string, fields ...any) {
	std.Info(ctx, msg, fields...)
}

// Warn 输出警告日志到全局记录器
func Warn(ctx context.Context, msg string, fields ...any) {
	std.Warn(ctx, msg, fields...)
}

// Error 输出错误日志到全局记录器
func Error(ctx context.Context, msg string, fields ...any) {
	std.Error(ctx, msg, fields...)
}

// Fatal 输出致命错误日志到全局记录器并退出程序
func Fatal(ctx context.Context, msg string, fields ...any) {
	std.Fatal(ctx, msg, fields...)
}

// Panic 输出 panic 日志到全局记录器并触发 panic
func Panic(ctx context.Context, msg string, fields ...any) {
	std.Panic(ctx, msg, fields...)
}

// Debugf 输出格式化调试日志到全局记录器
func Debugf(ctx context.Context, format string, args ...any) {
	std.Debugf(ctx, format, args...)
}

// Infof 输出格式化信息日志到全局记录器
func Infof(ctx context.Context, format string, args ...any) {
	std.Infof(ctx, format, args...)
}

// Warnf 输出格式化警告日志到全局记录器
func Warnf(ctx context.Context, format string, args ...any) {
	std.Warnf(ctx, format, args...)
}

// Errorf 输出格式化错误日志到全局记录器
func Errorf(ctx context.Context, format string, args ...any) {
	std.Errorf(ctx, format, args...)
}

// Fatalf 输出格式化致命错误日志到全局记录器并退出程序
func Fatalf(ctx context.Context, format string, args ...any) {
	std.Fatalf(ctx, format, args...)
}

// Panicf 输出格式化 panic 日志到全局记录器并触发 panic
func Panicf(ctx context.Context, format string, args ...any) {
	std.Panicf(ctx, format, args...)
}
