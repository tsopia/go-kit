package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// PoolStats 连接池统计信息
type PoolStats struct {
	OpenConnections   int
	IdleConnections   int
	WaitCount         int64
	WaitDuration      time.Duration
	MaxIdleClosed     int64
	MaxLifetimeClosed int64
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

// HealthChecker 定义健康探测接口，支持 context 取消。
type HealthChecker interface {
	Ping() error
	HealthCheck() error
	HealthCheckWithContext(ctx context.Context) *HealthStatus
}

type gormHealthChecker struct {
	dbProvider     func() *gorm.DB
	driverProvider func() string
	statsProvider  func() PoolStats
	hooks          Hooks
	logger         SimpleLogger
}

func newGormHealthChecker(dbProvider func() *gorm.DB, driverProvider func() string, statsProvider func() PoolStats, hooks Hooks, logger SimpleLogger) *gormHealthChecker {
	return &gormHealthChecker{
		dbProvider:     dbProvider,
		driverProvider: driverProvider,
		statsProvider:  statsProvider,
		hooks:          hooks,
		logger:         logger,
	}
}

func (h *gormHealthChecker) Ping() error {
	sqlDB, err := h.dbProvider().DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func (h *gormHealthChecker) HealthCheck() error {
	if err := h.Ping(); err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	stats := h.statsProvider()
	if stats.OpenConnections == 0 {
		return fmt.Errorf("没有可用的数据库连接")
	}

	return nil
}

func (h *gormHealthChecker) HealthCheckWithContext(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Healthy:   true,
		Timestamp: time.Now(),
		Driver:    h.driverProvider(),
		Stats:     h.statsProvider(),
	}

	if h.hooks.BeforeProbe != nil {
		if err := h.hooks.BeforeProbe(ctx); err != nil {
			status.Healthy = false
			status.Errors = append(status.Errors, fmt.Sprintf("Probe前置检查失败: %v", err))
			h.finishProbe(ctx, err)
			return status
		}
	}

	if ctx.Err() != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("Context错误: %v", ctx.Err()))
		h.finishProbe(ctx, ctx.Err())
		return status
	}

	if err := h.Ping(); err != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("连接失败: %v", err))
	}

	if status.Stats.OpenConnections == 0 {
		status.Healthy = false
		status.Errors = append(status.Errors, "无可用连接")
	}

	if err := h.runDriverSpecificCheck(ctx); err != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, err.Error())
	}

	h.finishProbe(ctx, nil)
	return status
}

func (h *gormHealthChecker) runDriverSpecificCheck(ctx context.Context) error {
	db := h.dbProvider().WithContext(ctx)
	var query string

	switch h.driverProvider() {
	case "mysql":
		query = "SELECT 1"
	case "postgres":
		query = "SELECT 1"
	case "sqlite":
		query = "SELECT 1"
	default:
		return fmt.Errorf("不支持的数据库驱动: %s", h.driverProvider())
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

func (h *gormHealthChecker) finishProbe(ctx context.Context, probeErr error) {
	if h.hooks.AfterProbe != nil {
		h.hooks.AfterProbe(ctx, probeErr)
	}
}
