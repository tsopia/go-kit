package storage

import (
	"context"
	"io"
	"time"
)

// Client 存储客户端接口
type Client interface {
	// 基础操作
	Upload(ctx context.Context, key string, reader io.Reader, opts ...UploadOption) error
	Download(ctx context.Context, key string, opts ...DownloadOption) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Stat(ctx context.Context, key string) (*ObjectInfo, error)

	// URL 签名
	SignedURL(ctx context.Context, key string, expire time.Duration, opts ...SignOption) (string, error)

	// 分片上传
	InitMultipart(ctx context.Context, key string, opts ...UploadOption) (*MultipartUpload, error)
	UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...UploadOption) (*PartInfo, error)
	CompleteMultipart(ctx context.Context, uploadID string, parts []*PartInfo, opts ...UploadOption) error
	AbortMultipart(ctx context.Context, uploadID string) error

	// 批量操作
	DeleteBatch(ctx context.Context, keys []string) error
}
