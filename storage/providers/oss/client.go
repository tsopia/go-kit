package oss

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/tsopia/go-kit/storage/providers"
)

// client OSS 客户端实现
type client struct {
	client *oss.Client
	bucket string
	config *providers.Config
}

// NewClient 创建 OSS 客户端
func NewClient(cfg *providers.Config) (providers.Client, error) {
	// 创建 OSS 配置
	ossCfg := oss.NewConfig().
		WithRegion(cfg.Region).
		WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret),
		)

	if cfg.Endpoint != "" {
		ossCfg = ossCfg.WithEndpoint(cfg.Endpoint)
	}

	c := oss.NewClient(ossCfg)

	return &client{
		client: c,
		bucket: cfg.Bucket,
		config: cfg,
	}, nil
}

func (c *client) Upload(ctx context.Context, key string, reader io.Reader, opts ...providers.UploadOptionFunc) error {
	options := &providers.UploadOption{}
	for _, opt := range opts {
		opt(options)
	}

	req := &oss.PutObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(key),
		Body:   reader,
	}

	if options.ContentType != "" {
		req.ContentType = oss.Ptr(options.ContentType)
	}
	if len(options.Metadata) > 0 {
		req.Metadata = make(map[string]string, len(options.Metadata))
		for key, value := range options.Metadata {
			req.Metadata[key] = value
		}
	}

	_, err := c.client.PutObject(ctx, req)
	return err
}

func (c *client) Download(ctx context.Context, key string, opts ...providers.DownloadOptionFunc) (io.ReadCloser, error) {
	req := &oss.GetObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(key),
	}

	resp, err := c.client.GetObject(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c *client) Delete(ctx context.Context, key string) error {
	req := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(key),
	}

	_, err := c.client.DeleteObject(ctx, req)
	return err
}

func (c *client) Exists(ctx context.Context, key string) (bool, error) {
	req := &oss.GetObjectMetaRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(key),
	}

	_, err := c.client.GetObjectMeta(ctx, req)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (c *client) Stat(ctx context.Context, key string) (*providers.ObjectInfo, error) {
	req := &oss.HeadObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(key),
	}

	resp, err := c.client.HeadObject(ctx, req)
	if err != nil {
		return nil, normalizeStatError(err)
	}

	return buildObjectInfo(key, resp), nil
}

func (c *client) SignedURL(ctx context.Context, key string, expire time.Duration, opts ...providers.SignOptionFunc) (string, error) {
	options := &providers.SignOption{}
	for _, opt := range opts {
		opt(options)
	}

	if expire == 0 {
		expire = c.config.DefaultSignExpire
		if expire == 0 {
			expire = 15 * time.Minute
		}
	}

	req := &oss.GetObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(key),
	}

	result, err := c.client.Presign(ctx, req, oss.PresignExpires(expire))
	if err != nil {
		return "", err
	}

	return result.URL, nil
}

func (c *client) AuthorizeDirectUpload(ctx context.Context, req providers.DirectUploadRequest) (*providers.DirectUploadAuthorization, error) {
	mode, err := selectOSSDirectUploadMode(req)
	if err != nil {
		return nil, err
	}

	if mode != providers.DirectUploadModePut {
		return nil, fmt.Errorf("%w: %s", providers.ErrUnsupportedDirectUploadMode, mode)
	}

	expire := c.resolveDirectUploadExpire(req.Expire)
	putReq := &oss.PutObjectRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(req.ObjectKey),
	}
	if req.ContentType != "" {
		putReq.ContentType = oss.Ptr(req.ContentType)
	}
	if len(req.Metadata) > 0 {
		putReq.Metadata = copyOSSStringMap(req.Metadata)
	}
	if req.Size != nil && req.Size.Exact > 0 {
		putReq.ContentLength = oss.Ptr(req.Size.Exact)
	}
	if err := applyOSSChecksum(putReq, req.Checksum); err != nil {
		return nil, err
	}

	result, err := c.client.Presign(ctx, putReq, oss.PresignExpires(expire))
	if err != nil {
		return nil, fmt.Errorf("presign put object: %w", err)
	}

	return &providers.DirectUploadAuthorization{
		Provider:  providers.TypeOSS,
		Mode:      providers.DirectUploadModePut,
		ObjectKey: req.ObjectKey,
		URL:       result.URL,
		Method:    result.Method,
		Headers:   copyOSSStringMap(result.SignedHeaders),
		ExpiresAt: result.Expiration,
		Constraints: providers.DirectUploadConstraints{
			ContentType: req.ContentType,
			Metadata:    copyOSSStringMap(req.Metadata),
			Size:        copyOSSSize(req.Size),
			Checksum:    copyOSSChecksum(req.Checksum),
		},
	}, nil
}

func (c *client) InitMultipart(ctx context.Context, key string, opts ...providers.UploadOptionFunc) (*providers.MultipartUpload, error) {
	req := &oss.InitiateMultipartUploadRequest{
		Bucket: oss.Ptr(c.bucket),
		Key:    oss.Ptr(key),
	}

	resp, err := c.client.InitiateMultipartUpload(ctx, req)
	if err != nil {
		return nil, err
	}

	return &providers.MultipartUpload{
		UploadID: *resp.UploadId,
		Key:      key,
		Bucket:   c.bucket,
	}, nil
}

