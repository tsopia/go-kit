# gokit-gen 重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重构 gokit-gen CLI 工具，修复 DSN 拼接安全问题，统一配置获取，共享数据库连接，补齐测试。

**Architecture:** 所有命令统一走 `解析配置 → database.New() → 按命令执行`。migrate 用 `db.SQLDB()`，gen 用 `db.GetDB()`，sync 共享同一连接。dbmigrate 包改为接受 `*sql.DB`，不再拼 DSN。

**Tech Stack:** Go 1.24, golang-migrate/v4, gorm.io/gen, go-sql-driver/mysql, net/url

---

## Phase 1：修复 database DSN 构造

### Task 1: 修复 buildMySQLDSN 使用 mysql.Config.FormatDSN()

**Files:**
- Modify: `database/connector.go:169-179`
- Modify: `database/connector.go` (imports)
- Test: `database/database_test.go` (现有)

- [ ] **Step 1: 确认当前测试通过**

Run: `go test -v ./database/ -run TestNew -count=1`
Expected: PASS（如果无此测试则跳过）

- [ ] **Step 2: 修改 buildMySQLDSN**

将 `database/connector.go:169-179` 的 `buildMySQLDSN` 替换为：

```go
func buildMySQLDSN(config *Config) string {
	mysqlCfg := mysql.Config{
		User:                 config.Username,
		Passwd:               config.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", config.Host, config.Port),
		DBName:               config.Database,
		Charset:              config.Charset,
		ParseTime:            true,
		Loc:                  getTimeLocation(config.Timezone),
		AllowNativePasswords: true,
	}
	return mysqlCfg.FormatDSN()
}

func getTimeLocation(tz string) *time.Location {
	if tz == "" {
		tz = "Local"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Local
	}
	return loc
}
```

在 `database/connector.go` 顶部 import 中添加 `"time"`。

- [ ] **Step 3: 确认编译通过**

Run: `go build ./database/...`
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add database/connector.go
git commit -m "fix(database): use mysql.Config.FormatDSN to handle special characters in password"
```

---

### Task 2: 修复 buildPostgresDSN 使用 url.URL

**Files:**
- Modify: `database/connector.go:181-191`
- Modify: `database/connector.go` (imports)

- [ ] **Step 1: 修改 buildPostgresDSN**

将 `database/connector.go:181-191` 的 `buildPostgresDSN` 替换为：

```go
func buildPostgresDSN(config *Config) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(config.Username, config.Password),
		Host:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Path:     config.Database,
		RawQuery: fmt.Sprintf("sslmode=%s&TimeZone=%s", config.SSLMode, config.Timezone),
	}
	return u.String()
}
```

在 `database/connector.go` 顶部 import 中添加 `"net/url"`。

- [ ] **Step 2: 确认编译通过**

Run: `go build ./database/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add database/connector.go
git commit -m "fix(database): use url.URL for postgres DSN to properly encode credentials"
```

---

### Task 3: 给 database.Config 添加 DSN 字段

**Files:**
- Modify: `database/config.go` (Config struct + SetDefaults)

- [ ] **Step 1: 添加 DSN 字段**

在 `database/config.go` 的 `Config` struct 中，`Database` 字段后添加：

```go
DSN string `mapstructure:"dsn" json:"dsn" yaml:"dsn"`
```

- [ ] **Step 2: 修改 SetDefaults 处理 DSN**

在 `database/config.go` 的 `SetDefaults()` 方法最前面添加 DSN 解析逻辑：

```go
func (c *Config) SetDefaults() {
	// 如果提供了 DSN，优先从 DSN 解析填充字段
	if c.DSN != "" {
		c.fillFromDSN()
	}

	// ... 原有逻辑不变
}
```

在 `config.go` 末尾添加：

```go
func (c *Config) fillFromDSN() {
	if strings.HasPrefix(c.DSN, "postgres://") || strings.HasPrefix(c.DSN, "postgresql://") {
		c.fillFromPostgresDSN()
	} else if strings.Contains(c.DSN, "@tcp(") {
		c.fillFromMySQLDSN()
	}
}

func (c *Config) fillFromMySQLDSN() {
	cfg, err := mysql.ParseDSN(c.DSN)
	if err != nil {
		return
	}
	if c.Driver == "" {
		c.Driver = "mysql"
	}
	if c.Username == "" {
		c.Username = cfg.User
	}
	if c.Password == "" {
		c.Password = cfg.Passwd
	}
	if c.Database == "" {
		c.Database = cfg.DBName
	}
}

