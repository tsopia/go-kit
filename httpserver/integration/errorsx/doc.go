// Package errorsx 提供 errors 包与 httpserver typed handler 之间的桥接映射。
//
// 适用场景：
//   - 团队统一使用 github.com/tsopia/go-kit/errors 作为业务错误出口
//   - 需要把错误码、错误名和 HTTP 状态稳定映射到 typed handler 响应
//   - 不希望 httpserver core 直接依赖 errors 包
package errorsx
