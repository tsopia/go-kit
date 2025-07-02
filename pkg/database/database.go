package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// 预定义错误
var (
	ErrMissingDriver     = errors.New("数据库驱动不能为空")
	ErrUnsupportedDriver = errors.New("不支持的数据库驱动")
	ErrMissingHost       = errors.New("数据库主机不能为空")
	ErrInvalidPort       = errors.New("数据库端口无效")
	ErrMissingUsername   = errors.New("数据库用户名不能为空")
	ErrMissingDatabase   = errors.New("数据库名不能为空")
	ErrMissingDBPath     = errors.New("SQLite数据库路径不能为空")
	ErrInvalidLogLevel   = errors.New("无效的日志级别")
	ErrInvalidCharset    = errors.New("无效的字符集")
	ErrInvalidSSLMode    = errors.New("无效的SSL模式")
	ErrInvalidConnPool   = errors.New("连接池配置无效")
	ErrInvalidTimeout    = errors.New("超时配置无效")
	ErrConnectionFailed  = errors.New("数据库连接失败")
	ErrTransactionFailed = errors.New("事务执行失败")
	ErrQueryFailed       = errors.New("查询执行失败")
	ErrMigrationFailed   = errors.New("数据库迁移失败")
)

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeConnection ErrorType = iota
	ErrorTypeValidation
	ErrorTypeQuery
	ErrorTypeTransaction
	ErrorTypeMigration
)

// DatabaseError 数据库错误结构
type DatabaseError struct {
	Type      ErrorType
	Operation string
	Err       error
	Context   map[string]interface{}
}

// Error 实现error接口
func (e *DatabaseError) Error() string {
	if e.Context != nil && len(e.Context) > 0 {
		return fmt.Sprintf("数据库错误 [%s]: %v (上下文: %v)", e.Operation, e.Err, e.Context)
	}
	return fmt.Sprintf("数据库错误 [%s]: %v", e.Operation, e.Err)
}

// Unwrap 支持errors.Unwrap
func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// Is 支持errors.Is
func (e *DatabaseError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewDatabaseError 创建数据库错误
func NewDatabaseError(errorType ErrorType, operation string, err error) *DatabaseError {
	return &DatabaseError{
		Type:      errorType,
		Operation: operation,
		Err:       err,
		Context:   make(map[string]interface{}),
	}
}

// WithContext 添加错误上下文
func (e *DatabaseError) WithContext(key string, value interface{}) *DatabaseError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// IsConnectionError 检查是否为连接错误
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	var dbErr *DatabaseError
	if errors.As(err, &dbErr) {
		return dbErr.Type == ErrorTypeConnection
	}

	return errors.Is(err, ErrConnectionFailed)
}

// IsValidationError 检查是否为验证错误
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	var dbErr *DatabaseError
	if errors.As(err, &dbErr) {
		return dbErr.Type == ErrorTypeValidation
	}

	// 检查是否为我们定义的验证错误
	return errors.Is(err, ErrMissingDriver) ||
		errors.Is(err, ErrUnsupportedDriver) ||
		errors.Is(err, ErrMissingHost) ||
		errors.Is(err, ErrInvalidPort) ||
		errors.Is(err, ErrMissingUsername) ||
		errors.Is(err, ErrMissingDatabase) ||
		errors.Is(err, ErrMissingDBPath) ||
		errors.Is(err, ErrInvalidLogLevel) ||
		errors.Is(err, ErrInvalidCharset) ||
		errors.Is(err, ErrInvalidSSLMode) ||
		errors.Is(err, ErrInvalidConnPool) ||
		errors.Is(err, ErrInvalidTimeout)
}

// 默认配置常量
const (
	DefaultMaxIdleConns     = 10
	DefaultMaxOpenConns     = 100
	DefaultConnMaxLifetime  = time.Hour
	DefaultConnMaxIdleTime  = 10 * time.Minute
	DefaultSlowThreshold    = time.Second
	DefaultLogLevel         = "silent"
	DefaultCharset          = "utf8mb4"
	DefaultTimezone         = "Local"
	DefaultPostgresSSLMode  = "disable"
	DefaultPostgresTimezone = "UTC"

	// 重试配置默认值
	DefaultRetryMaxAttempts   = 3
	DefaultRetryInitialDelay  = 1 * time.Second
	DefaultRetryMaxDelay      = 30 * time.Second
	DefaultRetryBackoffFactor = 2.0
	DefaultRetryJitterEnabled = true
)