func (c *Config) fillFromPostgresDSN() {
	u, err := url.Parse(c.DSN)
	if err != nil {
		return
	}
	if c.Driver == "" {
		c.Driver = "postgres"
	}
	if u.User != nil {
		if c.Username == "" {
			c.Username = u.User.Username()
		}
		if pw, ok := u.User.Password(); ok && c.Password == "" {
			c.Password = pw
		}
	}
	if c.Host == "" {
		c.Host = u.Hostname()
	}
	if c.Port == 0 {
		if p := u.Port(); p != "" {
			if port, err := strconv.Atoi(p); err == nil {
				c.Port = port
			}
		}
	}
	if c.Database == "" {
		c.Database = strings.TrimPrefix(u.Path, "/")
	}
	if c.SSLMode == "" {
		if v := u.Query().Get("sslmode"); v != "" {
			c.SSLMode = v
		}
	}
}
```

在 `config.go` 顶部 import 中确保包含：

```go
import (
	"net/url"
	"strconv"
	"strings"
	"time"
	// ... 其他已有的 import
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm/logger"
)
```

- [ ] **Step 3: 确认编译通过**

Run: `go build ./database/...`
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add database/config.go
git commit -m "feat(database): add DSN field to Config with auto-parse support"
```

---

## Phase 2：配置解析层

### Task 4: 创建 discover.go（Go 源码发现）

**Files:**
- Create: `cmd/gokit-gen/internal/discover.go`

- [ ] **Step 1: 创建 discover.go**

```go
package internal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tsopia/go-kit/database"
)

// DiscoverFromSource 从 Go 源码中发现 database.Config 字面量。
// 只提取 Driver/Host/Port/Username/Password/Database 六个字段。
// 只支持直接字面量赋值，不支持变量引用、函数调用或表达式拼接。
func DiscoverFromSource(workDir string) (*database.Config, error) {
	patterns := []string{
		filepath.Join(workDir, "main.go"),
		filepath.Join(workDir, "cmd", "*", "main.go"),
	}

	var configs []*database.Config

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, file := range matches {
			cfg, err := parseGoFile(file)
			if err != nil {
				continue
			}
			configs = append(configs, cfg)
		}
	}

	switch len(configs) {
	case 0:
		return nil, fmt.Errorf("未发现数据库配置")
	case 1:
		return configs[0], nil
	default:
		return nil, fmt.Errorf("发现多个候选配置，请显式指定 --dsn 或 --config")
	}
}

func parseGoFile(filename string) (*database.Config, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}

	var cfg database.Config
	found := false

	ast.Inspect(node, func(n ast.Node) bool {
		composit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		if isDatabaseConfig(composit.Type) {
			found = true
			extractFields(composit, &cfg)
		}

		return true
	})

	if !found {
		return nil, fmt.Errorf("no database.Config found")
	}

	return &cfg, nil
}

func isDatabaseConfig(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		if t.Sel.Name != "Config" {
			return false
		}
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == "database"
		}
	case *ast.StarExpr:
		return isDatabaseConfig(t.X)
	}
	return false
}

func extractFields(composit *ast.CompositeLit, cfg *database.Config) {
	for _, elt := range composit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "Driver":
			cfg.Driver = stringVal(kv.Value)
		case "Host":
			cfg.Host = stringVal(kv.Value)
		case "Port":
			cfg.Port = intVal(kv.Value)
		case "Username":
			cfg.Username = stringVal(kv.Value)
		case "Password":
			cfg.Password = stringVal(kv.Value)
		case "Database":
			cfg.Database = stringVal(kv.Value)
		}
	}
}

func stringVal(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		return strings.Trim(lit.Value, `"`)
	}
	return ""
}

func intVal(expr ast.Expr) int {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.INT {
		if v, err := strconv.Atoi(lit.Value); err == nil {
			return v
		}
	}
	return 0
}
```

- [ ] **Step 2: 确认编译通过**

Run: `go build ./cmd/gokit-gen/...`
Expected: 无错误（可能有 unused import 警告，暂忽略）

- [ ] **Step 3: Commit**

```bash
git add cmd/gokit-gen/internal/discover.go
git commit -m "feat(gokit-gen): add Go source discovery for database config"
```

---

### Task 5: 创建 config.go（统一配置获取）

**Files:**
- Create: `cmd/gokit-gen/internal/config.go`

- [ ] **Step 1: 创建 config.go**

```go
package internal

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/tsopia/go-kit/cfg"
	"github.com/tsopia/go-kit/database"
)

