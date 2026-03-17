package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"
	"github.com/tsopia/go-kit/storage/providers"
)

const maxS3PostContentLength int64 = 9223372036854775807

// client S3 客户端实现
type client struct {
	client     *awss3.Client
	bucket     string
	config     *providers.Config
	uploadKeys map[string]string // uploadID -> key
	mu         sync.RWMutex
}

// NewClient 创建 S3 客户端
func NewClient(cfg *providers.Config) (providers.Client, error) {
	creds := credentials.NewStaticCredentialsProvider(
		cfg.AccessKeyID,
		cfg.AccessKeySecret,
		"",
	)

	awsCfg := aws.Config{
		Region:      cfg.Region,
		Credentials: creds,
	}

	c := awss3.NewFromConfig(awsCfg, func(o *awss3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})

	return &client{
		client:     c,
		bucket:     cfg.Bucket,
		config:     cfg,
		uploadKeys: make(map[string]string),
	}, nil
}

func (c *client) Upload(ctx context.Context, key string, reader io.Reader, opts ...providers.UploadOptionFunc) error {
	options := &providers.UploadOption{}
	for _, opt := range opts {
		opt(options)
	}

	input := &awss3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   reader,
	}

	if options.ContentType != "" {
		input.ContentType = aws.String(options.ContentType)
	}
	if len(options.Metadata) > 0 {
		input.Metadata = make(map[string]string, len(options.Metadata))
		for key, value := range options.Metadata {
			input.Metadata[key] = value
		}
	}

	_, err := c.client.PutObject(ctx, input)
	return err
}

func (c *client) Download(ctx context.Context, key string, opts ...providers.DownloadOptionFunc) (io.ReadCloser, error) {
	input := &awss3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	resp, err := c.client.GetObject(ctx, input)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c *client) Delete(ctx context.Context, key string) error {
	input := &awss3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	_, err := c.client.DeleteObject(ctx, input)
	return err
}

func (c *client) Exists(ctx context.Context, key string) (bool, error) {
	input := &awss3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	_, err := c.client.HeadObject(ctx, input)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (c *client) Stat(ctx context.Context, key string) (*providers.ObjectInfo, error) {
	input := &awss3.HeadObjectInput{
		Bucket:       aws.String(c.bucket),
		Key:          aws.String(key),
		ChecksumMode: types.ChecksumModeEnabled,
	}

	resp, err := c.client.HeadObject(ctx, input)
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

	presignClient := awss3.NewPresignClient(c.client)

	method := "GET"
	if options.Method != "" {
		method = options.Method
	}

	var req *v4.PresignedHTTPRequest
	var err error

	switch method {
	case "GET", "get":
		req, err = presignClient.PresignGetObject(ctx, &awss3.GetObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(key),
		}, func(o *awss3.PresignOptions) {
			o.Expires = expire
		})
	case "PUT", "put":
		req, err = presignClient.PresignPutObject(ctx, &awss3.PutObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(key),
		}, func(o *awss3.PresignOptions) {
			o.Expires = expire
		})
	default:
		return "", fmt.Errorf("unsupported method: %s", method)
	}

	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func (c *client) AuthorizeDirectUpload(ctx context.Context, req providers.DirectUploadRequest) (*providers.DirectUploadAuthorization, error) {
	mode, err := selectS3DirectUploadMode(req)
	if err != nil {
		return nil, err
	}

	switch mode {
	case providers.DirectUploadModePut:
		return c.authorizePutDirectUpload(ctx, req)
	case providers.DirectUploadModePost:
		return c.authorizePostDirectUpload(ctx, req)
	default:
		return nil, fmt.Errorf("%w: %s", providers.ErrUnsupportedDirectUploadMode, mode)
	}
}

func (c *client) InitMultipart(ctx context.Context, key string, opts ...providers.UploadOptionFunc) (*providers.MultipartUpload, error) {
	input := &awss3.CreateMultipartUploadInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	resp, err := c.client.CreateMultipartUpload(ctx, input)
	if err != nil {
		return nil, err
	}

	uploadID := *resp.UploadId

	// 存储 uploadID -> key 的映射
	c.mu.Lock()
	c.uploadKeys[uploadID] = key
	c.mu.Unlock()

	return &providers.MultipartUpload{
		UploadID: uploadID,
		Key:      key,
		Bucket:   c.bucket,
	}, nil
}

