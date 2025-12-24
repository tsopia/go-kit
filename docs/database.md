# 数据库连接 (database)

支持MySQL、PostgreSQL、SQLite的数据库连接管理器，提供重试机制、连接池管理、健康检查等企业级特性。

## 🚀 特性

- ✅ 支持MySQL、PostgreSQL、SQLite
- ✅ 自动重试连接机制
- ✅ 连接池配置和健康检查
- ✅ 完整的错误处理
- ✅ 基于GORM的ORM支持
- ✅ 线程安全
- ✅ 事务支持

## 📖 快速开始

### 基本使用

```go
package main

import (
    "log"
    "github.com/tsopia/go-kit/database"
)

func main() {
    // 创建数据库配置
    config := &database.Config{
        Driver:   "mysql",
        Host:     "localhost",
        Port:     3306,
        Username: "root",
        Password: "password",
        Database: "myapp",
        Charset:  "utf8mb4",
        
        // 连接池配置
        MaxIdleConns:    10,
        MaxOpenConns:    100,
        ConnMaxLifetime: time.Hour,
        
        // 重试配置
        RetryEnabled:      true,
        RetryMaxAttempts:  3,
        RetryInitialDelay: 1 * time.Second,
        RetryMaxDelay:     30 * time.Second,
    }
    
    // 创建数据库连接
    db, err := database.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // 测试连接
    if err := db.Ping(); err != nil {
        log.Fatal(err)
    }
    
    log.Println("数据库连接成功")
}
```

### 不同数据库配置

#### MySQL

```go
config := &database.Config{
    Driver:   "mysql",
    Host:     "localhost",
    Port:     3306,
    Username: "root",
    Password: "password",
    Database: "myapp",
    Charset:  "utf8mb4",
    Timezone: "Local",
}
```

#### PostgreSQL

```go
config := &database.Config{
    Driver:   "postgres",
    Host:     "localhost",
    Port:     5432,
    Username: "postgres",
    Password: "password",
    Database: "myapp",
    SSLMode:  "disable",
    Timezone: "UTC",
}
```

#### SQLite

```go
config := &database.Config{
    Driver:   "sqlite",
    Database: "app.db", // 文件路径或 ":memory:" 用于内存数据库
}
```

## 🔧 API 参考

### 创建数据库连接

#### New
使用配置创建数据库连接

```go
db, err := database.New(config)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### 配置选项

#### Config 结构体

```go
type Config struct {
    // 基础连接配置
    Driver   string `mapstructure:"driver"`
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Username string `mapstructure:"username"`
    Password string `mapstructure:"password"`
    Database string `mapstructure:"database"`
    Charset  string `mapstructure:"charset"`
    SSLMode  string `mapstructure:"ssl_mode"`
    Timezone string `mapstructure:"timezone"`
    
    // 连接池配置
    MaxIdleConns    int           `mapstructure:"max_idle_conns"`
    MaxOpenConns    int           `mapstructure:"max_open_conns"`
    ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
    ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
    
    // GORM日志配置
    LogLevel                  string        `mapstructure:"log_level"`
    SlowThreshold             time.Duration `mapstructure:"slow_threshold"`
    IgnoreRecordNotFoundError bool          `mapstructure:"ignore_record_not_found_error"`
    ParameterizedQueries      bool          `mapstructure:"parameterized_queries"`
    Colorful                  bool          `mapstructure:"colorful"`
    
    // 重试配置
    RetryEnabled      bool          `mapstructure:"retry_enabled"`
    RetryMaxAttempts  int           `mapstructure:"retry_max_attempts"`
    RetryInitialDelay time.Duration `mapstructure:"retry_initial_delay"`
    RetryMaxDelay     time.Duration `mapstructure:"retry_max_delay"`
    RetryBackoffFactor float64      `mapstructure:"retry_backoff_factor"`
    RetryJitterEnabled bool         `mapstructure:"retry_jitter_enabled"`
    
    // 其他配置
    TablePrefix       string `mapstructure:"table_prefix"`
    SingularTable     bool   `mapstructure:"singular_table"`
    DisableForeignKey bool   `mapstructure:"disable_foreign_key"`
    PrepareStmt       bool   `mapstructure:"prepare_stmt"`
    DryRun            bool   `mapstructure:"dry_run"`
}
```

### 数据库操作

#### 获取GORM实例

```go
// 获取GORM数据库实例
gormDB := db.GetDB()

// 使用GORM进行查询
var users []User
result := gormDB.Find(&users)
if result.Error != nil {
    log.Printf("查询失败: %v", result.Error)
}
```

#### 带Context的操作

```go
// 使用Context进行数据库操作
ctx := context.Background()
gormDB := db.WithContext(ctx)

var user User
result := gormDB.Where("id = ?", userID).First(&user)
if result.Error != nil {
    log.Printf("查询用户失败: %v", result.Error)
}
```

#### 事务操作

```go
// 简单事务
err := db.Transaction(func(tx *gorm.DB) error {
    // 创建用户
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    
    // 创建用户配置
    if err := tx.Create(&userConfig).Error; err != nil {
        return err
    }
    
    return nil
})

