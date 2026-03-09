package internal

import (
	"fmt"

	"github.com/tsopia/go-kit/storage/providers/oss"
)

// NewClient 根据配置创建对应客户端
func NewClient(cfg *Config) (Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	switch cfg.Type {
	case TypeOSS:
		return oss.NewClient(cfg)
	case TypeCOS:
		return nil, fmt.Errorf("cos not implemented yet")
	case TypeS3:
		return nil, fmt.Errorf("s3 not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}
