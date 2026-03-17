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
type DirectUploadMode = providers.DirectUploadMode
type DirectUploadSize = providers.DirectUploadSize
type DirectUploadChecksumAlgorithm = providers.DirectUploadChecksumAlgorithm
type DirectUploadChecksum = providers.DirectUploadChecksum
type DirectUploadRequest = providers.DirectUploadRequest
type DirectUploadConstraints = providers.DirectUploadConstraints
type DirectUploadAuthorization = providers.DirectUploadAuthorization
type DirectUploadVerificationRequest = providers.DirectUploadVerificationRequest
type DirectUploadMismatch = providers.DirectUploadMismatch
type DirectUploadVerificationResult = providers.DirectUploadVerificationResult
type Config = providers.Config
type Client = providers.Client

const (
	TypeOSS = providers.TypeOSS
	TypeCOS = providers.TypeCOS
	TypeS3  = providers.TypeS3

	DirectUploadModeAuto = providers.DirectUploadModeAuto
	DirectUploadModePut  = providers.DirectUploadModePut
	DirectUploadModePost = providers.DirectUploadModePost

	DirectUploadChecksumMD5    = providers.DirectUploadChecksumMD5
	DirectUploadChecksumSHA256 = providers.DirectUploadChecksumSHA256
)
