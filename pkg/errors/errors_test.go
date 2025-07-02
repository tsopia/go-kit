package errors

import (
	"errors"
	"testing"
)

// === 核心功能测试 ===

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		message  []string
		expected string
	}{
		{
			name:     "默认消息",
			code:     CodeInvalidParam,
			message:  nil,
			expected: "参数无效",
		},
		{
			name:     "自定义消息",
			code:     CodeInvalidParam,
			message:  []string{"自定义参数错误"},
			expected: "自定义参数错误",
		},
		{
			name:     "空消息",
			code:     CodeUserNotFound,
			message:  []string{""},
			expected: "用户不存在",
		},
		{
			name:     "多个消息参数",
			code:     CodeNotFound,
			message:  []string{"资源不存在", "忽略的消息"},
			expected: "资源不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, tt.message...)
			if err == nil {
				t.Fatal("New() should return a non-nil error")
			}

			if err.GetMessage() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, err.GetMessage())
			}

			if err.Code != tt.code {
				t.Errorf("Expected %v, got %v", tt.code, err.Code)
			}
		})
	}
}

func TestNewWithDetails(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		message  string
		details  string
		expected string
	}{
		{
			name:     "带详细信息",
			code:     CodeInvalidParam,
			message:  "测试错误",
			details:  "详细信息",
			expected: "测试错误",
		},
		{
			name:     "空详细信息",
			code:     CodeNotFound,
			message:  "资源不存在",
			details:  "",
			expected: "资源不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewWithDetails(tt.code, tt.message, tt.details)
			if err == nil {
				t.Fatal("NewWithDetails() should return a non-nil error")
			}

			if err.GetMessage() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, err.GetMessage())
			}

			if err.Code != tt.code {
				t.Errorf("Expected %v, got %v", tt.code, err.Code)
			}

			if err.Details != tt.details {
				t.Errorf("Expected details '%s', got '%s'", tt.details, err.Details)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("原始错误")

	tests := []struct {
		name     string
		code     ErrorCode
		message  []string
		expected string
	}{
		{
			name:     "默认消息",
			code:     CodeNotFound,
			message:  nil,
			expected: "资源不存在",
		},
		{
			name:     "自定义消息",
			code:     CodeNotFound,
			message:  []string{"自定义包装错误"},
			expected: "自定义包装错误",
		},
		{
			name:     "空消息",
			code:     CodeUnauthorized,
			message:  []string{""},
			expected: "未授权",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedErr := Wrap(originalErr, tt.code, tt.message...)
			if wrappedErr == nil {
				t.Fatal("Wrap() should return a non-nil error")
			}

			if wrappedErr.GetMessage() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, wrappedErr.GetMessage())
			}

			if wrappedErr.Code != tt.code {
				t.Errorf("Expected %v, got %v", tt.code, wrappedErr.Code)
			}

			if wrappedErr.Cause != originalErr {
				t.Error("Cause should be the original error")
			}
		})
	}
}

