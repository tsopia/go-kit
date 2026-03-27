package httpserver

import (
	"fmt"
	"time"
)

const (
	DisableTimeout           time.Duration = -1
	defaultReadTimeout                     = 30 * time.Second
	defaultReadHeaderTimeout               = 5 * time.Second
	defaultWriteTimeout                    = 60 * time.Second
	defaultIdleTimeout                     = 60 * time.Second
	defaultShutdownTimeout                 = 10 * time.Second
	defaultDrainTimeout                    = 5 * time.Second
	defaultMaxHeaderBytes                  = 1 << 20
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

	// HandlerTimeout 用于 preset 自动挂载的 Timeout 中间件。
	//
	// 重要：仅在 preset.NewProductionServer 中生效！
	// 如果使用 NewServer 手动创建服务器，需要自行挂载：
	//   srv.Use(middleware.Timeout(duration))
	// 必须小于 WriteTimeout 才能生效（否则 Validate 会返回错误）。
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

func normalizeTimeoutValue(v *time.Duration, def time.Duration) {
	switch *v {
	case DisableTimeout:
		*v = 0
	case 0:
		*v = def
	}
}

func validateTimeoutValue(name string, v time.Duration) error {
	if v < DisableTimeout {
		return fmt.Errorf("%w: %s must be >= %v", ErrInvalidConfig, name, DisableTimeout)
	}
	return nil
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
	normalizeTimeoutValue(&c.ReadTimeout, defaultReadTimeout)
	normalizeTimeoutValue(&c.ReadHeaderTimeout, defaultReadHeaderTimeout)
	normalizeTimeoutValue(&c.WriteTimeout, defaultWriteTimeout)
	normalizeTimeoutValue(&c.IdleTimeout, defaultIdleTimeout)
	if c.MaxHeaderBytes <= 0 {
		c.MaxHeaderBytes = defaultMaxHeaderBytes
	}
	normalizeTimeoutValue(&c.ShutdownTimeout, defaultShutdownTimeout)
	normalizeTimeoutValue(&c.DrainTimeout, defaultDrainTimeout)
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
	if err := validateTimeoutValue("read timeout", c.ReadTimeout); err != nil {
		return err
	}
	if err := validateTimeoutValue("read header timeout", c.ReadHeaderTimeout); err != nil {
		return err
	}
	if err := validateTimeoutValue("write timeout", c.WriteTimeout); err != nil {
		return err
	}
	if err := validateTimeoutValue("idle timeout", c.IdleTimeout); err != nil {
		return err
	}
	if err := validateTimeoutValue("shutdown timeout", c.ShutdownTimeout); err != nil {
		return err
	}
	if err := validateTimeoutValue("drain timeout", c.DrainTimeout); err != nil {
		return err
	}
	if c.HandlerTimeout < 0 {
		return fmt.Errorf("%w: handler timeout must be >= 0", ErrInvalidConfig)
	}
	if c.HandlerTimeout > 0 && c.WriteTimeout > 0 && c.HandlerTimeout >= c.WriteTimeout {
		return fmt.Errorf("%w: handler timeout must be < write timeout", ErrInvalidConfig)
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
