package providers

import (
	"context"
	"fmt"
	"io"
	"time"
)

// ObjectInfo 对象元数据
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
}

// MultipartUpload 分片上传信息
type MultipartUpload struct {
	UploadID string
	Key      string
	Bucket   string
}

// PartInfo 分片信息
type PartInfo struct {
	PartNumber int
	ETag       string
}

// UploadOption 上传选项
type UploadOption struct {
	ContentType string
}

// UploadOptionFunc 上传选项函数
type UploadOptionFunc func(*UploadOption)

// DownloadOption 下载选项
type DownloadOption struct{}

// DownloadOptionFunc 下载选项函数
type DownloadOptionFunc func(*DownloadOption)

// SignOption 签名选项
type SignOption struct {
	Method string
}

// SignOptionFunc 签名选项函数
type SignOptionFunc func(*SignOption)

// Config 存储配置
type Config struct {
	Type              string
	AccessKeyID       string
	AccessKeySecret   string
	Bucket            string
	Region            string
	Endpoint          string
	DefaultSignExpire time.Duration
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("storage type is required")
	}
	if c.AccessKeyID == "" || c.AccessKeySecret == "" {
		return fmt.Errorf("access key is required")
	}
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	return nil
}

// Client 存储客户端接口
type Client interface {
	Upload(ctx context.Context, key string, reader io.Reader, opts ...UploadOptionFunc) error
	Download(ctx context.Context, key string, opts ...DownloadOptionFunc) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Stat(ctx context.Context, key string) (*ObjectInfo, error)
	SignedURL(ctx context.Context, key string, expire time.Duration, opts ...SignOptionFunc) (string, error)
	InitMultipart(ctx context.Context, key string, opts ...UploadOptionFunc) (*MultipartUpload, error)
	UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...UploadOptionFunc) (*PartInfo, error)
	CompleteMultipart(ctx context.Context, uploadID string, parts []*PartInfo, opts ...UploadOptionFunc) error
	AbortMultipart(ctx context.Context, uploadID string) error
	DeleteBatch(ctx context.Context, keys []string) error
}

// 存储类型常量
const (
	TypeOSS = "oss"
	TypeCOS = "cos"
	TypeS3  = "s3"
)