// LoadOptions 配置加载选项
type LoadOptions struct {
	DSN        string
	Driver     string
	ConfigPath string
	WorkDir    string
}

// LoadDatabaseConfig 从多种来源获取数据库配置，返回 *database.Config。
// 优先级：DSN > 配置文件 > Go 源码发现。
func LoadDatabaseConfig(opts LoadOptions) (*database.Config, error) {
	if opts.WorkDir == "" {
		opts.WorkDir = "."
	}

	// 来源 1：命令行 DSN
	if opts.DSN != "" {
		return loadFromDSN(opts.DSN, opts.Driver)
	}

	// 来源 2：配置文件
	if dbCfg, err := loadFromConfigFile(opts.ConfigPath); err == nil {
		return dbCfg, nil
	}

	// 来源 3：Go 源码发现
	return DiscoverFromSource(opts.WorkDir)
}

func loadFromDSN(dsn, driver string) (*database.Config, error) {
	// 自动推断 driver
	inferred := inferDriver(dsn)

	// 如果用户传了 --driver，必须与推断一致
	if driver != "" && inferred != "" && driver != inferred {
		return nil, fmt.Errorf("--driver=%s 与 DSN 推断的 driver=%s 不一致", driver, inferred)
	}

	if driver == "" {
		driver = inferred
	}
	if driver == "" {
		return nil, fmt.Errorf("无法推断数据库驱动，请通过 --driver 指定")
	}

	dbCfg := &database.Config{DSN: dsn, Driver: driver}

	// 解析 DSN 填充结构化字段（用于日志和验证）
	switch driver {
	case "mysql":
		parseMySQLDSNToConfig(dsn, dbCfg)
	case "postgres":
		parsePostgresDSNToConfig(dsn, dbCfg)
	}

	return dbCfg, nil
}

func inferDriver(dsn string) string {
	if strings.Contains(dsn, "@tcp(") || strings.Contains(dsn, "@tcp (") {
		return "mysql"
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return "postgres"
	}
	return ""
}

func parseMySQLDSNToConfig(dsn string, dbCfg *database.Config) {
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil {
		return
	}
	dbCfg.Username = parsed.User
	dbCfg.Password = parsed.Passwd
	dbCfg.Database = parsed.DBName
	dbCfg.Host, dbCfg.Port = parseAddr(parsed.Addr)
}

func parsePostgresDSNToConfig(dsn string, dbCfg *database.Config) {
	u, err := url.Parse(dsn)
	if err != nil {
		return
	}
	if u.User != nil {
		dbCfg.Username = u.User.Username()
		if pw, ok := u.User.Password(); ok {
			dbCfg.Password = pw
		}
	}
	dbCfg.Host = u.Hostname()
	if p := u.Port(); p != "" {
		fmt.Sscanf(p, "%d", &dbCfg.Port)
	}
	dbCfg.Database = strings.TrimPrefix(u.Path, "/")
	if v := u.Query().Get("sslmode"); v != "" {
		dbCfg.SSLMode = v
	}
}

func parseAddr(addr string) (host string, port int) {
	parts := strings.Split(addr, ":")
	host = parts[0]
	if len(parts) > 1 {
		fmt.Sscanf(parts[1], "%d", &port)
	}
	return
}

