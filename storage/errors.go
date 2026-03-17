package storage

import (
	"errors"

	"github.com/tsopia/go-kit/storage/providers"
)

var (
	ErrMissingClient                        = errors.New("storage: client not configured")
	ErrInvalidConfig                        = errors.New("storage: invalid configuration")
	ErrUnsupportedType                      = errors.New("storage: unsupported storage type")
	ErrObjectNotFound                       = providers.ErrObjectNotFound
	ErrBucketNotFound                       = providers.ErrBucketNotFound
	ErrAccessDenied                         = providers.ErrAccessDenied
	ErrInvalidCredentials                   = errors.New("storage: invalid credentials")
	ErrMultipartNotFound                    = errors.New("storage: multipart upload not found")
	ErrPartAlreadyExist                     = errors.New("storage: part already uploaded")
	ErrInvalidDirectUploadRequest           = errors.New("storage: invalid direct upload request")
	ErrUnsupportedDirectUploadConstraint    = providers.ErrUnsupportedDirectUploadConstraint
	ErrDirectUploadAuthorizationUnsupported = errors.New("storage: direct upload authorization not supported")
)
