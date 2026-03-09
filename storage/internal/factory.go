package internal

import (
	"fmt"

	"github.com/tsopia/go-kit/storage"
)

// NewClient 根据配置创建对应客户端
func NewClient(cfg *storage.Config) (storage.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	switch cfg.Type {
	case storage.TypeOSS:
		return nil, fmt.Errorf("oss not implemented yet")
	case storage.TypeCOS:
		return nil, fmt.Errorf("cos not implemented yet")
	case storage.TypeS3:
		return nil, fmt.Errorf("s3 not implemented yet")
	default:
		return nil, fmt.Errorf("%w: %s", storage.ErrUnsupportedType, cfg.Type)
	}
}