func loadFromConfigFile(configPath string) (*database.Config, error) {
	paths := []string{}
	if configPath != "" {
		paths = append(paths, configPath)
	}

	provider, err := cfg.New(paths...)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %w", err)
	}

	sub := provider.Sub("database")
	if sub == nil {
		return nil, fmt.Errorf("配置文件中未找到 database 段")
	}

	// 检查是否有 dsn 字段
	if dsn, _ := sub.GetString("dsn"); dsn != "" {
		driver, _ := sub.GetString("driver")
		return loadFromDSN(dsn, driver)
	}

	// 结构化字段直接反序列化
	var dbCfg database.Config
	if err := sub.Unmarshal(&dbCfg); err != nil {
		return nil, fmt.Errorf("解析 database 配置失败: %w", err)
	}

	if dbCfg.Driver == "" {
		return nil, fmt.Errorf("配置文件 database 段缺少 driver")
	}

	return &dbCfg, nil
}
```

- [ ] **Step 2: 确认编译通过**

Run: `go build ./cmd/gokit-gen/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add cmd/gokit-gen/internal/config.go
git commit -m "feat(gokit-gen): add unified config resolution (DSN > config file > source discovery)"
```

---

### Task 6: 创建 config_test.go

**Files:**
- Create: `cmd/gokit-gen/internal/config_test.go`

- [ ] **Step 1: 创建 config_test.go**

```go
package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInferDriver(t *testing.T) {
	tests := []struct {
		dsn    string
		expect string
	}{
		{"root:pass@tcp(127.0.0.1:3306)/demo", "mysql"},
		{"postgres://user:pass@localhost:5432/demo", "postgres"},
		{"postgresql://user:pass@localhost:5432/demo", "postgres"},
		{"unknown://something", ""},
	}

	for _, tt := range tests {
		t.Run(tt.dsn, func(t *testing.T) {
			got := inferDriver(tt.dsn)
			if got != tt.expect {
				t.Errorf("inferDriver(%q) = %q, want %q", tt.dsn, got, tt.expect)
			}
		})
	}
}

func TestLoadFromDSN_MySQL(t *testing.T) {
	cfg, err := loadFromDSN("root:P@ss:123@tcp(127.0.0.1:3306)/demo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Driver != "mysql" {
		t.Errorf("driver = %q, want mysql", cfg.Driver)
	}
	if cfg.Username != "root" {
		t.Errorf("username = %q, want root", cfg.Username)
	}
	if cfg.Password != "P@ss:123" {
		t.Errorf("password = %q, want P@ss:123", cfg.Password)
	}
	if cfg.Database != "demo" {
		t.Errorf("database = %q, want demo", cfg.Database)
	}
}

func TestLoadFromDSN_Postgres(t *testing.T) {
	cfg, err := loadFromDSN("postgres://admin:secret@db.example.com:5432/mydb?sslmode=require", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Driver != "postgres" {
		t.Errorf("driver = %q, want postgres", cfg.Driver)
	}
	if cfg.Username != "admin" {
		t.Errorf("username = %q, want admin", cfg.Username)
	}
	if cfg.Password != "secret" {
		t.Errorf("password = %q, want secret", cfg.Password)
	}
	if cfg.Database != "mydb" {
		t.Errorf("database = %q, want mydb", cfg.Database)
	}
	if cfg.SSLMode != "require" {
		t.Errorf("sslmode = %q, want require", cfg.SSLMode)
	}
}

func TestLoadFromDSN_DriverMismatch(t *testing.T) {
	_, err := loadFromDSN("root:pass@tcp(127.0.0.1:3306)/demo", "postgres")
	if err == nil {
		t.Fatal("expected error for driver mismatch")
	}
}

func TestLoadFromDSN_UnknownFormat(t *testing.T) {
	_, err := loadFromDSN("something-random", "")
	if err == nil {
		t.Fatal("expected error for unknown DSN format")
	}
}

func TestDiscoverFromSource_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := DiscoverFromSource(tmpDir)
	if err == nil {
		t.Fatal("expected error when no source files found")
	}
}

