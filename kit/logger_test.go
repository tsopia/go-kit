package kit

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want Level
	}{
		{
			name: "默认配置",
			opts: Options{},
			want: InfoLevel,
		},
		{
			name: "Debug级别",
			opts: Options{Level: "debug"},
			want: DebugLevel,
		},
		{
			name: "Error级别",
			opts: Options{Level: "error"},
			want: ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.opts)
			if logger == nil {
				t.Fatal("New() returned nil")
			}
			if logger.currentLevel() != tt.want {
				t.Errorf("currentLevel() = %v, want %v", logger.currentLevel(), tt.want)
			}
		})
	}
}

func TestLogger_SetLevel(t *testing.T) {
	logger := New(Options{Level: "info"})

	tests := []struct {
		name  string
		level string
		want  Level
	}{
		{"Debug", "debug", DebugLevel},
		{"Info", "info", InfoLevel},
		{"Warn", "warn", WarnLevel},
		{"Error", "error", ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger.SetLevel(tt.level)
			if got := logger.currentLevel(); got != tt.want {
				t.Errorf("SetLevel(%v) then currentLevel() = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Level:  "debug",
		Format: FormatJSON,
		Output: &buf,
	})

	tests := []struct {
		name     string
		logFunc  func()
		contains string
	}{
		{
			name: "Debug日志",
			logFunc: func() {
				logger.Debug(context.Background(), "debug message", "key", "value")
			},
			contains: "debug message",
		},
		{
			name: "Info日志",
			logFunc: func() {
				logger.Info(context.Background(), "info message")
			},
			contains: "info message",
		},
		{
			name: "Warn日志",
			logFunc: func() {
				logger.Warn(context.Background(), "warn message")
			},
			contains: "warn message",
		},
		{
			name: "Error日志",
			logFunc: func() {
				logger.Error(context.Background(), "error message")
			},
			contains: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()
			output := buf.String()
			if !strings.Contains(output, tt.contains) {
				t.Errorf("log output does not contain %q, got: %s", tt.contains, output)
			}
		})
	}
}

func TestLogger_ContextExtraction(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Level:  "info",
		Format: FormatJSON,
		Output: &buf,
		ContextKeys: &ContextKeys{
			Trace:   []string{"trace_id", "x-trace-id"},
			Request: []string{"request_id", "x-request-id"},
			User:    []string{"user_id"},
		},
	})

	tests := []struct {
		name     string
		ctx      context.Context
		contains []string
	}{
		{
			name: "提取 trace_id",
			ctx:  context.WithValue(context.Background(), "trace_id", "trace-123"),
			contains: []string{
				`"trace_id":"trace-123"`,
			},
		},
		{
			name: "提取多个字段",
			ctx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "trace_id", "trace-456")
				ctx = context.WithValue(ctx, "request_id", "req-789")
				ctx = context.WithValue(ctx, "user_id", "user-001")
				return ctx
			}(),
			contains: []string{
				`"trace_id":"trace-456"`,
				`"request_id":"req-789"`,
				`"user_id":"user-001"`,
			},
		},
		{
			name:     "空上下文",
			ctx:      context.Background(),
			contains: []string{},
		},
		{
			name: "备用 key",
			ctx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "x-trace-id", "x-trace-abc")
				return ctx
			}(),
			contains: []string{
				`"trace_id":"x-trace-abc"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger.Info(tt.ctx, "test message")
			output := buf.String()

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("log output does not contain %q, got: %s", want, output)
				}
			}
		})
	}
}

func TestLogger_FormattedLog(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Level:  "debug",
		Format: FormatText,
		Output: &buf,
	})

	tests := []struct {
		name     string
		logFunc  func()
		contains string
	}{
		{
			name:     "Debugf",
			logFunc:  func() { logger.Debugf(context.Background(), "debug %s %d", "test", 42) },
			contains: "debug test 42",
		},
		{
			name:     "Infof",
			logFunc:  func() { logger.Infof(context.Background(), "info %s", "message") },
			contains: "info message",
		},
		{
			name:     "Warnf",
			logFunc:  func() { logger.Warnf(context.Background(), "warn %d", 123) },
			contains: "warn 123",
		},
		{
			name:     "Errorf",
			logFunc:  func() { logger.Errorf(context.Background(), "error %v", "test") },
			contains: "error test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()
			output := buf.String()
			if !strings.Contains(output, tt.contains) {
				t.Errorf("log output does not contain %q, got: %s", tt.contains, output)
			}
		})
	}
}

func TestLogger_StackTrace(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Level:     "info",
		Format:    FormatJSON,
		Output:    &buf,
		AddCaller: true,
		StackTrace: StackTraceConfig{
			Enabled:     true,
			Level:       ErrorLevel,
			Depth:       10,
			SkipRuntime: true,
		},
	})

	// Error 级别应该有堆栈
	buf.Reset()
	logger.Error(context.Background(), "error with stack")
	output := buf.String()
	if !strings.Contains(output, "stack_trace") {
		t.Errorf("Error log should contain stack_trace, got: %s", output)
	}
	if !strings.Contains(output, "caller") {
		t.Errorf("Error log should contain caller, got: %s", output)
	}

	// Info 级别不应该有堆栈
	buf.Reset()
	logger.Info(context.Background(), "info without stack")
	output = buf.String()
	if strings.Contains(output, "stack_trace") {
		t.Errorf("Info log should not contain stack_trace, got: %s", output)
	}
}

func TestLogger_WithCtx(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Level:  "info",
		Format: FormatJSON,
		Output: &buf,
	})

	ctx := context.WithValue(context.Background(), "trace_id", "chained-trace")
	ctxLogger := logger.WithCtx(ctx)

	ctxLogger.Info("message 1")
	output1 := buf.String()
	if !strings.Contains(output1, `"trace_id":"chained-trace"`) {
		t.Errorf("first message should contain trace_id, got: %s", output1)
	}

	buf.Reset()
	ctxLogger.Info("message 2")
	output2 := buf.String()
	if !strings.Contains(output2, `"trace_id":"chained-trace"`) {
		t.Errorf("second message should contain trace_id, got: %s", output2)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     Level
		wantWarn bool
	}{
		{"debug lowercase", "debug", DebugLevel, false},
		{"DEBUG uppercase", "DEBUG", DebugLevel, false},
		{"Debug mixed", "Debug", DebugLevel, false},
		{"info lowercase", "info", InfoLevel, false},
		{"INFO uppercase", "INFO", InfoLevel, false},
		{"warn lowercase", "warn", WarnLevel, false},
		{"warning alias", "warning", WarnLevel, false},
		{"error lowercase", "error", ErrorLevel, false},
		{"fatal lowercase", "fatal", FatalLevel, false},
		{"panic lowercase", "panic", PanicLevel, false},
		{"empty string", "", InfoLevel, false},
		{"invalid value", "invalid", InfoLevel, true},
		{"typo debg", "debg", InfoLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{FatalLevel, "FATAL"},
		{PanicLevel, "PANIC"},
		{Level(100), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormat_String(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatJSON, "json"},
		{FormatText, "text"},
		{Format("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("Format.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobalFunctions(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{
		Level:  "info",
		Format: FormatJSON,
		Output: &buf,
	})

	tests := []struct {
		name     string
		logFunc  func()
		contains string
	}{
		{
			name:     "全局 Info",
			logFunc:  func() { Info(context.Background(), "global info") },
			contains: "global info",
		},
		{
			name:     "全局 Warn",
			logFunc:  func() { Warn(context.Background(), "global warn") },
			contains: "global warn",
		},
		{
			name:     "全局 Error",
			logFunc:  func() { Error(context.Background(), "global error") },
			contains: "global error",
		},
		{
			name:     "全局 Infof",
			logFunc:  func() { Infof(context.Background(), "formatted %s", "info") },
			contains: "formatted info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()
			output := buf.String()
			if !strings.Contains(output, tt.contains) {
				t.Errorf("log output does not contain %q, got: %s", tt.contains, output)
			}
		})
	}
}

func TestSetDefaultContextKeys(t *testing.T) {
	// 保存原始值
	original := getDefaultContextKeys()
	defer SetDefaultContextKeys(original)

	// 设置新的默认值
	newKeys := ContextKeys{
		Trace:   []string{"custom-trace"},
		Request: []string{"custom-request"},
		User:    []string{"custom-user"},
	}
	SetDefaultContextKeys(newKeys)

	// 验证新值
	got := getDefaultContextKeys()
	if len(got.Trace) != 1 || got.Trace[0] != "custom-trace" {
		t.Errorf("Trace keys not set correctly, got: %v", got.Trace)
	}
	if len(got.Request) != 1 || got.Request[0] != "custom-request" {
		t.Errorf("Request keys not set correctly, got: %v", got.Request)
	}
}

func TestParseFields(t *testing.T) {
	tests := []struct {
		name   string
		fields []any
		want   int // 期望的 attr 数量
	}{
		{
			name:   "空字段",
			fields: []any{},
			want:   0,
		},
		{
			name:   "一对kv",
			fields: []any{"key", "value"},
			want:   1,
		},
		{
			name:   "多对kv",
			fields: []any{"k1", "v1", "k2", 42, "k3", true},
			want:   3,
		},
		{
			name:   "奇数个字段（最后一个被忽略）",
			fields: []any{"k1", "v1", "k2"},
			want:   1,
		},
		{
			name:   "非字符串key被跳过",
			fields: []any{"k1", "v1", 123, "v2"},
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFields(tt.fields)
			if len(got) != tt.want {
				t.Errorf("parseFields() returned %d attrs, want %d", len(got), tt.want)
			}
		})
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Level:  "warn",
		Format: FormatText,
		Output: &buf,
	})

	// Debug 和 Info 应该被过滤
	logger.Debug(context.Background(), "debug msg")
	logger.Info(context.Background(), "info msg")
	if buf.Len() > 0 {
		t.Errorf("Debug and Info should be filtered, got: %s", buf.String())
	}

	// Warn 和 Error 应该输出
	buf.Reset()
	logger.Warn(context.Background(), "warn msg")
	if !strings.Contains(buf.String(), "warn msg") {
		t.Errorf("Warn should be logged, got: %s", buf.String())
	}

	buf.Reset()
	logger.Error(context.Background(), "error msg")
	if !strings.Contains(buf.String(), "error msg") {
		t.Errorf("Error should be logged, got: %s", buf.String())
	}
}

func TestWebHookTrigger(t *testing.T) {
	// 测试 webhook 触发（模拟）
	filterCalled := make(chan Level, 10)

	testWebhook := &WebhookConfig{
		Name:   "test",
		URL:    "http://localhost:9999/test",
		Method: "POST",
		Filter: func(_ context.Context, record LogRecord) bool {
			filterCalled <- record.Level
			return record.Level >= ErrorLevel
		},
	}

	var buf bytes.Buffer
	logger := New(Options{
		Level:      "info",
		Format:     FormatJSON,
		Output:     &buf,
		Webhooks:   []*WebhookConfig{testWebhook},
		StackTrace: StackTraceConfig{Enabled: false},
	})

	// 记录不同级别的日志
	logger.Info(context.Background(), "info message")
	logger.Error(context.Background(), "error message")

	// 收集 filter 调用结果
	var infoCalled, errorCalled bool
	timeout := time.After(500 * time.Millisecond)
	done := false
	for !done {
		select {
		case level := <-filterCalled:
			switch level {
			case InfoLevel:
				infoCalled = true
			case ErrorLevel:
				errorCalled = true
			}
			if infoCalled && errorCalled {
				done = true
			}
		case <-timeout:
			done = true
		}
	}

	if !infoCalled {
		t.Error("Info level should call Filter")
	}
	if !errorCalled {
		t.Error("Error level should call Filter")
	}
}

func TestAddWebhook(t *testing.T) {
	// 重置全局 logger
	Init(Options{Level: "error"})

	var triggered bool
	AddWebhook(&WebhookConfig{
		Name: "added-webhook",
		URL:  "http://test",
		Filter: func(_ context.Context, _ LogRecord) bool {
			triggered = true
			return true
		},
	})

	// 触发日志
	Error(context.Background(), "test error")
	time.Sleep(100 * time.Millisecond)

	if !triggered {
		t.Error("Added webhook should be triggered")
	}
}

func TestLogger_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Level:  "info",
		Format: FormatJSON,
		Output: &buf,
	})

	logger.Info(context.Background(), "test message", "key1", "value1", "key2", 42)

	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Output should be valid JSON: %v, got: %s", err, output)
	}

	if result["msg"] != "test message" {
		t.Errorf("JSON msg field = %v, want test message", result["msg"])
	}
	if result["key1"] != "value1" {
		t.Errorf("JSON key1 field = %v, want value1", result["key1"])
	}
	if result["key2"] != float64(42) {
		t.Errorf("JSON key2 field = %v, want 42", result["key2"])
	}
}

func TestCustomContextKeys(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Level:  "info",
		Format: FormatJSON,
		Output: &buf,
		ContextKeys: &ContextKeys{
			Trace:   []string{"uber-trace-id"},
			Request: []string{"x-request-id"},
			Custom: map[string][]string{
				"session_id": {"session_id", "sid"},
				"tenant_id":  {"tenant_id", "x-tenant-id"},
			},
		},
	})

	ctx := context.Background()
	ctx = context.WithValue(ctx, "uber-trace-id", "uber-123")
	ctx = context.WithValue(ctx, "session_id", "sess-456")
	ctx = context.WithValue(ctx, "x-tenant-id", "tenant-789")

	logger.Info(ctx, "custom keys test")
	output := buf.String()

	if !strings.Contains(output, `"trace_id":"uber-123"`) {
		t.Errorf("should extract trace_id from uber-trace-id, got: %s", output)
	}
	if !strings.Contains(output, `"session_id":"sess-456"`) {
		t.Errorf("should extract session_id, got: %s", output)
	}
	if !strings.Contains(output, `"tenant_id":"tenant-789"`) {
		t.Errorf("should extract tenant_id from x-tenant-id, got: %s", output)
	}
}

func TestLogger_Sync(t *testing.T) {
	logger := New(Options{})
	// Sync 应该返回 nil
	if err := logger.Sync(); err != nil {
		t.Errorf("Sync() = %v, want nil", err)
	}
}

func TestLogger_Write(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Level:  "info",
		Format: FormatText,
		Output: &buf,
	})

	// 测试 io.Writer 接口
	n, err := logger.Write([]byte("test message"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != len("test message") {
		t.Errorf("Write() n = %d, want %d", n, len("test message"))
	}

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Write should log message, got: %s", output)
	}
}
