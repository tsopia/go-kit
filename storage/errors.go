package storage

import "errors"

var (
	ErrMissingClient                    = errors.New("storage: client not configured")
	ErrInvalidConfig                    = errors.New("storage: invalid configuration")
	ErrUnsupportedType                  = errors.New("storage: unsupported storage type")
	ErrObjectNotFound                   = errors.New("storage: object not found")
	ErrBucketNotFound                   = errors.New("storage: bucket not found")
	ErrAccessDenied                     = errors.New("storage: access denied")
	ErrInvalidCredentials               = errors.New("storage: invalid credentials")
	ErrMultipartNotFound                = errors.New("storage: multipart upload not found")
	ErrPartAlreadyExist                 = errors.New("storage: part already uploaded")
	ErrInvalidDirectUploadRequest       = errors.New("storage: invalid direct upload request")
	ErrDirectUploadAuthorizationUnsupported = errors.New("storage: direct upload authorization not supported")
)