// Config 数据库配置
type Config struct {
	// 基础连接配置
	Driver   string `mapstructure:"driver" json:"driver" yaml:"driver"`
	Host     string `mapstructure:"host" json:"host" yaml:"host"`
	Port     int    `mapstructure:"port" json:"port" yaml:"port"`
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	Database string `mapstructure:"database" json:"database" yaml:"database"`
	Charset  string `mapstructure:"charset" json:"charset" yaml:"charset"`
	SSLMode  string `mapstructure:"ssl_mode" json:"ssl_mode" yaml:"ssl_mode"`
	Timezone string `mapstructure:"timezone" json:"timezone" yaml:"timezone"`

	// 连接池配置
	MaxIdleConns    int           `mapstructure:"max_idle_conns" json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" json:"max_open_conns" yaml:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" json:"conn_max_idle_time" yaml:"conn_max_idle_time"`

	// GORM日志配置
	CustomLogger              logger.Interface `mapstructure:"-" json:"-" yaml:"-"`
	LogLevel                  string           `mapstructure:"log_level" json:"log_level" yaml:"log_level"`
	SlowThreshold             time.Duration    `mapstructure:"slow_threshold" json:"slow_threshold" yaml:"slow_threshold"`
	IgnoreRecordNotFoundError bool             `mapstructure:"ignore_record_not_found_error" json:"ignore_record_not_found_error" yaml:"ignore_record_not_found_error"`
	ParameterizedQueries      bool             `mapstructure:"parameterized_queries" json:"parameterized_queries" yaml:"parameterized_queries"`
	Colorful                  bool             `mapstructure:"colorful" json:"colorful" yaml:"colorful"`

	// 连接重试配置
	RetryMaxAttempts   int           `mapstructure:"retry_max_attempts" json:"retry_max_attempts" yaml:"retry_max_attempts"`
	RetryInitialDelay  time.Duration `mapstructure:"retry_initial_delay" json:"retry_initial_delay" yaml:"retry_initial_delay"`
	RetryMaxDelay      time.Duration `mapstructure:"retry_max_delay" json:"retry_max_delay" yaml:"retry_max_delay"`
	RetryBackoffFactor float64       `mapstructure:"retry_backoff_factor" json:"retry_backoff_factor" yaml:"retry_backoff_factor"`
	RetryJitterEnabled bool          `mapstructure:"retry_jitter_enabled" json:"retry_jitter_enabled" yaml:"retry_jitter_enabled"`
	RetryEnabled       bool          `mapstructure:"retry_enabled" json:"retry_enabled" yaml:"retry_enabled"`

	// 其他配置
	TablePrefix       string `mapstructure:"table_prefix" json:"table_prefix" yaml:"table_prefix"`
	SingularTable     bool   `mapstructure:"singular_table" json:"singular_table" yaml:"singular_table"`
	DisableForeignKey bool   `mapstructure:"disable_foreign_key" json:"disable_foreign_key" yaml:"disable_foreign_key"`
	PrepareStmt       bool   `mapstructure:"prepare_stmt" json:"prepare_stmt" yaml:"prepare_stmt"`
	DryRun            bool   `mapstructure:"dry_run" json:"dry_run" yaml:"dry_run"`
}

