package slog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/tsopia/go-kit/constants"
)

// ContextKey 定义context key类型以避免冲突
type ContextKey string

// 默认日志文件路径配置
var (
	// DefaultLogFile 默认日志文件路径
	DefaultLogFile = "app.log"
	// DefaultLogDir 默认日志目录
	DefaultLogDir = "logs"
)

// SetDefaultLogFile 设置默认日志文件路径
func SetDefaultLogFile(filepath string) {
	DefaultLogFile = filepath
}

// GetDefaultLogFile 获取默认日志文件路径
func GetDefaultLogFile() string {
	return DefaultLogFile
}

// SetDefaultLogDir 设置默认日志目录
func SetDefaultLogDir(dir string) {
	DefaultLogDir = dir
}

// GetDefaultLogDir 获取默认日志目录
func GetDefaultLogDir() string {
	return DefaultLogDir
}

// GetDefaultLogPath 获取完整的默认日志路径
func GetDefaultLogPath() string {
	return filepath.Join(DefaultLogDir, DefaultLogFile)
}

// CleanupLogFiles 清理日志文件（主要用于测试）
func CleanupLogFiles() error {
	logPath := GetDefaultLogPath()
	if err := os.Remove(logPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.Remove(DefaultLogDir); err != nil && !os.IsNotExist(err) {
		if !isDirectoryNotEmpty(err) {
			return err
		}
	}
	return nil
}

// CleanupLogFile 清理指定的日志文件
func CleanupLogFile(filepath string) error {
	if err := os.Remove(filepath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// isDirectoryNotEmpty 检查错误是否是因为目录不为空
func isDirectoryNotEmpty(err error) bool {
	return true
}

// EnsureLogDir 确保日志目录存在
func EnsureLogDir() error {
	return os.MkdirAll(DefaultLogDir, 0o755)
}

// EnsureLogDirForPath 确保指定路径的日志目录存在
func EnsureLogDirForPath(logPath string) error {
	dir := filepath.Dir(logPath)
	return os.MkdirAll(dir, 0o755)
}

// Level 日志级别
type Level int8

const (
	DebugLevel Level = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// Format 日志格式类型
type Format string

const (
	// FormatJSON JSON格式输出
	FormatJSON Format = "json"
	// FormatConsole 控制台格式输出（带颜色）
	FormatConsole Format = "console"
	// FormatText 文本格式输出（不带颜色）
	FormatText Format = "text"
)

// String 返回格式字符串
func (f Format) String() string {
	return string(f)
}

// ParseFormat 解析日志格式
func ParseFormat(format string) Format {
	switch format {
	case "json":
		return FormatJSON
	case "console":
		return FormatConsole
	case "text":
		return FormatText
	default:
		return FormatConsole
	}
}

// String 返回日志级别字符串
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	default:
		return "unknown"
	}
}

// ParseLevel 解析日志级别
func ParseLevel(level string) Level {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// RotateConfig 日志轮转配置
type RotateConfig struct {
	Filename   string // 日志文件名
	MaxSize    int    // 最大文件大小(MB)
	MaxBackups int    // 最大备份数量
	MaxAge     int    // 最大保留天数
	Compress   bool   // 是否压缩
	LocalTime  bool   // 是否使用本地时间
}

// SamplingConfig 采样配置
type SamplingConfig struct {
	Initial    int           // 初始采样数量
	Thereafter int           // 后续采样数量
	Tick       time.Duration // 采样周期
}

// Hook 日志钩子函数
type Hook func(entry Entry) error

// Entry slog Hook 数据
type Entry struct {
	Level      slog.Level
	Time       time.Time
	Message    string
	Attributes []slog.Attr
}

// Options 日志选项
type Options struct {
	Level            Level                  // 日志级别
	Format           Format                 // 输出格式 (FormatJSON, FormatConsole, FormatText)
	TimeFormat       string                 // 时间格式
	Caller           bool                   // 是否显示调用者信息
	Stacktrace       bool                   // 是否显示堆栈跟踪
	EnableFileOutput bool                   // 是否启用文件输出
	Color            bool                   // 是否启用彩色输出（仅控制台/文本格式）
	Sampling         *SamplingConfig        // 采样配置
	Rotate           *RotateConfig          // 日志轮转配置
	Fields           map[string]interface{} // 默认字段
	Hooks            []Hook                 // 钩子函数
}

// ContextExtractor 上下文信息提取器
type ContextExtractor interface {
	Extract(ctx context.Context) map[string]interface{}
}

// DefaultContextExtractor 默认上下文提取器
type DefaultContextExtractor struct{}

// Extract 从context中提取信息
func (d *DefaultContextExtractor) Extract(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	if traceID := constants.TraceIDFromContext(ctx); traceID != "" {
		fields["trace_id"] = traceID
	} else if traceID := ctx.Value(ContextKey("trace_id")); traceID != nil {
		fields["trace_id"] = traceID
	} else if traceID := ctx.Value("traceId"); traceID != nil {
		fields["trace_id"] = traceID
	} else if traceID := ctx.Value("x-trace-id"); traceID != nil {
		fields["trace_id"] = traceID
	}

	if requestID := constants.RequestIDFromContext(ctx); requestID != "" {
		fields["request_id"] = requestID
	} else if requestID := ctx.Value(ContextKey("request_id")); requestID != nil {
		fields["request_id"] = requestID
	} else if requestID := ctx.Value("requestId"); requestID != nil {
		fields["request_id"] = requestID
	} else if requestID := ctx.Value("x-request-id"); requestID != nil {
		fields["request_id"] = requestID
	}

	if spanID := ctx.Value(ContextKey("span_id")); spanID != nil {
		fields["span_id"] = spanID
	}
	if spanID := ctx.Value("spanId"); spanID != nil {
		fields["span_id"] = spanID
	}

	if userID := ctx.Value(ContextKey("user_id")); userID != nil {
		fields["user_id"] = userID
	}
	if userID := ctx.Value("userId"); userID != nil {
		fields["user_id"] = userID
	}

	return fields
}

// Logger 日志管理器
type Logger struct {
	logger       *slog.Logger
	levelVar     *slog.LevelVar
	config       Options
	ctx          context.Context
	ctxExtractor ContextExtractor
	hooks        []Hook
}

// New 创建默认 slog Logger
func New() *Logger {
	return NewWithOptions(Options{
		Level:      InfoLevel,
		Format:     FormatText,
		TimeFormat: time.RFC3339,
		Caller:     true,
		Stacktrace: true,
	})
}

// NewWithOptions 根据选项创建 slog Logger
func NewWithOptions(opts Options) *Logger {
	extractor := &DefaultContextExtractor{}
	handler, levelVar := newSlogHandler(opts, extractor, opts.Hooks)
	slogLogger := slog.New(handler)

	return &Logger{
		logger:       slogLogger,
		levelVar:     levelVar,
		config:       opts,
		ctx:          context.Background(),
		ctxExtractor: extractor,
		hooks:        opts.Hooks,
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	if l.levelVar != nil {
		l.levelVar.Set(convertSlogLevel(level))
	}
}

// GetLevel 获取日志级别
func (l *Logger) GetLevel() Level {
	if l.levelVar == nil {
		return InfoLevel
	}
	switch l.levelVar.Level() {
	case slog.LevelDebug:
		return DebugLevel
	case slog.LevelInfo:
		return InfoLevel
	case slog.LevelWarn:
		return WarnLevel
	case slog.LevelError:
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// IsEnabled 检查日志级别是否启用
func (l *Logger) IsEnabled(level Level) bool {
	if l.levelVar == nil {
		return true
	}
	return convertSlogLevel(level) >= l.levelVar.Level()
}

func (l *Logger) log(level slog.Level, msg string, attrs ...slog.Attr) {
	if l.logger == nil {
		return
	}
	l.logger.LogAttrs(l.ctx, level, msg, attrs...)
}

// Debug 输出调试日志
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.log(slog.LevelDebug, msg, kvToAttrs(fields...)...)
}

// Info 输出信息日志
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.log(slog.LevelInfo, msg, kvToAttrs(fields...)...)
}

// Warn 输出警告日志
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.log(slog.LevelWarn, msg, kvToAttrs(fields...)...)
}

// Error 输出错误日志
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.log(slog.LevelError, msg, kvToAttrs(fields...)...)
}

// Fatal 输出致命错误日志并退出
func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.log(slog.LevelError, msg, kvToAttrs(fields...)...)
	os.Exit(1)
}

// Panic 输出panic日志并panic
func (l *Logger) Panic(msg string, fields ...interface{}) {
	l.log(slog.LevelError, msg, kvToAttrs(fields...)...)
	panic(msg)
}

// Debugf 输出格式化调试日志
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.IsEnabled(DebugLevel) {
		l.log(slog.LevelDebug, fmt.Sprintf(format, args...))
	}
}

