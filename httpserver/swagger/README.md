# httpserver/swagger

`httpserver/swagger` 提供 Swagger UI 的 Gin 路由挂载能力，适合和 `httpserver` 一起使用。

这个子包只负责在运行时暴露 Swagger UI：

- 不运行 `swag init`
- 不生成 OpenAPI 文档
- 不接管业务鉴权
- 不自动推断业务接口文档

## 安装

```bash
go get github.com/tsopia/go-kit/httpserver/swagger
go install github.com/swaggo/swag/cmd/swag@latest
```

## 文档生成

推荐统一生成到 `internal/docs`：

```bash
swag init -g cmd/server/main.go -o internal/docs
```

主程序中显式导入生成包：

```go
import _ "your/module/internal/docs"
```

## 推荐接入方式

```go
import (
	_ "your/module/internal/docs"

	"github.com/tsopia/go-kit/httpserver"
	httpswagger "github.com/tsopia/go-kit/httpserver/swagger"
)

func main() {
	srv := httpserver.NewServer(nil)

	public := srv.Group("")
	protected := srv.Group("/api/v1")
	protected.Use(AuthMiddleware())

	httpswagger.Register(public, httpswagger.Config{})
}
```

默认行为：

- 路由默认挂载到 `/swagger/*any`
- 文档地址默认使用 `doc.json`
- Swagger 默认公开访问

如果需要自定义路径：

```go
httpswagger.Register(public, httpswagger.Config{
	Path:   "/docs/*any",
	DocURL: "/openapi/doc.json",
})
```

## 鉴权约定

推荐做法：

- Swagger 注册在公共路由组
- 业务鉴权只挂在受保护的路由组
- 不要把业务鉴权直接全局挂到 `srv.Use(...)`

如果历史项目已经使用全局鉴权，需要在中间件内对白名单路径放行：

- `/swagger/`
- `/swagger/index.html`
- `/swagger/doc.json`

## AI 与注释规范

推荐把 `swaggo` 注释写在 transport 层 typed handler 上，再通过 `httpserver.Handle...` 注册路由。

JSON 接口模板：

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
// @Failure 400 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 500 {object} map[string]string
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
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
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
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users [get]
func (m *UserModule) listUsers(ctx context.Context, req ListUsersRequest) (ListUsersResponse, error) {
	return m.user.List(ctx, req)
}
```
