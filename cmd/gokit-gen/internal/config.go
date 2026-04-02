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
	inferred := inferDriver(dsn)

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
	host, port := parseAddr(parsed.Addr)
	dbCfg.Host = host
	dbCfg.Port = port
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
		var port int
		fmt.Sscanf(p, "%d", &port)
		dbCfg.Port = port
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