func TestWrapWithDetails(t *testing.T) {
	originalErr := errors.New("原始错误")

	tests := []struct {
		name     string
		code     ErrorCode
		message  string
		details  string
		expected string
	}{
		{
			name:     "带详细信息",
			code:     CodeNotFound,
			message:  "包装错误",
			details:  "详细信息",
			expected: "包装错误",
		},
		{
			name:     "空详细信息",
			code:     CodeDatabaseError,
			message:  "数据库错误",
			details:  "",
			expected: "数据库错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedErr := WrapWithDetails(originalErr, tt.code, tt.message, tt.details)
			if wrappedErr == nil {
				t.Fatal("WrapWithDetails() should return a non-nil error")
			}

			if wrappedErr.GetMessage() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, wrappedErr.GetMessage())
			}

			if wrappedErr.Code != tt.code {
				t.Errorf("Expected %v, got %v", tt.code, wrappedErr.Code)
			}

			if wrappedErr.Details != tt.details {
				t.Errorf("Expected details '%s', got '%s'", tt.details, wrappedErr.Details)
			}

			if wrappedErr.Cause != originalErr {
				t.Error("Cause should be the original error")
			}
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "默认消息",
			err:      New(CodeInvalidParam),
			expected: "[INVALID_PARAM] 参数无效",
		},
		{
			name:     "自定义消息",
			err:      New(CodeInvalidParam, "自定义错误"),
			expected: "[INVALID_PARAM] 自定义错误",
		},
		{
			name:     "带详细信息",
			err:      NewWithDetails(CodeInvalidParam, "测试错误", "详细信息"),
			expected: "[INVALID_PARAM] 测试错误: 详细信息",
		},
		{
			name:     "空详细信息",
			err:      NewWithDetails(CodeNotFound, "资源不存在", ""),
			expected: "[NOT_FOUND] 资源不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorStr := tt.err.Error()
			if errorStr != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, errorStr)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	originalErr := errors.New("原始错误")
	wrappedErr := Wrap(originalErr, CodeNotFound, "包装错误")

	unwrapped := wrappedErr.Unwrap()
	if unwrapped != originalErr {
		t.Error("Unwrap() should return the original error")
	}

	// 测试没有Cause的错误
	simpleErr := New(CodeInvalidParam)
	unwrapped = simpleErr.Unwrap()
	if unwrapped != nil {
		t.Error("Unwrap() should return nil for error without cause")
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{
			name:     "自定义错误",
			err:      New(CodeInvalidParam, "测试错误"),
			expected: CodeInvalidParam,
		},
		{
			name:     "包装的错误",
			err:      Wrap(errors.New("原始错误"), CodeNotFound, "包装错误"),
			expected: CodeNotFound,
		},
		{
			name:     "普通错误",
			err:      errors.New("原始错误"),
			expected: CodeInternalServer,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: ErrorCode{},
		},
		{
			name:     "嵌套包装错误",
			err:      Wrap(Wrap(errors.New("根错误"), CodeDatabaseError), CodeNotFound),
			expected: CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GetCode(tt.err)
			if !code.Equal(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, code)
			}
		})
	}
}

