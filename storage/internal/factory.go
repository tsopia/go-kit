package internal

import (
	"fmt"

	"github.com/tsopia/go-kit/storage/providers"
	"github.com/tsopia/go-kit/storage/providers/cos"
	"github.com/tsopia/go-kit/storage/providers/oss"
)

// NewClient 根据配置创建对应客户端
func NewClient(cfg *providers.Config) (providers.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	switch cfg.Type {
	case providers.TypeOSS:
		return oss.NewClient(cfg)
	case providers.TypeCOS:
		return cos.NewClient(cfg)
	case providers.TypeS3:
		return nil, fmt.Errorf("s3 not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}
