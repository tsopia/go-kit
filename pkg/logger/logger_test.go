package logger

import (
	"context"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	logger := New()
	if logger == nil {
		t.Fatal("New() should return a non-nil logger")
	}
}

func TestSetLevel(t *testing.T) {
	logger := New()

	// 测试各种日志级别
	levels := []Level{DebugLevel, InfoLevel, WarnLevel, ErrorLevel, FatalLevel}

	for _, level := range levels {
		// 这里主要测试方法不会panic
		logger.SetLevel(level)
	}
}

func TestNewWithOptions(t *testing.T) {
	opts := Options{
		Level:  InfoLevel,
		Format: FormatJSON,
	}

	logger := NewWithOptions(opts)
	if logger == nil {
		t.Fatal("NewWithOptions() should return a non-nil logger")
	}
}

func TestLogMethods(t *testing.T) {
	logger := New()
	logger.SetLevel(DebugLevel)

	// 测试各种日志方法
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")

	// 测试带字段的日志
	logger.Info("Message with fields",
		"key1", "value1",
		"key2", 42,
	)
}

func TestWith(t *testing.T) {
	logger := New()

	// 创建带字段的logger
	childLogger := logger.With(
		"service", "test",
		"version", 1,
	)

	if childLogger == nil {
		t.Fatal("With() should return a non-nil logger")
	}

	// 测试子logger
	childLogger.Info("Child logger message")
}

func TestSync(t *testing.T) {
	logger := New()
	logger.Sync()
}

func TestGetZap(t *testing.T) {
	logger := New()
	zapLogger := logger.GetZap()

	if zapLogger == nil {
		t.Fatal("GetZap() should return a non-nil zap logger")
	}

	// 测试直接使用zap logger
	zapLogger.Info("Direct zap message")
}

func TestNewDevelopment(t *testing.T) {
	logger := NewDevelopment()
	if logger == nil {
		t.Fatal("NewDevelopment() should return a non-nil logger")
	}

	logger.Info("Development logger message")
}

func TestNewProduction(t *testing.T) {
	logger := NewProduction()
	if logger == nil {
		t.Fatal("NewProduction() should return a non-nil logger")
	}

	logger.Info("Production logger message")
}

func TestNewNop(t *testing.T) {
	logger := NewNop()
	if logger == nil {
		t.Fatal("NewNop() should return a non-nil logger")
	}

	// Nop logger不应该输出任何内容
	logger.Info("This should not be logged")
}

func TestLogLevels(t *testing.T) {
	logger := New()

	// 测试不同级别的日志
	testCases := []struct {
		level Level
		name  string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
	}

	for _, tc := range testCases {
		logger.SetLevel(tc.level)
		logger.Info("Testing level: " + tc.name)
	}
}

func TestMultipleFields(t *testing.T) {
	logger := New()

	logger.Info("Message with multiple fields",
		"string_field", "value",
		"int_field", 42,
		"bool_field", true,
		"float_field", 3.14,
	)
}

func TestErrorLogging(t *testing.T) {
	logger := New()

	// 测试错误日志
	err := &testError{msg: "test error"}
	logger.Error("Error occurred", "error", err)
}

// 测试用的错误类型
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// 测试格式化日志方法
func TestFormattedLogMethods(t *testing.T) {
	logger := New()
	logger.SetLevel(DebugLevel)

	// 测试各种格式化日志方法
	logger.Debugf("Debug message: %s", "test")
	logger.Infof("Info message: %d", 42)
	logger.Warnf("Warning message: %v", true)
	logger.Errorf("Error message: %s", "something went wrong")

	// 测试复杂格式化
	logger.Infof("User %s (ID: %d) logged in from %s at %s",
		"john", 123, "192.168.1.1", "2023-01-01 10:00:00")
}

// 测试格式化全局函数
func TestFormattedGlobalFunctions(t *testing.T) {
	SetLevel(DebugLevel)

	// 测试全局格式化函数
	Debugf("Global debug: %s", "test")
	Infof("Global info: %d", 42)
	Warnf("Global warn: %v", true)
	Errorf("Global error: %s", "test error")
}

// 测试性能：格式化 vs 结构化
func TestFormattingPerformance(t *testing.T) {
	logger := New()
	logger.SetLevel(InfoLevel) // 设置为Info级别，Debug不会输出

	// 测试当日志级别不启用时，格式化方法是否避免了不必要的字符串格式化
	logger.Debugf("This should not be formatted: %s %d %v", "expensive", 12345, map[string]interface{}{"key": "value"})

	// 这个测试主要是确保方法不会panic
	if !logger.IsEnabled(DebugLevel) {
		t.Log("Debug level is disabled, formatting should be skipped")
	}
}

