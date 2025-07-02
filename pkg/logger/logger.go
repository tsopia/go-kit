package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go-kit/pkg/constants"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
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
	// 删除默认日志文件
	logPath := GetDefaultLogPath()
	if err := os.Remove(logPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// 删除默认日志目录（如果为空）
	if err := os.Remove(DefaultLogDir); err != nil && !os.IsNotExist(err) {
		// 如果目录不为空，这是正常的，不返回错误
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
	// 在不同操作系统上，"目录不为空"的错误消息可能不同
	// 这里简单处理，如果删除失败就认为目录不为空
	return true
}

// EnsureLogDir 确保日志目录存在
func EnsureLogDir() error {
	return os.MkdirAll(DefaultLogDir, 0755)
}

// EnsureLogDirForPath 确保指定路径的日志目录存在
func EnsureLogDirForPath(logPath string) error {
	dir := filepath.Dir(logPath)
	return os.MkdirAll(dir, 0755)
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

// Options 日志选项
type Options struct {
	Level            Level                  // 日志级别
	Format           Format                 // 输出格式 (FormatJSON, FormatConsole, FormatText)
	TimeFormat       string                 // 时间格式
	Caller           bool                   // 是否显示调用者信息
	Stacktrace       bool                   // 是否显示堆栈跟踪
	EnableFileOutput bool                   // 是否启用文件输出
	Sampling         *SamplingConfig        // 采样配置
	Rotate           *RotateConfig          // 日志轮转配置
	Fields           map[string]interface{} // 默认字段
	Hooks            []Hook                 // 钩子函数
}

// SamplingConfig 采样配置
type SamplingConfig struct {
	Initial    int           // 初始采样数量
	Thereafter int           // 后续采样数量
	Tick       time.Duration // 采样周期
}

// Hook 日志钩子函数
type Hook func(entry zapcore.Entry) error

// ContextExtractor 上下文信息提取器
type ContextExtractor interface {
	Extract(ctx context.Context) map[string]interface{}
}

// DefaultContextExtractor 默认上下文提取器
type DefaultContextExtractor struct{}

// Extract 从context中提取信息
func (d *DefaultContextExtractor) Extract(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	// 提取 trace_id（优先使用 constants 包定义的键，保持向后兼容性）
	if traceID := constants.TraceIDFromContext(ctx); traceID != "" {
		fields["trace_id"] = traceID
	} else if traceID := ctx.Value(ContextKey("trace_id")); traceID != nil {
		fields["trace_id"] = traceID
	} else if traceID := ctx.Value("traceId"); traceID != nil {
		fields["trace_id"] = traceID
	} else if traceID := ctx.Value("x-trace-id"); traceID != nil {
		fields["trace_id"] = traceID
	}

	// 提取 request_id（优先使用 constants 包定义的键，保持向后兼容性）
	if requestID := constants.RequestIDFromContext(ctx); requestID != "" {
		fields["request_id"] = requestID
	} else if requestID := ctx.Value(ContextKey("request_id")); requestID != nil {
		fields["request_id"] = requestID
	} else if requestID := ctx.Value("requestId"); requestID != nil {
		fields["request_id"] = requestID
	} else if requestID := ctx.Value("x-request-id"); requestID != nil {
		fields["request_id"] = requestID
	}

	// 提取span信息
	if spanID := ctx.Value(ContextKey("span_id")); spanID != nil {
		fields["span_id"] = spanID
	}
	if spanID := ctx.Value("spanId"); spanID != nil {
		fields["span_id"] = spanID
	}

	// 提取request_id
	if requestID := ctx.Value(ContextKey("request_id")); requestID != nil {
		fields["request_id"] = requestID
	}
	if requestID := ctx.Value("requestId"); requestID != nil {
		fields["request_id"] = requestID
	}
	if requestID := ctx.Value("x-request-id"); requestID != nil {
		fields["request_id"] = requestID
	}

	// 提取user_id
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
	zap          *zap.Logger
	sugar        *zap.SugaredLogger
	level        zap.AtomicLevel
	config       Options
	mu           sync.RWMutex
	hooks        []Hook
	ctx          context.Context  // 当前上下文
	ctxExtractor ContextExtractor // 上下文信息提取器
}

// New 创建新的日志管理器
func New() *Logger {
	return NewWithOptions(Options{
		Level:      InfoLevel,
		Format:     FormatConsole,
		TimeFormat: time.RFC3339,
		Caller:     true,
		Stacktrace: true,
	})
}

// NewWithOptions 根据选项创建日志管理器
func NewWithOptions(opts Options) *Logger {
	logger := &Logger{
		level:        zap.NewAtomicLevelAt(convertLevel(opts.Level)),
		config:       opts,
		hooks:        opts.Hooks,
		ctx:          context.Background(),
		ctxExtractor: &DefaultContextExtractor{},
	}

	// 构建编码器配置
	encoderConfig := logger.buildEncoderConfig()

	// 构建编码器
	var encoder zapcore.Encoder
	switch opts.Format {
	case FormatJSON:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case FormatConsole:
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	case FormatText:
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 构建输出
	writer := logger.buildWriter()

	// 构建核心
	core := zapcore.NewCore(encoder, writer, logger.level)

	// 应用采样
	if opts.Sampling != nil {
		core = zapcore.NewSamplerWithOptions(core, opts.Sampling.Tick, opts.Sampling.Initial, opts.Sampling.Thereafter)
	}

	// 构建zap logger
	zapLogger := zap.New(core)

	// 添加调用者信息
	if opts.Caller {
		zapLogger = zapLogger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(1))
	}

	// 添加堆栈跟踪
	if opts.Stacktrace {
		zapLogger = zapLogger.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel))
	}

	// 添加默认字段
	if opts.Fields != nil {
		fields := make([]zap.Field, 0, len(opts.Fields))
		for key, value := range opts.Fields {
			fields = append(fields, zap.Any(key, value))
		}
		zapLogger = zapLogger.With(fields...)
	}

	logger.zap = zapLogger
	logger.sugar = zapLogger.Sugar()

	return logger
}

