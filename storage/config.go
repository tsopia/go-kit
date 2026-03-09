package storage

import "time"

// Type 存储类型
type Type string

const (
	TypeOSS Type = "oss"
	TypeCOS Type = "cos"
	TypeS3  Type = "s3"
)

// Config 存储配置
type Config struct {
	Type     Type   `yaml:"type" json:"type"`
	Bucket   string `yaml:"bucket" json:"bucket"`
	Region   string `yaml:"region" json:"region"`
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// 凭证
	AccessKeyID     string `yaml:"access_key_id" json:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret" json:"access_key_secret"`
	SecretAccessKey string `yaml:"secret_access_key" json:"secret_access_key"`
	SessionToken    string `yaml:"session_token" json:"session_token"`

	// 连接控制
	Timeout           time.Duration `yaml:"timeout" json:"timeout"`
	MaxRetries        int           `yaml:"max_retries" json:"max_retries"`
	MaxPartSize       int64         `yaml:"max_part_size" json:"max_part_size"`
	PartSize          int64         `yaml:"part_size" json:"part_size"`
	DefaultSignExpire time.Duration `yaml:"default_sign_expire" json:"default_sign_expire"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Type == "" {
		return ErrInvalidConfig
	}
	if c.Bucket == "" {
		return ErrInvalidConfig
	}
	if c.Region == "" {
		return ErrInvalidConfig
	}
	if c.AccessKeyID == "" {
		return ErrInvalidConfig
	}
	return nil
}