// SetDefaults 设置默认值
func (c *Config) SetDefaults() {
	if c.LogLevel == "" {
		c.LogLevel = DefaultLogLevel
	}
	if c.SlowThreshold == 0 {
		c.SlowThreshold = DefaultSlowThreshold
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = DefaultMaxIdleConns
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = DefaultMaxOpenConns
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = DefaultConnMaxLifetime
	}
	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = DefaultConnMaxIdleTime
	}

	// 重试配置默认值
	if c.RetryMaxAttempts == 0 {
		c.RetryMaxAttempts = DefaultRetryMaxAttempts
	}
	if c.RetryInitialDelay == 0 {
		c.RetryInitialDelay = DefaultRetryInitialDelay
	}
	if c.RetryMaxDelay == 0 {
		c.RetryMaxDelay = DefaultRetryMaxDelay
	}
	if c.RetryBackoffFactor == 0 {
		c.RetryBackoffFactor = DefaultRetryBackoffFactor
	}
	// 默认启用重试和抖动
	if c.RetryMaxAttempts > 1 {
		c.RetryEnabled = true
	}
	if c.RetryEnabled && !c.RetryJitterEnabled {
		c.RetryJitterEnabled = DefaultRetryJitterEnabled
	}

	// 数据库特定默认值
	switch c.Driver {
	case "mysql":
		if c.Charset == "" {
			c.Charset = DefaultCharset
		}
		if c.Timezone == "" {
			c.Timezone = DefaultTimezone
		}
	case "postgres":
		if c.SSLMode == "" {
			c.SSLMode = DefaultPostgresSSLMode
		}
		if c.Timezone == "" {
			c.Timezone = DefaultPostgresTimezone
		}
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证驱动
	if c.Driver == "" {
		return ErrMissingDriver
	}

	switch c.Driver {
	case "mysql", "postgres", "sqlite":
		// 支持的驱动
	default:
		return fmt.Errorf("%w: %s (支持的驱动: mysql, postgres, sqlite)", ErrUnsupportedDriver, c.Driver)
	}

	// 验证日志级别
	if c.LogLevel != "" && !IsValidLogLevel(c.LogLevel) {
		return fmt.Errorf("%w: %s", ErrInvalidLogLevel, c.LogLevel)
	}

	// 根据驱动类型进行特定验证
	switch c.Driver {
	case "mysql":
		if err := c.validateMySQL(); err != nil {
			return err
		}
	case "postgres":
		if err := c.validatePostgreSQL(); err != nil {
			return err
		}
	case "sqlite":
		if err := c.validateSQLite(); err != nil {
			return err
		}
	}

	// 验证连接池配置
	if err := c.validateConnectionPool(); err != nil {
		return err
	}

	// 验证时间配置
	if err := c.validateTimeouts(); err != nil {
		return err
	}

	// 验证重试配置
	if err := c.validateRetryConfig(); err != nil {
		return err
	}

	return nil
}

// validateMySQL 验证MySQL特定配置
func (c *Config) validateMySQL() error {
	if c.Host == "" {
		return ErrMissingHost
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("%w: 端口必须在1-65535范围内，当前值: %d", ErrInvalidPort, c.Port)
	}
	if c.Username == "" {
		return ErrMissingUsername
	}
	if c.Database == "" {
		return ErrMissingDatabase
	}

	// 验证MySQL字符集
	if c.Charset != "" && !isValidMySQLCharset(c.Charset) {
		return fmt.Errorf("%w: %s (支持的字符集: utf8, utf8mb4, latin1, gbk)", ErrInvalidCharset, c.Charset)
	}

	// 验证数据库名格式
	if !isValidDatabaseName(c.Database) {
		return fmt.Errorf("%w: 数据库名包含非法字符", ErrMissingDatabase)
	}

	return nil
}

// validatePostgreSQL 验证PostgreSQL特定配置
func (c *Config) validatePostgreSQL() error {
	if c.Host == "" {
		return ErrMissingHost
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("%w: 端口必须在1-65535范围内，当前值: %d", ErrInvalidPort, c.Port)
	}
	if c.Username == "" {
		return ErrMissingUsername
	}
	if c.Database == "" {
		return ErrMissingDatabase
	}

	// 验证PostgreSQL SSL模式
	if c.SSLMode != "" && !isValidPostgreSQLSSLMode(c.SSLMode) {
		return fmt.Errorf("%w: %s (支持的SSL模式: disable, require, verify-ca, verify-full)", ErrInvalidSSLMode, c.SSLMode)
	}

	// 验证数据库名格式
	if !isValidDatabaseName(c.Database) {
		return fmt.Errorf("%w: 数据库名包含非法字符", ErrMissingDatabase)
	}

	return nil
}

// validateSQLite 验证SQLite特定配置
func (c *Config) validateSQLite() error {
	if c.Database == "" {
		return ErrMissingDBPath
	}

	// 内存数据库特殊处理
	if c.Database == ":memory:" {
		return nil
	}

	// 检查SQLite文件目录是否存在
	if dir := filepath.Dir(c.Database); dir != "." {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("%w: SQLite数据库目录不存在: %s", ErrMissingDBPath, dir)
		}
	}

	return nil
}

