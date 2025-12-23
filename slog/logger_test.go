package slog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func withTempLogFile(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "slog.log")
	return logPath, func() {
		_ = os.RemoveAll(dir)
	}
}

func TestSlogContextExtraction(t *testing.T) {
	logPath, cleanup := withTempLogFile(t)
	defer cleanup()

	opts := Options{
		Level:            InfoLevel,
		Format:           FormatJSON,
		EnableFileOutput: true,
		Rotate: &RotateConfig{
			Filename: logPath,
		},
	}

	logger := NewWithOptions(opts)
	ctx := context.Background()
	ctx = context.WithValue(ctx, ContextKey("trace_id"), "trace-ctx")
	ctx = context.WithValue(ctx, ContextKey("request_id"), "req-ctx")

	logger.WithContext(ctx).Info("context message")

	bytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志失败: %v", err)
	}

	content := string(bytes)
	if !strings.Contains(content, "trace-ctx") || !strings.Contains(content, "req-ctx") {
		t.Fatalf("日志应包含上下文字段，内容: %s", content)
	}
}

func TestSlogHookCalled(t *testing.T) {
	logPath, cleanup := withTempLogFile(t)
	defer cleanup()

	called := false
	hook := func(entry Entry) error {
		called = true
		return nil
	}

	opts := Options{
		Level:            InfoLevel,
		Format:           FormatText,
		EnableFileOutput: true,
		Rotate:           &RotateConfig{Filename: logPath},
		Hooks:            []Hook{hook},
	}

	logger := NewWithOptions(opts)
	logger.Info("hook message")

	if !called {
		t.Fatalf("Hook 应该被调用")
	}
}

func TestSlogSampling(t *testing.T) {
	logPath, cleanup := withTempLogFile(t)
	defer cleanup()

	opts := Options{
		Level:            InfoLevel,
		Format:           FormatText,
		EnableFileOutput: true,
		Rotate:           &RotateConfig{Filename: logPath},
		Sampling: &SamplingConfig{
			Initial:    1,
			Thereafter: 3,
			Tick:       time.Second,
		},
	}

	logger := NewWithOptions(opts)
	for i := 0; i < 7; i++ {
		logger.Info("sampling message", "idx", i)
	}

	bytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志失败: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	if len(lines) >= 7 {
		t.Fatalf("采样应减少日志数量，得到 %d 行", len(lines))
	}
}

func TestSlogRotation(t *testing.T) {
	logPath, cleanup := withTempLogFile(t)
	defer cleanup()

	opts := Options{
		Level:            InfoLevel,
		Format:           FormatText,
		EnableFileOutput: true,
		Rotate: &RotateConfig{
			Filename:   logPath,
			MaxSize:    1,
			MaxBackups: 2,
			MaxAge:     1,
		},
	}

	logger := NewWithOptions(opts)
	for i := 0; i < 2000; i++ {
		logger.Info(strings.Repeat("x", 100))
	}

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("日志文件不存在: %v", err)
	}
}

func TestSlogLevelChange(t *testing.T) {
	logPath, cleanup := withTempLogFile(t)
	defer cleanup()

	opts := Options{
		Level:            WarnLevel,
		Format:           FormatJSON,
		EnableFileOutput: true,
		Rotate:           &RotateConfig{Filename: logPath},
	}

	logger := NewWithOptions(opts)
	logger.Info("should be skipped")
	logger.SetLevel(InfoLevel)
	logger.Info("should be logged")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志失败: %v", err)
	}

	if !strings.Contains(string(content), "should be logged") {
		t.Fatalf("级别调整后日志应写入")
	}
}

func TestSlogFormatJSONAndText(t *testing.T) {
	jsonPath, cleanupJSON := withTempLogFile(t)
	defer cleanupJSON()
	textPath, cleanupText := withTempLogFile(t)
	defer cleanupText()

	jsonLogger := NewWithOptions(Options{
		Level:            InfoLevel,
		Format:           FormatJSON,
		EnableFileOutput: true,
		Rotate:           &RotateConfig{Filename: jsonPath},
	})
	jsonLogger.Info("json message")

	textLogger := NewWithOptions(Options{
		Level:            InfoLevel,
		Format:           FormatText,
		EnableFileOutput: true,
		Rotate:           &RotateConfig{Filename: textPath},
	})
	textLogger.Info("text message")

	jsonContent, _ := os.ReadFile(jsonPath)
	if !strings.HasPrefix(strings.TrimSpace(string(jsonContent)), "{") {
		t.Fatalf("JSON 格式应为 JSON 文本")
	}

	textContent, _ := os.ReadFile(textPath)
	if strings.HasPrefix(strings.TrimSpace(string(textContent)), "{") {
		t.Fatalf("Text 格式不应为 JSON")
	}
}

func TestSlogCallerAndStacktrace(t *testing.T) {
	logPath, cleanup := withTempLogFile(t)
	defer cleanup()

	opts := Options{
		Level:            ErrorLevel,
		Format:           FormatText,
		EnableFileOutput: true,
		Rotate:           &RotateConfig{Filename: logPath},
		Caller:           true,
		Stacktrace:       true,
	}

	logger := NewWithOptions(opts)
	logger.Error("error with stack")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志失败: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "stacktrace") {
		t.Fatalf("日志应包含 stacktrace，内容: %s", text)
	}
	if !strings.Contains(text, "logger_test.go") {
		t.Fatalf("日志应包含调用者信息")
	}
}

func TestGlobalFunctionsUseDefaultSlog(t *testing.T) {
	logPath, cleanup := withTempLogFile(t)
	defer cleanup()

	SetupSlogWithOptions(Options{
		Level:            InfoLevel,
		Format:           FormatJSON,
		EnableFileOutput: true,
		Rotate: &RotateConfig{
			Filename: logPath,
		},
	})

	Info("global info", "k", "v")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志失败: %v", err)
	}

	if !strings.Contains(string(content), "global info") {
		t.Fatalf("全局 slog 输出应写入文件")
	}
}