// === 错误检查测试 ===

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     ErrorCode
		expected bool
	}{
		{
			name:     "匹配的错误码",
			err:      New(CodeInvalidParam, "测试错误"),
			code:     CodeInvalidParam,
			expected: true,
		},
		{
			name:     "不匹配的错误码",
			err:      New(CodeInvalidParam, "测试错误"),
			code:     CodeNotFound,
			expected: false,
		},
		{
			name:     "普通错误",
			err:      errors.New("普通错误"),
			code:     CodeInternalServer,
			expected: true, // 普通错误默认返回CodeInternalServer
		},
		{
			name:     "nil错误",
			err:      nil,
			code:     CodeInvalidParam,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Is(tt.err, tt.code)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsInternalServer(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "内部服务器错误",
			err:      New(CodeInternalServer, "内部错误"),
			expected: true,
		},
		{
			name:     "非内部服务器错误",
			err:      New(CodeInvalidParam, "参数错误"),
			expected: false,
		},
		{
			name:     "普通错误",
			err:      errors.New("普通错误"),
			expected: true, // 普通错误默认返回CodeInternalServer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInternalServer(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "未找到错误",
			err:      New(CodeNotFound, "未找到"),
			expected: true,
		},
		{
			name:     "非未找到错误",
			err:      New(CodeInvalidParam, "参数错误"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "未授权错误",
			err:      New(CodeUnauthorized, "未授权"),
			expected: true,
		},
		{
			name:     "非未授权错误",
			err:      New(CodeInvalidParam, "参数错误"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUnauthorized(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsDatabaseError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "数据库错误",
			err:      New(CodeDatabaseError, "数据库错误"),
			expected: true,
		},
		{
			name:     "记录不存在错误",
			err:      New(CodeRecordNotFound, "记录不存在"),
			expected: true,
		},
		{
			name:     "重复键错误",
			err:      New(CodeDuplicateKey, "重复键"),
			expected: true,
		},
		{
			name:     "外键违反错误",
			err:      New(CodeForeignKeyViolation, "外键违反"),
			expected: true,
		},
		{
			name:     "非数据库错误",
			err:      New(CodeInvalidParam, "参数错误"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDatabaseError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsExternalServiceError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "外部服务错误",
			err:      New(CodeExternalServiceError, "外部服务错误"),
			expected: true,
		},
		{
			name:     "网络错误",
			err:      New(CodeNetworkError, "网络错误"),
			expected: true,
		},
		{
			name:     "超时错误",
			err:      New(CodeTimeoutError, "超时错误"),
			expected: true,
		},
		{
			name:     "非外部服务错误",
			err:      New(CodeInvalidParam, "参数错误"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsExternalServiceError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// === 工具函数测试 ===

func TestStringToCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ErrorCode
	}{
		{
			name:     "大写匹配",
			input:    "INVALID_PARAM",
			expected: CodeInvalidParam,
		},
		{
			name:     "小写匹配",
			input:    "invalid_param",
			expected: CodeInvalidParam,
		},
		{
			name:     "混合大小写",
			input:    "Not_Found",
			expected: CodeNotFound,
		},
		{
			name:     "未知错误码",
			input:    "UNKNOWN_ERROR",
			expected: NewErrorCode(9999, "UNKNOWN_ERROR", "未知错误"),
		},
		{
			name:     "空字符串",
			input:    "",
			expected: NewErrorCode(9999, "", "未知错误"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringToCode(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("StringToCode(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStringToCodeWithFound(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ErrorCode
		found    bool
	}{
		{
			name:     "找到匹配",
			input:    "INVALID_PARAM",
			expected: CodeInvalidParam,
			found:    true,
		},
		{
			name:     "小写匹配",
			input:    "invalid_param",
			expected: CodeInvalidParam,
			found:    true,
		},
		{
			name:     "未找到",
			input:    "UNKNOWN_ERROR",
			expected: NewErrorCode(9999, "UNKNOWN_ERROR", "未知错误"),
			found:    false,
		},
		{
			name:     "空字符串",
			input:    "",
			expected: NewErrorCode(9999, "", "未知错误"),
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := StringToCodeWithFound(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("StringToCodeWithFound(%q) = %v, want %v", tt.input, result, tt.expected)
			}
			if found != tt.found {
				t.Errorf("StringToCodeWithFound(%q) found = %v, want %v", tt.input, found, tt.found)
			}
		})
	}
}

func TestNewErrorCode(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		nameStr        string
		defaultMessage []string
		expected       ErrorCode
	}{
		{
			name:           "带默认消息",
			code:           5001,
			nameStr:        "CUSTOM_ERROR",
			defaultMessage: []string{"自定义错误"},
			expected: ErrorCode{
				Code:           5001,
				Name:           "CUSTOM_ERROR",
				DefaultMessage: "自定义错误",
			},
		},
		{
			name:           "无默认消息",
			code:           5002,
			nameStr:        "ANOTHER_ERROR",
			defaultMessage: nil,
			expected: ErrorCode{
				Code: 5002,
				Name: "ANOTHER_ERROR",
			},
		},
		{
			name:           "空默认消息",
			code:           5003,
			nameStr:        "EMPTY_ERROR",
			defaultMessage: []string{""},
			expected: ErrorCode{
				Code: 5003,
				Name: "EMPTY_ERROR",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewErrorCode(tt.code, tt.nameStr, tt.defaultMessage...)
			if result.Code != tt.expected.Code {
				t.Errorf("Expected Code %d, got %d", tt.expected.Code, result.Code)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Expected Name %s, got %s", tt.expected.Name, result.Name)
			}
			if result.DefaultMessage != tt.expected.DefaultMessage {
				t.Errorf("Expected DefaultMessage %s, got %s", tt.expected.DefaultMessage, result.DefaultMessage)
			}
		})
	}
}

// === 便利函数测试 ===

func TestConvenienceFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(...string) *Error
		expected ErrorCode
	}{
		{
			name:     "InvalidParam",
			fn:       InvalidParam,
			expected: CodeInvalidParam,
		},
		{
			name:     "NotFound",
			fn:       NotFound,
			expected: CodeNotFound,
		},
		{
			name:     "Internal",
			fn:       Internal,
			expected: CodeInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试默认消息
			err := tt.fn()
			if err.Code != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, err.Code)
			}
			if err.GetMessage() == "" {
				t.Error("Expected default message to be set")
			}

			// 测试自定义消息
			err2 := tt.fn("自定义消息")
			if err2.Code != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, err2.Code)
			}
			if err2.GetMessage() != "自定义消息" {
				t.Errorf("Expected '自定义消息', got '%s'", err2.GetMessage())
			}
		})
	}
}

// === 链式调用测试 ===

func TestFluentInterface(t *testing.T) {
	t.Run("完整链式调用", func(t *testing.T) {
		err := New(CodeInvalidParam, "test error").
			WithContext("user_id", "123").
			WithContext("request_id", "req-456").
			WithDetails("additional details").
			WithStack()

		if err.Context["user_id"] != "123" {
			t.Errorf("Expected context user_id=123, got %v", err.Context["user_id"])
		}

		if err.Context["request_id"] != "req-456" {
			t.Errorf("Expected context request_id=req-456, got %v", err.Context["request_id"])
		}

		if err.Details != "additional details" {
			t.Errorf("Expected details 'additional details', got %s", err.Details)
		}

		if err.Stack == "" {
			t.Error("Expected stack to be set")
		}
	})

	t.Run("重复调用WithStack", func(t *testing.T) {
		err := New(CodeInvalidParam, "test")
		stack1 := err.WithStack().Stack
		stack2 := err.WithStack().Stack

		if stack1 != stack2 {
			t.Error("Repeated WithStack() calls should return the same stack")
		}
	})

	t.Run("重复调用WithContext", func(t *testing.T) {
		err := New(CodeInvalidParam, "test")
		err.WithContext("key", "value1")
		err.WithContext("key", "value2")

		if err.Context["key"] != "value2" {
			t.Errorf("Expected context key=value2, got %v", err.Context["key"])
		}
	})
}

func TestGetContext(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected map[string]interface{}
	}{
		{
			name: "有上下文的错误",
			err:  New(CodeInvalidParam, "test").WithContext("key", "value"),
			expected: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:     "无上下文的错误",
			err:      New(CodeInvalidParam, "test"),
			expected: nil,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := GetContext(tt.err)
			if tt.expected == nil {
				if context != nil {
					t.Errorf("Expected nil context, got %v", context)
				}
			} else {
				if context["key"] != tt.expected["key"] {
					t.Errorf("Expected context %v, got %v", tt.expected, context)
				}
			}
		})
	}
}

