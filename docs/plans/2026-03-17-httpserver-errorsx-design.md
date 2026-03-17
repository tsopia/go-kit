# HTTPServer ErrorsX Integration Design

## 背景

团队计划把 `errors` 包作为统一业务错误出口，而 `httpserver` 的 typed handler 已经具备稳定的默认错误模型与 `WithErrorMapper(...)` 扩展点。

当前缺口在于：

- 业务项目需要重复编写 `WithErrorMapper(...)`
- 相同 `errors` 定义可能在不同服务里映射出不同的 HTTP 状态和响应体
- `httpserver` core 不应直接依赖 `errors`

## 目标

新增一个独立桥接包，把 `errors` 包的业务错误映射成稳定的 HTTP 响应，供 `httpserver` typed handler 直接复用。

## 非目标

- 不修改 `httpserver` core 对 `errors` 的依赖边界
- 不在 `preset` 中默认自动启用
- 不扩展 `errors` 包本身的数据模型

## 包位置

建议新增：

```text
httpserver/integration/errorsx/
```

这是一个 integration 层能力，而不是 `httpserver` core 或 `errors` core 的一部分。

## 响应结构

桥接包使用独立响应结构，而不是复用 `httpserver.ErrorResponse`：

```go
type ResponseBody struct {
    Code    int            `json:"code"`
    Name    string         `json:"name"`
    Message string         `json:"message"`
    Details map[string]any `json:"details,omitempty"`
}
```

原因：

- `httpserver.ErrorResponse.Code` 当前是字符串语义
- `errors` 包已经有明确的数字业务码 `Code(err)`
- 这层桥接需要优先表达团队统一业务错误码

## 映射规则

### 命中 `errors` coded error

返回：

- HTTP 状态：`errors.HTTPCode(err)`
- body.code：`errors.Code(err)`
- body.name：`errors.Name(err)`
- body.message：`err.Error()`

### 未命中

返回内部错误兜底，不泄露原始内部错误：

- HTTP 状态：`errors.HTTPCode(errors.Internal.New(...))` 语义等价的 `500`
- body.code：`errors.Internal.Code`
- body.name：`errors.Internal.Name`
- body.message：`"internal server error"`

第一版不主动填充 `details`，只保留扩展位。

## 对外 API

```go
func Response(err error) (int, ResponseBody)
func Mapper() httpserver.ErrorMapper
```

用法：

```go
srv.POST("/users", httpserver.HandleJSON(
    createUser,
    httpserver.WithErrorMapper(errorsx.Mapper()),
))
```

## 文档与能力清单

按仓库规范，同步更新：

- `.ai/capabilities.yaml`
- `AGENTS.md`
- `httpserver/README.md`
- `httpserver/integration/errorsx/doc.go`
- `httpserver/integration/errorsx/README.md`

## 测试策略

覆盖：

- `errors.InvalidParam` 映射为 `400 + code/name/message`
- `errors.NotFound` 映射为 `404`
- 普通 `error` 回退为内部错误，不透传底层 message
- `Mapper()` 可直接挂到 `httpserver.HandleJSON(...)`