// Infof 输出格式化信息日志
func (l *Logger) Infof(format string, args ...interface{}) {
	if l.IsEnabled(InfoLevel) {
		l.log(slog.LevelInfo, fmt.Sprintf(format, args...))
	}
}

// Warnf 输出格式化警告日志
func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.IsEnabled(WarnLevel) {
		l.log(slog.LevelWarn, fmt.Sprintf(format, args...))
	}
}

// Errorf 输出格式化错误日志
func (l *Logger) Errorf(format string, args ...interface{}) {
	if l.IsEnabled(ErrorLevel) {
		l.log(slog.LevelError, fmt.Sprintf(format, args...))
	}
}

// Fatalf 输出格式化致命错误日志并退出
func (l *Logger) Fatalf(format string, args ...interface{}) {
	if l.IsEnabled(FatalLevel) {
		l.log(slog.LevelError, fmt.Sprintf(format, args...))
		os.Exit(1)
	}
}

// Panicf 输出格式化panic日志并panic
func (l *Logger) Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.log(slog.LevelError, msg)
	panic(msg)
}

func kvToAttrs(fields ...interface{}) []slog.Attr {
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

func attrsToArgs(attrs []slog.Attr) []any {
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	return args
}

// With 创建带字段的日志记录器
func (l *Logger) With(fields ...interface{}) *Logger {
	return &Logger{
		logger:       l.logger.With(attrsToArgs(kvToAttrs(fields...))...),
		levelVar:     l.levelVar,
		config:       l.config,
		ctx:          l.ctx,
		ctxExtractor: l.ctxExtractor,
		hooks:        l.hooks,
	}
}

// WithFields 创建带字段的日志记录器
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	attrs := make([]slog.Attr, 0, len(fields))
	for key, value := range fields {
		attrs = append(attrs, slog.Any(key, value))
	}
	return &Logger{
		logger:       l.logger.With(attrsToArgs(attrs)...),
		levelVar:     l.levelVar,
		config:       l.config,
		ctx:          l.ctx,
		ctxExtractor: l.ctxExtractor,
		hooks:        l.hooks,
	}
}

