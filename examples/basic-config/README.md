# 基本配置使用示例

本示例展示了如何使用简化的配置管理器。

## 运行示例

```bash
cd examples/basic-config
go run main.go
```

## 环境变量前缀配置

### 情况1: 使用自定义前缀

```bash
# 设置应用名称作为环境变量前缀
export APP_NAME=myapp

# 设置配置值（使用 MYAPP_ 前缀）
export MYAPP_SERVER_HOST=localhost
export MYAPP_SERVER_PORT=8080
export MYAPP_DEBUG=true
export MYAPP_DATABASE_HOST=localhost
export MYAPP_DATABASE_PORT=3306
export MYAPP_DATABASE_NAME=myapp_db

# 运行程序
go run main.go
```

### 情况2: 不使用前缀

```bash
# 不设置 APP_NAME 环境变量

# 设置配置值（无前缀）
export SERVER_HOST=localhost
export SERVER_PORT=8080
export DEBUG=true
export DATABASE_HOST=localhost
export DATABASE_PORT=3306
export DATABASE_NAME=myapp_db

# 运行程序
go run main.go
```

## 配置文件示例

创建 `config.yaml` 文件：

```yaml
server:
  host: "127.0.0.1"
  port: 8080

database:
  host: "localhost"
  port: 3306
  username: "root"
  password: "password"
  name: "myapp"

debug: false
```

## 优先级

配置的优先级从高到低：
1. 环境变量
2. 配置文件
3. 程序中设置的值

## 环境变量映射规则

- 配置键中的点号 (`.`) 会替换为下划线 (`_`)
- 前缀会自动转换为大写
- 如果设置了 `APP_NAME=myapp`：
  - `server.host` → `MYAPP_SERVER_HOST`
  - `database.password` → `MYAPP_DATABASE_PASSWORD`
- 如果没有设置 `APP_NAME`：
  - `server.host` → `SERVER_HOST`
  - `database.password` → `DATABASE_PASSWORD` 