# HTTPServer 包

`httpserver` 是基于 Gin 的 HTTP 传输层封装，目标是保留 Gin 的灵活性，同时提供更稳定的服务启动、健康检查和 typed handler 能力，方便业务项目统一接入，也方便 AI coding 理解和生成代码。

## 适用场景

- 需要一个轻量 HTTP 服务启动器
- 需要保留 Gin 原生路由写法
- 需要模块化注册路由
- 需要统一的 JSON / Query / URI typed handler 写法
- 需要把日志、指标、审计接入到自己的实现，而不是绑定框架内置日志

## 安装

```go
go get github.com/tsopia/go-kit/httpserver
```

## 核心能力

- `NewServer` 创建 HTTP 服务
- `Start` / `Serve` / `Run` / `Shutdown` 生命周期管理
- `HealthCheckPort` 支持独立健康检查端口
- `Hooks` 支持生命周期事件观测
- `RouteModule` 支持按模块注册路由
- `Handle` / `HandleJSON` 支持 typed handler
- `DecodeJSON` / `DecodeQuery` / `DecodeURI` / `ComposeDecoder` 支持请求解码组合
- `httpserver/middleware` 子包支持通用 Recovery、Timeout、TraceID、RequestID、AccessLog、CORS 等中间件
- `httpserver/observability/prometheus` 与 `httpserver/observability/otel` 子包支持指标和 tracing 集成
- `httpserver/preset` 子包支持官方推荐的默认装配
- `httpserver/integration/errorsx` 子包支持 `errors` 包到 typed handler 的统一错误映射
- `httpserver/swagger` 子包支持 Swagger UI 路由挂载

## 快速开始

### 直接使用 Gin 风格路由

```go
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/httpserver"
)

func main() {
	srv := httpserver.NewServer(&httpserver.Config{
		Host: "0.0.0.0",
		Port: 8080,
	})

	srv.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
```

### 推荐的 typed handler 写法

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/httpserver"
)

type LoginRequest struct {
	Email string `json:"email"`
}

func (r LoginRequest) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	return nil
}

type LoginResponse struct {
	Token string `json:"token"`
}

type AuthService struct{}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	return LoginResponse{Token: "token-123"}, nil
}

type UserModule struct {
	auth *AuthService
}

func NewUserModule(auth *AuthService) *UserModule {
	return &UserModule{auth: auth}
}

func (m *UserModule) RegisterRoutes(r gin.IRoutes) {
	r.POST("/login", httpserver.HandleJSON(
		m.Login,
		httpserver.WithSuccessStatus(http.StatusOK),
	))
}

func (m *UserModule) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	return m.auth.Login(ctx, req)
}