// buildEncoderConfig 构建编码器配置
func (l *Logger) buildEncoderConfig() zapcore.EncoderConfig {
	config := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 自定义时间格式
	if l.config.TimeFormat != "" {
		config.EncodeTime = zapcore.TimeEncoderOfLayout(l.config.TimeFormat)
	}

	// 根据格式调整编码器
	switch l.config.Format {
	case FormatConsole:
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncodeCaller = zapcore.ShortCallerEncoder
	case FormatJSON:
		config.EncodeLevel = zapcore.LowercaseLevelEncoder
		config.EncodeCaller = zapcore.ShortCallerEncoder
	}

	return config
}

// buildWriter 构建输出写入器
func (l *Logger) buildWriter() zapcore.WriteSyncer {
	// 始终输出到stdout
	writers := []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}

	// 如果启用文件输出，添加文件写入器
	if l.config.EnableFileOutput {
		if l.config.Rotate != nil {
			writers = append(writers, zapcore.AddSync(l.buildRotateWriter()))
		} else {
			// 如果没有轮转配置，使用默认文件
			logPath := GetDefaultLogPath()
			// 确保日志目录存在
			if err := EnsureLogDirForPath(logPath); err == nil {
				file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				if err == nil {
					writers = append(writers, zapcore.AddSync(file))
				}
			}
		}
	}

	// 如果只有一个写入器，直接返回
	if len(writers) == 1 {
		return writers[0]
	}

	// 多个写入器，使用MultiWriteSyncer
	return zapcore.NewMultiWriteSyncer(writers...)
}

// buildRotateWriter 构建轮转写入器
func (l *Logger) buildRotateWriter() io.Writer {
	return &lumberjack.Logger{
		Filename:   l.config.Rotate.Filename,
		MaxSize:    l.config.Rotate.MaxSize,
		MaxBackups: l.config.Rotate.MaxBackups,
		MaxAge:     l.config.Rotate.MaxAge,
		Compress:   l.config.Rotate.Compress,
		LocalTime:  l.config.Rotate.LocalTime,
	}
}

// convertLevel 转换日志级别
func convertLevel(level Level) zapcore.Level {
	switch level {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level.SetLevel(convertLevel(level))
}

// GetLevel 获取日志级别
func (l *Logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()

	switch l.level.Level() {
	case zapcore.DebugLevel:
		return DebugLevel
	case zapcore.InfoLevel:
		return InfoLevel
	case zapcore.WarnLevel:
		return WarnLevel
	case zapcore.ErrorLevel:
		return ErrorLevel
	case zapcore.FatalLevel:
		return FatalLevel
	default:
		return InfoLevel
	}
}

// Debug 输出调试日志
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.executeHooks(zapcore.DebugLevel, msg)
	l.sugar.Debugw(msg, fields...)
}