func (c *client) UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...providers.UploadOptionFunc) (*providers.PartInfo, error) {
	// 从 map 获取 key
	c.mu.RLock()
	key, ok := c.uploadKeys[uploadID]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("upload not found: %s", uploadID)
	}

	input := &awss3.UploadPartInput{
		Bucket:     aws.String(c.bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(int32(partNum)),
		Body:       reader,
	}

	resp, err := c.client.UploadPart(ctx, input)
	if err != nil {
		return nil, err
	}

	return &providers.PartInfo{
		PartNumber: partNum,
		ETag:       *resp.ETag,
	}, nil
}

func (c *client) CompleteMultipart(ctx context.Context, uploadID string, parts []*providers.PartInfo, opts ...providers.UploadOptionFunc) error {
	// 从 map 获取 key
	c.mu.RLock()
	key, ok := c.uploadKeys[uploadID]
	c.mu.RUnlock()
	if !ok {
		return fmt.Errorf("upload not found: %s", uploadID)
	}

	// 构建 CompletedParts
	var completedParts []types.CompletedPart
	for _, part := range parts {
		completedParts = append(completedParts, types.CompletedPart{
			PartNumber: aws.Int32(int32(part.PartNumber)),
			ETag:       aws.String(part.ETag),
		})
	}

	input := &awss3.CompleteMultipartUploadInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}

	_, err := c.client.CompleteMultipartUpload(ctx, input)
	if err != nil {
		return err
	}

	// 完成后删除映射
	c.mu.Lock()
	delete(c.uploadKeys, uploadID)
	c.mu.Unlock()

	return nil
}

func (c *client) AbortMultipart(ctx context.Context, uploadID string) error {
	// 从 map 获取 key
	c.mu.RLock()
	key, ok := c.uploadKeys[uploadID]
	c.mu.RUnlock()
	if !ok {
		return fmt.Errorf("upload not found: %s", uploadID)
	}

	input := &awss3.AbortMultipartUploadInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	}

	_, err := c.client.AbortMultipartUpload(ctx, input)
	if err != nil {
		return err
	}

	// 中止后删除映射
	c.mu.Lock()
	delete(c.uploadKeys, uploadID)
	c.mu.Unlock()

	return nil
}

func (c *client) DeleteBatch(ctx context.Context, keys []string) error {
	var objs []types.ObjectIdentifier
	for _, k := range keys {
		objs = append(objs, types.ObjectIdentifier{Key: aws.String(k)})
	}

	input := &awss3.DeleteObjectsInput{
		Bucket: aws.String(c.bucket),
		Delete: &types.Delete{
			Objects: objs,
		},
	}

	_, err := c.client.DeleteObjects(ctx, input)
	return err
}

func selectS3DirectUploadMode(req providers.DirectUploadRequest) (providers.DirectUploadMode, error) {
	switch req.Mode {
	case "", providers.DirectUploadModeAuto:
		if requiresS3PostDirectUpload(req) {
			return providers.DirectUploadModePost, nil
		}
		return providers.DirectUploadModePut, nil
	case providers.DirectUploadModePut:
		if requiresS3PostDirectUpload(req) {
			return "", fmt.Errorf("%w: size range requires post mode", providers.ErrUnsupportedDirectUploadConstraint)
		}
		return providers.DirectUploadModePut, nil
	case providers.DirectUploadModePost:
		return providers.DirectUploadModePost, nil
	default:
		return "", fmt.Errorf("%w: %s", providers.ErrUnsupportedDirectUploadMode, req.Mode)
	}
}

func requiresS3PostDirectUpload(req providers.DirectUploadRequest) bool {
	if req.Size == nil {
		return false
	}

	return req.Size.Exact == 0 && (req.Size.Min > 0 || req.Size.Max > 0)
}

