package storage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/tsopia/go-kit/storage/providers"
)

// AuthorizeDirectUpload 生成客户端直传授权结果。
func AuthorizeDirectUpload(ctx context.Context, req DirectUploadRequest) (*DirectUploadAuthorization, error) {
	return AuthorizeDirectUploadWithClient(ctx, GetClient(), req)
}

// AuthorizeDirectUploadWithClient 使用指定客户端生成客户端直传授权结果。
func AuthorizeDirectUploadWithClient(ctx context.Context, c Client, req DirectUploadRequest) (*DirectUploadAuthorization, error) {
	if c == nil {
		return nil, ErrMissingClient
	}

	normalized, err := normalizeDirectUploadRequest(req)
	if err != nil {
		return nil, err
	}

	authorizer, ok := c.(providers.DirectUploadAuthorizer)
	if !ok {
		return nil, ErrDirectUploadAuthorizationUnsupported
	}

	return authorizer.AuthorizeDirectUpload(ctx, normalized)
}

// VerifyDirectUploadObject 校验客户端直传后的对象状态。
func VerifyDirectUploadObject(ctx context.Context, req DirectUploadVerificationRequest) (*DirectUploadVerificationResult, error) {
	return VerifyDirectUploadObjectWithClient(ctx, GetClient(), req)
}

// VerifyDirectUploadObjectWithClient 使用指定客户端校验客户端直传后的对象状态。
func VerifyDirectUploadObjectWithClient(ctx context.Context, c Client, req DirectUploadVerificationRequest) (*DirectUploadVerificationResult, error) {
	if c == nil {
		return nil, ErrMissingClient
	}

	normalizedReq, err := normalizeDirectUploadVerificationRequest(req)
	if err != nil {
		return nil, err
	}

	info, err := c.Stat(ctx, normalizedReq.ObjectKey)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			return &DirectUploadVerificationResult{
				Exists:  false,
				Matched: false,
			}, nil
		}
		return nil, fmt.Errorf("stat object: %w", err)
	}
	if info == nil {
		return &DirectUploadVerificationResult{
			Exists:  false,
			Matched: false,
		}, nil
	}

	result := &DirectUploadVerificationResult{
		Exists:  true,
		Matched: true,
		Object:  info,
	}

	appendMismatch := func(field, expected, actual string) {
		result.Matched = false
		result.Mismatches = append(result.Mismatches, DirectUploadMismatch{
			Field:    field,
			Expected: expected,
			Actual:   actual,
		})
	}

	if normalizedReq.ObjectKey != "" && info.Key != normalizedReq.ObjectKey {
		appendMismatch("object_key", normalizedReq.ObjectKey, info.Key)
	}

	if normalizedReq.ContentType != "" && info.ContentType != normalizedReq.ContentType {
		appendMismatch("content_type", normalizedReq.ContentType, info.ContentType)
	}

	if normalizedReq.Size != nil {
		switch {
		case normalizedReq.Size.Exact > 0 && info.Size != normalizedReq.Size.Exact:
			appendMismatch("size", strconv.FormatInt(normalizedReq.Size.Exact, 10), strconv.FormatInt(info.Size, 10))
		case normalizedReq.Size.Min > 0 && info.Size < normalizedReq.Size.Min:
			appendMismatch("size_min", strconv.FormatInt(normalizedReq.Size.Min, 10), strconv.FormatInt(info.Size, 10))
		case normalizedReq.Size.Max > 0 && info.Size > normalizedReq.Size.Max:
			appendMismatch("size_max", strconv.FormatInt(normalizedReq.Size.Max, 10), strconv.FormatInt(info.Size, 10))
		}
	}

	for key, expected := range normalizedReq.Metadata {
		actual := ""
		if info.Metadata != nil {
			actual = info.Metadata[key]
		}
		if actual != expected {
			appendMismatch("metadata."+key, expected, actual)
		}
	}

	if normalizedReq.Checksum != nil {
		algorithm := string(normalizedReq.Checksum.Algorithm)
		actual := ""
		if info.Checksums != nil {
			actual = info.Checksums[algorithm]
		}
		if actual == "" {
			return nil, fmt.Errorf("%w: checksum %q is not available from provider metadata", ErrUnsupportedDirectUploadConstraint, algorithm)
		}
		if actual != normalizedReq.Checksum.Value {
			appendMismatch("checksum."+algorithm, normalizedReq.Checksum.Value, actual)
		}
	}

	return result, nil
}

func normalizeDirectUploadVerificationRequest(req DirectUploadVerificationRequest) (DirectUploadVerificationRequest, error) {
	req.ObjectKey = strings.TrimSpace(req.ObjectKey)
	if req.ObjectKey == "" {
		return DirectUploadVerificationRequest{}, fmt.Errorf("%w: object key is required", ErrInvalidDirectUploadRequest)
	}

	req.ContentType = strings.TrimSpace(req.ContentType)

	if req.Size != nil {
		normalizedSize, err := normalizeDirectUploadSize(*req.Size)
		if err != nil {
			return DirectUploadVerificationRequest{}, err
		}
		req.Size = &normalizedSize
	}

	if req.Metadata != nil {
		normalizedMetadata, err := normalizeDirectUploadMetadata(req.Metadata)
		if err != nil {
			return DirectUploadVerificationRequest{}, err
		}
		req.Metadata = normalizedMetadata
	}

	if req.Checksum != nil {
		normalizedChecksum, err := normalizeDirectUploadChecksum(*req.Checksum)
		if err != nil {
			return DirectUploadVerificationRequest{}, err
		}
		req.Checksum = &normalizedChecksum
	}

	return req, nil
}

