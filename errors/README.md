# errors

Go-Kit 错误处理包，提供统一的错误码定义、层级匹配和错误实例化能力。

## 特性

- **显式/自动注册**：手动指定 code，或自动分配（从 4000 开始）
- **层级匹配**：通过 `Class` 字段建立错误层次，支持 `errors.Is` 向上匹配
- **HTTP 状态码**：错误定义可绑定 HTTP 状态码，自动继承父类
- **标准库兼容**：`errors.Is`、`errors.As`、`Unwrap` 完全兼容
- **文档生成**：支持导出错误码列表生成文档

## 安装

```bash
go get github.com/tsopia/go-kit/errors
```

## 快速开始

```go
package main

import (
    "errors"
    "fmt"

    kiterr "github.com/tsopia/go-kit/errors"
)

func main() {
    // 1. 注册错误（显式 code）
    ErrUserNotFound := kiterr.Register(3001, "USER_NOT_FOUND").WithHTTP(404)

    // 2. 创建错误
    err := ErrUserNotFound.New("用户 123 不存在")

    // 3. 判断错误
    if errors.Is(err, ErrUserNotFound) {
        fmt.Println("用户不存在")
    }

    // 4. 获取信息
    fmt.Println(kiterr.Code(err))     // 3001
    fmt.Println(kiterr.Name(err))     // USER_NOT_FOUND
    fmt.Println(kiterr.HTTPCode(err)) // 404
}
```

## API 说明

### 错误注册

```go
// 显式注册（推荐用于 API 暴露的错误）
errDef := kiterr.Register(3001, "USER_NOT_FOUND")

// 自动分配 code（从 4000 开始）
autoDef := kiterr.NewDefinition("EMAIL_NOT_VERIFIED")
```

### 创建错误实例

```go
// 简单错误
err := errDef.New("用户不存在")

// 格式化消息
err := errDef.Newf("用户 %s 不存在", userID)

// 包装底层错误
err := errDef.Wrap(dbErr, "查询用户失败")

// 格式化包装
err := errDef.Wrapf(dbErr, "查询用户 %s 失败", userID)
```

### 层级匹配

```go
// 建立层级：NotFound -> UserNotFound -> UserDeleted
base := kiterr.NotFound  // code=2001, http=404
child := kiterr.Register(3101, "USER_NOT_FOUND")
child.Class = base

// 匹配
err := child.New("用户不存在")
errors.Is(err, child)  // true
errors.Is(err, base)   // true（层级匹配）
```

### 辅助函数

| 函数 | 说明 |
|------|------|
| `Code(err)` | 获取错误码，非 codedError 返回 0 |
| `Name(err)` | 获取错误名，非 codedError 返回 "" |
| `HTTPCode(err)` | 获取 HTTP 状态码，默认 500 |
| `Export()` | 导出所有注册的错误定义 |

### 预定义错误

```go
kiterr.NotFound      // 2001, 404
kiterr.InvalidParam  // 2002, 400
kiterr.Unauthorized  // 2003, 401
kiterr.Forbidden     // 2004, 403
kiterr.Internal      // 2005, 500
kiterr.Timeout       // 2006, 504
kiterr.BadGateway    // 2007, 502
```

## 错误码范围

| 范围 | 用途 |
|------|------|
| 1000-1999 | 保留 |
| 2000-2999 | 工具库 Sentinel |
| 3000-3999 | 推荐用户显式注册 |
| 4000+ | 自动分配起始 |

## 导出错误码文档

```go
// errors/export_test.go
func TestGenerateDoc(t *testing.T) {
    if os.Getenv("GEN_ERR_DOC") != "1" {
        t.Skip()
    }
    // 生成 docs/error-codes.md
}
```

运行：
```bash
GEN_ERR_DOC=1 go test ./errors -run TestGenerateDoc -v
```

## 完整示例

```go
package main

import (
    "errors"
    "fmt"

    kiterr "github.com/tsopia/go-kit/errors"
)

func main() {
    // 定义业务错误
    ErrOrder := kiterr.Register(5100, "ORDER_ERROR").WithHTTP(400)
    ErrOrderNotFound := kiterr.Register(5101, "ORDER_NOT_FOUND")
    ErrOrderNotFound.Class = ErrOrder

    // 模拟底层错误
    dbErr := fmt.Errorf("db connection timeout")

    // 创建包装错误
    err := ErrOrderNotFound.Wrap(dbErr, "查询订单 12345 失败")

    // 判断和处理
    switch {
    case errors.Is(err, ErrOrderNotFound):
        fmt.Printf("[%d] %s: 订单不存在\n", kiterr.Code(err), kiterr.Name(err))
    case errors.Is(err, ErrOrder):
        fmt.Printf("[%d] %s: 订单相关错误\n", kiterr.Code(err), kiterr.Name(err))
    }

    // 获取 HTTP 状态码返回给客户端
    httpCode := kiterr.HTTPCode(err)
    fmt.Printf("HTTP Status: %d\n", httpCode)
}
```

## 测试

```bash
go test ./errors -v
```

## 许可证

MIT