// validateConnectionPool 验证连接池配置
func (c *Config) validateConnectionPool() error {
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("%w: 最大空闲连接数不能为负数", ErrInvalidConnPool)
	}
	if c.MaxOpenConns < 0 {
		return fmt.Errorf("%w: 最大打开连接数不能为负数", ErrInvalidConnPool)
	}
	if c.MaxOpenConns > 0 && c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("%w: 最大空闲连接数(%d)不能大于最大打开连接数(%d)", ErrInvalidConnPool, c.MaxIdleConns, c.MaxOpenConns)
	}

	return nil
}

// validateTimeouts 验证超时配置
func (c *Config) validateTimeouts() error {
	if c.SlowThreshold < 0 {
		return fmt.Errorf("%w: 慢查询阈值不能为负数", ErrInvalidTimeout)
	}
	if c.ConnMaxLifetime < 0 {
		return fmt.Errorf("%w: 连接最大生存时间不能为负数", ErrInvalidTimeout)
	}
	if c.ConnMaxIdleTime < 0 {
		return fmt.Errorf("%w: 连接最大空闲时间不能为负数", ErrInvalidTimeout)
	}

	return nil
}

// validateRetryConfig 验证重试配置
func (c *Config) validateRetryConfig() error {
	if c.RetryMaxAttempts < 0 {
		return fmt.Errorf("%w: 重试最大次数不能为负数", ErrInvalidTimeout)
	}
	if c.RetryMaxAttempts > 100 {
		return fmt.Errorf("%w: 重试最大次数不能超过100次", ErrInvalidTimeout)
	}
	if c.RetryInitialDelay < 0 {
		return fmt.Errorf("%w: 初始重试延迟不能为负数", ErrInvalidTimeout)
	}
	if c.RetryMaxDelay < 0 {
		return fmt.Errorf("%w: 最大重试延迟不能为负数", ErrInvalidTimeout)
	}
	if c.RetryMaxDelay > 0 && c.RetryInitialDelay > c.RetryMaxDelay {
		return fmt.Errorf("%w: 初始重试延迟不能大于最大重试延迟", ErrInvalidTimeout)
	}
	if c.RetryBackoffFactor < 1.0 {
		return fmt.Errorf("%w: 重试退避因子不能小于1.0", ErrInvalidTimeout)
	}
	if c.RetryBackoffFactor > 10.0 {
		return fmt.Errorf("%w: 重试退避因子不能大于10.0", ErrInvalidTimeout)
	}

	return nil
}

// isValidMySQLCharset 验证MySQL字符集
func isValidMySQLCharset(charset string) bool {
	validCharsets := []string{"utf8", "utf8mb4", "latin1", "gbk", "gb2312", "ascii"}
	for _, valid := range validCharsets {
		if strings.EqualFold(charset, valid) {
			return true
		}
	}
	return false
}

// isValidPostgreSQLSSLMode 验证PostgreSQL SSL模式
func isValidPostgreSQLSSLMode(sslMode string) bool {
	validModes := []string{"disable", "require", "verify-ca", "verify-full"}
	for _, valid := range validModes {
		if strings.EqualFold(sslMode, valid) {
			return true
		}
	}
	return false
}

// isValidDatabaseName 验证数据库名格式
func isValidDatabaseName(name string) bool {
	// 数据库名只能包含字母、数字、下划线和连字符
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched && len(name) <= 64
}

// PoolStats 连接池统计信息
type PoolStats struct {
	OpenConnections   int
	IdleConnections   int
	WaitCount         int64
	WaitDuration      time.Duration
	MaxIdleClosed     int64
	MaxLifetimeClosed int64
}

// Database 数据库管理器
type Database struct {
	config *Config
	db     *gorm.DB
	mu     sync.RWMutex
}