// 带Context的事务
err := db.TransactionWithContext(ctx, func(tx *gorm.DB) error {
    // 事务操作...
    return nil
})
```

### 健康检查

#### 基本健康检查

```go
// 检查数据库连接
if err := db.HealthCheck(); err != nil {
    log.Printf("数据库健康检查失败: %v", err)
}

// 带Context的健康检查
status := db.HealthCheckWithContext(ctx)
if !status.Healthy {
    log.Printf("数据库不健康: %v", status.Errors)
}
```

#### 连接池统计

```go
// 获取连接池统计信息
stats := db.Stats()
log.Printf("连接池统计: 打开=%d, 空闲=%d, 等待=%d",
    stats.OpenConnections,
    stats.IdleConnections,
    stats.WaitCount,
)
```

### 数据库迁移

```go
// 自动迁移数据库表
err := db.AutoMigrate(&User{}, &UserConfig{})
if err != nil {
    log.Printf("数据库迁移失败: %v", err)
}
```

## 🏗️ 最佳实践

### 1. 配置管理

#### 从配置文件加载

```go
type DatabaseConfig struct {
    Driver   string `mapstructure:"driver"`
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Username string `mapstructure:"username"`
    Password string `mapstructure:"password"`
    Database string `mapstructure:"database"`
    
    // 连接池配置
    MaxIdleConns    int           `mapstructure:"max_idle_conns"`
    MaxOpenConns    int           `mapstructure:"max_open_conns"`
    ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
    
    // 重试配置
    RetryEnabled      bool          `mapstructure:"retry_enabled"`
    RetryMaxAttempts  int           `mapstructure:"retry_max_attempts"`
    RetryInitialDelay time.Duration `mapstructure:"retry_initial_delay"`
}

func loadDatabaseConfig() (*database.Config, error) {
    var cfg struct {
        Database DatabaseConfig `mapstructure:"database"`
    }
    
    if err := config.LoadConfig(&cfg); err != nil {
        return nil, err
    }
    
    return &database.Config{
        Driver:            cfg.Database.Driver,
        Host:              cfg.Database.Host,
        Port:              cfg.Database.Port,
        Username:          cfg.Database.Username,
        Password:          cfg.Database.Password,
        Database:          cfg.Database.Database,
        MaxIdleConns:      cfg.Database.MaxIdleConns,
        MaxOpenConns:      cfg.Database.MaxOpenConns,
        ConnMaxLifetime:   cfg.Database.ConnMaxLifetime,
        RetryEnabled:      cfg.Database.RetryEnabled,
        RetryMaxAttempts:  cfg.Database.RetryMaxAttempts,
        RetryInitialDelay: cfg.Database.RetryInitialDelay,
    }, nil
}
```

### 2. 错误处理

```go
// 检查特定类型的错误
if database.IsConnectionError(err) {
    log.Printf("数据库连接错误: %v", err)
    // 尝试重连或降级处理
}

if database.IsValidationError(err) {
    log.Printf("配置验证错误: %v", err)
    // 检查配置参数
}

// 获取详细的错误信息
var dbErr *database.DatabaseError
if errors.As(err, &dbErr) {
    log.Printf("数据库错误 [%s]: %v", dbErr.Operation, dbErr.Err)
    log.Printf("错误上下文: %v", dbErr.Context)
}
```

### 3. 连接池优化

```go
// 生产环境连接池配置
config := &database.Config{
    // 基础配置...
    
    // 连接池配置
    MaxIdleConns:    10,              // 空闲连接数
    MaxOpenConns:    100,             // 最大连接数
    ConnMaxLifetime: time.Hour,       // 连接最大生存时间
    ConnMaxIdleTime: 10 * time.Minute, // 空闲连接超时
    
    // 重试配置
    RetryEnabled:      true,
    RetryMaxAttempts:  3,
    RetryInitialDelay: 1 * time.Second,
    RetryMaxDelay:     30 * time.Second,
    RetryBackoffFactor: 2.0,
    RetryJitterEnabled: true,
}
```

### 4. 健康检查

```go
// 定期健康检查
func startHealthCheck(db *database.Database) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if err := db.HealthCheck(); err != nil {
                log.Printf("数据库健康检查失败: %v", err)
                
                // 获取连接池统计
                stats := db.Stats()
                log.Printf("连接池状态: 打开=%d, 空闲=%d", 
                    stats.OpenConnections, stats.IdleConnections)
            }
        }
    }
}
```

### 5. 事务管理

```go
// 复杂事务示例
func createUserWithProfile(db *database.Database, user *User, profile *Profile) error {
    return db.Transaction(func(tx *gorm.DB) error {
        // 创建用户
        if err := tx.Create(user).Error; err != nil {
            return err
        }
        
        // 设置用户ID
        profile.UserID = user.ID
        
        // 创建用户档案
        if err := tx.Create(profile).Error; err != nil {
            return err
        }
        
        // 创建默认设置
        settings := &UserSettings{
            UserID: user.ID,
            Theme:  "default",
        }
        
        if err := tx.Create(settings).Error; err != nil {
            return err
        }
        
        return nil
    })
}
```

### 6. 模型定义

```go
// 用户模型
type User struct {
    ID        uint      `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
    
    Name     string `gorm:"size:100;not null"`
    Email    string `gorm:"size:100;uniqueIndex;not null"`
    Password string `gorm:"size:255;not null"`
    
    // 关联
    Profile   Profile   `gorm:"foreignKey:UserID"`
    Settings  UserSettings `gorm:"foreignKey:UserID"`
}

// 用户档案
type Profile struct {
    ID        uint      `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    
    UserID uint `gorm:"not null"`
    Avatar  string `gorm:"size:255"`
    Bio     string `gorm:"size:500"`
}