// WithContext 创建带上下文的日志记录器
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		ctx = context.Background()
	}
	ctxFields := l.ctxExtractor.Extract(ctx)

	newLogger := &Logger{
		logger:       l.logger,
		levelVar:     l.levelVar,
		config:       l.config,
		ctx:          ctx,
		ctxExtractor: l.ctxExtractor,
		hooks:        l.hooks,
	}

	if len(ctxFields) > 0 {
		attrs := make([]slog.Attr, 0, len(ctxFields))
		for key, value := range ctxFields {
			attrs = append(attrs, slog.Any(key, value))
		}
		newLogger.logger = l.logger.With(attrsToArgs(attrs)...)
	}
	return newLogger
}

// WithError 创建带错误字段的日志记录器
func (l *Logger) WithError(err error) *Logger {
	return l.With("error", err)
}

// Named 创建命名的日志记录器
func (l *Logger) Named(name string) *Logger {
	return l.With("logger", name)
}

// Sync 同步日志缓冲区（对 slog 无操作）
func (l *Logger) Sync() error {
	return nil
}

// GetSlog 获取底层 slog.Logger
func (l *Logger) GetSlog() *slog.Logger {
	return l.logger
}

// AddHook 添加钩子函数
func (l *Logger) AddHook(hook Hook) {
	l.hooks = append(l.hooks, hook)
}

// RemoveHooks 移除所有钩子函数
func (l *Logger) RemoveHooks() {
	l.hooks = nil
}

// Clone 克隆日志记录器
func (l *Logger) Clone() *Logger {
	return &Logger{
		logger:       l.logger,
		levelVar:     l.levelVar,
		config:       l.config,
		ctx:          l.ctx,
		ctxExtractor: l.ctxExtractor,
		hooks:        append([]Hook(nil), l.hooks...),
	}
}

