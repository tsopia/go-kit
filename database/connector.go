package database

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	mysqlcfg "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Connector 负责建立数据库连接并完成基础配置（重试、命名策略、连接池）。
type Connector interface {
	Connect(ctx context.Context) (*gorm.DB, error)
}

// PoolConfigurator 允许对底层 *sql.DB 进行池参数配置。
type PoolConfigurator interface {
	Configure(db *gorm.DB) error
}

type defaultConnector struct {
	config *Config
	hooks  Hooks
	logger SimpleLogger
}

func newDefaultConnector(config *Config, hooks Hooks, logger SimpleLogger) *defaultConnector {
	return &defaultConnector{config: config, hooks: hooks, logger: logger}
}

func (c *defaultConnector) Connect(ctx context.Context) (*gorm.DB, error) {
	if c.hooks.BeforeConnect != nil {
		c.hooks.BeforeConnect(c.config)
	}

	db, err := c.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	if err := c.configurePool(db); err != nil {
		return nil, fmt.Errorf("配置连接池失败: %w", err)
	}

	if c.hooks.AfterConnect != nil {
		c.hooks.AfterConnect(db)
	}

	return db, nil
}

func (c *defaultConnector) connect(ctx context.Context) (*gorm.DB, error) {
	if c.config.RetryEnabled && c.config.RetryMaxAttempts > 1 {
		return c.connectWithRetry(ctx)
	}
	return c.connectOnce()
}

func (c *defaultConnector) connectOnce() (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch c.config.Driver {
	case "mysql":
		dsn := buildMySQLDSN(c.config)
		dialector = mysql.Open(dsn)

	case "postgres":
		dsn := buildPostgresDSN(c.config)
		dialector = postgres.Open(dsn)

	case "sqlite":
		dsn := c.config.Database
		dialector = sqlite.Open(dsn)

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDriver, c.config.Driver)
	}

	namingStrategy := buildNamingStrategy(c.config)

	gormConfig := &gorm.Config{
		Logger:                                   newGormLogger(c.config),
		NamingStrategy:                           namingStrategy,
		DisableForeignKeyConstraintWhenMigrating: c.config.DisableForeignKey,
		PrepareStmt:                              c.config.PrepareStmt,
		DryRun:                                   c.config.DryRun,
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, NewDatabaseError(ErrorTypeConnection, "gorm.Open", err).
			WithContext("driver", c.config.Driver)
	}

	return db, nil
}

func (c *defaultConnector) connectWithRetry(ctx context.Context) (*gorm.DB, error) {
	config := c.config
	var lastErr error

	for attempt := 1; attempt <= config.RetryMaxAttempts; attempt++ {
		db, err := c.connectOnce()
		if err == nil {
			return db, nil
		}

		lastErr = err

		if attempt == config.RetryMaxAttempts {
			break
		}

		delay := calculateRetryDelay(config, attempt-1)
		c.logWarn("database connect failed; will retry", "attempt", attempt, "max_attempts", config.RetryMaxAttempts, "delay", delay, "error", err)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("数据库连接重试被取消: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return nil, fmt.Errorf("数据库连接失败，已重试%d次: %w", config.RetryMaxAttempts, lastErr)
}

func (c *defaultConnector) configurePool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxIdleConns(c.config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(c.config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(c.config.ConnMaxIdleTime)

	return nil
}

func (c *defaultConnector) logWarn(msg string, fields ...interface{}) {
	if c.logger != nil {
		c.logger.Warn(msg, fields...)
	}
}

func calculateRetryDelay(config *Config, attempt int) time.Duration {
	delay := float64(config.RetryInitialDelay) * math.Pow(config.RetryBackoffFactor, float64(attempt))

	if config.RetryMaxDelay > 0 && time.Duration(delay) > config.RetryMaxDelay {
		delay = float64(config.RetryMaxDelay)
	}

	if config.RetryJitterEnabled {
		jitter := rand.Float64() * 0.1
		delay = delay * (1 + jitter)
	}

	return time.Duration(delay)
}

func buildMySQLDSN(config *Config) string {
	cfg := mysqlcfg.Config{
		User:                 config.Username,
		Passwd:               config.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", config.Host, config.Port),
		DBName:               config.Database,
		ParseTime:            true,
		Loc:                  getTimeLocation(config.Timezone),
		AllowNativePasswords: true,
		Params:               map[string]string{"charset": config.Charset},
	}
	return cfg.FormatDSN()
}

func getTimeLocation(tz string) *time.Location {
	if tz == "" {
		tz = "Local"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Local
	}
	return loc
}

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

func buildNamingStrategy(config *Config) schema.NamingStrategy {
	return schema.NamingStrategy{
		TablePrefix:   config.TablePrefix,
		SingularTable: config.SingularTable,
	}
}

func newGormLogger(config *Config) logger.Interface {
	if config.CustomLogger != nil {
		return config.CustomLogger
	}
	logLevel := getLogLevel(config.LogLevel)

	logConfig := logger.Config{
		SlowThreshold:             config.SlowThreshold,
		LogLevel:                  logLevel,
		IgnoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
		ParameterizedQueries:      config.ParameterizedQueries,
		Colorful:                  config.Colorful,
	}

	return logger.New(newStdLogWriter(), logConfig)
}

func newStdLogWriter() logger.Writer {
	return logWrapper{underlying: stdLogger{}}
}

type logWrapper struct {
	underlying stdLogger
}

func (l logWrapper) Printf(msg string, data ...interface{}) {
	l.underlying.Info(fmt.Sprintf(msg, data...))
}