// New 创建新的数据库管理器
func New(config *Config) (*Database, error) {
	// 设置默认值
	config.SetDefaults()

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	db, err := connect(config)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	database := &Database{
		config: config,
		db:     db,
	}

	// 配置连接池
	if err := database.configurePool(); err != nil {
		// 如果连接池配置失败，关闭已建立的连接
		if closeErr := database.Close(); closeErr != nil {
			return nil, fmt.Errorf("配置连接池失败: %w (关闭连接时发生额外错误: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("配置连接池失败: %w", err)
	}

	return database, nil
}

// connect 连接数据库
func connect(config *Config) (*gorm.DB, error) {
	if config.RetryEnabled && config.RetryMaxAttempts > 1 {
		return connectWithRetry(config)
	}
	return connectOnce(config)
}

// connectOnce 单次连接数据库
func connectOnce(config *Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch config.Driver {
	case "mysql":
		dsn := buildMySQLDSN(config)
		dialector = mysql.Open(dsn)

	case "postgres":
		dsn := buildPostgresDSN(config)
		dialector = postgres.Open(dsn)

	case "sqlite":
		dialector = sqlite.Open(config.Database)

	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", config.Driver)
	}

	// 配置GORM
	gormConfig := &gorm.Config{
		Logger:                                   newGormLogger(config),
		NamingStrategy:                           buildNamingStrategy(config),
		DisableForeignKeyConstraintWhenMigrating: config.DisableForeignKey,
		PrepareStmt:                              config.PrepareStmt,
		DryRun:                                   config.DryRun,
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// connectWithRetry 带重试的连接数据库
func connectWithRetry(config *Config) (*gorm.DB, error) {
	var lastErr error

	for attempt := 1; attempt <= config.RetryMaxAttempts; attempt++ {
		db, err := connectOnce(config)
		if err == nil {
			if attempt > 1 {
				log.Printf("数据库连接成功，尝试次数: %d", attempt)
			}
			return db, nil
		}

		lastErr = err

		// 如果是最后一次尝试，不需要等待
		if attempt == config.RetryMaxAttempts {
			break
		}

		// 计算延迟时间
		delay := calculateRetryDelay(config, attempt-1)
		log.Printf("数据库连接失败 (尝试 %d/%d): %v, %v后重试",
			attempt, config.RetryMaxAttempts, err, delay)

		// 等待后重试
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("数据库连接失败，已重试%d次: %w", config.RetryMaxAttempts, lastErr)
}

// calculateRetryDelay 计算重试延迟时间
func calculateRetryDelay(config *Config, attempt int) time.Duration {
	// 计算指数退避延迟
	delay := float64(config.RetryInitialDelay) * math.Pow(config.RetryBackoffFactor, float64(attempt))

	// 限制最大延迟
	if config.RetryMaxDelay > 0 && time.Duration(delay) > config.RetryMaxDelay {
		delay = float64(config.RetryMaxDelay)
	}

	// 添加抖动以避免雷群效应
	if config.RetryJitterEnabled {
		jitter := rand.Float64() * 0.1 // 10%的抖动
		delay = delay * (1 + jitter)
	}

	return time.Duration(delay)
}

// buildMySQLDSN 构建MySQL DSN
func buildMySQLDSN(config *Config) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.Charset,
		config.Timezone,
	)
}

// buildPostgresDSN 构建PostgreSQL DSN
func buildPostgresDSN(config *Config) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		config.Host,
		config.Port,
		config.Username,
		config.Password,
		config.Database,
		config.SSLMode,
		config.Timezone,
	)
}

// buildNamingStrategy 构建命名策略
func buildNamingStrategy(config *Config) schema.NamingStrategy {
	return schema.NamingStrategy{
		TablePrefix:   config.TablePrefix,
		SingularTable: config.SingularTable,
	}
}

// newGormLogger 创建GORM日志记录器
func newGormLogger(config *Config) logger.Interface {
	if config.CustomLogger != nil {
		return config.CustomLogger
	}
	logLevel := getLogLevel(config.LogLevel)

	// 创建日志配置
	logConfig := logger.Config{
		SlowThreshold:             config.SlowThreshold,
		LogLevel:                  logLevel,
		IgnoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
		ParameterizedQueries:      config.ParameterizedQueries,
		Colorful:                  config.Colorful,
	}

	// 处理日志输出
	var writer logger.Writer = log.New(os.Stdout, "\r\n", log.LstdFlags)

	return logger.New(writer, logConfig)
}

// configurePool 配置连接池
func (d *Database) configurePool() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(d.config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(d.config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(d.config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(d.config.ConnMaxIdleTime)

	return nil
}

// GetDB 获取GORM数据库实例
func (d *Database) GetDB() *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

// SetLogger 设置自定义日志记录器
func (d *Config) SetCustomLogger(l SimpleLogger, level string) {
	d.CustomLogger = NewGormLogger(l, level)
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping 测试数据库连接
func (d *Database) Ping() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Stats 获取连接池统计信息
func (d *Database) Stats() PoolStats {
	sqlDB, err := d.db.DB()
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

// HealthStatus 健康检查状态
type HealthStatus struct {
	Healthy   bool      `json:"healthy"`
	Timestamp time.Time `json:"timestamp"`
	Driver    string    `json:"driver"`
	Errors    []string  `json:"errors,omitempty"`
	Warnings  []string  `json:"warnings,omitempty"`
	Stats     PoolStats `json:"stats"`
}

// HealthCheck 健康检查
func (d *Database) HealthCheck() error {
	// 基本连接检查
	if err := d.Ping(); err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 检查连接池状态
	stats := d.Stats()
	if stats.OpenConnections == 0 {
		return fmt.Errorf("没有可用的数据库连接")
	}

	return nil
}

// HealthCheckWithContext 带Context的健康检查
func (d *Database) HealthCheckWithContext(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Healthy:   true,
		Timestamp: time.Now(),
		Driver:    d.GetDriver(),
		Stats:     d.Stats(),
	}

	// 检查Context是否已取消
	if ctx.Err() != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("Context错误: %v", ctx.Err()))
		return status
	}

	// 检查基本连接
	if err := d.Ping(); err != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("连接失败: %v", err))
	}

	// 检查连接池状态
	if status.Stats.OpenConnections == 0 {
		status.Healthy = false
		status.Errors = append(status.Errors, "无可用连接")
	}

	// 检查连接池健康状态
	if status.Stats.OpenConnections > 0 {
		// 检查等待时间是否过长
		if status.Stats.WaitCount > 0 && status.Stats.WaitDuration > 5*time.Second {
			status.Warnings = append(status.Warnings,
				fmt.Sprintf("连接等待时间过长: %v", status.Stats.WaitDuration))
		}

		// 检查连接池使用率
		if status.Stats.OpenConnections > 0 && status.Stats.IdleConnections == 0 {
			status.Warnings = append(status.Warnings, "连接池使用率过高，无空闲连接")
		}
	}

	// 执行简单查询测试
	if err := d.performQueryTest(ctx); err != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("查询测试失败: %v", err))
	}

	return status
}