func TestGetStack(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool // true表示期望有堆栈，false表示期望无堆栈
	}{
		{
			name:     "有堆栈的错误",
			err:      New(CodeInvalidParam, "test").WithStack(),
			expected: true,
		},
		{
			name:     "无堆栈的错误",
			err:      New(CodeInvalidParam, "test"),
			expected: false,
		},
		{
			name:     "nil错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := GetStack(tt.err)
			if tt.expected {
				if stack == "" {
					t.Error("Expected non-empty stack")
				}
			} else {
				if stack != "" {
					t.Errorf("Expected empty stack, got %s", stack)
				}
			}
		})
	}
}

// === 格式化函数测试 ===

func TestNewf(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "简单格式化",
			code:     CodeInvalidParam,
			format:   "user %s not found",
			args:     []interface{}{"john"},
			expected: "user john not found",
		},
		{
			name:     "多个参数",
			code:     CodeNotFound,
			format:   "user %s with id %d not found",
			args:     []interface{}{"john", 123},
			expected: "user john with id 123 not found",
		},
		{
			name:     "无参数",
			code:     CodeUnauthorized,
			format:   "unauthorized access",
			args:     nil,
			expected: "unauthorized access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Newf(tt.code, tt.format, tt.args...)
			if err.Message != tt.expected {
				t.Errorf("Expected message '%s', got '%s'", tt.expected, err.Message)
			}
			if err.Code != tt.code {
				t.Errorf("Expected code %v, got %v", tt.code, err.Code)
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	originalErr := errors.New("original")

	tests := []struct {
		name     string
		code     ErrorCode
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "简单格式化",
			code:     CodeInvalidParam,
			format:   "failed to process user %s",
			args:     []interface{}{"john"},
			expected: "failed to process user john",
		},
		{
			name:     "多个参数",
			code:     CodeDatabaseError,
			format:   "failed to save user %s with id %d",
			args:     []interface{}{"john", 123},
			expected: "failed to save user john with id 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Wrapf(originalErr, tt.code, tt.format, tt.args...)
			if err.Message != tt.expected {
				t.Errorf("Expected message '%s', got '%s'", tt.expected, err.Message)
			}
			if err.Code != tt.code {
				t.Errorf("Expected code %v, got %v", tt.code, err.Code)
			}
			if err.Cause != originalErr {
				t.Error("Expected cause to be original error")
			}
		})
	}
}

