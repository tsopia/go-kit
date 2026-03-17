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

// DirectUploadMode 客户端直传授权模式。
type DirectUploadMode string

const (
	DirectUploadModeAuto DirectUploadMode = "auto"
	DirectUploadModePut  DirectUploadMode = "put"
	DirectUploadModePost DirectUploadMode = "post"
)

// DirectUploadChecksumAlgorithm 描述客户端直传校验算法。
type DirectUploadChecksumAlgorithm string

const (
	DirectUploadChecksumMD5    DirectUploadChecksumAlgorithm = "md5"
	DirectUploadChecksumSHA256 DirectUploadChecksumAlgorithm = "sha256"
)

// DirectUploadSize 描述客户端直传的对象大小约束。
type DirectUploadSize struct {
	Exact int64
	Min   int64
	Max   int64
}

// DirectUploadChecksum 描述客户端直传的内容校验约束。
type DirectUploadChecksum struct {
	Algorithm DirectUploadChecksumAlgorithm
	Value     string
}

// DirectUploadRequest 描述客户端直传授权请求。
type DirectUploadRequest struct {
	ObjectKey   string
	Expire      time.Duration
	ContentType string
	Metadata    map[string]string
	Size        *DirectUploadSize
	Checksum    *DirectUploadChecksum
	Mode        DirectUploadMode
}

// DirectUploadConstraints 描述实际生效的上传约束。
type DirectUploadConstraints struct {
	ContentType string
	Metadata    map[string]string
	Size        *DirectUploadSize
	Checksum    *DirectUploadChecksum
}

// DirectUploadAuthorization 描述客户端直传授权结果。
type DirectUploadAuthorization struct {
	Provider    string
	Mode        DirectUploadMode
	ObjectKey   string
	URL         string
	Method      string
	Headers     map[string]string
	FormFields  map[string]string
	ExpiresAt   time.Time
	Constraints DirectUploadConstraints
}

// DirectUploadVerificationRequest 描述上传后对象校验请求。
type DirectUploadVerificationRequest struct {
	ObjectKey   string
	ContentType string
	Metadata    map[string]string
	Size        *DirectUploadSize
	Checksum    *DirectUploadChecksum
}

// DirectUploadMismatch 描述对象事实与期望约束的不匹配项。
type DirectUploadMismatch struct {
	Field    string
	Expected string
	Actual   string
}

// DirectUploadVerificationResult 描述上传后对象校验结果。
type DirectUploadVerificationResult struct {
	Exists     bool
	Matched    bool
	Mismatches []DirectUploadMismatch
	Object     *ObjectInfo
}

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
