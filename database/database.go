package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	"gorm.io/gorm"
)

// Database 数据库管理器，组合 Connector、Executor、HealthChecker 等职责。
type Database struct {
	config *Config
	db     *gorm.DB

	mu     sync.RWMutex
	hooks  Hooks
	logger SimpleLogger

	connector     Connector
	executor      Executor
	healthChecker HealthChecker
}

type stdLogger struct{}

func (l stdLogger) Info(msg string, fields ...interface{})  { log.Printf("INFO %s %v", msg, fields) }
func (l stdLogger) Warn(msg string, fields ...interface{})  { log.Printf("WARN %s %v", msg, fields) }
func (l stdLogger) Error(msg string, fields ...interface{}) { log.Printf("ERROR %s %v", msg, fields) }

// New 创建新的数据库管理器
func New(config *Config) (*Database, error) {
	return NewWithOptions(config)
}

// NewWithOptions 支持通过可选参数注入日志、钩子或自定义组件。
func NewWithOptions(config *Config, opts ...Option) (*Database, error) {
	config.SetDefaults()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	database := &Database{config: config}

	for _, opt := range opts {
		opt(database)
	}

	if database.logger == nil {
		database.logger = stdLogger{}
	}

	if database.connector == nil {
		database.connector = newDefaultConnector(config, database.hooks, database.logger)
	}

	db, err := database.connector.Connect(context.Background())
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	database.db = db

	if database.executor == nil {
		database.executor = newGormExecutor(database.GetDB, database.logger)
	}
	if database.healthChecker == nil {
		database.healthChecker = newGormHealthChecker(database.GetDB, database.GetDriver, database.Stats, database.hooks, database.logger)
	}

	return database, nil
}

// GetDB 获取GORM数据库实例
func (d *Database) GetDB() *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

// WithContext 返回带有Context的GORM实例
func (d *Database) WithContext(ctx context.Context) *gorm.DB {
	return d.GetDB().WithContext(ctx)
}

// Exec 执行写操作
func (d *Database) Exec(ctx context.Context, query string, args ...interface{}) error {
	return d.executor.Exec(ctx, query, args...)
}

// Query 执行查询
func (d *Database) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return d.executor.Query(ctx, dest, query, args...)
}

// Tx 启动事务
func (d *Database) Tx(ctx context.Context, fn func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	return d.executor.Tx(ctx, fn, opts...)
}

// BeginTx 开启事务
func (d *Database) BeginTx(ctx context.Context, opts ...*sql.TxOptions) (*gorm.DB, error) {
	return d.executor.BeginTx(ctx, opts...)
}

// Raw 提供受控的底层 *gorm.DB 访问。
func (d *Database) Raw() *gorm.DB {
	return d.GetDB()
}

// SQLDB 返回底层 *sql.DB。
func (d *Database) SQLDB() (*sql.DB, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db.DB()
}

// AutoMigrate 自动迁移数据库表
func (d *Database) AutoMigrate(dst ...interface{}) error {
	return d.db.AutoMigrate(dst...)
}

// IsConnected 检查数据库连接状态
func (d *Database) IsConnected() bool {
	return d.Ping() == nil
}

// Ping 测试数据库连接
func (d *Database) Ping() error {
	return d.healthChecker.Ping()
}

// HealthCheck 健康检查
func (d *Database) HealthCheck() error {
	return d.healthChecker.HealthCheck()
}

// HealthCheckWithContext 带Context的健康检查
func (d *Database) HealthCheckWithContext(ctx context.Context) *HealthStatus {
	return d.healthChecker.HealthCheckWithContext(ctx)
}

// GetConfig 获取数据库配置（返回副本，防止外部修改）
func (d *Database) GetConfig() Config {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return *d.config
}

// GetDriver 获取数据库驱动类型
func (d *Database) GetDriver() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config.Driver
}

// IsReadOnly 检查是否为只读模式（DryRun模式）
func (d *Database) IsReadOnly() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config.DryRun
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	return d.CloseWithContext(context.Background())
}

// CloseWithContext 允许调用方传入上下文，配合生命周期钩子完成优雅关闭。
func (d *Database) CloseWithContext(ctx context.Context) error {
	if d.hooks.BeforeClose != nil {
		if err := d.hooks.BeforeClose(ctx, d.db); err != nil {
			return err
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	closeErr := sqlDB.Close()

	if d.hooks.AfterClose != nil {
		d.hooks.AfterClose(ctx, closeErr)
	}

	return closeErr
}

// Stats 获取连接池统计信息
func (d *Database) Stats() PoolStats {
	sqlDB, err := d.SQLDB()
	if err != nil {
		return PoolStats{}
	}

	stats := sqlDB.Stats()
	return PoolStats{
		OpenConnections:   stats.OpenConnections,
		IdleConnections:   stats.Idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		MaxIdleClosed:     stats.MaxIdleClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
	}
}

// Transaction 事务便利方法，自动处理提交和回滚
func (d *Database) Transaction(fn func(*gorm.DB) error) error {
	return d.db.Transaction(fn)
}

// TransactionWithContext 带Context的事务便利方法
func (d *Database) TransactionWithContext(ctx context.Context, fn func(*gorm.DB) error) error {
	return d.db.WithContext(ctx).Transaction(fn)
}

// Option helpers for custom components
func WithConnector(connector Connector) Option {
	return func(d *Database) {
		d.connector = connector
	}
}

func WithExecutor(executor Executor) Option {
	return func(d *Database) {
		d.executor = executor
	}
}

func WithHealthChecker(healthChecker HealthChecker) Option {
	return func(d *Database) {
		d.healthChecker = healthChecker
	}
}