func (c *client) authorizePutDirectUpload(ctx context.Context, req providers.DirectUploadRequest) (*providers.DirectUploadAuthorization, error) {
	if requiresS3PostDirectUpload(req) {
		return nil, fmt.Errorf("%w: size range requires post mode", providers.ErrUnsupportedDirectUploadConstraint)
	}

	expire := c.resolveDirectUploadExpire(req.Expire)
	basePresigned, err := awss3.NewPresignClient(c.client).PresignPutObject(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(req.ObjectKey),
	}, func(o *awss3.PresignOptions) {
		o.Expires = expire
	})
	if err != nil {
		return nil, fmt.Errorf("resolve put object url: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, basePresigned.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("build put object request: %w", err)
	}
	httpReq.URL.RawQuery = ""
	httpReq.URL.Opaque = "//" + httpReq.URL.Host + httpReq.URL.EscapedPath()
	if req.ContentType != "" {
		httpReq.Header.Set("Content-Type", req.ContentType)
	}
	for key, value := range req.Metadata {
		httpReq.Header.Set("X-Amz-Meta-"+key, value)
	}
	if req.Size != nil && req.Size.Exact > 0 {
		httpReq.ContentLength = req.Size.Exact
	}
	if err := applyS3PutHeaders(httpReq, req.Checksum); err != nil {
		return nil, err
	}

	creds := aws.Credentials{
		AccessKeyID:     c.config.AccessKeyID,
		SecretAccessKey: c.config.AccessKeySecret,
		Source:          "go-kit-storage",
	}
	presignedURL, signedHeaders, err := v4.NewSigner().PresignHTTP(
		ctx,
		creds,
		httpReq,
		"UNSIGNED-PAYLOAD",
		"s3",
		c.config.Region,
		time.Now(),
		func(o *v4.SignerOptions) {
			o.DisableHeaderHoisting = true
			o.DisableURIPathEscaping = true
		},
	)
	if err != nil {
		return nil, fmt.Errorf("presign put object: %w", err)
	}

	return &providers.DirectUploadAuthorization{
		Provider:    providers.TypeS3,
		Mode:        providers.DirectUploadModePut,
		ObjectKey:   req.ObjectKey,
		URL:         presignedURL,
		Method:      http.MethodPut,
		Headers:     flattenHeaders(signedHeaders),
		ExpiresAt:   time.Now().Add(expire),
		Constraints: buildDirectUploadConstraints(req),
	}, nil
}

func (c *client) authorizePostDirectUpload(ctx context.Context, req providers.DirectUploadRequest) (*providers.DirectUploadAuthorization, error) {
	if req.Checksum != nil {
		return nil, fmt.Errorf("%w: checksum requires put mode", providers.ErrUnsupportedDirectUploadConstraint)
	}

	expire := c.resolveDirectUploadExpire(req.Expire)
	conditions := make([]any, 0, 1+len(req.Metadata))
	formFields := make(map[string]string)

	if req.ContentType != "" {
		conditions = append(conditions, map[string]string{"Content-Type": req.ContentType})
		formFields["Content-Type"] = req.ContentType
	}
	for key, value := range req.Metadata {
		headerKey := "x-amz-meta-" + key
		conditions = append(conditions, map[string]string{headerKey: value})
		formFields[headerKey] = value
	}
	if req.Size != nil {
		minSize, maxSize, ok := s3PostContentLengthRange(*req.Size)
		if !ok {
			return nil, fmt.Errorf("%w: invalid size constraint for post mode", providers.ErrUnsupportedDirectUploadConstraint)
		}
		conditions = append(conditions, []any{"content-length-range", minSize, maxSize})
	}

	presigned, err := awss3.NewPresignClient(c.client).PresignPostObject(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(req.ObjectKey),
	}, func(o *awss3.PresignPostOptions) {
		o.Expires = expire
		o.Conditions = conditions
	})
	if err != nil {
		return nil, fmt.Errorf("presign post object: %w", err)
	}

	for key, value := range presigned.Values {
		formFields[key] = value
	}

	return &providers.DirectUploadAuthorization{
		Provider:    providers.TypeS3,
		Mode:        providers.DirectUploadModePost,
		ObjectKey:   req.ObjectKey,
		URL:         presigned.URL,
		Method:      http.MethodPost,
		FormFields:  formFields,
		ExpiresAt:   time.Now().Add(expire),
		Constraints: buildDirectUploadConstraints(req),
	}, nil
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

