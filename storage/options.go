package storage

import "time"

// UploadOption 上传选项
type UploadOption func(*uploadOptions)

type uploadOptions struct {
	ContentType   string
	ContentLength int64
	Metadata      map[string]string
	Headers       map[string]string
	StorageClass  string
}

func WithContentType(ct string) UploadOption {
	return func(o *uploadOptions) {
		o.ContentType = ct
	}
}

func WithMetadata(key, value string) UploadOption {
	return func(o *uploadOptions) {
		if o.Metadata == nil {
			o.Metadata = make(map[string]string)
		}
		o.Metadata[key] = value
	}
}

func WithStorageClass(class string) UploadOption {
	return func(o *uploadOptions) {
		o.StorageClass = class
	}
}

// DownloadOption 下载选项
type DownloadOption func(*downloadOptions)

type downloadOptions struct {
	RangeStart int64
	RangeEnd   int64
}

func WithRange(start, end int64) DownloadOption {
	return func(o *downloadOptions) {
		o.RangeStart = start
		o.RangeEnd = end
	}
}

// SignOption 签名选项
type SignOption func(*signOptions)

type signOptions struct {
	Expire      time.Duration
	Method      string
	ContentType string
	Headers     map[string]string
}

func WithExpire(d time.Duration) SignOption {
	return func(o *signOptions) {
		o.Expire = d
	}
}

func WithMethod(method string) SignOption {
	return func(o *signOptions) {
		o.Method = method
	}
}
