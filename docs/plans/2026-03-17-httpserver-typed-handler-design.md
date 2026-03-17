# HTTPServer Typed Handler Design

## 背景

当前 `httpserver` 的 typed handler 已具备基础能力：

- `Handle` / `HandleJSON`
- `DecodeJSON` / `DecodeQuery` / `DecodeURI`
- `ComposeDecoder`
- `Validate()` / `Validate(context.Context)` 约定
- `WithErrorMapper`

但在三个方面仍然偏弱：

1. 默认错误响应结构不稳定，只返回松散的 `{"error": "..."}`
2. 解码入口存在底层零件，但高层快捷入口不足
3. 校验只能返回普通 `error`，不方便表达字段级错误

## 目标

在不改变 `httpserver`“轻量 Gin 封装”定位的前提下，收敛 typed handler 的请求处理 pipeline，统一默认错误模型，补齐常用快捷入口，并增强校验表达能力。

## 非目标

- 不引入重量级校验框架
- 不自动猜测请求来源
- 不直接依赖 `errors` 包
- 不引入新的 endpoint DSL 或 builder 风格 API

## 设计

### 1. 请求处理 pipeline

typed handler 内部统一为以下顺序：

1. Decode
2. Validate
3. Invoke
4. Encode success
5. Render error

### 2. 错误模型

新增稳定默认错误响应：

```go
type ErrorResponse struct {
    Code    string         `json:"code"`
    Message string         `json:"message"`
    Details map[string]any `json:"details,omitempty"`
}
```

默认错误码使用稳定字符串，而不是在基础库里引入数字业务码体系。

默认映射：

- decode 错误：`400 invalid_request`
- validate 错误：`422 validation_failed`
- 未知错误：`500 internal_error`

### 3. HTTP 业务错误接口

新增标准业务错误接口，允许业务逻辑显式控制 HTTP 语义：

```go
type HTTPError interface {
    error
    StatusCode() int
    ErrorCode() string
    ErrorMessage() string
    ErrorDetails() map[string]any
}
```

渲染顺序为：

1. transport/request 错误
2. `HTTPError`
3. 自定义 `ErrorMapper`
4. 默认 `500`

### 4. 校验模型

保留现有：

- `Validate() error`
- `Validate(context.Context) error`

新增：

```go
type ValidationField struct {
    Field   string `json:"field"`
    Code    string `json:"code,omitempty"`
    Message string `json:"message"`
}

type ValidationError struct {
    Message string
    Fields  []ValidationField
}
```

并增加显式 validator 链：

```go
type RequestValidator[Req any] func(context.Context, Req) error
```

通过 `WithValidators(...)` 追加额外校验器，既支持请求对象自校验，也支持装配层上下文校验。

### 5. 解码与快捷入口

保留底层能力：

- `Handle`
- `HandleJSON`
- `DecodeJSON`
- `DecodeQuery`
- `DecodeURI`
- `ComposeDecoder`

新增高层快捷入口：

- `HandleQuery`
- `HandleURI`
- `HandleQueryURI`

它们只是对已有 decoder 组合的稳定快捷封装，不引入新的黑盒行为。

## 兼容策略

- 保留现有 `Handle`、`HandleJSON`、`WithDecoder`、`WithErrorMapper`
- `WithEncoder`、`WithSuccessStatus` 行为不变
- 默认错误响应从旧的 `{"error": "..."}` 收敛到 `ErrorResponse`
- 业务项目如果已有错误体系，可继续通过 `WithErrorMapper` 接入

## 测试策略

覆盖以下关键路径：

- 默认 decode/validate/internal 错误响应
- `ValidationError` 结构化字段输出
- `HTTPError` 映射优先级
- `WithValidators(...)` 与请求自校验组合
- `HandleQuery` / `HandleURI` / `HandleQueryURI`
- 旧入口兼容
