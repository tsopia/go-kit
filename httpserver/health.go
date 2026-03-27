package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 健康检查相关结构体和函数

// HealthStatus 健康检查状态
type HealthStatus struct {
	Status    string                 `json:"status"`            // healthy, unhealthy
	Timestamp int64                  `json:"timestamp"`         // Unix时间戳
	Version   string                 `json:"version,omitempty"` // 应用版本
	Uptime    int64                  `json:"uptime,omitempty"`  // 运行时间（秒）
	Checks    map[string]interface{} `json:"checks,omitempty"`  // 详细检查结果
	Error     string                 `json:"error,omitempty"`   // 错误信息
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
	GetName() string
}

// HealthCheckManager 健康检查管理器
type HealthCheckManager struct {
	checkers     []HealthChecker
	startTime    time.Time
	version      string
	checkTimeout time.Duration
}

// HealthCheckOption 描述 HealthCheckManager 的可选配置。
type HealthCheckOption func(*HealthCheckManager)

// WithCheckTimeout 设置单个 checker 的超时时间。
func WithCheckTimeout(timeout time.Duration) HealthCheckOption {
	return func(hcm *HealthCheckManager) {
		hcm.checkTimeout = timeout
	}
}

const defaultCheckTimeout = 5 * time.Second

// NewHealthCheckManager 创建健康检查管理器
func NewHealthCheckManager(version string, opts ...HealthCheckOption) *HealthCheckManager {
	hcm := &HealthCheckManager{
		checkers:     make([]HealthChecker, 0),
		startTime:    time.Now(),
		version:      version,
		checkTimeout: defaultCheckTimeout,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(hcm)
		}
	}
	return hcm
}

// AddChecker 添加健康检查器
func (hcm *HealthCheckManager) AddChecker(checker HealthChecker) {
	hcm.checkers = append(hcm.checkers, checker)
}

type checkResult struct {
	name       string
	err        error
	durationMs int64
}

// CheckHealth 并发执行所有健康检查
func (hcm *HealthCheckManager) CheckHealth(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Version:   hcm.version,
		Uptime:    int64(time.Since(hcm.startTime).Seconds()),
		Checks:    make(map[string]interface{}),
	}

	if len(hcm.checkers) == 0 {
		return status
	}

	results := make(chan checkResult, len(hcm.checkers))

	for _, checker := range hcm.checkers {
		go func(c HealthChecker) {
			checkCtx, cancel := context.WithTimeout(context.Background(), hcm.checkTimeout)
			defer cancel()
			start := time.Now()
			err := c.CheckHealth(checkCtx)
			results <- checkResult{
				name:       c.GetName(),
				err:        err,
				durationMs: time.Since(start).Milliseconds(),
			}
		}(checker)
	}

	for range hcm.checkers {
		r := <-results
		if r.err != nil {
			status.Status = "unhealthy"
			status.Checks[r.name] = map[string]interface{}{
				"status":      "unhealthy",
				"error":       r.err.Error(),
				"duration_ms": r.durationMs,
			}
		} else {
			status.Checks[r.name] = map[string]interface{}{
				"status":      "healthy",
				"duration_ms": r.durationMs,
			}
		}
	}

	return status
}

// DefaultHealthHandler 默认健康检查处理器
func DefaultHealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		status := &HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now().Unix(),
			Version:   "1.0.0",
		}

		c.JSON(http.StatusOK, status)
	}
}

// HealthHandlerWithManager 带管理器的健康检查处理器
func HealthHandlerWithManager(manager *HealthCheckManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := ContextFromGin(c)
		status := manager.CheckHealth(ctx)

		// 根据健康状态返回相应的HTTP状态码
		if status.Status == "healthy" {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
	}
}

// EnableHealthCheck 启用健康检查并注册默认健康检查路由。
func (s *Server) EnableHealthCheck() {
	s.config.EnableHealthCheck = true
	s.healthHandler = DefaultHealthHandler()
	s.registerHealthRoute()
	s.registerProbeRoutes()
}

// EnableHealthCheckWithManager 启用带管理器的健康检查并注册探针路由。
func (s *Server) EnableHealthCheckWithManager(manager *HealthCheckManager) {
	s.config.EnableHealthCheck = true
	s.healthHandler = HealthHandlerWithManager(manager)
	s.registerHealthRoute()
	s.registerProbeRoutes()
}

// SetHealthCheckPath 设置健康检查路径。
// 必须在健康检查路由注册或服务器启动前调用，否则返回错误。
func (s *Server) SetHealthCheckPath(path string) error {
	if s.healthRouteRegistered || s.readinessRouteRegistered || s.livenessRouteRegistered || s.server != nil || s.healthServer != nil {
		return fmt.Errorf("health check path must be configured before route registration or startup")
	}
	s.config.HealthCheckPath = path
	return nil
}

// GetHealthCheckPath 获取健康检查路径
func (s *Server) GetHealthCheckPath() string {
	return s.config.HealthCheckPath
}

// 内置健康检查器

// DatabaseHealthChecker 数据库健康检查器
type DatabaseHealthChecker struct {
	name string
	db   interface {
		Ping() error
	}
}

// NewDatabaseHealthChecker 创建数据库健康检查器
func NewDatabaseHealthChecker(name string, db interface {
	Ping() error
}) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		name: name,
		db:   db,
	}
}

// CheckHealth 检查数据库健康状态
func (dhc *DatabaseHealthChecker) CheckHealth(ctx context.Context) error {
	return dhc.db.Ping()
}

// GetName 获取检查器名称
func (dhc *DatabaseHealthChecker) GetName() string {
	return dhc.name
}

// HTTPHealthChecker HTTP服务健康检查器
type HTTPHealthChecker struct {
	name   string
	url    string
	client *http.Client
}

// NewHTTPHealthChecker 创建HTTP服务健康检查器
func NewHTTPHealthChecker(name, url string, timeout time.Duration) *HTTPHealthChecker {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &HTTPHealthChecker{
		name:   name,
		url:    url,
		client: &http.Client{Timeout: timeout},
	}
}

// CheckHealth 检查HTTP服务健康状态
func (hhc *HTTPHealthChecker) CheckHealth(ctx context.Context) (err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", hhc.url, nil)
	if err != nil {
		return err
	}

	resp, err := hhc.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("HTTP服务返回状态码: %d", resp.StatusCode)
}

// GetName 获取检查器名称
func (hhc *HTTPHealthChecker) GetName() string {
	return hhc.name
}

// CustomHealthChecker 自定义健康检查器
type CustomHealthChecker struct {
	name    string
	checker func(ctx context.Context) error
}

// NewCustomHealthChecker 创建自定义健康检查器
func NewCustomHealthChecker(name string, checker func(ctx context.Context) error) *CustomHealthChecker {
	return &CustomHealthChecker{
		name:    name,
		checker: checker,
	}
}

// CheckHealth 执行自定义健康检查
func (chc *CustomHealthChecker) CheckHealth(ctx context.Context) error {
	return chc.checker(ctx)
}

// GetName 获取检查器名称
func (chc *CustomHealthChecker) GetName() string {
	return chc.name
}