// performQueryTest 执行简单查询测试
func (d *Database) performQueryTest(ctx context.Context) error {
	db := d.WithContext(ctx)

	// 根据不同数据库类型执行不同的测试查询
	var query string
	switch d.GetDriver() {
	case "mysql":
		query = "SELECT 1"
	case "postgres":
		query = "SELECT 1"
	case "sqlite":
		query = "SELECT 1"
	default:
		return fmt.Errorf("不支持的数据库驱动: %s", d.GetDriver())
	}

	var result int
	if err := db.Raw(query).Scan(&result).Error; err != nil {
		return err
	}

	if result != 1 {
		return fmt.Errorf("查询结果异常: 期望1，得到%d", result)
	}

	return nil
}

// AutoMigrate 自动迁移数据库表
func (d *Database) AutoMigrate(dst ...interface{}) error {
	return d.db.AutoMigrate(dst...)
}

// IsConnected 检查数据库连接状态
func (d *Database) IsConnected() bool {
	return d.Ping() == nil
}

// WithContext 返回带有Context的GORM实例
func (d *Database) WithContext(ctx context.Context) *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db.WithContext(ctx)
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

// Transaction 事务便利方法，自动处理提交和回滚
func (d *Database) Transaction(fn func(*gorm.DB) error) error {
	return d.db.Transaction(fn)
}

// TransactionWithContext 带Context的事务便利方法
func (d *Database) TransactionWithContext(ctx context.Context, fn func(*gorm.DB) error) error {
	return d.db.WithContext(ctx).Transaction(fn)
}

// SafeString 返回安全的配置字符串（密码已脱敏）
func (c *Config) SafeString() string {
	safe := *c
	if safe.Password != "" {
		safe.Password = "***"
	}
	return fmt.Sprintf("Driver:%s Host:%s Port:%d Username:%s Database:%s",
		safe.Driver, safe.Host, safe.Port, safe.Username, safe.Database)
}