// Info 输出信息日志
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.executeHooks(zapcore.InfoLevel, msg)
	l.sugar.Infow(msg, fields...)
}

// Warn 输出警告日志
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.executeHooks(zapcore.WarnLevel, msg)
	l.sugar.Warnw(msg, fields...)
}

// Error 输出错误日志
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.executeHooks(zapcore.ErrorLevel, msg)
	l.sugar.Errorw(msg, fields...)
}

// Fatal 输出致命错误日志并退出
func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.executeHooks(zapcore.FatalLevel, msg)
	l.sugar.Fatalw(msg, fields...)
}

// Panic 输出panic日志并panic
func (l *Logger) Panic(msg string, fields ...interface{}) {
	l.executeHooks(zapcore.PanicLevel, msg)
	l.sugar.Panicw(msg, fields...)
}

// === 格式化日志方法 ===

// Debugf 输出格式化调试日志
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.IsEnabled(DebugLevel) {
		msg := fmt.Sprintf(format, args...)
		l.executeHooks(zapcore.DebugLevel, msg)
		l.sugar.Debug(msg)
	}
}

// Infof 输出格式化信息日志
func (l *Logger) Infof(format string, args ...interface{}) {
	if l.IsEnabled(InfoLevel) {
		msg := fmt.Sprintf(format, args...)
		l.executeHooks(zapcore.InfoLevel, msg)
		l.sugar.Info(msg)
	}
}

// Warnf 输出格式化警告日志
func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.IsEnabled(WarnLevel) {
		msg := fmt.Sprintf(format, args...)
		l.executeHooks(zapcore.WarnLevel, msg)
		l.sugar.Warn(msg)
	}
}

// Errorf 输出格式化错误日志
func (l *Logger) Errorf(format string, args ...interface{}) {
	if l.IsEnabled(ErrorLevel) {
		msg := fmt.Sprintf(format, args...)
		l.executeHooks(zapcore.ErrorLevel, msg)
		l.sugar.Error(msg)
	}
}

// Fatalf 输出格式化致命错误日志并退出
func (l *Logger) Fatalf(format string, args ...interface{}) {
	if l.IsEnabled(FatalLevel) {
		msg := fmt.Sprintf(format, args...)
		l.executeHooks(zapcore.FatalLevel, msg)
		l.sugar.Fatal(msg)
	}
}

// Panicf 输出格式化panic日志并panic
func (l *Logger) Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.executeHooks(zapcore.PanicLevel, msg)
	l.sugar.Panic(msg)
}

// executeHooks 执行钩子函数
func (l *Logger) executeHooks(level zapcore.Level, msg string) {
	if len(l.hooks) == 0 {
		return
	}

	entry := zapcore.Entry{
		Level:   level,
		Time:    time.Now(),
		Message: msg,
	}

	for _, hook := range l.hooks {
		if err := hook(entry); err != nil {
			// 钩子执行失败，记录到标准错误
			fmt.Fprintf(os.Stderr, "日志钩子执行失败: %v\n", err)
		}
	}
}

// With 创建带字段的日志记录器
func (l *Logger) With(fields ...interface{}) *Logger {
	newLogger := &Logger{
		zap:          l.zap.Sugar().With(fields...).Desugar(),
		level:        l.level,
		config:       l.config,
		hooks:        l.hooks,
		ctx:          l.ctx,
		ctxExtractor: l.ctxExtractor,
	}
	newLogger.sugar = newLogger.zap.Sugar()
	return newLogger
}

// WithFields 创建带字段的日志记录器
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}

	newLogger := &Logger{
		zap:          l.zap.With(zapFields...),
		level:        l.level,
		config:       l.config,
		hooks:        l.hooks,
		ctx:          l.ctx,
		ctxExtractor: l.ctxExtractor,
	}
	newLogger.sugar = newLogger.zap.Sugar()
	return newLogger
}

// WithContext 创建带上下文的日志记录器
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		ctx = context.Background()
	}

	// 从上下文中提取字段
	ctxFields := l.ctxExtractor.Extract(ctx)

	// 创建新的logger
	newLogger := &Logger{
		zap:          l.zap,
		level:        l.level,
		config:       l.config,
		hooks:        l.hooks,
		ctx:          ctx,
		ctxExtractor: l.ctxExtractor,
	}

	// 如果有上下文字段，添加到logger中
	if len(ctxFields) > 0 {
		zapFields := make([]zap.Field, 0, len(ctxFields))
		for key, value := range ctxFields {
			zapFields = append(zapFields, zap.Any(key, value))
		}
		newLogger.zap = l.zap.With(zapFields...)
	}

	newLogger.sugar = newLogger.zap.Sugar()
	return newLogger
}