// 测试错误情况
func TestFormattedErrorCases(t *testing.T) {
	logger := New()

	// 测试正常的格式化
	logger.Infof("Normal format: %s %d", "test", 42)

	// 测试nil参数
	logger.Infof("Nil value: %v", nil)

	// 测试空字符串
	logger.Infof("Empty string: %s", "")

	// 测试多种类型
	logger.Infof("Mixed types: %s %d %f %v", "string", 123, 3.14, true)
}

// 测试WithContext功能
func TestWithContext(t *testing.T) {
	logger := New()

	// 创建带trace信息的context
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKey("trace_id"), "12345")
	ctx = context.WithValue(ctx, ContextKey("request_id"), "req-67890")
	ctx = context.WithValue(ctx, ContextKey("user_id"), 123)

	// 创建带context的logger
	ctxLogger := logger.WithContext(ctx)

	// 测试日志输出
	ctxLogger.Info("用户操作")
	ctxLogger.Error("操作失败")

	// 测试链式调用
	ctxLogger.WithFields(map[string]interface{}{
		"action": "login",
		"ip":     "192.168.1.1",
	}).Info("用户登录")
}

// 测试全局WithContext功能
func TestGlobalWithContext(t *testing.T) {
	// 创建带trace信息的context
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKey("trace_id"), "global-12345")
	ctx = context.WithValue(ctx, ContextKey("span_id"), "span-67890")

	// 使用全局WithContext
	// ctxLogger :=
	WithContext(ctx).Info("全局上下文日志")

	// 测试格式化方法
	WithContext(ctx).Infof("用户 %d 执行操作", 123)
}

// 测试初始化功能
func TestInit(t *testing.T) {
	// 保存原始logger
	originalLogger := GetDefaultLogger()
	defer InitWithLogger(originalLogger)

	// 测试Init
	Init(Options{
		Level:  DebugLevel,
		Format: FormatJSON,
	})

	// 测试初始化后的功能
	Info("测试初始化后的Info方法")
	Error("测试初始化后的Error方法")
	Infof("测试初始化后的Infof方法: %s", "test")
}

// 测试包级别方法
func TestPackageLevelMethods(t *testing.T) {
	// 测试小写方法
	Debug("Debug message")
	Info("Info message")
	Warn("Warning message")
	Error("Error message")

	// 测试小写格式化方法
	Debugf("Debug: %s", "test")
	Infof("Info: %d", 42)
	Warnf("Warning: %v", true)
	Errorf("Error: %s", "test error")
}

// 自定义提取器用于测试
type CustomExtractor struct{}

func (c *CustomExtractor) Extract(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	// 提取自定义字段
	if sessionID := ctx.Value(ContextKey("session_id")); sessionID != nil {
		fields["session_id"] = sessionID
	}
	if correlationID := ctx.Value(ContextKey("correlation_id")); correlationID != nil {
		fields["correlation_id"] = correlationID
	}

	return fields
}

// 测试自定义ContextExtractor
func TestCustomContextExtractor(t *testing.T) {
	// 设置自定义提取器
	originalExtractor := GetContextExtractor()
	SetContextExtractor(&CustomExtractor{})
	defer SetContextExtractor(originalExtractor)

	// 创建带自定义字段的context
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKey("session_id"), "sess-123")
	ctx = context.WithValue(ctx, ContextKey("correlation_id"), "corr-456")

	// 测试自定义提取器
	ctxLogger := WithContext(ctx)
	ctxLogger.Info("自定义提取器测试")
}

// 测试context为nil的情况
func TestWithContextNil(t *testing.T) {
	logger := New()

	// 测试nil context
	ctxLogger := logger.WithContext(nil)
	ctxLogger.Info("nil context测试")

	// 应该不会panic
	if ctxLogger == nil {
		t.Error("WithContext(nil) should not return nil")
	}
}

// 测试context字段覆盖
func TestContextFieldOverride(t *testing.T) {
	logger := New()

	// 创建带字段的context
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKey("trace_id"), "ctx-trace-123")
	ctx = context.WithValue(ctx, ContextKey("user_id"), 456)

	// 创建带context的logger
	// ctxLogger :=

	// 添加额外字段（可能会覆盖context字段）
	logger.WithContext(ctx).WithFields(map[string]interface{}{
		"user_id": 789, // 覆盖context中的user_id
		"action":  "test",
	}).Info("字段覆盖测试")
}

