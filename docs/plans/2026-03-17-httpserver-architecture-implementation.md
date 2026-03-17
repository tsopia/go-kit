# httpserver Architecture Evolution Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `httpserver` 从“单文件轻封装”演进为“薄 core + 可选 middleware/observability/preset”的分层结构，同时保持现有 Gin-first API 和向后兼容。

**Architecture:** 先收敛 core 内部结构和生命周期模型，再引入 readiness/liveness 和 draining 语义，随后把通用中间件与观测集成拆到子包，并通过兼容包装器和官方 preset 保持老项目平滑迁移。所有新增子包都遵循仓库要求：先更新 `.ai/capabilities.yaml`，再创建包目录与代码，最后更新 `AGENTS.md`。

**Tech Stack:** Go 1.24, Gin 1.11, standard library `net/http`, table-driven tests, `go test`, `golangci-lint`

---

**References:**
- Read [`docs/plans/2026-03-17-httpserver-architecture-design.md`](docs/plans/2026-03-17-httpserver-architecture-design.md) before touching code.
- Check current implementation in `httpserver/server.go`, `httpserver/options.go`, `httpserver/handler.go`, `httpserver/README.md`.
- Follow project requirements in `AGENTS.md` and `.ai/capabilities.yaml`.
- Use `@superpowers:test-driven-development` for Tasks 1-5.
- Use `@superpowers:verification-before-completion` before claiming the work is complete.

### Task 1: 收敛 core 文件布局与配置校验

**Files:**
- Create: `httpserver/config.go`
- Create: `httpserver/lifecycle.go`
- Create: `httpserver/errors.go`
- Modify: `httpserver/server.go`
- Modify: `httpserver/options.go`
- Modify: `httpserver/README.md`
- Test: `httpserver/server_test.go`
- Test: `httpserver/lifecycle_test.go`

**Step 1: Write the failing tests**

新增 table-driven tests，锁定配置默认化和校验行为：

```go
func TestConfigNormalizeAndValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Host:              "127.0.0.1",
				Port:              8080,
				ReadTimeout:       time.Second,
				ReadHeaderTimeout: time.Second,
				WriteTimeout:      time.Second,
				IdleTimeout:       time.Second,
				ShutdownTimeout:   time.Second,
				DrainTimeout:      time.Second,
				HealthCheckPath:   "/health",
				ReadinessPath:     "/readyz",
				LivenessPath:      "/livez",
			},
		},
		{
			name: "invalid port",
			cfg: Config{
				Host: "127.0.0.1",
				Port: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			cfg.Normalize()
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestConfigNormalizeAndValidate' -v`

Expected: FAIL because `ReadHeaderTimeout`、`DrainTimeout`、`ReadinessPath`、`LivenessPath`、`Normalize()`、`Validate()` do not exist yet.

**Step 3: Write minimal implementation**

实现 `config.go`，只做配置字段迁移和校验，不改变现有外部入口：

```go
func (c *Config) Normalize() {
	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.ReadHeaderTimeout <= 0 {
		c.ReadHeaderTimeout = 5 * time.Second
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

func (c *Config) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("validate config: invalid port %d", c.Port)
	}
	if c.HealthCheckPort < 0 || c.HealthCheckPort > 65535 {
		return fmt.Errorf("validate config: invalid health check port %d", c.HealthCheckPort)
	}
	if c.HealthCheckPort != 0 && c.HealthCheckPort == c.Port {
		return fmt.Errorf("validate config: health check port conflicts with port")
	}
	return nil
}
```

同时把重复生命周期辅助函数从 `server.go` 移到 `lifecycle.go`，保持 `NewServer(...)`、`Engine()`、`Use()`、`Group()` 现有签名不变。

