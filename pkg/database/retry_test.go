package database

import (
	"testing"
	"time"
)

func TestRetryConfig_Defaults(t *testing.T) {
	config := &Config{
		Driver:   "sqlite",
		Database: ":memory:",
	}

	config.SetDefaults()

	// 验证默认值
	if config.RetryMaxAttempts != DefaultRetryMaxAttempts {
		t.Errorf("期望 RetryMaxAttempts = %d, 实际 = %d", DefaultRetryMaxAttempts, config.RetryMaxAttempts)
	}
	if config.RetryInitialDelay != DefaultRetryInitialDelay {
		t.Errorf("期望 RetryInitialDelay = %v, 实际 = %v", DefaultRetryInitialDelay, config.RetryInitialDelay)
	}
	if config.RetryMaxDelay != DefaultRetryMaxDelay {
		t.Errorf("期望 RetryMaxDelay = %v, 实际 = %v", DefaultRetryMaxDelay, config.RetryMaxDelay)
	}
	if config.RetryBackoffFactor != DefaultRetryBackoffFactor {
		t.Errorf("期望 RetryBackoffFactor = %f, 实际 = %f", DefaultRetryBackoffFactor, config.RetryBackoffFactor)
	}
	if !config.RetryEnabled {
		t.Error("期望 RetryEnabled = true, 实际 = false")
	}
	if !config.RetryJitterEnabled {
		t.Error("期望 RetryJitterEnabled = true, 实际 = false")
	}
}

func TestRetryConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "有效的重试配置",
			config: &Config{
				Driver:             "sqlite",
				Database:           ":memory:",
				RetryMaxAttempts:   3,
				RetryInitialDelay:  time.Second,
				RetryMaxDelay:      10 * time.Second,
				RetryBackoffFactor: 2.0,
			},
			wantErr: false,
		},
		{
			name: "负数重试次数",
			config: &Config{
				Driver:           "sqlite",
				Database:         ":memory:",
				RetryMaxAttempts: -1,
			},
			wantErr: true,
		},
		{
			name: "过大的重试次数",
			config: &Config{
				Driver:           "sqlite",
				Database:         ":memory:",
				RetryMaxAttempts: 101,
			},
			wantErr: true,
		},
		{
			name: "负数初始延迟",
			config: &Config{
				Driver:            "sqlite",
				Database:          ":memory:",
				RetryMaxAttempts:  3,
				RetryInitialDelay: -time.Second,
			},
			wantErr: true,
		},
		{
			name: "初始延迟大于最大延迟",
			config: &Config{
				Driver:            "sqlite",
				Database:          ":memory:",
				RetryMaxAttempts:  3,
				RetryInitialDelay: 10 * time.Second,
				RetryMaxDelay:     5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "退避因子小于1",
			config: &Config{
				Driver:             "sqlite",
				Database:           ":memory:",
				RetryMaxAttempts:   3,
				RetryBackoffFactor: 0.5,
			},
			wantErr: true,
		},
		{
			name: "退避因子过大",
			config: &Config{
				Driver:             "sqlite",
				Database:           ":memory:",
				RetryMaxAttempts:   3,
				RetryBackoffFactor: 11.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.SetDefaults()
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateRetryDelay(t *testing.T) {
	config := &Config{
		RetryInitialDelay:  time.Second,
		RetryMaxDelay:      10 * time.Second,
		RetryBackoffFactor: 2.0,
		RetryJitterEnabled: false, // 禁用抖动以便测试
	}

	tests := []struct {
		name    string
		attempt int
		expect  time.Duration
	}{
		{
			name:    "第一次重试",
			attempt: 0,
			expect:  time.Second,
		},
		{
			name:    "第二次重试",
			attempt: 1,
			expect:  2 * time.Second,
		},
		{
			name:    "第三次重试",
			attempt: 2,
			expect:  4 * time.Second,
		},
		{
			name:    "达到最大延迟",
			attempt: 10,
			expect:  10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := calculateRetryDelay(config, tt.attempt)
			if delay != tt.expect {
				t.Errorf("calculateRetryDelay() = %v, 期望 %v", delay, tt.expect)
			}
		})
	}
}

func TestCalculateRetryDelay_WithJitter(t *testing.T) {
	config := &Config{
		RetryInitialDelay:  time.Second,
		RetryMaxDelay:      10 * time.Second,
		RetryBackoffFactor: 2.0,
		RetryJitterEnabled: true, // 启用抖动
	}

	// 测试抖动是否在合理范围内
	baseDelay := time.Second
	delay := calculateRetryDelay(config, 0)

	// 抖动应该在10%范围内
	minDelay := baseDelay
	maxDelay := time.Duration(float64(baseDelay) * 1.1)

	if delay < minDelay || delay > maxDelay {
		t.Errorf("抖动延迟 %v 超出预期范围 [%v, %v]", delay, minDelay, maxDelay)
	}
}

func TestRetryDisabled(t *testing.T) {
	config := &Config{
		Driver:           "sqlite",
		Database:         ":memory:",
		RetryEnabled:     false,
		RetryMaxAttempts: 1,
	}

	config.SetDefaults()

	// 验证禁用重试时的行为
	if config.RetryEnabled {
		t.Error("期望 RetryEnabled = false, 实际 = true")
	}
}

func TestSuccessfulConnection(t *testing.T) {
	config := &Config{
		Driver:           "sqlite",
		Database:         ":memory:",
		RetryMaxAttempts: 3,
	}

	config.SetDefaults()

	db, err := New(config)
	if err != nil {
		t.Fatalf("期望成功连接，但收到错误: %v", err)
	}
	defer db.Close()

	// 验证连接是否正常
	if err := db.Ping(); err != nil {
		t.Errorf("Ping() 失败: %v", err)
	}
}