// 全局日志实例
var defaultLogger = New()

// InitSlog 使用指定选项初始化全局 logger 并替换 slog.Default
func InitSlog(opts Options) {
	SetupSlogWithOptions(opts)
}

// SetupSlogWithOptions 设置全局 logger 并替换 slog.Default
func SetupSlogWithOptions(opts Options) {
	defaultLogger = NewWithOptions(opts)
	slog.SetDefault(defaultLogger.GetSlog())
}

// SetDefaultLogger 设置全局 logger
func SetDefaultLogger(logger *Logger) {
	if logger != nil {
		defaultLogger = logger
		slog.SetDefault(logger.GetSlog())
	}
}

// GetDefaultLogger 获取全局 logger
func GetDefaultLogger() *Logger {
	return defaultLogger
}

// SetContextExtractor 设置全局上下文提取器
func SetContextExtractor(extractor ContextExtractor) {
	if extractor != nil {
		defaultLogger.ctxExtractor = extractor
	}
}

// GetContextExtractor 获取全局上下文提取器
func GetContextExtractor() ContextExtractor {
	return defaultLogger.ctxExtractor
}

// 全局函数
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

func GetLevel() Level {
	return defaultLogger.GetLevel()
}

func Debug(msg string, fields ...interface{}) {
	slog.Default().LogAttrs(context.Background(), slog.LevelDebug, msg, kvToAttrs(fields...)...)
}

func Info(msg string, fields ...interface{}) {
	slog.Default().LogAttrs(context.Background(), slog.LevelInfo, msg, kvToAttrs(fields...)...)
}

func Warn(msg string, fields ...interface{}) {
	slog.Default().LogAttrs(context.Background(), slog.LevelWarn, msg, kvToAttrs(fields...)...)
}

func Error(msg string, fields ...interface{}) {
	slog.Default().LogAttrs(context.Background(), slog.LevelError, msg, kvToAttrs(fields...)...)
}

func Fatal(msg string, fields ...interface{}) {
	slog.Default().LogAttrs(context.Background(), slog.LevelError, msg, kvToAttrs(fields...)...)
	os.Exit(1)
}

func Panic(msg string, fields ...interface{}) {
	slog.Default().LogAttrs(context.Background(), slog.LevelError, msg, kvToAttrs(fields...)...)
	panic(msg)
}

// === 格式化全局函数 ===

func Debugf(format string, args ...interface{}) {
	if defaultLogger.IsEnabled(DebugLevel) {
		slog.Default().LogAttrs(context.Background(), slog.LevelDebug, fmt.Sprintf(format, args...))
	}
}

func Infof(format string, args ...interface{}) {
	if defaultLogger.IsEnabled(InfoLevel) {
		slog.Default().LogAttrs(context.Background(), slog.LevelInfo, fmt.Sprintf(format, args...))
	}
}

func Warnf(format string, args ...interface{}) {
	if defaultLogger.IsEnabled(WarnLevel) {
		slog.Default().LogAttrs(context.Background(), slog.LevelWarn, fmt.Sprintf(format, args...))
	}
}

func Errorf(format string, args ...interface{}) {
	if defaultLogger.IsEnabled(ErrorLevel) {
		slog.Default().LogAttrs(context.Background(), slog.LevelError, fmt.Sprintf(format, args...))
	}
}

func Fatalf(format string, args ...interface{}) {
	if defaultLogger.IsEnabled(FatalLevel) {
		slog.Default().LogAttrs(context.Background(), slog.LevelError, fmt.Sprintf(format, args...))
		os.Exit(1)
	}
}

func Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	slog.Default().LogAttrs(context.Background(), slog.LevelError, msg)
	panic(msg)
}

func With(fields ...interface{}) *Logger {
	return defaultLogger.With(fields...)
}

func WithFields(fields map[string]interface{}) *Logger {
	return defaultLogger.WithFields(fields)
}

func WithError(err error) *Logger {
	return defaultLogger.WithError(err)
}

func Named(name string) *Logger {
	return defaultLogger.Named(name)
}

func Sync() error {
	return defaultLogger.Sync()
}

func FromContext(ctx context.Context) *Logger {
	return defaultLogger.WithContext(ctx)
}
