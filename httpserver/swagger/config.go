package swagger

// Config 描述 Swagger UI 的路由挂载配置。
type Config struct {
	Path   string
	DocURL string
}

// DefaultConfig 返回 Swagger UI 的默认挂载配置。
func DefaultConfig() Config {
	return Config{
		Path:   "/swagger/*any",
		DocURL: "doc.json",
	}
}

func applyDefaults(cfg Config) Config {
	defaults := DefaultConfig()
	if cfg.Path == "" {
		cfg.Path = defaults.Path
	}
	if cfg.DocURL == "" {
		cfg.DocURL = defaults.DocURL
	}

	return cfg
}
