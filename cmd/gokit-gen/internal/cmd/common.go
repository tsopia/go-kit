package cmd

import (
	"fmt"

	"github.com/tsopia/go-kit/cmd/gokit-gen/internal/discover"
	"github.com/tsopia/go-kit/database"
)

func connectDatabase(cfg *discover.Config) (*database.Database, error) {
	// 如果提供了 DSN，需要先解析成字段
	if cfg.DSN != "" {
		parsedCfg, err := discover.ParseDSN(cfg.DSN)
		if err != nil {
			return nil, fmt.Errorf("parse DSN: %w", err)
		}
		// 使用解析后的配置
		cfg = parsedCfg
	}

	// 校验必需的 driver 字段
	if cfg.Driver == "" {
		return nil, fmt.Errorf("database driver is required (provide --driver or config)")
	}

	// 设置默认端口
	if cfg.Port == 0 {
		switch cfg.Driver {
		case "mysql":
			cfg.Port = 3306
		case "postgres":
			cfg.Port = 5432
		}
	}

	dbCfg := &database.Config{
		Driver:   cfg.Driver,
		Host:     cfg.Host,
		Port:     cfg.Port,
		Database: cfg.Database,
		Username: cfg.Username,
		Password: cfg.Password,
	}

	// 传递 SSL 模式参数（如果有）
	if cfg.Params != nil {
		if sslmode, ok := cfg.Params["sslmode"]; ok {
			dbCfg.SSLMode = sslmode
		}
	}

	return database.New(dbCfg)
}
