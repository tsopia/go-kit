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
type DirectUploadMode = internal.DirectUploadMode
type DirectUploadSize = internal.DirectUploadSize
type DirectUploadChecksumAlgorithm = internal.DirectUploadChecksumAlgorithm
type DirectUploadChecksum = internal.DirectUploadChecksum
type DirectUploadRequest = internal.DirectUploadRequest
type DirectUploadConstraints = internal.DirectUploadConstraints
type DirectUploadAuthorization = internal.DirectUploadAuthorization
type DirectUploadVerificationRequest = internal.DirectUploadVerificationRequest
type DirectUploadMismatch = internal.DirectUploadMismatch
type DirectUploadVerificationResult = internal.DirectUploadVerificationResult
type Config = internal.Config
type Client = internal.Client

const (
	TypeOSS = internal.TypeOSS
	TypeCOS = internal.TypeCOS
	TypeS3  = internal.TypeS3

	DirectUploadModeAuto = internal.DirectUploadModeAuto
	DirectUploadModePut  = internal.DirectUploadModePut
	DirectUploadModePost = internal.DirectUploadModePost

	DirectUploadChecksumMD5    = internal.DirectUploadChecksumMD5
	DirectUploadChecksumSHA256 = internal.DirectUploadChecksumSHA256
)

// WithContentType 设置 Content-Type
func WithContentType(ct string) UploadOptionFunc {
	return func(o *UploadOption) {
		o.ContentType = ct
	}
}

// WithMetadata 设置对象 metadata。
func WithMetadata(key, value string) UploadOptionFunc {
	return func(o *UploadOption) {
		if o.Metadata == nil {
			o.Metadata = make(map[string]string)
		}
		o.Metadata[key] = value
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
