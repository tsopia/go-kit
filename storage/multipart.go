package storage

import "time"

// MultipartUpload 分片上传会话
type MultipartUpload struct {
	UploadID string
	Key      string
	Bucket   string
}

// PartInfo 分片信息
type PartInfo struct {
	PartNumber int
	ETag       string
	Size       int64
}

// ObjectInfo 对象元数据
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
	Metadata     map[string]string
}