// 测试默认日志文件路径配置
func TestDefaultLogFileConfig(t *testing.T) {
	// 保存原始值
	originalLogFile := GetDefaultLogFile()
	originalLogDir := GetDefaultLogDir()

	// 测试结束后恢复原始值
	defer func() {
		SetDefaultLogFile(originalLogFile)
		SetDefaultLogDir(originalLogDir)
	}()

	// 测试设置和获取默认日志文件
	SetDefaultLogFile("test.log")
	if GetDefaultLogFile() != "test.log" {
		t.Errorf("Expected test.log, got %s", GetDefaultLogFile())
	}

	// 测试设置和获取默认日志目录
	SetDefaultLogDir("testlogs")
	if GetDefaultLogDir() != "testlogs" {
		t.Errorf("Expected testlogs, got %s", GetDefaultLogDir())
	}

	// 测试获取完整路径
	expectedPath := "testlogs/test.log"
	if GetDefaultLogPath() != expectedPath {
		t.Errorf("Expected %s, got %s", expectedPath, GetDefaultLogPath())
	}
}

// 测试日志文件清理功能
func TestLogFileCleanup(t *testing.T) {
	// 设置测试用的日志路径
	SetDefaultLogDir("test_logs")
	SetDefaultLogFile("test.log")

	// 测试结束后清理并恢复原始值
	defer func() {
		CleanupLogFiles()
		SetDefaultLogDir("logs")
		SetDefaultLogFile("app.log")
	}()

	// 创建一个带文件输出的logger
	logger := NewWithOptions(Options{
		Level:            InfoLevel,
		Format:           FormatJSON,
		EnableFileOutput: true,
	})

	// 写入一些日志
	logger.Info("测试日志")
	logger.Error("测试错误")
	logger.Sync()

	// 检查文件是否存在
	logPath := GetDefaultLogPath()
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("日志文件应该存在: %s", logPath)
	}

	// 清理日志文件
	if err := CleanupLogFiles(); err != nil {
		t.Errorf("清理日志文件失败: %v", err)
	}

	// 检查文件是否被删除
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("日志文件应该被删除: %s", logPath)
	}
}

// 测试特定日志文件清理
func TestCleanupLogFile(t *testing.T) {
	testFile := "test_specific.log"

	// 创建测试文件
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	file.Close()

	// 确认文件存在
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("测试文件应该存在: %s", testFile)
	}

	// 清理文件
	if err := CleanupLogFile(testFile); err != nil {
		t.Errorf("清理文件失败: %v", err)
	}

	// 确认文件被删除
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("文件应该被删除: %s", testFile)
	}
}

// 测试确保日志目录存在
func TestEnsureLogDir(t *testing.T) {
	// 设置测试目录
	SetDefaultLogDir("test_ensure_dir")

	// 测试结束后清理
	defer func() {
		os.RemoveAll("test_ensure_dir")
		SetDefaultLogDir("logs")
	}()

	// 确保目录存在
	if err := EnsureLogDir(); err != nil {
		t.Errorf("确保目录存在失败: %v", err)
	}

	// 检查目录是否存在
	if _, err := os.Stat("test_ensure_dir"); os.IsNotExist(err) {
		t.Errorf("目录应该存在: test_ensure_dir")
	}
}

// 测试为指定路径确保目录存在
func TestEnsureLogDirForPath(t *testing.T) {
	testPath := "test_path_dir/subdir/test.log"

	// 测试结束后清理
	defer os.RemoveAll("test_path_dir")

	// 确保路径的目录存在
	if err := EnsureLogDirForPath(testPath); err != nil {
		t.Errorf("确保路径目录存在失败: %v", err)
	}

	// 检查目录是否存在
	if _, err := os.Stat("test_path_dir/subdir"); os.IsNotExist(err) {
		t.Errorf("目录应该存在: test_path_dir/subdir")
	}
}

// 测试格式常量
func TestFormatConstants(t *testing.T) {
	// 测试格式常量值
	if FormatJSON.String() != "json" {
		t.Errorf("Expected 'json', got '%s'", FormatJSON.String())
	}

	if FormatConsole.String() != "console" {
		t.Errorf("Expected 'console', got '%s'", FormatConsole.String())
	}

	if FormatText.String() != "text" {
		t.Errorf("Expected 'text', got '%s'", FormatText.String())
	}
}

// 测试格式解析
func TestParseFormat(t *testing.T) {
	testCases := []struct {
		input    string
		expected Format
	}{
		{"json", FormatJSON},
		{"console", FormatConsole},
		{"text", FormatText},
		{"unknown", FormatConsole}, // 默认值
		{"", FormatConsole},        // 空字符串默认值
	}

	for _, tc := range testCases {
		result := ParseFormat(tc.input)
		if result != tc.expected {
			t.Errorf("ParseFormat('%s') = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

// 测试不同格式的日志输出
func TestDifferentFormats(t *testing.T) {
	formats := []Format{FormatJSON, FormatConsole, FormatText}

	for _, format := range formats {
		logger := NewWithOptions(Options{
			Level:  InfoLevel,
			Format: format,
		})

		// 测试不会panic
		logger.Info("测试格式", "format", format.String())
		logger.Sync()
	}
}