**Step 4: Run tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestConfigNormalizeAndValidate|TestDefaultConfig' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/config.go httpserver/lifecycle.go httpserver/errors.go httpserver/server.go httpserver/options.go httpserver/README.md httpserver/server_test.go httpserver/lifecycle_test.go
git commit -m "refactor(httpserver): normalize core config and lifecycle structure"
```

### Task 2: 引入状态机、readiness/liveness 与 draining

**Files:**
- Create: `httpserver/readiness.go`
- Modify: `httpserver/server.go`
- Modify: `httpserver/options.go`
- Modify: `httpserver/README.md`
- Test: `httpserver/health_server_test.go`
- Test: `httpserver/lifecycle_test.go`

**Step 1: Write the failing tests**

增加 table-driven tests，覆盖状态切换和健康端点：

```go
func TestReadinessAndLivenessEndpoints(t *testing.T) {
	tests := []struct {
		name            string
		manualReady     bool
		action          func(*Server)
		wantReadyStatus int
		wantLiveStatus  int
	}{
		{
			name:            "auto ready server returns ready",
			manualReady:     false,
			action:          func(*Server) {},
			wantReadyStatus: http.StatusOK,
			wantLiveStatus:  http.StatusOK,
		},
		{
			name:      "draining server becomes unready",
			manualReady: false,
			action: func(s *Server) {
				s.MarkDraining()
			},
			wantReadyStatus: http.StatusServiceUnavailable,
			wantLiveStatus:  http.StatusOK,
		},
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestReadinessAndLivenessEndpoints' -v`

Expected: FAIL because `State`、`MarkDraining()`、`ReadinessPath`、`LivenessPath`、`WithManualReadiness()` and handlers do not exist yet.

**Step 3: Write minimal implementation**

实现集中状态切换和 endpoint：

```go
type State string

const (
	StateNew      State = "new"
	StateStarting State = "starting"
	StateReady    State = "ready"
	StateDraining State = "draining"
	StateStopping State = "stopping"
	StateStopped  State = "stopped"
	StateFailed   State = "failed"
)

func (s *Server) MarkReady() { ... }
func (s *Server) MarkDraining() { ... }
func (s *Server) State() State { ... }
```

实现规则：

- `Start` / `Run` / `RunTLS` / `Serve` 共用一套内部启动流程
- `DefaultConfig()` 保持自动 ready，`WithManualReadiness()` 用于关闭自动 ready
- `WaitForShutdown()` 先 `MarkDraining()`，等待 `DrainTimeout`，再调用 `Shutdown(ctx)`
- `/health` 暂时保持与 readiness 一致，确保兼容

**Step 4: Run tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestReadinessAndLivenessEndpoints|TestHealthCheckServingModes' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/readiness.go httpserver/server.go httpserver/options.go httpserver/README.md httpserver/health_server_test.go httpserver/lifecycle_test.go
git commit -m "feat(httpserver): add readiness lifecycle state machine"
```

### Task 3: 抽出 `httpserver/middleware` 并保留兼容包装器

**Files:**
- Modify: `.ai/capabilities.yaml`
- Modify: `AGENTS.md`
- Create: `httpserver/middleware/doc.go`
- Create: `httpserver/middleware/README.md`
- Create: `httpserver/middleware/recovery.go`
- Create: `httpserver/middleware/timeout.go`
- Create: `httpserver/middleware/trace_id.go`
- Create: `httpserver/middleware/request_id.go`
- Create: `httpserver/middleware/cors.go`
- Create: `httpserver/middleware/security_headers.go`
- Create: `httpserver/middleware/max_body_size.go`
- Modify: `httpserver/server.go`
- Modify: `httpserver/README.md`
- Test: `httpserver/middleware/recovery_test.go`
- Test: `httpserver/middleware/timeout_test.go`
- Test: `httpserver/server_test.go`

**Step 1: Update capability metadata before code**

先更新 `.ai/capabilities.yaml`，新增 `middleware` 能力条目；再更新 `AGENTS.md` 的“库能力速查表”。

建议新增能力片段：

```yaml
- name: middleware
  description: HTTP 中间件集合（Recovery、Timeout、TraceID、CORS、安全头等）
  import: github.com/tsopia/go-kit/httpserver/middleware
  scenarios:
    - name: 挂载恢复中间件
      snippet: srv.Use(middleware.Recovery())
    - name: 挂载超时中间件
      snippet: srv.Use(middleware.Timeout(2 * time.Second))
  dependencies: [httpserver]
```

**Step 2: Write the failing tests**

先写中间件包测试，再写主包兼容测试：

```go
func TestRecovery(t *testing.T) { ... }
func TestTimeout(t *testing.T) { ... }
func TestLegacyTraceIDMiddlewareDelegatesToMiddlewarePackage(t *testing.T) { ... }
```

**Step 3: Run tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./httpserver ./httpserver/middleware -run 'TestRecovery|TestTimeout|TestLegacyTraceIDMiddlewareDelegatesToMiddlewarePackage' -v`

Expected: FAIL because package and wrappers do not exist yet.

**Step 4: Write minimal implementation**

实现 `httpserver/middleware` 包，并让主包兼容方法转调：

```go
func TraceIDMiddleware() gin.HandlerFunc {
	return middleware.TraceID()
}

func RequestIDMiddleware() gin.HandlerFunc {
	return middleware.RequestID()
}

func CORSMiddleware() gin.HandlerFunc {
	return middleware.CORS(middleware.CORSConfig{})
}
```

实现规则：

- 新包默认只输出 `gin.HandlerFunc`
- `Recovery` 使用 `defer` + `recover` 返回 500，禁止吞掉 panic 不留痕
- `Timeout` 基于 `context.WithTimeout`，只处理 handler 超时响应，不引入 goroutine 泄漏
- 旧 API 暂时保留，README 迁移说明优先推荐新包

**Step 5: Run tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver ./httpserver/middleware -v`

Expected: PASS

**Step 6: Commit**

```bash
git add .ai/capabilities.yaml AGENTS.md httpserver/middleware httpserver/server.go httpserver/README.md httpserver/server_test.go
git commit -m "feat(httpserver): extract reusable middleware package"
```

### Task 4: 新增 `httpserver/observability` 子包

**Files:**
- Modify: `.ai/capabilities.yaml`
- Modify: `AGENTS.md`
- Create: `httpserver/observability/prometheus/doc.go`
- Create: `httpserver/observability/prometheus/README.md`
- Create: `httpserver/observability/prometheus/middleware.go`
- Create: `httpserver/observability/prometheus/register.go`
- Create: `httpserver/observability/otel/doc.go`
- Create: `httpserver/observability/otel/README.md`
- Create: `httpserver/observability/otel/middleware.go`
- Modify: `httpserver/README.md`
- Test: `httpserver/observability/prometheus/register_test.go`
- Test: `httpserver/observability/otel/middleware_test.go`

**Step 1: Update capability metadata before code**

在 `.ai/capabilities.yaml` 中新增 `httpserver-prometheus` 和 `httpserver-otel` 条目，并更新 `AGENTS.md`：

```yaml
- name: httpserver-prometheus
  description: Prometheus 指标中间件与 /metrics 路由注册
  import: github.com/tsopia/go-kit/httpserver/observability/prometheus
  dependencies: [httpserver, middleware]

- name: httpserver-otel
  description: OpenTelemetry HTTP tracing 中间件
  import: github.com/tsopia/go-kit/httpserver/observability/otel
  dependencies: [httpserver, middleware]
```

**Step 2: Write the failing tests**

示例测试：

```go
func TestRegisterMetricsRoute(t *testing.T) { ... }
func TestTracingMiddlewarePropagatesSpanContext(t *testing.T) { ... }
```

**Step 3: Run tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/observability/... -v`

Expected: FAIL because packages do not exist yet.

**Step 4: Write minimal implementation**

实现规则：

- Prometheus 中间件只做请求计数、延迟、状态码标签
- `/metrics` 必须显式通过 `Register(...)` 注册
- OTel 中间件只负责 span 创建和上下文传播，不绑定 exporter 初始化
- README 说明“provider/meter/tracer 由应用装配层提供”

**Step 5: Run tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/observability/... -v`

Expected: PASS

**Step 6: Commit**

```bash
git add .ai/capabilities.yaml AGENTS.md httpserver/observability httpserver/README.md
git commit -m "feat(httpserver): add observability extension packages"
```

### Task 5: 新增 `httpserver/preset` 并提供官方装配

**Files:**
- Modify: `.ai/capabilities.yaml`
- Modify: `AGENTS.md`
- Create: `httpserver/preset/doc.go`
- Create: `httpserver/preset/README.md`
- Create: `httpserver/preset/production.go`
- Create: `httpserver/preset/development.go`
- Modify: `httpserver/README.md`
- Test: `httpserver/preset/production_test.go`

**Step 1: Update capability metadata before code**

新增 `preset` 能力条目：

```yaml
- name: preset
  description: 官方推荐的 HTTP server 默认装配
  import: github.com/tsopia/go-kit/httpserver/preset
  scenarios:
    - name: 创建生产默认服务器
      snippet: srv := preset.NewProductionServer(cfg)
  dependencies: [httpserver, middleware]
```

并同步更新 `AGENTS.md` 的“库能力速查表”。

**Step 2: Write the failing tests**

示例测试：

```go
func TestNewProductionServerAppliesExpectedMiddlewareOrder(t *testing.T) { ... }
```

**Step 3: Run tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/preset -v`

Expected: FAIL because package does not exist yet.

**Step 4: Write minimal implementation**

实现规则：

- `preset` 只做装配，不重新实现中间件逻辑
- 生产默认链路建议顺序：`Recovery -> RequestID -> TraceID -> Timeout -> SecurityHeaders`
- 不在 `preset` 中偷偷注册 metrics、swagger、pprof
- 为使用方保留继续 `Use(...)` 和 `Group(...)` 的空间

**Step 5: Run tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/preset -v`

Expected: PASS

**Step 6: Commit**

```bash
git add .ai/capabilities.yaml AGENTS.md httpserver/preset httpserver/README.md
git commit -m "feat(httpserver): add official preset assembly"
```

### Task 6: 完成文档、验证与迁移说明

**Files:**
- Modify: `httpserver/doc.go`
- Modify: `httpserver/README.md`
- Modify: `AGENTS.md`
- Modify: `.ai/capabilities.yaml`
- Modify: `docs/plans/2026-03-17-httpserver-architecture-design.md`

**Step 1: Update user-facing docs**

补齐：

- `core / middleware / observability / preset` 的边界说明
- 从旧 `httpserver.TraceIDMiddleware()` 迁移到 `middleware.TraceID()` 的示例
- readiness/liveness 新端点使用说明
- preset 的推荐装配示例

**Step 2: Run full verification**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/... -v`

Expected: PASS

Run: `GOCACHE=/tmp/go-build go test ./...`

Expected: PASS

Run: `golangci-lint run`

Expected: PASS

Run: `gokit list`

Expected: 新增能力条目可见，且描述与 `.ai/capabilities.yaml` 一致。

**Step 3: Commit**

```bash
git add httpserver/doc.go httpserver/README.md AGENTS.md .ai/capabilities.yaml docs/plans/2026-03-17-httpserver-architecture-design.md
git commit -m "docs(httpserver): document architecture layering and migration"
```

## Notes

- 每一阶段都必须保持 `NewServer(...)`、`Engine()`、`Use()`、`Group()`、`RegisterModules()` 的兼容。
- 所有新增测试必须使用 table-driven tests。
- 如果某个子包在实现阶段被证明暂时不值得引入，应先回到设计文档修订，而不是跳过 `.ai/capabilities.yaml` / `AGENTS.md` 约束直接写代码。