func normalizeDirectUploadRequest(req DirectUploadRequest) (DirectUploadRequest, error) {
	req.ObjectKey = strings.TrimSpace(req.ObjectKey)
	if req.ObjectKey == "" {
		return DirectUploadRequest{}, fmt.Errorf("%w: object key is required", ErrInvalidDirectUploadRequest)
	}

	if req.Expire < 0 {
		return DirectUploadRequest{}, fmt.Errorf("%w: expire must not be negative", ErrInvalidDirectUploadRequest)
	}

	req.ContentType = strings.TrimSpace(req.ContentType)
	normalizedMode, err := normalizeDirectUploadMode(req.Mode)
	if err != nil {
		return DirectUploadRequest{}, err
	}
	req.Mode = normalizedMode

	if req.Size != nil {
		normalizedSize, err := normalizeDirectUploadSize(*req.Size)
		if err != nil {
			return DirectUploadRequest{}, err
		}
		req.Size = &normalizedSize
	}

	if req.Metadata != nil {
		normalizedMetadata, err := normalizeDirectUploadMetadata(req.Metadata)
		if err != nil {
			return DirectUploadRequest{}, err
		}
		req.Metadata = normalizedMetadata
	}

	if req.Checksum != nil {
		normalizedChecksum, err := normalizeDirectUploadChecksum(*req.Checksum)
		if err != nil {
			return DirectUploadRequest{}, err
		}
		req.Checksum = &normalizedChecksum
	}

	return req, nil
}

func normalizeDirectUploadMode(mode DirectUploadMode) (DirectUploadMode, error) {
	switch DirectUploadMode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case "", DirectUploadModeAuto:
		return DirectUploadModeAuto, nil
	case DirectUploadModePut:
		return DirectUploadModePut, nil
	case DirectUploadModePost:
		return DirectUploadModePost, nil
	default:
		return "", fmt.Errorf("%w: unsupported direct upload mode %q", ErrInvalidDirectUploadRequest, mode)
	}
}

func normalizeDirectUploadSize(size DirectUploadSize) (DirectUploadSize, error) {
	if size.Exact > 0 && (size.Min != 0 || size.Max != 0) {
		return DirectUploadSize{}, fmt.Errorf("%w: exact size cannot be combined with min/max", ErrInvalidDirectUploadRequest)
	}

	if size.Exact < 0 || size.Min < 0 || size.Max < 0 {
		return DirectUploadSize{}, fmt.Errorf("%w: size values must not be negative", ErrInvalidDirectUploadRequest)
	}

	if size.Exact == 0 && size.Min == 0 && size.Max == 0 {
		return DirectUploadSize{}, fmt.Errorf("%w: size constraint requires exact or min/max", ErrInvalidDirectUploadRequest)
	}

	if size.Exact == 0 && size.Min > 0 && size.Max > 0 && size.Min > size.Max {
		return DirectUploadSize{}, fmt.Errorf("%w: min size must be less than or equal to max size", ErrInvalidDirectUploadRequest)
	}

	return size, nil
}

func normalizeDirectUploadMetadata(metadata map[string]string) (map[string]string, error) {
	normalized := make(map[string]string, len(metadata))
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			return nil, fmt.Errorf("%w: metadata key and value are required", ErrInvalidDirectUploadRequest)
		}
		if existing, ok := normalized[trimmedKey]; ok && existing != trimmedValue {
			return nil, fmt.Errorf("%w: duplicate metadata key %q", ErrInvalidDirectUploadRequest, trimmedKey)
		}
		normalized[trimmedKey] = trimmedValue
	}

	return normalized, nil
}

func normalizeDirectUploadChecksum(checksum DirectUploadChecksum) (DirectUploadChecksum, error) {
	checksum.Algorithm = DirectUploadChecksumAlgorithm(strings.ToLower(strings.TrimSpace(string(checksum.Algorithm))))
	checksum.Value = strings.TrimSpace(checksum.Value)
	if checksum.Algorithm == "" || checksum.Value == "" {
		return DirectUploadChecksum{}, fmt.Errorf("%w: checksum algorithm and value are required", ErrInvalidDirectUploadRequest)
	}

	switch checksum.Algorithm {
	case DirectUploadChecksumMD5, DirectUploadChecksumSHA256:
		return checksum, nil
	default:
		return DirectUploadChecksum{}, fmt.Errorf("%w: unsupported checksum algorithm %q", ErrInvalidDirectUploadRequest, checksum.Algorithm)
	}
}
