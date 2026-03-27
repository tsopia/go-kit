package database

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm/logger"
)

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

	MaxRetryAttempts = 100
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
	RetryConfigured    bool          `mapstructure:"retry_configured" json:"retry_configured" yaml:"retry_configured"`

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
	explicitRetryConfig := c.hasExplicitRetryConfig()
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
	// 未显式提供重试配置时，沿用默认启用策略。
	if !explicitRetryConfig && c.RetryMaxAttempts > 1 {
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

func (c *Config) hasExplicitRetryConfig() bool {
	return c.RetryConfigured ||
		c.RetryMaxAttempts != 0 ||
		c.RetryInitialDelay != 0 ||
		c.RetryMaxDelay != 0 ||
		c.RetryBackoffFactor != 0
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

// SetCustomLogger 允许在 Config 层设置 GORM 日志实现，便于与外部 logger 对齐。
func (c *Config) SetCustomLogger(l SimpleLogger, level string) {
	c.CustomLogger = NewGormLogger(l, level)
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

func (c *Config) validateConnectionPool() error {
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("%w: 最大空闲连接数不能为负数", ErrInvalidConnPool)
	}
	if c.MaxOpenConns < 0 {
		return fmt.Errorf("%w: 最大打开连接数不能为负数", ErrInvalidConnPool)
	}
	if c.MaxOpenConns > 0 && c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("%w: 最大空闲连接数不能超过最大打开连接数", ErrInvalidConnPool)
	}
	if c.ConnMaxLifetime < 0 {
		return fmt.Errorf("%w: 连接最大生命周期不能为负数", ErrInvalidConnPool)
	}
	if c.ConnMaxIdleTime < 0 {
		return fmt.Errorf("%w: 连接最大空闲时间不能为负数", ErrInvalidConnPool)
	}

	return nil
}

func (c *Config) validateTimeouts() error {
	if c.SlowThreshold < 0 {
		return fmt.Errorf("%w: 慢查询阈值不能为负数", ErrInvalidTimeout)
	}
	if c.RetryInitialDelay < 0 || c.RetryMaxDelay < 0 {
		return fmt.Errorf("%w: 重试延迟不能为负数", ErrInvalidTimeout)
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

func (c *Config) validateRetryConfig() error {
	if c.RetryMaxAttempts < 1 {
		return fmt.Errorf("%w: 重试次数必须大于等于1", ErrInvalidTimeout)
	}
	if c.RetryMaxAttempts > MaxRetryAttempts {
		return fmt.Errorf("%w: 重试次数不能超过%d", ErrInvalidTimeout, MaxRetryAttempts)
	}
	if c.RetryEnabled && c.RetryMaxAttempts == 1 {
		return fmt.Errorf("%w: 启用重试时，重试次数必须大于1", ErrInvalidTimeout)
	}
	return nil
}

func isValidMySQLCharset(charset string) bool {
	validCharsets := []string{"utf8", "utf8mb4", "latin1", "gbk", "gb2312", "ascii"}
	for _, valid := range validCharsets {
		if strings.EqualFold(charset, valid) {
			return true
		}
	}
	return false
}

func isValidPostgreSQLSSLMode(sslMode string) bool {
	validModes := []string{"disable", "require", "verify-ca", "verify-full"}
	for _, valid := range validModes {
		if strings.EqualFold(sslMode, valid) {
			return true
		}
	}
	return false
}

func isValidDatabaseName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched && len(name) <= 64
}
