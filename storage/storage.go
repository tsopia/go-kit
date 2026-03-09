package storage

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/tsopia/go-kit/storage/internal"
)

var (
	_client Client
	_mu     sync.RWMutex
)

// Configure 初始化全局客户端
func Configure(cfg *Config) error {
	_mu.Lock()
	defer _mu.Unlock()

	client, err := internal.NewClient(cfg)
	if err != nil {
		return err
	}
	_client = client
	return nil
}

// GetClient 获取全局客户端
func GetClient() Client {
	_mu.RLock()
	defer _mu.RUnlock()
	return _client
}

// Upload 上传文件
func Upload(ctx context.Context, key string, reader io.Reader, opts ...UploadOptionFunc) error {
	return UploadWithClient(ctx, GetClient(), key, reader, opts...)
}

// UploadWithClient 使用指定客户端上传
func UploadWithClient(ctx context.Context, c Client, key string, reader io.Reader, opts ...UploadOptionFunc) error {
	if c == nil {
		return ErrMissingClient
	}
	return c.Upload(ctx, key, reader, opts...)
}

// Download 下载文件
func Download(ctx context.Context, key string, opts ...DownloadOptionFunc) (io.ReadCloser, error) {
	return DownloadWithClient(ctx, GetClient(), key, opts...)
}

// DownloadWithClient 使用指定客户端下载
func DownloadWithClient(ctx context.Context, c Client, key string, opts ...DownloadOptionFunc) (io.ReadCloser, error) {
	if c == nil {
		return nil, ErrMissingClient
	}
	return c.Download(ctx, key, opts...)
}

// Delete 删除文件
func Delete(ctx context.Context, key string) error {
	return DeleteWithClient(ctx, GetClient(), key)
}

// DeleteWithClient 使用指定客户端删除
func DeleteWithClient(ctx context.Context, c Client, key string) error {
	if c == nil {
		return ErrMissingClient
	}
	return c.Delete(ctx, key)
}

// Exists 检查文件是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	return ExistsWithClient(ctx, GetClient(), key)
}

// ExistsWithClient 使用指定客户端检查存在
func ExistsWithClient(ctx context.Context, c Client, key string) (bool, error) {
	if c == nil {
		return false, ErrMissingClient
	}
	return c.Exists(ctx, key)
}

// SignedURL 生成签名 URL
func SignedURL(ctx context.Context, key string, expire time.Duration, opts ...SignOptionFunc) (string, error) {
	return SignedURLWithClient(ctx, GetClient(), key, expire, opts...)
}

// SignedURLWithClient 使用指定客户端生成签名 URL
func SignedURLWithClient(ctx context.Context, c Client, key string, expire time.Duration, opts ...SignOptionFunc) (string, error) {
	if c == nil {
		return "", ErrMissingClient
	}
	return c.SignedURL(ctx, key, expire, opts...)
}

// InitMultipart 初始化分片上传
func InitMultipart(ctx context.Context, key string, opts ...UploadOptionFunc) (*MultipartUpload, error) {
	return InitMultipartWithClient(ctx, GetClient(), key, opts...)
}

// InitMultipartWithClient 使用指定客户端初始化分片上传
func InitMultipartWithClient(ctx context.Context, c Client, key string, opts ...UploadOptionFunc) (*MultipartUpload, error) {
	if c == nil {
		return nil, ErrMissingClient
	}
	return c.InitMultipart(ctx, key, opts...)
}

// UploadPart 上传分片
func UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...UploadOptionFunc) (*PartInfo, error) {
	return UploadPartWithClient(ctx, GetClient(), uploadID, partNum, reader, opts...)
}

// UploadPartWithClient 使用指定客户端上传分片
func UploadPartWithClient(ctx context.Context, c Client, uploadID string, partNum int, reader io.Reader, opts ...UploadOptionFunc) (*PartInfo, error) {
	if c == nil {
		return nil, ErrMissingClient
	}
	return c.UploadPart(ctx, uploadID, partNum, reader, opts...)
}

// CompleteMultipart 完成分片上传
func CompleteMultipart(ctx context.Context, uploadID string, parts []*PartInfo, opts ...UploadOptionFunc) error {
	return CompleteMultipartWithClient(ctx, GetClient(), uploadID, parts, opts...)
}

// CompleteMultipartWithClient 使用指定客户端完成分片上传
func CompleteMultipartWithClient(ctx context.Context, c Client, uploadID string, parts []*PartInfo, opts ...UploadOptionFunc) error {
	if c == nil {
		return ErrMissingClient
	}
	return c.CompleteMultipart(ctx, uploadID, parts, opts...)
}

// AbortMultipart 中止分片上传
func AbortMultipart(ctx context.Context, uploadID string) error {
	return AbortMultipartWithClient(ctx, GetClient(), uploadID)
}

// AbortMultipartWithClient 使用指定客户端中止分片上传
func AbortMultipartWithClient(ctx context.Context, c Client, uploadID string) error {
	if c == nil {
		return ErrMissingClient
	}
	return c.AbortMultipart(ctx, uploadID)
}

// DeleteBatch 批量删除文件
func DeleteBatch(ctx context.Context, keys []string) error {
	return DeleteBatchWithClient(ctx, GetClient(), keys)
}

// DeleteBatchWithClient 使用指定客户端批量删除
func DeleteBatchWithClient(ctx context.Context, c Client, keys []string) error {
	if c == nil {
		return ErrMissingClient
	}
	return c.DeleteBatch(ctx, keys)
}
