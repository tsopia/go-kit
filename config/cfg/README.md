# cfg

`cfg` 提供基于 [Viper](https://github.com/spf13/viper) 的配置加载与读取能力，兼顾结构体绑定和简单的 `cfg.GetXxx` 访问方式，同时支持默认值与类型检查。

## 快速开始

```go
import "your/module/path/config/cfg"

// 可选：将配置绑定到结构体
var conf struct {
    App struct {
        Name string `mapstructure:"name"`
        Port int    `mapstructure:"port"`
    } `mapstructure:"app"`
}

// 加载配置文件并绑定（target 可以为 nil）
manager, err := cfg.Load(&conf) // 自动查找 config.yml 或 .env
if err != nil {
    panic(err)
}

// 直接通过 manager 读取
name, err := manager.String("app.name")            // 如果缺失返回 ErrNotFound
port, err := manager.Int("app.port", 8080)         // 缺失时落回默认值

// 或使用包级函数读取（依赖默认管理器）
debug, err := cfg.Bool("flags.debug", false)
```

## 默认值与错误处理

- Getter 的最后一个参数可以传入默认值，键缺失时直接返回该值且不报错。
- 未提供默认值且键缺失，返回 `ErrNotFound`。
- 类型不匹配时返回 `ErrTypeMismatch`。
- 未加载配置时调用包级 Getter，会返回 `ErrNotLoaded`。

## 支持的 Getter

与 Viper 常用接口保持一致：`String`、`Bool`、`Int`、`Int64`、`Float64`、`Duration`、`Time`、`StringSlice`、`StringMap`、`StringMapString`，每个都支持可选默认值。可通过 `IsSet` 判断键是否存在。

## 配置文件查找规则

1. 如果 `cfg.Load` 指定了路径且文件存在，优先使用该路径。
2. 未指定路径时，会在工作目录、上级目录及各自的 `configs/`、`config/` 目录中依次查找 `config.yml`，找不到再查 `.env`。
3. 环境变量可以覆盖配置：
   - 若设置了 `APP_NAME`，则使用其大写作为前缀（如 `APP_NAME=myapp`，环境变量为 `MYAPP_APP_NAME`）。
   - 键名中的 `.` 会转换为 `_`，并允许空值覆盖（例如 `app.name` 对应 `APP_NAME` 前缀下的 `APP_NAME` 环境变量）。

## 线程安全

`Manager` 对读取操作加读写锁，满足并发读取场景；不支持热更新。