func TestDiscoverFromSource_SingleConfig(t *testing.T) {
	tmpDir := t.TempDir()
	mainGo := `package main
import "github.com/tsopia/go-kit/database"
func main() {
	db, _ := database.New(&database.Config{
		Driver:   "mysql",
		Host:     "127.0.0.1",
		Port:     3306,
		Username: "root",
		Password: "pass",
		Database: "demo",
	})
	_ = db
}`
	err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := DiscoverFromSource(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Driver != "mysql" {
		t.Errorf("driver = %q, want mysql", cfg.Driver)
	}
	if cfg.Database != "demo" {
		t.Errorf("database = %q, want demo", cfg.Database)
	}
}

func TestDiscoverFromSource_MultipleConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "cmd", "server"), 0755)

	mainGo := `package main
import "github.com/tsopia/go-kit/database"
func main() {
	db, _ := database.New(&database.Config{Driver: "mysql", Host: "localhost", Port: 3306, Username: "root", Password: "a", Database: "db1"})
	_ = db
}`
	cmdGo := `package main
import "github.com/tsopia/go-kit/database"
func main() {
	db, _ := database.New(&database.Config{Driver: "postgres", Host: "localhost", Port: 5432, Username: "admin", Password: "b", Database: "db2"})
	_ = db
}`
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	os.WriteFile(filepath.Join(tmpDir, "cmd", "server", "main.go"), []byte(cmdGo), 0644)

	_, err := DiscoverFromSource(tmpDir)
	if err == nil {
		t.Fatal("expected error for multiple configs")
	}
}
```

- [ ] **Step 2: 运行测试确认通过**

Run: `go test -v ./cmd/gokit-gen/internal/ -run TestInfer -count=1`
Expected: PASS

- [ ] **Step 3: 运行全部测试**

Run: `go test -v ./cmd/gokit-gen/internal/ -count=1`
Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/gokit-gen/internal/config_test.go
git commit -m "test(gokit-gen): add config resolution tests (DSN, driver inference, source discovery)"
```

---

## Phase 3：dbmigrate 重写

### Task 7: 重写 dbmigrate/migrate.go

**Files:**
- Rewrite: `dbmigrate/migrate.go`

- [ ] **Step 1: 删除旧文件，创建新文件**

删除旧 `dbmigrate/migrate.go`，写入：

```go
package dbmigrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"
)

// Config 数据库迁移配置。
type Config struct {
	// SourcePath migration 文件目录路径。
	SourcePath string

	// DB 已建立的 *sql.DB 连接。
	DB *sql.DB

	// DriverName 数据库驱动名称：mysql | postgres | sqlite。
	DriverName string
}

// Status 迁移状态。
type Status struct {
	Version uint
	Dirty   bool
}

// Up 执行所有待执行的 up migration。
func Up(ctx context.Context, cfg Config) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// Down 回退一个版本。
func Down(ctx context.Context, cfg Config) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

// UpTo 迁移到指定版本。
func UpTo(ctx context.Context, cfg Config, version uint) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate to version %d: %w", version, err)
	}
	return nil
}

// DownTo 回退到指定版本。
func DownTo(ctx context.Context, cfg Config, version uint) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate down to version %d: %w", version, err)
	}
	return nil
}

// Version 返回当前迁移版本和脏状态。
func Version(ctx context.Context, cfg Config) (uint, bool, error) {
	m, err := createMigrate(cfg)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	v, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get version: %w", err)
	}
	return v, dirty, nil
}

// Status 返回迁移状态。
func Status(ctx context.Context, cfg Config) (Status, error) {
	v, dirty, err := Version(ctx, cfg)
	if err != nil {
		return Status{}, err
	}
	return Status{Version: v, Dirty: dirty}, nil
}

// Force 强制设置版本号，用于修复脏状态。
func Force(ctx context.Context, cfg Config, version int) error {
	m, err := createMigrate(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("force version %d: %w", version, err)
	}
	return nil
}

func createMigrate(cfg Config) (*migrate.Migrate, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if cfg.SourcePath == "" {
		return nil, fmt.Errorf("source path is empty")
	}

	source, err := (&file.File{}).Open("file://" + cfg.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("open migration source: %w", err)
	}

	dbDriver, err := openDriver(cfg.DB, cfg.DriverName)
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithInstance("file", source, cfg.DriverName, dbDriver)
	if err != nil {
		return nil, fmt.Errorf("create migrate instance: %w", err)
	}
	return m, nil
}
```

注意：`openDriver` 需要一个单独的文件来避免 import 冲突。

- [ ] **Step 2: 创建 dbmigrate/driver.go**

```go
package dbmigrate

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
)

func openDriver(db *sql.DB, driverName string) (database.Driver, error) {
	switch driverName {
	case "mysql":
		return mysql.WithInstance(db, &mysql.Config{})
	case "postgres":
		return postgres.WithInstance(db, &postgres.Config{})
	case "sqlite":
		return sqlite3.WithInstance(db, &sqlite3.Config{})
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driverName)
	}
}
```

- [ ] **Step 3: 更新 go.mod 依赖**

Run: `go mod tidy`
Expected: 自动添加/清理依赖

- [ ] **Step 4: 确认编译通过**

