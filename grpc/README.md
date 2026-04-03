# gRPC 包

> **待实现** — 当前为占位目录，尚未提供 gRPC 相关能力。

## 计划

gRPC 包计划提供以下能力：

- 基于 `google.golang.org/grpc` 的客户端/服务端封装
- 服务注册与发现集成
- 拦截器（认证、日志、指标）
- 健康检查

如需使用 gRPC，可暂时直接依赖官方库：

```go
import "google.golang.org/grpc"
```
