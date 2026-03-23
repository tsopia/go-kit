package httpserver

import (
	"fmt"
	"time"
)

const (
	defaultReadTimeout       = 30 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
	defaultWriteTimeout      = 60 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
	defaultDrainTimeout      = 5 * time.Second
	defaultMaxHeaderBytes    = 1 << 20
)

// Config 服务器配置。
type Config struct {
	Host              string
	Port              int
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ShutdownTimeout   time.Duration
	DrainTimeout      time.Duration

	EnableHealthCheck bool
	HealthCheckPath   string
	ReadinessPath     string
	LivenessPath      string
	HealthCheckPort   int

	// HandlerTimeout 用于 preset 自动挂载的中间件超时。
	// 只在 preset.NewProductionServer 中使用，非 preset 场景无效。
	// 必须小于 WriteTimeout 才能生效。
	HandlerTimeout time.Duration
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		Host:              "0.0.0.0",
		Port:              8080,
		ReadTimeout:       defaultReadTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
		ShutdownTimeout:   defaultShutdownTimeout,
		DrainTimeout:      defaultDrainTimeout,
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
		ReadinessPath:     "/readyz",
		LivenessPath:      "/livez",
		HealthCheckPort:   0,
	}
}

// Normalize 为零值填充默认配置。
func (c *Config) Normalize() {
	if c == nil {
		return
	}

	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = defaultReadTimeout
	}
	if c.ReadHeaderTimeout <= 0 {
		c.ReadHeaderTimeout = defaultReadHeaderTimeout
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = defaultWriteTimeout
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = defaultIdleTimeout
	}
	if c.MaxHeaderBytes <= 0 {
		c.MaxHeaderBytes = defaultMaxHeaderBytes
	}
	if c.ShutdownTimeout <= 0 {
		c.ShutdownTimeout = defaultShutdownTimeout
	}
	if c.DrainTimeout <= 0 {
		c.DrainTimeout = defaultDrainTimeout
	}
	if c.HealthCheckPath == "" {
		c.HealthCheckPath = "/health"
	}
	if c.ReadinessPath == "" {
		c.ReadinessPath = "/readyz"
	}
	if c.LivenessPath == "" {
		c.LivenessPath = "/livez"
	}
}

// Validate 校验服务器配置。
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: config is nil", ErrInvalidConfig)
	}

	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("%w: invalid port %d", ErrInvalidConfig, c.Port)
	}
	if c.HealthCheckPort < 0 || c.HealthCheckPort > 65535 {
		return fmt.Errorf("%w: invalid health check port %d", ErrInvalidConfig, c.HealthCheckPort)
	}
	if c.HealthCheckPort != 0 && c.HealthCheckPort == c.Port {
		return fmt.Errorf("%w: health check port conflicts with port %d", ErrInvalidConfig, c.Port)
	}
	if c.ReadTimeout < 0 {
		return fmt.Errorf("%w: read timeout must be >= 0", ErrInvalidConfig)
	}
	if c.ReadHeaderTimeout < 0 {
		return fmt.Errorf("%w: read header timeout must be >= 0", ErrInvalidConfig)
	}
	if c.WriteTimeout < 0 {
		return fmt.Errorf("%w: write timeout must be >= 0", ErrInvalidConfig)
	}
	if c.IdleTimeout < 0 {
		return fmt.Errorf("%w: idle timeout must be >= 0", ErrInvalidConfig)
	}
	if c.ShutdownTimeout < 0 {
		return fmt.Errorf("%w: shutdown timeout must be >= 0", ErrInvalidConfig)
	}
	if c.DrainTimeout < 0 {
		return fmt.Errorf("%w: drain timeout must be >= 0", ErrInvalidConfig)
	}
	if c.HealthCheckPath == "" {
		return fmt.Errorf("%w: health check path is required", ErrInvalidConfig)
	}
	if c.ReadinessPath == "" {
		return fmt.Errorf("%w: readiness path is required", ErrInvalidConfig)
	}
	if c.LivenessPath == "" {
		return fmt.Errorf("%w: liveness path is required", ErrInvalidConfig)
	}

	return nil
}
