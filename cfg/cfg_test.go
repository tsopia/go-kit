package cfg

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// resetGlobalForTesting 重置全局状态（测试辅助函数）
func resetGlobalForTesting() {
	globalMu.Lock()
	defer globalMu.Unlock()
	global = nil
	globalErr = nil
	globalOnce = sync.Once{}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestNew_ExplicitPath 测试显式指定配置文件路径
func TestNew_ExplicitPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantErr   bool
		errString string
	}{
		{
			name: "valid config file",
			path: "testdata/config.yml",
		},
		{
			name:      "non-existent file",
			path:      "testdata/non-existent.yml",
			wantErr:   true,
			errString: "config file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := New(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errString != "" && !contains(err.Error(), tt.errString) {
					t.Errorf("expected error containing %q, got %q", tt.errString, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if provider == nil {
				t.Error("expected provider, got nil")
			}
		})
	}
}

// TestProvider_GetMethods 测试各种 Get 方法
func TestProvider_GetMethods(t *testing.T) {
	provider, err := New("testdata/config.yml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Test GetString
	t.Run("GetString", func(t *testing.T) {
		val, err := provider.GetString("app.name")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != "test-app" {
			t.Errorf("got %q, want %q", val, "test-app")
		}
	})

	// Test GetInt
	t.Run("GetInt", func(t *testing.T) {
		val, err := provider.GetInt("db.port")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != 5432 {
			t.Errorf("got %d, want %d", val, 5432)
		}
	})

	// Test GetBool
	t.Run("GetBool", func(t *testing.T) {
		val, err := provider.GetBool("app.debug")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !val {
			t.Errorf("got %v, want true", val)
		}
	})

	// Test GetDuration
	t.Run("GetDuration", func(t *testing.T) {
		val, err := provider.GetDuration("server.timeout")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != 30*time.Second {
			t.Errorf("got %v, want %v", val, 30*time.Second)
		}
	})
}

// TestProvider_GetWithDefault 测试带默认值的 Get 方法
func TestProvider_GetWithDefault(t *testing.T) {
	provider, err := New("testdata/config.yml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Test existing key with default
	t.Run("existing key with default", func(t *testing.T) {
		val, err := provider.GetString("app.name", "default-name")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != "test-app" {
			t.Errorf("got %q, want %q", val, "test-app")
		}
	})

	// Test non-existing key with default
	t.Run("non-existing key with default", func(t *testing.T) {
		val, err := provider.GetString("nonexistent.key", "default-value")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != "default-value" {
			t.Errorf("got %q, want %q", val, "default-value")
		}
	})

	// Test non-existing key without default
	t.Run("non-existing key without default", func(t *testing.T) {
		_, err := provider.GetString("nonexistent.key")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

// TestProvider_ErrNotFound 测试键不存在时的错误
func TestProvider_ErrNotFound(t *testing.T) {
	provider, err := New("testdata/config.yml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	_, err = provider.GetString("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestProvider_Exists 测试键存在检查
func TestProvider_Exists(t *testing.T) {
	provider, err := New("testdata/config.yml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	if !provider.Exists("app.name") {
		t.Error("expected app.name to exist")
	}

	if provider.Exists("nonexistent") {
		t.Error("expected nonexistent to not exist")
	}
}

// TestNewFromMap 测试基于 map 的 Provider
func TestNewFromMap(t *testing.T) {
	data := map[string]any{
		"app": map[string]any{
			"name":    "test-app",
			"version": "1.0.0",
			"debug":   true,
		},
		"db": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
	}

	provider := NewFromMap(data)
	if provider == nil {
		t.Fatal("NewFromMap returned nil")
	}

	// Test GetString
	val, err := provider.GetString("app.name")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != "test-app" {
		t.Errorf("got %q, want %q", val, "test-app")
	}

	// Test GetInt
	port, err := provider.GetInt("db.port")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if port != 5432 {
		t.Errorf("got %d, want %d", port, 5432)
	}
}

func TestMapProvider_Unmarshal(t *testing.T) {
	data := map[string]any{
		"app": map[string]any{
			"name":  "test-app",
			"debug": true,
		},
		"db": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
	}
	provider := NewFromMap(data)

	type config struct {
		App struct {
			Name  string `mapstructure:"name"`
			Debug bool   `mapstructure:"debug"`
		} `mapstructure:"app"`
		DB struct {
			Host string `mapstructure:"host"`
			Port int    `mapstructure:"port"`
		} `mapstructure:"db"`
	}

	var cfg config
	if err := provider.Unmarshal(&cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.App.Name != "test-app" {
		t.Errorf("got %q, want %q", cfg.App.Name, "test-app")
	}
	if !cfg.App.Debug {
		t.Errorf("got %v, want %v", cfg.App.Debug, true)
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("got %q, want %q", cfg.DB.Host, "localhost")
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("got %d, want %d", cfg.DB.Port, 5432)
	}
}

func TestMapProvider_UnmarshalNilTarget(t *testing.T) {
	provider := NewFromMap(map[string]any{"app": map[string]any{"name": "test-app"}})
	if err := provider.Unmarshal(nil); err == nil {
		t.Fatal("expected error for nil target")
	}
}

// TestEmptyProvider 测试空 Provider
func TestEmptyProvider(t *testing.T) {
	empty := EmptyProvider()
	if empty == nil {
		t.Fatal("EmptyProvider returned nil")
	}

	// Test without default - should return ErrNotFound
	_, err := empty.GetString("key")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Test with default - should return default
	val, err := empty.GetString("key", "default")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != "default" {
		t.Errorf("got %q, want %q", val, "default")
	}

	// Test Exists
	if empty.Exists("key") {
		t.Error("expected Exists to return false")
	}
}

// TestSub_ChainedCalls 测试链式调用安全
func TestSub_ChainedCalls(t *testing.T) {
	provider, err := New("testdata/config.yml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Chain calls to non-existent keys
	sub := provider.Sub("nonexistent").Sub("nested")
	val, err := sub.GetInt("key")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if val != 0 {
		t.Errorf("expected 0, got %d", val)
	}
}

// TestGlobalFunctions 测试全局便捷函数
func TestGlobalFunctions(t *testing.T) {
	resetGlobalForTesting()

	// Before Init - should return ErrNotFound or default
	t.Run("before init", func(t *testing.T) {
		_, err := GetString("key")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
		if errors.Is(err, ErrNotInitialized) {
			t.Errorf("expected not ErrNotInitialized, got %v", err)
		}

		val, err := GetString("key", "default")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != "default" {
			t.Errorf("got %q, want %q", val, "default")
		}
	})

	// Init
	if err := Init("testdata/config.yml"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// After Init
	t.Run("after init", func(t *testing.T) {
		val, err := GetString("app.name")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != "test-app" {
			t.Errorf("got %q, want %q", val, "test-app")
		}
	})
}

// TestEnvOverride 测试环境变量覆盖配置值
func TestEnvOverride(t *testing.T) {
	t.Setenv("CFG_TEST_DB_HOST", "env-host")
	t.Setenv("CFG_TEST_DB_PORT", "3306")

	provider, err := NewWithPrefix("CFG_TEST", "testdata/config.yml")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	host, err := provider.GetString("db.host")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if host != "env-host" {
		t.Errorf("got %q, want %q", host, "env-host")
	}

	port, err := provider.GetInt("db.port")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if port != 3306 {
		t.Errorf("got %d, want %d", port, 3306)
	}
}

// TestProvider_GetSizeInBytes 测试字节大小获取
func TestProvider_GetSizeInBytes(t *testing.T) {
	data := map[string]any{
		"size1": "1GB",
		"size2": "100MB",
		"size3": 1024,
	}
	provider := NewFromMap(data)

	size1, err := provider.GetSizeInBytes("size1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if size1 != 1024*1024*1024 {
		t.Errorf("got %d, want %d", size1, 1024*1024*1024)
	}

	size2, err := provider.GetSizeInBytes("size2")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if size2 != 100*1024*1024 {
		t.Errorf("got %d, want %d", size2, 100*1024*1024)
	}
}

// TestCheckDefaultCount 测试默认值参数个数检查
func TestCheckDefaultCount(t *testing.T) {
	provider := EmptyProvider()

	// Multiple defaults should return error
	_, err := provider.GetInt("key", 1, 2, 3)
	if err == nil {
		t.Error("expected error for multiple defaults")
	}
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("expected ErrTypeMismatch, got %v", err)
	}
}