func (c *client) UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...providers.UploadOptionFunc) (*providers.PartInfo, error) {
	req := &oss.UploadPartRequest{
		Bucket:     oss.Ptr(c.bucket),
		Key:        oss.Ptr(uploadID),
		UploadId:   oss.Ptr(uploadID),
		PartNumber: int32(partNum),
		Body:       reader,
	}

	resp, err := c.client.UploadPart(ctx, req)
	if err != nil {
		return nil, err
	}

	return &providers.PartInfo{
		PartNumber: partNum,
		ETag:       *resp.ETag,
	}, nil
}

func (c *client) CompleteMultipart(ctx context.Context, uploadID string, parts []*providers.PartInfo, opts ...providers.UploadOptionFunc) error {
	ossParts := make([]oss.UploadPart, len(parts))
	for i, p := range parts {
		ossParts[i] = oss.UploadPart{
			PartNumber: int32(p.PartNumber),
			ETag:       oss.Ptr(p.ETag),
		}
	}

	req := &oss.CompleteMultipartUploadRequest{
		Bucket:   oss.Ptr(c.bucket),
		Key:      oss.Ptr(uploadID),
		UploadId: oss.Ptr(uploadID),
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
			Parts: ossParts,
		},
	}

	_, err := c.client.CompleteMultipartUpload(ctx, req)
	return err
}

func (c *client) AbortMultipart(ctx context.Context, uploadID string) error {
	return fmt.Errorf("abort multipart not implemented")
}

func (c *client) DeleteBatch(ctx context.Context, keys []string) error {
	objs := make([]oss.DeleteObject, len(keys))
	for i, k := range keys {
		objs[i] = oss.DeleteObject{Key: oss.Ptr(k)}
	}

	req := &oss.DeleteMultipleObjectsRequest{
		Bucket:  oss.Ptr(c.bucket),
		Objects: objs,
	}

	_, err := c.client.DeleteMultipleObjects(ctx, req)
	return err
}

func selectOSSDirectUploadMode(req providers.DirectUploadRequest) (providers.DirectUploadMode, error) {
	switch req.Mode {
	case "", providers.DirectUploadModeAuto, providers.DirectUploadModePut:
		if req.Size != nil && req.Size.Exact == 0 && (req.Size.Min > 0 || req.Size.Max > 0) {
			return "", fmt.Errorf("%w: size range is not supported by oss put presign", providers.ErrUnsupportedDirectUploadConstraint)
		}
		return providers.DirectUploadModePut, nil
	case providers.DirectUploadModePost:
		return "", fmt.Errorf("%w: post mode is not supported by oss provider", providers.ErrUnsupportedDirectUploadMode)
	default:
		return "", fmt.Errorf("%w: %s", providers.ErrUnsupportedDirectUploadMode, req.Mode)
	}
}

func (c *client) resolveDirectUploadExpire(expire time.Duration) time.Duration {
	if expire == 0 {
		expire = c.config.DefaultSignExpire
		if expire == 0 {
			expire = 15 * time.Minute
		}
	}

	return expire
}

func applyOSSChecksum(req *oss.PutObjectRequest, checksum *providers.DirectUploadChecksum) error {
	if checksum == nil {
		return nil
	}

	switch checksum.Algorithm {
	case providers.DirectUploadChecksumMD5:
		req.ContentMD5 = oss.Ptr(checksum.Value)
		return nil
	default:
		return fmt.Errorf("%w: checksum algorithm %q", providers.ErrUnsupportedDirectUploadConstraint, checksum.Algorithm)
	}
}

func copyOSSStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}

	return cloned
}

func copyOSSSize(size *providers.DirectUploadSize) *providers.DirectUploadSize {
	if size == nil {
		return nil
	}

	cloned := *size
	return &cloned
}

func copyOSSChecksum(checksum *providers.DirectUploadChecksum) *providers.DirectUploadChecksum {
	if checksum == nil {
		return nil
	}

	cloned := *checksum
	return &cloned
}

func buildObjectInfo(key string, resp *oss.HeadObjectResult) *providers.ObjectInfo {
	if resp == nil {
		return nil
	}

	contentType := ""
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	}

	info := &providers.ObjectInfo{
		Key:         key,
		Size:        resp.ContentLength,
		ContentType: contentType,
		Metadata:    copyOSSStringMap(resp.Metadata),
	}
	if resp.LastModified != nil {
		info.LastModified = *resp.LastModified
	}
	if resp.ETag != nil {
		info.ETag = *resp.ETag
	}
	if resp.ContentMD5 != nil && *resp.ContentMD5 != "" {
		info.Checksums = map[string]string{
			string(providers.DirectUploadChecksumMD5): *resp.ContentMD5,
		}
	}

	return info
}

func normalizeStatError(err error) error {
	if err == nil {
		return nil
	}

	var serviceErr *oss.ServiceError
	if errors.As(err, &serviceErr) {
		switch serviceErr.StatusCode {
		case 404:
			if serviceErr.Code == "NoSuchBucket" {
				return providers.ErrBucketNotFound
			}
			return providers.ErrObjectNotFound
		case 403:
			return providers.ErrAccessDenied
		}
	}

	return err
}
