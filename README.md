# Go-Kit

一个现代化、高性能的Go工具包，提供企业级应用开发所需的核心功能模块。

## 🚀 特性

- **配置管理** - 基于Viper的灵活配置系统，支持多格式和环境变量
- **HTTP客户端** - 功能强大的HTTP客户端，支持重试、熔断、调试
- **数据库连接** - 支持MySQL、PostgreSQL、SQLite，带重试机制
- **日志记录** - 基于Zap的高性能结构化日志
- **错误处理** - 统一的错误码系统和错误包装
- **HTTP服务器** - 基于Gin的轻量级HTTP服务器
- **工具函数** - 常用工具函数和常量定义

## 📦 模块概览

| 模块 | 描述 | 文档 |
|------|------|------|
| [config](./docs/config.md) | 配置管理系统 | [📖 详细文档](./docs/config.md) |
| [httpclient](./docs/httpclient.md) | HTTP客户端 | [📖 详细文档](./docs/httpclient.md) |
| [database](./docs/database.md) | 数据库连接管理 | [📖 详细文档](./docs/database.md) |
| [logger](./docs/logger.md) | 日志记录系统 | [📖 详细文档](./docs/logger.md) |
| [errors](./docs/errors.md) | 错误处理系统 | [📖 详细文档](./docs/errors.md) |
| [httpserver](./docs/httpserver.md) | HTTP服务器 | [📖 详细文档](./docs/httpserver.md) |
| [constants](./docs/constants.md) | 常量定义 | [📖 详细文档](./docs/constants.md) |
| [utils](./docs/utils.md) | 工具函数 | [📖 详细文档](./docs/utils.md) |
| [llm](./docs/llm.md) | 大模型客户端统一封装（Eino 兼容） | [📖 详细文档](./docs/llm.md) |

## 🎯 快速开始

### 安装

```bash
go mod init your-project
go get github.com/spf13/viper
go get go.uber.org/zap
go get github.com/gin-gonic/gin
go get gorm.io/gorm
```

### 基本使用

```go
package main

import (
    "log"
    "github.com/tsopia/go-kit/config"
    "github.com/tsopia/go-kit/logger"
    "github.com/tsopia/go-kit/httpclient"
)

// 配置结构
type AppConfig struct {
    Server struct {
        Host string `yaml:"host"`
        Port int    `yaml:"port"`
    } `yaml:"server"`
    Database struct {
        Host string `yaml:"host"`
        Port int    `yaml:"port"`
        Name string `yaml:"name"`
    } `yaml:"database"`
}

func main() {
    // 1. 加载配置
    var cfg AppConfig
    if err := config.LoadConfig(&cfg); err != nil {
        log.Fatal(err)
    }

    // 2. 初始化日志
    logger.SetupProduction()
    logger.Info("应用启动", "port", cfg.Server.Port)

    // 3. 创建HTTP客户端
    client := httpclient.NewClient()
    resp, err := client.Get("https://api.example.com/health")
    if err != nil {
        logger.Error("健康检查失败", "error", err)
    }

    logger.Info("应用运行中", "status", "ok")
}
```

## 📋 示例项目

查看 `examples/` 目录获取完整的使用示例：

- [基础配置示例](./examples/basic-config/) - 配置管理基础用法
- [HTTP服务器示例](./examples/http-server/) - HTTP服务器搭建
- [数据库连接示例](./examples/database-simple/) - 数据库连接管理
- [错误处理示例](./examples/errors-demo/) - 错误处理最佳实践
- [HTTP客户端示例](./examples/httpclient-ctx/) - HTTP客户端使用

## 🏗️ 架构设计

### 模块化设计
每个功能模块都是独立的，可以单独使用或组合使用：

```
go-kit/
├── config/      # 配置管理
├── logger/      # 日志系统
├── httpclient/  # HTTP客户端
├── database/    # 数据库连接
├── errors/      # 错误处理
├── httpserver/  # HTTP服务器
├── constants/   # 常量定义
├── utils/       # 工具函数
├── examples/    # 使用示例
└── docs/       # 详细文档
```

### 依赖关系
- 所有模块都可以独立使用
- 通过 `constants` 解决共享常量问题
- 避免循环依赖，保持清晰的模块边界

## 🔧 环境要求

- Go 1.21+
- 支持的操作系统：Linux, macOS, Windows

## 📚 文档导航

### 核心模块
- [配置管理](./docs/config.md) - 灵活的配置加载和环境变量支持
- [HTTP客户端](./docs/httpclient.md) - 功能强大的HTTP客户端，支持重试和调试
- [数据库连接](./docs/database.md) - 多数据库支持，带重试和连接池管理
- [日志系统](./docs/logger.md) - 高性能结构化日志，支持追踪
- [错误处理](./docs/errors.md) - 统一的错误码系统和错误包装
- [HTTP服务器](./docs/httpserver.md) - 基于Gin的轻量级服务器
- [常量定义](./docs/constants.md) - 共享常量和工具函数
- [工具函数](./docs/utils.md) - 常用工具函数集合

### 最佳实践
- [配置最佳实践](./docs/config.md#最佳实践)
- [日志最佳实践](./docs/logger.md#最佳实践)
- [错误处理最佳实践](./docs/errors.md#最佳实践)
- [HTTP客户端最佳实践](./docs/httpclient.md#最佳实践)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 开发环境设置

```bash
git clone https://github.com/your-username/go-kit.git
cd go-kit
go mod tidy
go test ./...
```

### 代码规范

- 遵循 Go 官方代码规范
- 所有新功能需要包含测试
- 提交前运行 `go test ./...`
- 保持文档与代码同步

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

感谢以下开源项目的支持：

- [Viper](https://github.com/spf13/viper) - 配置管理
- [Zap](https://github.com/uber-go/zap) - 高性能日志
- [Gin](https://github.com/gin-gonic/gin) - HTTP框架
- [GORM](https://gorm.io/) - ORM框架

---

**Go-Kit** - 让Go开发更简单、更高效 🚀 