// WithError 创建带错误字段的日志记录器
func (l *Logger) WithError(err error) *Logger {
	return l.With("error", err)
}

// Named 创建命名的日志记录器
func (l *Logger) Named(name string) *Logger {
	newLogger := &Logger{
		zap:          l.zap.Named(name),
		level:        l.level,
		config:       l.config,
		hooks:        l.hooks,
		ctx:          l.ctx,
		ctxExtractor: l.ctxExtractor,
	}
	newLogger.sugar = newLogger.zap.Sugar()
	return newLogger
}

// Sync 同步日志缓冲区
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// GetZap 获取底层zap日志记录器
func (l *Logger) GetZap() *zap.Logger {
	return l.zap
}

// GetSugar 获取底层sugar日志记录器
func (l *Logger) GetSugar() *zap.SugaredLogger {
	return l.sugar
}

// AddHook 添加钩子函数
func (l *Logger) AddHook(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, hook)
}

// RemoveHooks 移除所有钩子函数
func (l *Logger) RemoveHooks() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = nil
}

// IsEnabled 检查日志级别是否启用
func (l *Logger) IsEnabled(level Level) bool {
	return l.level.Enabled(convertLevel(level))
}

// Clone 克隆日志记录器
func (l *Logger) Clone() *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return &Logger{
		zap:          l.zap,
		sugar:        l.sugar,
		level:        l.level,
		config:       l.config,
		hooks:        append([]Hook(nil), l.hooks...),
		ctx:          l.ctx,
		ctxExtractor: l.ctxExtractor,
	}
}

// NewDevelopment 创建开发环境日志记录器
func NewDevelopment() *Logger {
	return NewWithOptions(Options{
		Level:      DebugLevel,
		Format:     FormatConsole,
		TimeFormat: "2006-01-02 15:04:05",
		Caller:     true,
		Stacktrace: true,
	})
}

// NewProduction 创建生产环境日志记录器
func NewProduction() *Logger {
	return NewWithOptions(Options{
		Level:            InfoLevel,
		Format:           FormatJSON,
		TimeFormat:       time.RFC3339,
		Caller:           false,
		Stacktrace:       false,
		EnableFileOutput: true,
		Rotate: &RotateConfig{
			Filename:   GetDefaultLogPath(),
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
			LocalTime:  true,
		},
	})
}

// NewNop 创建空操作日志记录器
func NewNop() *Logger {
	return &Logger{
		zap:          zap.NewNop(),
		sugar:        zap.NewNop().Sugar(),
		level:        zap.NewAtomicLevelAt(zapcore.InfoLevel),
		ctx:          context.Background(),
		ctxExtractor: &DefaultContextExtractor{},
	}
}

// 全局日志实例
var defaultLogger = New()

// Init 初始化全局日志记录器
func Init(opts Options) {
	defaultLogger = NewWithOptions(opts)
}

// InitWithLogger 使用指定的日志记录器初始化全局实例
func InitWithLogger(logger *Logger) {
	if logger != nil {
		defaultLogger = logger
	}
}

// WithContext 创建带上下文的全局日志记录器
func WithContext(ctx context.Context) *Logger {
	return defaultLogger.WithContext(ctx)
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
	defaultLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...interface{}) {
	defaultLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...interface{}) {
	defaultLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...interface{}) {
	defaultLogger.Error(msg, fields...)
}

func Fatal(msg string, fields ...interface{}) {
	defaultLogger.Fatal(msg, fields...)
}

func Panic(msg string, fields ...interface{}) {
	defaultLogger.Panic(msg, fields...)
}

// === 格式化全局函数 ===

func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	defaultLogger.Panicf(format, args...)
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

func IsEnabled(level Level) bool {
	return defaultLogger.IsEnabled(level)
}

func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

func GetDefaultLogger() *Logger {
	return defaultLogger
}

// 便捷的预定义日志记录器
func SetupDevelopment() {
	defaultLogger = NewDevelopment()
}

func SetupProduction() {
	defaultLogger = NewProduction()
}

func SetupWithOptions(opts Options) {
	defaultLogger = NewWithOptions(opts)
}

// FromContext 从 context.Context 创建带有上下文字段的 logger
// 自动提取 trace_id 和 request_id 等字段
func FromContext(ctx context.Context) *Logger {
	return defaultLogger.WithContext(ctx)
}
