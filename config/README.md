# config 包

> **已废弃**，请使用 [`cfg`](../cfg/) 包替代。

`config` 是基于 Viper 的配置管理旧版封装，已被 `cfg` 包取代。

## 迁移指南

| 旧版 (config) | 新版 (cfg) |
|---------------|------------|
| `config.LoadConfig(&c)` | `cfg.Init()` + `cfg.Default()` |
| `config.GetStringWithDefault(key, def)` | `cfg.GetString(key, def)` |
| `config.GetIntWithDefault(key, def)` | `cfg.GetInt(key, def)` |
| `config.GetBoolWithDefault(key, def)` | `cfg.GetBool(key, def)` |
| `config.GetClient()` | `cfg.Default()` |

## 主要区别

- `cfg` 提供线程安全的 `Provider` 实例，支持多配置源
- `cfg` 入口为 `New()` / `NewWithPrefix()` 创建实例，或 `Init()` / `InitWithPrefix()` 初始化全局实例
- `cfg` 使用可选参数（`defaultValue ...T`）替代 `WithDefault` 后缀
- `cfg` 导出明确的错误变量（`ErrNotInitialized`、`ErrNotFound`、`ErrTypeMismatch`）

详细文档见 [`cfg/README.md`](../cfg/README.md)。
