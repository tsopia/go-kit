# errorsx

`errorsx` 是 `errors` 包与 `httpserver` typed handler 之间的桥接层。

它的职责很单一：

- 把 `errors.Code(err)`、`errors.Name(err)`、`errors.HTTPCode(err)` 映射为稳定 HTTP 响应
- 提供可直接挂到 `httpserver.WithErrorMapper(...)` 的适配器

## 安装

```go
go get github.com/tsopia/go-kit/httpserver/integration/errorsx
```

## 使用

```go
srv.POST("/users", httpserver.HandleJSON(
    createUser,
    httpserver.WithErrorMapper(errorsx.Mapper()),
))
```

默认响应示例：

```json
{
  "code": 2002,
  "name": "INVALID_PARAM",
  "message": "email is required"
}
```

普通 `error` 会回退到内部错误响应，不直接暴露底层错误消息。