func main() {
	srv := httpserver.NewServer(nil, httpserver.WithModules(
		NewUserModule(&AuthService{}),
	))

	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
```

## 配置

```go
type Config struct {
	Host            string
	Port            int
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ShutdownTimeout time.Duration
	DrainTimeout      time.Duration

	EnableHealthCheck bool
	HealthCheckPath   string
	ReadinessPath     string
	LivenessPath      string
	HealthCheckPort   int
}
```

默认值可以通过 `httpserver.DefaultConfig()` 获取。
如果需要对传入配置补默认值或做启动前校验，可使用 `(*Config).Normalize()` 和 `(*Config).Validate()`。

## 生命周期与 hooks

如果你需要日志、指标或审计，使用 `Hooks` 接入你自己的实现：

```go
srv := httpserver.NewServer(cfg, httpserver.WithHooks(httpserver.Hooks{
	OnStarted: func(ctx context.Context, event httpserver.LifecycleEvent) {
		log.Printf("server started addr=%s health=%s", event.Addr, event.HealthAddr)
	},
	OnServeError: func(ctx context.Context, event httpserver.LifecycleEvent) {
		log.Printf("server error addr=%s err=%v", event.Addr, event.Err)
	},
}))
```

`httpserver` 不依赖 `kit` 或任何具体日志包，日志方案由使用方决定。

## 通用中间件

如果你需要可复用的 Recovery、Timeout、TraceID、RequestID、AccessLog、CORS 或安全响应头，优先使用 `httpserver/middleware` 子包，而不是继续向 `Server` 主类型增加能力。

```go
srv := httpserver.NewServer(cfg)
srv.Use(middleware.AccessLog())
srv.Use(middleware.Recovery())
srv.Use(middleware.Timeout(2 * time.Second))
srv.Use(middleware.TraceID())
```

如果需要调试请求体和响应体，可以显式开启 payload capture：

```go
srv.Use(middleware.AccessLog(middleware.AccessLogConfig{
	CapturePayload: true,
	MaxBodyBytes:   8 << 10,
	Multipart: middleware.MultipartConfig{
		Mode: middleware.MultipartFormFieldsOnly,
	},
}))
```

## 可观测性扩展

如果你需要指标和 tracing，请使用独立的 observability 子包，而不是把这些能力塞进 `httpserver` core。

```go
public := srv.Group("")
srv.Use(prometheus.Middleware())
srv.Use(otel.Middleware(otel.Config{
	TracerName: "user-service",
}))
prometheus.Register(public, prometheus.Config{
	Path: "/metrics",
})
```

## 官方默认装配

如果你希望快速获得一套一致的 HTTP 默认链路，可以直接使用 `httpserver/preset`：

```go
srv := preset.NewProductionServer(cfg)
```

## 健康检查

共享主端口：

```go
srv := httpserver.NewServer(&httpserver.Config{
	Host:              "0.0.0.0",
	Port:              8080,
	EnableHealthCheck: true,
	HealthCheckPath:   "/healthz",
})
```

独立健康检查端口：

```go
srv := httpserver.NewServer(&httpserver.Config{
	Host:              "0.0.0.0",
	Port:              8080,
	EnableHealthCheck: true,
	HealthCheckPath:   "/healthz",
	HealthCheckPort:   18080,
})
```

默认情况下，服务器启动后会自动进入 ready 状态，同时暴露：

- `HealthCheckPath`，默认 `/health`
- `ReadinessPath`，默认 `/readyz`
- `LivenessPath`，默认 `/livez`

如果你需要在预热完成后再接流量，可以通过 `httpserver.WithManualReadiness()` 关闭自动 ready，然后在合适时机调用 `srv.MarkReady()`。

## 模块化路由

如果希望按业务模块组织路由，实现 `RouteModule` 即可：

```go
type RouteModule interface {
	RegisterRoutes(r gin.IRoutes)
}
```

推荐在应用装配层完成依赖注入，再把模块交给 `httpserver`：

```go
userRepo := NewUserRepo(db)
authSvc := NewAuthService(userRepo)
userModule := NewUserModule(authSvc)

srv := httpserver.NewServer(cfg, httpserver.WithModules(userModule))
```

## Typed handler

常用快捷入口：

```go
r.POST("/login", httpserver.HandleJSON(login))
r.GET("/users", httpserver.HandleQuery(listUsers))
r.GET("/users/:id", httpserver.HandleURI(getUser))
r.GET("/users/:id", httpserver.HandleQueryURI(getUserDetail))
```

如果你需要更细粒度的解码控制，仍然可以保留底层 decoder 组合：

```go
r.GET("/users", httpserver.Handle(
	listUsers,
	httpserver.WithDecoder(httpserver.DecodeQuery[ListUsersRequest]()),
))

httpserver.ComposeDecoder(
	httpserver.DecodeURI[Req](),
	httpserver.DecodeQuery[Req](),
)
```

请求对象如果实现了以下任一方法，会自动执行校验：

```go
Validate() error
Validate(context.Context) error
```

如果还需要补充装配层校验，可以追加显式 validator：

```go
r.POST("/register", httpserver.HandleJSON(
	register,
	httpserver.WithValidators(func(ctx context.Context, req RegisterRequest) error {
		if strings.HasSuffix(req.Email, "@company.com") {
			return nil
		}

		return &httpserver.ValidationError{
			Message: "request validation failed",
			Fields: []httpserver.ValidationField{
				{
					Field:   "email",
					Message: "must use company email",
				},
			},
		}
	}),
))
```

默认错误响应统一为：

```json
{
  "code": "validation_failed",
  "message": "email is required"
}
```

字段级校验会额外返回 `details.fields`。如果业务错误需要自己控制 HTTP 状态码和错误体语义，可以返回实现了 `HTTPError` 接口的错误，或者继续通过 `WithErrorMapper(...)` 做项目级映射。

如果团队统一使用 `github.com/tsopia/go-kit/errors` 作为业务错误出口，推荐直接使用 `httpserver/integration/errorsx`：

```go
import (
	"github.com/tsopia/go-kit/httpserver"
	"github.com/tsopia/go-kit/httpserver/integration/errorsx"
)

r.POST("/users", httpserver.HandleJSON(
	createUser,
	httpserver.WithErrorMapper(errorsx.Mapper()),
))
```

默认会把 `errors` 包中的业务错误映射成：

```json
{
  "code": 2002,
  "name": "INVALID_PARAM",
  "message": "email is required"
}
```

## Swagger 集成

推荐通过 `httpserver/swagger` 子包挂载 Swagger UI，而不是把文档路由和业务鉴权揉在一起。

安装依赖：

```bash
go get github.com/tsopia/go-kit/httpserver/swagger
go install github.com/swaggo/swag/cmd/swag@latest
```

生成文档：

```bash
swag init -g cmd/server/main.go -o internal/docs
```

推荐路由组织：

```go
import (
	_ "your/module/internal/docs"

	"github.com/tsopia/go-kit/httpserver"
	httpswagger "github.com/tsopia/go-kit/httpserver/swagger"
)

srv := httpserver.NewServer(nil)

public := srv.Group("")
protected := srv.Group("/api/v1")
protected.Use(AuthMiddleware())

httpswagger.Register(public, httpswagger.Config{})
```

推荐约定：

- Swagger 默认公开访问，注册在 `public` 路由组
- 业务鉴权只挂到受保护的 `Group()`，不要直接全局 `srv.Use(AuthMiddleware())`
- 历史项目如果已经使用全局鉴权，中间件需要对白名单路径 `/swagger/` 放行
- `swaggo` 注释写在 transport 层 typed handler 上，不写在 service 层

### Swagger 注释模板

推荐把 `swaggo` 注释写在被 `Handle...` 包装的 handler 上：

```go
type LoginRequest struct {
	Email string `json:"email" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// login godoc
// @Summary 用户登录
// @Description 使用邮箱登录并返回访问令牌
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 422 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /auth/login [post]
func (m *UserModule) login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	return m.auth.Login(ctx, req)
}

r.POST("/auth/login", httpserver.HandleJSON(m.login))
```

鉴权接口模板：

```go
// profile godoc
// @Summary 获取当前用户信息
// @Tags user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ProfileResponse
// @Failure 401 {object} httpserver.ErrorResponse
// @Failure 403 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /users/me [get]
func (m *UserModule) profile(ctx context.Context, req ProfileRequest) (ProfileResponse, error) {
	return m.user.Profile(ctx, req)
}
```

Query 接口模板：

```go
type ListUsersRequest struct {
	Page int `form:"page"`
	Size int `form:"size"`
}

// listUsers godoc
// @Summary 用户列表
// @Tags user
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Security BearerAuth
// @Success 200 {object} ListUsersResponse
// @Failure 400 {object} httpserver.ErrorResponse
// @Failure 500 {object} httpserver.ErrorResponse
// @Router /users [get]
func (m *UserModule) listUsers(ctx context.Context, req ListUsersRequest) (ListUsersResponse, error) {
	return m.user.List(ctx, req)
}
```

## 推荐实践

- 需要灵活性时，直接使用 Gin 原生 handler
- 需要统一接口契约时，优先使用 `HandleJSON`、`HandleQuery`、`HandleURI`、`HandleQueryURI`
- 模块依赖在业务装配层注入，不在 `httpserver` 内做容器
- 日志、指标、审计统一通过 `Hooks` 接入
- Swagger 建议通过 `httpserver/swagger` 注册在公共路由组

## 更多说明

更完整的设计说明与用法参考见：

- `docs/httpserver.md`
- `examples/http-server`
- `examples/http-server-improved`
- `examples/http-health-check`
