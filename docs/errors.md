# 错误处理（errors）

`errors` 包提供统一的错误定义、错误实例化、层级匹配和错误码导出能力。

## 1. 快速开始

下面示例可直接运行：

```go
package main

import (
	"errors"
	"fmt"

	kiterr "github.com/tsopia/go-kit/errors"
)

func main() {
	// 1) 定义错误（显式 code）
	ErrUserNotFound := kiterr.Register(5101, "USER_NOT_FOUND").WithHTTP(404)
	ErrUserReadFailed := kiterr.Register(5102, "USER_READ_FAILED").WithHTTP(500)
	ErrUserReadFailed.Class = ErrUserNotFound

	// 2) 创建错误
	errA := ErrUserNotFound.New("用户不存在")
	errB := ErrUserReadFailed.Wrap(fmt.Errorf("db timeout"), "读取用户失败")

	// 3) 判断错误
	fmt.Println(errors.Is(errA, ErrUserNotFound))   // true
	fmt.Println(errors.Is(errB, ErrUserReadFailed)) // true
	fmt.Println(errors.Is(errB, ErrUserNotFound))   // true（层级匹配）

	// 4) 读取错误信息
	fmt.Println(kiterr.Code(errB))     // 5102
	fmt.Println(kiterr.Name(errB))     // USER_READ_FAILED
	fmt.Println(kiterr.HTTPCode(errB)) // 500
}
```

## 2. 显式注册错误（Register）

`Register(code, name)` 用于手动指定错误码与错误名；`code` 与 `name` 在同一 Registry 内必须唯一，重复会 panic。

```go
package main

import (
	"fmt"

	kiterr "github.com/tsopia/go-kit/errors"
)

func main() {
	ErrOrderNotFound := kiterr.Register(5201, "ORDER_NOT_FOUND").WithHTTP(404)
	fmt.Println(ErrOrderNotFound.Code) // 5201
	fmt.Println(ErrOrderNotFound.Name) // ORDER_NOT_FOUND
	fmt.Println(ErrOrderNotFound.HTTP) // 404
}
```

## 3. 自动分配错误（NewDefinition）

`NewDefinition(name)` 会在全局注册表中自动分配 code（默认从 4000 开始）。同名会返回同一个定义。

```go
package main

import (
	"fmt"

	kiterr "github.com/tsopia/go-kit/errors"
)

func main() {
	def1 := kiterr.NewDefinition("PAYMENT_FAILED")
	def2 := kiterr.NewDefinition("PAYMENT_FAILED")
	def3 := kiterr.NewDefinition("PAYMENT_TIMEOUT")

	fmt.Println(def1.Code)      // 4000（首次自动分配）
	fmt.Println(def1 == def2)   // true（同名复用）
	fmt.Println(def3.Code >= 4001) // true
}
```

## 4. 创建错误（New / Newf / Wrap / Wrapf）

错误创建入口都在 `*Definition` 上：

```go
package main

import (
	"fmt"

	kiterr "github.com/tsopia/go-kit/errors"
)

func main() {
	ErrPaymentFailed := kiterr.Register(5301, "PAYMENT_FAILED")
	cause := fmt.Errorf("upstream timeout")

	err1 := ErrPaymentFailed.New("支付失败")
	err2 := ErrPaymentFailed.Newf("支付失败，order_id=%s", "o_1001")
	err3 := ErrPaymentFailed.Wrap(cause, "调用支付网关失败")
	err4 := ErrPaymentFailed.Wrapf(cause, "调用支付网关失败，gateway=%s", "alipay")

	fmt.Println(err1)
	fmt.Println(err2)
	fmt.Println(err3)
	fmt.Println(err4)
}
```

## 5. 判断错误（errors.Is + 层级匹配）

`codedError` 支持：
- 与当前 Definition 精确匹配
- 沿 `Class` 向上做祖先匹配
- 同时保留 `Wrap` 的底层 `cause` 链

```go
package main

import (
	"errors"
	"fmt"

	kiterr "github.com/tsopia/go-kit/errors"
)

func main() {
	base := kiterr.Register(5400, "BIZ_ERROR")
	parent := kiterr.Register(5401, "ORDER_ERROR")
	parent.Class = base
	child := kiterr.Register(5402, "ORDER_EXPIRED")
	child.Class = parent

	cause := fmt.Errorf("db timeout")
	err := child.Wrap(cause, "订单已过期")

	fmt.Println(errors.Is(err, child))  // true
	fmt.Println(errors.Is(err, parent)) // true
	fmt.Println(errors.Is(err, base))   // true
	fmt.Println(errors.Is(err, cause))  // true
}
```

## 6. 获取错误信息（Code / Name / HTTPCode）

```go
package main

import (
	"fmt"

	kiterr "github.com/tsopia/go-kit/errors"
)

func main() {
	ErrForbidden := kiterr.Register(5501, "FORBIDDEN").WithHTTP(403)
	err := ErrForbidden.New("无权限访问")

	fmt.Println(kiterr.Code(err))     // 5501
	fmt.Println(kiterr.Name(err))     // FORBIDDEN
	fmt.Println(kiterr.HTTPCode(err)) // 403

	fmt.Println(kiterr.Code(fmt.Errorf("plain error")))     // 0
	fmt.Println(kiterr.Name(fmt.Errorf("plain error")))     // ""
	fmt.Println(kiterr.HTTPCode(fmt.Errorf("plain error"))) // 500
}
```

## 7. 导出错误码文档（TestGenerateDoc）

`errors/export_test.go` 已内置 `TestGenerateDoc`：
- 仅当 `GEN_ERR_DOC=1` 时执行
- 从 `errors.Export()` 读取定义
- 生成 `docs/error-codes.md`

执行命令：

```bash
GEN_ERR_DOC=1 go test ./errors -run TestGenerateDoc -v
```

如需在业务中查看导出数据：

```go
package main

import (
	"fmt"

	kiterr "github.com/tsopia/go-kit/errors"
)

func main() {
	kiterr.Register(5601, "COUPON_INVALID")
	kiterr.Register(5602, "COUPON_EXPIRED")

	for _, info := range kiterr.Export() {
		fmt.Printf("code=%d name=%s class=%s\n", info.Code, info.Name, info.Class)
	}
}
```