// 用户设置
type UserSettings struct {
    ID        uint      `gorm:"primarykey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    
    UserID uint `gorm:"not null"`
    Theme  string `gorm:"size:50;default:'default'"`
}
```

### 7. 查询优化

```go
// 使用预加载避免N+1问题
func getUsersWithProfiles(db *database.Database) ([]User, error) {
    var users []User
    err := db.GetDB().
        Preload("Profile").
        Preload("Settings").
        Find(&users).Error
    
    return users, err
}

// 使用索引优化查询
func getUserByEmail(db *database.Database, email string) (*User, error) {
    var user User
    err := db.GetDB().
        Where("email = ?", email).
        First(&user).Error
    
    if err != nil {
        return nil, err
    }
    
    return &user, nil
}
```

## 🧪 测试

### 单元测试

```go
func TestDatabaseConnection(t *testing.T) {
    // 使用SQLite内存数据库进行测试
    config := &database.Config{
        Driver:   "sqlite",
        Database: ":memory:",
    }
    
    db, err := database.New(config)
    if err != nil {
        t.Fatalf("创建数据库连接失败: %v", err)
    }
    defer db.Close()
    
    // 测试连接
    if err := db.Ping(); err != nil {
        t.Fatalf("数据库连接测试失败: %v", err)
    }
    
    // 测试迁移
    if err := db.AutoMigrate(&User{}); err != nil {
        t.Fatalf("数据库迁移失败: %v", err)
    }
}
```

### 集成测试

```go
func TestDatabaseTransaction(t *testing.T) {
    config := &database.Config{
        Driver:   "sqlite",
        Database: ":memory:",
    }
    
    db, err := database.New(config)
    if err != nil {
        t.Fatalf("创建数据库连接失败: %v", err)
    }
    defer db.Close()
    
    // 迁移表
    if err := db.AutoMigrate(&User{}); err != nil {
        t.Fatalf("数据库迁移失败: %v", err)
    }
    
    // 测试事务
    err = db.Transaction(func(tx *gorm.DB) error {
        user := &User{
            Name:     "测试用户",
            Email:    "test@example.com",
            Password: "password",
        }
        
        if err := tx.Create(user).Error; err != nil {
            return err
        }
        
        // 验证用户已创建
        var count int64
        if err := tx.Model(&User{}).Count(&count).Error; err != nil {
            return err
        }
        
        if count != 1 {
            return fmt.Errorf("期望1个用户，实际%d个", count)
        }
        
        return nil
    })
    
    if err != nil {
        t.Fatalf("事务测试失败: %v", err)
    }
}
```

## 🔍 故障排除

### 常见问题

#### 1. 连接失败

```bash
# 检查数据库服务是否运行
systemctl status mysql
systemctl status postgresql

# 检查网络连接
telnet localhost 3306
telnet localhost 5432

# 检查用户权限
mysql -u root -p -e "SHOW GRANTS FOR 'user'@'localhost'"
```

#### 2. 连接池问题

```go
// 监控连接池状态
stats := db.Stats()
log.Printf("连接池状态: 打开=%d, 空闲=%d, 等待=%d",
    stats.OpenConnections,
    stats.IdleConnections,
    stats.WaitCount,
)

// 如果等待连接过多，考虑增加连接池大小
config.MaxOpenConns = 200
config.MaxIdleConns = 20
```

#### 3. 重试配置

```go
// 调整重试配置
config.RetryEnabled = true
config.RetryMaxAttempts = 5
config.RetryInitialDelay = 2 * time.Second
config.RetryMaxDelay = 60 * time.Second
config.RetryBackoffFactor = 1.5
config.RetryJitterEnabled = true
```

### 性能优化

```go
// 1. 使用连接池
config.MaxIdleConns = 10
config.MaxOpenConns = 100
config.ConnMaxLifetime = time.Hour
config.ConnMaxIdleTime = 10 * time.Minute

// 2. 启用预处理语句
config.PrepareStmt = true

// 3. 配置慢查询日志
config.SlowThreshold = 1 * time.Second
config.LogLevel = "warn"

// 4. 使用索引
// 在模型上添加索引标签
type User struct {
    ID    uint   `gorm:"primarykey"`
    Email string `gorm:"size:100;uniqueIndex"`
    Name  string `gorm:"size:100;index"`
}
```

## 📚 相关链接

- [GORM官方文档](https://gorm.io/)
- [示例项目](./examples/database-simple/)
- [返回首页](../README.md) 