func applyS3PutHeaders(req *http.Request, checksum *providers.DirectUploadChecksum) error {
	if checksum == nil {
		return nil
	}

	switch checksum.Algorithm {
	case providers.DirectUploadChecksumMD5:
		req.Header.Set("Content-MD5", checksum.Value)
		return nil
	case providers.DirectUploadChecksumSHA256:
		req.Header.Set("X-Amz-Checksum-Sha256", checksum.Value)
		return nil
	default:
		return fmt.Errorf("%w: checksum algorithm %q", providers.ErrUnsupportedDirectUploadConstraint, checksum.Algorithm)
	}
}

func s3PostContentLengthRange(size providers.DirectUploadSize) (int64, int64, bool) {
	switch {
	case size.Exact > 0:
		return size.Exact, size.Exact, true
	case size.Min > 0 && size.Max > 0:
		return size.Min, size.Max, true
	case size.Min > 0:
		return size.Min, maxS3PostContentLength, true
	case size.Max > 0:
		return 0, size.Max, true
	default:
		return 0, 0, false
	}
}

func flattenHeaders(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	flattened := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		flattened[key] = values[0]
	}

	return flattened
}

func copyStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}

	return cloned
}

func buildDirectUploadConstraints(req providers.DirectUploadRequest) providers.DirectUploadConstraints {
	constraints := providers.DirectUploadConstraints{
		ContentType: req.ContentType,
		Metadata:    copyStringMap(req.Metadata),
	}
	if req.Size != nil {
		size := *req.Size
		constraints.Size = &size
	}
	if req.Checksum != nil {
		checksum := *req.Checksum
		constraints.Checksum = &checksum
	}

	return constraints
}

func buildObjectInfo(key string, resp *awss3.HeadObjectOutput) *providers.ObjectInfo {
	if resp == nil {
		return nil
	}

	info := &providers.ObjectInfo{
		Key:       key,
		Metadata:  copyStringMap(resp.Metadata),
		Checksums: buildObjectChecksums(resp),
	}
	if resp.ContentLength != nil {
		info.Size = *resp.ContentLength
	}
	if resp.LastModified != nil {
		info.LastModified = *resp.LastModified
	}
	if resp.ETag != nil {
		info.ETag = *resp.ETag
	}
	if resp.ContentType != nil {
		info.ContentType = *resp.ContentType
	}

	return info
}

func buildObjectChecksums(resp *awss3.HeadObjectOutput) map[string]string {
	checksums := map[string]string{}
	if resp.ChecksumSHA256 != nil {
		checksums[string(providers.DirectUploadChecksumSHA256)] = *resp.ChecksumSHA256
	}
	if resp.ChecksumSHA1 != nil {
		checksums["sha1"] = *resp.ChecksumSHA1
	}
	if resp.ChecksumCRC32 != nil {
		checksums["crc32"] = *resp.ChecksumCRC32
	}
	if resp.ChecksumCRC32C != nil {
		checksums["crc32c"] = *resp.ChecksumCRC32C
	}
	if resp.ChecksumCRC64NVME != nil {
		checksums["crc64nvme"] = *resp.ChecksumCRC64NVME
	}
	if len(checksums) == 0 {
		return nil
	}

	return checksums
}

func normalizeStatError(err error) error {
	if err == nil {
		return nil
	}

	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return providers.ErrObjectNotFound
	}

	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return providers.ErrObjectNotFound
	}

	var noSuchBucket *types.NoSuchBucket
	if errors.As(err, &noSuchBucket) {
		return providers.ErrBucketNotFound
	}

	var accessDenied *types.AccessDenied
	if errors.As(err, &accessDenied) {
		return providers.ErrAccessDenied
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey":
			return providers.ErrObjectNotFound
		case "NoSuchBucket":
			return providers.ErrBucketNotFound
		case "AccessDenied":
			return providers.ErrAccessDenied
		}
	}

	return err
}
