package internal

import (
	"github.com/tsopia/go-kit/storage/providers"
)

// 重新导出 providers 包中的类型
type ObjectInfo = providers.ObjectInfo
type MultipartUpload = providers.MultipartUpload
type PartInfo = providers.PartInfo
type UploadOption = providers.UploadOption
type UploadOptionFunc = providers.UploadOptionFunc
type DownloadOption = providers.DownloadOption
type DownloadOptionFunc = providers.DownloadOptionFunc
type SignOption = providers.SignOption
type SignOptionFunc = providers.SignOptionFunc
type Config = providers.Config
type Client = providers.Client

const (
	TypeOSS = providers.TypeOSS
	TypeCOS = providers.TypeCOS
	TypeS3  = providers.TypeS3
)
