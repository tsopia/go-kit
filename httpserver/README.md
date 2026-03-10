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
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxHeaderBytes  int
	ShutdownTimeout time.Duration

	EnableHealthCheck bool
	HealthCheckPath   string
	HealthCheckPort   int
}
```

默认值可以通过 `httpserver.DefaultConfig()` 获取。

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

JSON body：

```go
r.POST("/login", httpserver.HandleJSON(login))
```

Query / URI / 自定义 decoder：

```go
r.GET("/users", httpserver.Handle(
	listUsers,
	httpserver.WithDecoder(httpserver.DecodeQuery[ListUsersRequest]()),
))
```

解码器可以组合：

```go
httpserver.ComposeDecoder(
	httpserver.DecodeURI[Req](),
	httpserver.DecodeQuery[Req](),
)
```

如果请求结构实现了以下任一方法，会自动执行校验：

```go
Validate() error
Validate(context.Context) error
```

## 推荐实践

- 需要灵活性时，直接使用 Gin 原生 handler
- 需要统一接口契约时，优先使用 `Handle` / `HandleJSON`
- 模块依赖在业务装配层注入，不在 `httpserver` 内做容器
- 日志、指标、审计统一通过 `Hooks` 接入

## 更多说明

更完整的设计说明与用法参考见：

- `docs/httpserver.md`
- `examples/http-server`
- `examples/http-server-improved`
- `examples/http-health-check`