Run: `go build ./dbmigrate/...`
Expected: 无错误

- [ ] **Step 5: Commit**

```bash
git add dbmigrate/migrate.go dbmigrate/driver.go go.sum
git commit -m "refactor(dbmigrate): rewrite to accept *sql.DB, use NewWithInstance, remove DSN building"
```

---

### Task 8: 创建 dbmigrate 单元测试

**Files:**
- Create: `dbmigrate/migrate_test.go`

- [ ] **Step 1: 创建测试文件**

```go
package dbmigrate

import (
	"database/sql"
	"testing"
)

func TestCreateMigrate_NilDB(t *testing.T) {
	_, err := createMigrate(Config{
		SourcePath: "migrations",
		DB:         nil,
		DriverName: "mysql",
	})
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
}

func TestCreateMigrate_EmptySourcePath(t *testing.T) {
	db, _ := sql.Open("mysql", "")
	defer db.Close()

	_, err := createMigrate(Config{
		SourcePath: "",
		DB:         db,
		DriverName: "mysql",
	})
	if err == nil {
		t.Fatal("expected error for empty source path")
	}
}

func TestOpenDriver_Unsupported(t *testing.T) {
	db, _ := sql.Open("mysql", "")
	defer db.Close()

	_, err := openDriver(db, "oracle")
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestOpenDriver_MySQL(t *testing.T) {
	db, _ := sql.Open("mysql", "root:invalid@tcp(localhost:3306)/test")
	defer db.Close()

	d, err := openDriver(db, "mysql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d == nil {
		t.Fatal("expected non-nil driver")
	}
}

func TestOpenDriver_Postgres(t *testing.T) {
	db, _ := sql.Open("postgres", "host=localhost port=5432 user=test dbname=test sslmode=disable")
	defer db.Close()

	d, err := openDriver(db, "postgres")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d == nil {
		t.Fatal("expected non-nil driver")
	}
}

func TestOpenDriver_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Skipf("sqlite3 driver not available: %v", err)
	}
	defer db.Close()

	d, err := openDriver(db, "sqlite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d == nil {
		t.Fatal("expected non-nil driver")
	}
}
```

- [ ] **Step 2: 运行测试**

Run: `go test -v ./dbmigrate/ -run "TestCreateMigrate|TestOpenDriver" -count=1`
Expected: TestCreateMigrate_NilDB PASS, TestCreateMigrate_EmptySourcePath PASS, TestOpenDriver_Unsupported PASS, TestOpenDriver_MySQL/Postgres/SQLite PASS

- [ ] **Step 3: Commit**

```bash
git add dbmigrate/migrate_test.go
git commit -m "test(dbmigrate): add unit tests for config validation and driver dispatch"
```

---

## Phase 4：gen + sync + CLI

### Task 9: 创建 gen.go

**Files:**
- Create: `cmd/gokit-gen/gen.go`

- [ ] **Step 1: 创建 gen.go**

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gen"
	"gorm.io/gorm"
)

type genOptions struct {
	DB      *gorm.DB
	OutPath string
	Tables  []string
}

func runGen(_ context.Context, opts genOptions) (err error) {
	if opts.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("gorm gen panic: %v", r)
		}
	}()

	cfg := gen.Config{
		OutPath: opts.OutPath,
		Mode:    gen.WithDefaultQuery | gen.WithQueryInterface,
	}

	gormGen := gen.NewGenerator(cfg)
	gormGen.UseDB(opts.DB)

	if len(opts.Tables) > 0 {
		for _, table := range opts.Tables {
			table = strings.TrimSpace(table)
			if table != "" {
				gormGen.ApplyBasic(gormGen.GenerateModel(table))
			}
		}
	} else {
		gormGen.ApplyBasic(gormGen.GenerateAllTable()...)
	}

	gormGen.Execute()
	return nil
}
```

- [ ] **Step 2: 确认编译通过**

Run: `go build ./cmd/gokit-gen/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add cmd/gokit-gen/gen.go
git commit -m "feat(gokit-gen): add gen command with panic recovery"
```

---

### Task 10: 重写 main.go

**Files:**
- Rewrite: `cmd/gokit-gen/main.go`

- [ ] **Step 1: 重写 main.go**

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tsopia/go-kit/cmd/gokit-gen/internal"
	"github.com/tsopia/go-kit/database"
	"github.com/tsopia/go-kit/dbmigrate"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()
	subcommand := os.Args[1]

	switch subcommand {
	case "migrate":
		if err := runMigrate(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "gen":
		if err := runGenCmd(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "sync":
		if err := runSync(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`gokit-gen - Code generation tool for go-kit projects

