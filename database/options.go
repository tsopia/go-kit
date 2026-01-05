package database

import (
	"context"

	"gorm.io/gorm"
)

// Hooks 定义生命周期扩展点，便于调用方在关键节点插入自定义逻辑。
type Hooks struct {
	BeforeConnect func(cfg *Config)
	AfterConnect  func(db *gorm.DB)

	BeforeClose func(ctx context.Context, db *gorm.DB) error
	AfterClose  func(ctx context.Context, closeErr error)

	BeforeProbe func(ctx context.Context) error
	AfterProbe  func(ctx context.Context, probeErr error)
}

// Option 允许在构建 Database 时注入可选配置，例如 Logger、Hooks 等。
type Option func(*Database)

// WithHooks 设置生命周期钩子。
func WithHooks(h Hooks) Option {
	return func(d *Database) {
		d.hooks = h
	}
}

// WithLogger 设置用于启动、重试和健康检查日志的 SimpleLogger。
func WithLogger(l SimpleLogger) Option {
	return func(d *Database) {
		d.logger = l
	}
}