// === 边界条件测试 ===

func TestEdgeCases(t *testing.T) {
	t.Run("空字符串消息", func(t *testing.T) {
		err := New(CodeInvalidParam, "")
		if err.GetMessage() != "参数无效" {
			t.Errorf("Expected default message for empty string, got '%s'", err.GetMessage())
		}
	})

	t.Run("nil错误包装", func(t *testing.T) {
		wrapped := Wrap(nil, CodeNotFound)
		if wrapped.Cause != nil {
			t.Error("Expected nil cause when wrapping nil error")
		}
	})

	t.Run("空上下文", func(t *testing.T) {
		err := New(CodeInvalidParam, "test")
		err.WithContext("", "value")
		if err.Context[""] != "value" {
			t.Error("Expected empty key to work in context")
		}
	})

	t.Run("nil上下文值", func(t *testing.T) {
		err := New(CodeInvalidParam, "test")
		err.WithContext("key", nil)
		if err.Context["key"] != nil {
			t.Error("Expected nil value to work in context")
		}
	})
}

// === 性能测试 ===

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New(CodeInvalidParam, "benchmark test")
	}
}

func BenchmarkWrap(b *testing.B) {
	originalErr := errors.New("original error")
	for i := 0; i < b.N; i++ {
		Wrap(originalErr, CodeNotFound, "benchmark test")
	}
}

func BenchmarkStringToCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		StringToCode("INVALID_PARAM")
	}
}

func BenchmarkStringToCodeWithFound(b *testing.B) {
	for i := 0; i < b.N; i++ {
		StringToCodeWithFound("INVALID_PARAM")
	}
}

func BenchmarkGetCode(b *testing.B) {
	err := New(CodeInvalidParam, "test")
	for i := 0; i < b.N; i++ {
		GetCode(err)
	}
}

func BenchmarkWithStack(b *testing.B) {
	err := New(CodeInvalidParam, "test")
	for i := 0; i < b.N; i++ {
		err.WithStack()
	}
}

// === 并发安全测试 ===

func TestConcurrency(t *testing.T) {
	t.Run("并发创建错误", func(t *testing.T) {
		const numGoroutines = 100
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				err := New(CodeInvalidParam, "concurrent test")
				if err == nil {
					t.Error("Expected non-nil error")
				}
				done <- true
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	t.Run("并发上下文操作", func(t *testing.T) {
		const numGoroutines = 50
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				// 每个goroutine创建独立的错误实例
				err := New(CodeInvalidParam, "test")
				err.WithContext("key", id)

				// 验证上下文设置成功
				context := GetContext(err)
				if context == nil || context["key"] != id {
					t.Errorf("Expected context key=%d, got %v", id, context)
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}
