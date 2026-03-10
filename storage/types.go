package storage

import (
	"time"

	"github.com/tsopia/go-kit/storage/internal"
)

// 重新导出 internal 包中的类型
type ObjectInfo = internal.ObjectInfo
type MultipartUpload = internal.MultipartUpload
type PartInfo = internal.PartInfo
type UploadOption = internal.UploadOption
type UploadOptionFunc = internal.UploadOptionFunc
type DownloadOption = internal.DownloadOption
type DownloadOptionFunc = internal.DownloadOptionFunc
type SignOption = internal.SignOption
type SignOptionFunc = internal.SignOptionFunc
type Config = internal.Config
type Client = internal.Client

const (
	TypeOSS = internal.TypeOSS
	TypeCOS = internal.TypeCOS
	TypeS3  = internal.TypeS3
)

// WithContentType 设置 Content-Type
func WithContentType(ct string) UploadOptionFunc {
	return func(o *UploadOption) {
		o.ContentType = ct
	}
}

// WithSignMethod 设置签名方法
func WithSignMethod(method string) SignOptionFunc {
	return func(o *SignOption) {
		o.Method = method
	}
}

// DefaultSignExpire 默认签名过期时间
const DefaultSignExpire = 15 * time.Minute
