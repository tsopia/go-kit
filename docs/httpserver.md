# HTTP服务器 (httpserver)

`httpserver` 是基于 Gin 的轻量 HTTP 传输层封装。它保留 Gin 直写能力，同时提供可选的 typed handler 契约，方便项目统一接口写法，也更适合 AI coding。

## 特性

- 基于 Gin 的路由与中间件能力
- `Start` / `Run` / `Serve` / `Shutdown` 生命周期管理
- 可选独立 `HealthCheckPort`
- 生命周期 hooks，便于接入任意日志与观测系统
- `RouteModule` 批量注册模块路由
- `Handle` / `HandleJSON` typed handler 契约
- Trace ID / Request ID / CORS 中间件

## 快速开始

### 方式一：直接使用 Gin 风格路由

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

	srv.POST("/users", func(c *gin.Context) {
		var req struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, gin.H{"name": req.Name})
	})

	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
```

### 方式二：推荐的 typed handler 写法

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

type AuthService struct{}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	return LoginResponse{Token: "token-123"}, nil
}

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

当前 `Config` 只包含传输层需要的字段：

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

默认值可通过 `httpserver.DefaultConfig()` 获取。

## 生命周期

### `NewServer`

```go
srv := httpserver.NewServer(cfg, opts...)
```

- 保留现有 Gin 路由注册方式
- 支持通过 `Option` 注入 hooks、模块等扩展能力

### `Start`

```go
if err := srv.Start(); err != nil {
	return fmt.Errorf("start server: %w", err)
}
```

- 非阻塞启动
- 同步返回监听失败错误
- 不再 `panic`

### `Serve`

```go
ln, err := net.Listen("tcp", "127.0.0.1:8080")
if err != nil {
	return fmt.Errorf("listen: %w", err)
}

if err := srv.Serve(ln); err != nil {
	return fmt.Errorf("serve: %w", err)
}
```

- 使用调用方提供的 `net.Listener`
- 会同时启动独立健康检查端口（如果配置了 `HealthCheckPort`）

### `Run`

```go
if err := srv.Run(); err != nil {
	return fmt.Errorf("run server: %w", err)
}
```

- 阻塞运行
- 内部自己创建 listener

### `Errors`

```go
select {
case err := <-srv.Errors():
	log.Printf("serve error: %v", err)
default:
}
```

- 暴露运行期服务错误
- 适合配合 `Start()` 使用

### `Shutdown`

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
	return fmt.Errorf("shutdown server: %w", err)
}
```

- 同时关闭主服务和独立健康检查服务

## 生命周期 hooks

`httpserver` 不依赖 `kit` 或任何具体日志库。需要日志、指标或审计时，用 hooks 接入你自己的实现。

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

## 健康检查

### 共享主端口

```go
srv := httpserver.NewServer(&httpserver.Config{
	Host:              "0.0.0.0",
	Port:              8080,
	EnableHealthCheck: true,
	HealthCheckPath:   "/healthz",
	HealthCheckPort:   0,
})
```

此时健康检查接口和业务接口共用主端口。

### 独立健康检查端口

```go
srv := httpserver.NewServer(&httpserver.Config{
	Host:              "0.0.0.0",
	Port:              8080,
	EnableHealthCheck: true,
	HealthCheckPath:   "/healthz",
	HealthCheckPort:   18080,
})
```

此时：

- 业务接口监听 `8080`
- 健康检查只监听 `18080`

### 带健康检查管理器

```go
manager := httpserver.NewHealthCheckManager("1.0.0")
manager.AddChecker(httpserver.NewDatabaseHealthChecker("mysql", mysqlDB))
manager.AddChecker(httpserver.NewHTTPHealthChecker("payment", "http://payment/healthz", 5*time.Second))

srv := httpserver.NewServer(&httpserver.Config{
	Host:              "0.0.0.0",
	Port:              8080,
	EnableHealthCheck: true,
	HealthCheckPath:   "/healthz",
	HealthCheckPort:   18080,
})

srv.EnableHealthCheckWithManager(manager)
```

## 模块化路由

如果你希望项目按业务模块组织路由，可以实现 `RouteModule`：

```go
type RouteModule interface {
	RegisterRoutes(r gin.IRoutes)
}
```

推荐由应用装配层做构造函数注入：

```go
db, err := database.New(dbConfig)
if err != nil {
	return fmt.Errorf("init db: %w", err)
}

userRepo := NewUserRepo(db)
authSvc := NewAuthService(userRepo)
userModule := NewUserModule(authSvc)

srv := httpserver.NewServer(cfg, httpserver.WithModules(userModule))
```

这样 `httpserver` 只负责路由与传输层，不负责依赖注入容器。

## Typed handler

### `HandleJSON`

适用于 JSON body 请求：

```go
r.POST("/login", httpserver.HandleJSON(login))
```

### `Handle`

适用于自定义 decoder：

```go
r.GET("/users", httpserver.Handle(
	listUsers,
	httpserver.WithDecoder(httpserver.DecodeQuery[ListUsersRequest]()),
))
```

### 可用 decoder

```go
httpserver.DecodeJSON[Req]()
httpserver.DecodeQuery[Req]()
httpserver.DecodeURI[Req]()
httpserver.ComposeDecoder(
	httpserver.DecodeURI[Req](),
	httpserver.DecodeQuery[Req](),
)
```

### 自动校验

如果请求类型实现以下任一方法，handler 会自动调用：

```go
Validate() error
Validate(context.Context) error
```

### 错误映射

默认行为：

- 解码失败 -> `400`
- 校验失败 -> `422`
- 业务错误 -> `500`

也可以自定义：

```go
httpserver.WithErrorMapper(func(err error) (int, any) {
	if errors.Is(err, ErrDuplicateUser) {
		return http.StatusConflict, gin.H{"error": "duplicate"}
	}

	return http.StatusInternalServerError, gin.H{"error": err.Error()}
})
```

### 成功状态码与自定义编码器

```go
httpserver.HandleJSON(
	createUser,
	httpserver.WithSuccessStatus(http.StatusCreated),
	httpserver.WithEncoder(func(c *gin.Context, status int, resp any) {
		c.JSON(status, gin.H{
			"data": resp,
		})
	}),
)
```

## 中间件与上下文

### Trace / Request ID

```go
srv.Use(
	httpserver.TraceIDMiddleware(),
	httpserver.RequestIDMiddleware(),
)
```

### CORS

```go
srv.Use(httpserver.CORSMiddleware())
```

### 从 Gin 提取请求上下文

```go
func handler(c *gin.Context) {
	ctx := httpserver.ContextFromGin(c)
	_ = ctx
}
```

## 推荐实践

- 需要灵活时，直接使用 Gin 风格路由
- 需要统一接口契约时，优先用 `Handle` / `HandleJSON`
- 依赖装配放到 `internal/app` 或类似的 composition root
- 模块通过 `RouteModule` 接入，不在 `httpserver` 里做容器
- 日志、指标、审计通过 hooks 接入，不强绑任何具体实现
