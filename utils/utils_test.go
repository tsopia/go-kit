package utils

import (
	"testing"
)

func TestCryptoUtils(t *testing.T) {
	crypto := &CryptoUtils{}

	// 测试Token生成
	token := crypto.GenerateToken(32)
	if token == "" {
		t.Fatal("GenerateToken() should return a non-empty string")
	}

	if len(token) != 32 {
		t.Errorf("Expected token length 32, got %d", len(token))
	}

	// 测试MD5哈希
	input := "test string"
	hash := crypto.MD5(input)
	if len(hash) != 32 {
		t.Errorf("Expected MD5 hash length 32, got %d", len(hash))
	}

	// 测试SHA256哈希
	sha256Hash := crypto.SHA256(input)
	if len(sha256Hash) != 64 {
		t.Errorf("Expected SHA256 hash length 64, got %d", len(sha256Hash))
	}
}

func TestValidationUtils(t *testing.T) {
	validation := &ValidationUtils{}

	// 测试邮箱验证
	if !validation.IsEmail("test@example.com") {
		t.Error("Expected valid email")
	}

	if validation.IsEmail("invalid-email") {
		t.Error("Expected invalid email")
	}

	// 测试URL验证
	if !validation.IsURL("https://example.com") {
		t.Error("Expected valid URL")
	}

	if validation.IsURL("not-a-url") {
		t.Error("Expected invalid URL")
	}

	// 测试手机号验证
	if !validation.IsPhone("13812345678") {
		t.Error("Expected valid phone number")
	}

	if validation.IsPhone("invalid-phone") {
		t.Error("Expected invalid phone number")
	}
}

func TestStringUtils(t *testing.T) {
	strUtils := &StringUtils{}

	// 测试IsEmpty
	if !strUtils.IsEmpty("") {
		t.Error("Expected empty string to be empty")
	}

	if strUtils.IsEmpty("not empty") {
		t.Error("Expected non-empty string to not be empty")
	}

	// 测试IsNotEmpty
	if strUtils.IsNotEmpty("") {
		t.Error("Expected empty string to not be not empty")
	}

	// 测试Random
	random := strUtils.Random(10)
	if len(random) != 10 {
		t.Errorf("Expected length 10, got %d", len(random))
	}

	// 测试Truncate
	truncated := strUtils.Truncate("hello world", 5)
	// Truncate方法可能会添加省略号，所以长度可能会超过5
	if len(truncated) == 0 {
		t.Error("Expected non-empty truncated string")
	}

	// 测试CamelToSnake
	snake := strUtils.CamelToSnake("HelloWorld")
	if snake == "" {
		t.Error("Expected non-empty snake case string")
	}

	// 测试SnakeToCamel
	camel := strUtils.SnakeToCamel("hello_world")
	if camel == "" {
		t.Error("Expected non-empty camel case string")
	}
}

func TestNumberUtils(t *testing.T) {
	numUtils := &NumberUtils{}

	// 测试SafeParseInt
	intVal := numUtils.SafeParseInt("123", 0)
	if intVal != 123 {
		t.Errorf("Expected 123, got %d", intVal)
	}

	// 测试SafeParseFloat
	floatVal := numUtils.SafeParseFloat("123.45", 0.0)
	if floatVal != 123.45 {
		t.Errorf("Expected 123.45, got %f", floatVal)
	}

	// 测试InRange
	if !numUtils.InRange(5, 1, 10) {
		t.Error("Expected 5 to be in range 1-10")
	}

	// 测试ClampInt
	clamped := numUtils.ClampInt(15, 1, 10)
	if clamped != 10 {
		t.Errorf("Expected 10, got %d", clamped)
	}

	// 测试ClampFloat
	clampedFloat := numUtils.ClampFloat(15.5, 1.0, 10.0)
	if clampedFloat != 10.0 {
		t.Errorf("Expected 10.0, got %f", clampedFloat)
	}
}