Usage:
  gokit-gen <command> [flags]

Commands:
  migrate    Database migration operations
  gen        Generate GORM models from database
  sync       Run migrate then gen (recommended)

Global Flags:
  --dsn              Database DSN (auto-detects driver)
  --driver           Database driver: mysql, postgres, sqlite (optional, validated against DSN)
  --config           Config file path
  --migration-path   Migration files directory (default: migrations)
  --out              Output directory for generated code (default: internal/model)
  --tables           Comma-separated list of tables to generate (default: all)

Examples:
  gokit-gen sync                                                        # Auto-discover config
  gokit-gen sync --out ./pkg/model                                      # Custom output directory
  gokit-gen migrate up                                                  # Run pending migrations
  gokit-gen migrate down                                                # Rollback one migration
  gokit-gen migrate status                                              # Show migration status
  gokit-gen gen --tables user,order                                     # Generate specific tables
  gokit-gen sync --dsn "root:pass@tcp(localhost:3306)/db"               # MySQL DSN
  gokit-gen sync --dsn "postgres://user:pass@host:5432/db"              # PostgreSQL DSN
`)
}

// --- 配置加载与连接 ---

func loadAndConnect(args []string) (*database.Database, *database.Config, error) {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN")
	driver := fs.String("driver", "", "Database driver")
	config := fs.String("config", "", "Config file path")
	fs.Parse(args)

	dbCfg, err := internal.LoadDatabaseConfig(internal.LoadOptions{
		DSN:        *dsn,
		Driver:     *driver,
		ConfigPath: *config,
		WorkDir:    ".",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	db, err := database.New(dbCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("connect database: %w", err)
	}
	return db, dbCfg, nil
}

// --- migrate 命令 ---

func runMigrate(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN")
	driver := fs.String("driver", "", "Database driver")
	config := fs.String("config", "", "Config file path")
	source := fs.String("migration-path", "migrations", "Migration files directory")
	fs.Parse(args)

	if len(fs.Args()) == 0 {
		return fmt.Errorf("migrate command required: up, down, status, version, force <version>")
	}
	command := fs.Args()[0]

	dbCfg, err := internal.LoadDatabaseConfig(internal.LoadOptions{
		DSN:        *dsn,
		Driver:     *driver,
		ConfigPath: *config,
		WorkDir:    ".",
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := database.New(dbCfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	sqlDB, err := db.SQLDB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	mCfg := dbmigrate.Config{
		SourcePath: *source,
		DB:         sqlDB,
		DriverName: dbCfg.Driver,
	}

	switch command {
	case "up":
		if err := dbmigrate.Up(ctx, mCfg); err != nil {
			return err
		}
		fmt.Println("Migration completed: up")
	case "down":
		if err := dbmigrate.Down(ctx, mCfg); err != nil {
			return err
		}
		fmt.Println("Migration completed: down")
	case "status":
		st, err := dbmigrate.Status(ctx, mCfg)
		if err != nil {
			return err
		}
		if st.Version == 0 {
			fmt.Println("Migration status: no migrations applied")
		} else {
			dirty := ""
			if st.Dirty {
				dirty = " (dirty)"
			}
			fmt.Printf("Migration status: version %d%s\n", st.Version, dirty)
		}
	case "version":
		v, dirty, err := dbmigrate.Version(ctx, mCfg)
		if err != nil {
			return err
		}
		fmt.Printf("Version: %d (dirty: %v)\n", v, dirty)
	case "force":
		if len(fs.Args()) < 2 {
			return fmt.Errorf("force requires a version argument")
		}
		var version int
		if _, err := fmt.Sscanf(fs.Args()[1], "%d", &version); err != nil {
			return fmt.Errorf("invalid version: %s", fs.Args()[1])
		}
		if err := dbmigrate.Force(ctx, mCfg, version); err != nil {
			return err
		}
		fmt.Printf("Forced version: %d\n", version)
	default:
		return fmt.Errorf("unknown migrate command: %s", command)
	}

	return nil
}

// --- gen 命令 ---

func runGenCmd(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN")
	driver := fs.String("driver", "", "Database driver")
	config := fs.String("config", "", "Config file path")
	out := fs.String("out", "internal/model", "Output directory")
	tables := fs.String("tables", "", "Comma-separated table names")
	fs.Parse(args)

	db, dbCfg, err := internal.LoadDatabaseConfig(internal.LoadOptions{
		DSN:        *dsn,
		Driver:     *driver,
		ConfigPath: *config,
		WorkDir:    ".",
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	dbInst, err := database.New(dbCfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer dbInst.Close()

	_ = db // loadAndConnect 不再使用，改为上面手动建连

	if err := runGen(ctx, genOptions{
		DB:      dbInst.GetDB(),
		OutPath: *out,
		Tables:  parseTables(*tables),
	}); err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	fmt.Printf("Generated code to: %s\n", *out)
	return nil
}

// --- sync 命令 ---

func runSync(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN")
	driver := fs.String("driver", "", "Database driver")
	config := fs.String("config", "", "Config file path")
	migrateSource := fs.String("migration-path", "migrations", "Migration files directory")
	out := fs.String("out", "internal/model", "Output directory")
	tables := fs.String("tables", "", "Comma-separated table names")
	fs.Parse(args)

	// 解析配置，只建一次连接
	dbCfg, err := internal.LoadDatabaseConfig(internal.LoadOptions{
		DSN:        *dsn,
		Driver:     *driver,
		ConfigPath: *config,
		WorkDir:    ".",
	})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := database.New(dbCfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	// Step 1: migrate up
	sqlDB, err := db.SQLDB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	if err := dbmigrate.Up(ctx, dbmigrate.Config{
		SourcePath: *migrateSource,
		DB:         sqlDB,
		DriverName: dbCfg.Driver,
	}); err != nil {
		return fmt.Errorf("migrate failed: %w", err)
	}
	fmt.Println("Migration completed: up")

	// Step 2: gen
	if err := runGen(ctx, genOptions{
		DB:      db.GetDB(),
		OutPath: *out,
		Tables:  parseTables(*tables),
	}); err != nil {
		return fmt.Errorf("gen failed: %w", err)
	}

	fmt.Printf("Generated code to: %s\n", *out)
	fmt.Println("Sync completed successfully")
	return nil
}

func parseTables(tables string) []string {
	if tables == "" {
		return nil
	}
	return strings.Split(tables, ",")
}
```

- [ ] **Step 2: 确认编译通过**

Run: `go build ./cmd/gokit-gen/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add cmd/gokit-gen/main.go
git commit -m "refactor(gokit-gen): rewrite CLI with unified config, shared connections"
```

---

## Phase 5：删除旧代码

### Task 11: 删除旧文件

**Files:**
- Delete: `cmd/gokit-gen/internal/discover/discover.go`
- Delete: `cmd/gokit-gen/internal/cmd/common.go`
- Delete: `cmd/gokit-gen/internal/cmd/migrate.go`
- Delete: `cmd/gokit-gen/internal/cmd/gen.go`
- Delete: `cmd/gokit-gen/internal/cmd/sync.go`
- Delete: `cmd/gokit-gen/internal/generator/generator.go`

- [ ] **Step 1: 删除旧文件**

```bash
rm -rf cmd/gokit-gen/internal/discover/
rm -rf cmd/gokit-gen/internal/cmd/
rm -rf cmd/gokit-gen/internal/generator/
```

- [ ] **Step 2: 确认编译通过**

Run: `go build ./cmd/gokit-gen/...`
Expected: 无错误

- [ ] **Step 3: 运行全部测试**

Run: `go test -v ./cmd/gokit-gen/... ./dbmigrate/...`
Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add -A cmd/gokit-gen/internal/
git commit -m "chore(gokit-gen): remove old discover, cmd, and generator packages"
```

---

### Task 12: 清理依赖并最终验证

- [ ] **Step 1: 清理依赖**

Run: `go mod tidy`

- [ ] **Step 2: 全仓编译**

Run: `go build ./...`
Expected: 无错误

- [ ] **Step 3: 全仓测试**

Run: `go test -v ./cmd/gokit-gen/... ./dbmigrate/...`
Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: tidy dependencies after gokit-gen refactor"
```
