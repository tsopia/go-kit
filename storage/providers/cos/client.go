package cos

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/tsopia/go-kit/storage/providers"
)

// client COS 客户端实现
type client struct {
	client     *cos.Client
	config     *providers.Config
	uploadKeys map[string]string // uploadID -> key 映射，用于分片上传
}

// NewClient 创建 COS 客户端
func NewClient(cfg *providers.Config) (providers.Client, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region))
	if err != nil {
		return nil, err
	}

	if cfg.Endpoint != "" {
		u, err = url.Parse(cfg.Endpoint)
		if err != nil {
			return nil, err
		}
	}

	baseURL := &cos.BaseURL{BucketURL: u}

	// 使用 AuthorizationTransport 设置密钥
	authTransport := &cos.AuthorizationTransport{
		SecretID:  cfg.AccessKeyID,
		SecretKey: cfg.AccessKeySecret,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
		},
	}

	httpClient := &http.Client{
		Transport: authTransport,
		Timeout:   30 * time.Second,
	}

	c := cos.NewClient(baseURL, httpClient)

	return &client{
		client:     c,
		config:     cfg,
		uploadKeys: make(map[string]string),
	}, nil
}

func (c *client) Upload(ctx context.Context, key string, reader io.Reader, opts ...providers.UploadOptionFunc) error {
	options := &providers.UploadOption{}
	for _, opt := range opts {
		opt(options)
	}

	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{},
	}
	if options.ContentType != "" {
		opt.ContentType = options.ContentType
	}
	if len(options.Metadata) > 0 {
		header := http.Header{}
		for key, value := range options.Metadata {
			header.Set("x-cos-meta-"+key, value)
		}
		opt.XCosMetaXXX = &header
	}

	_, err := c.client.Object.Put(ctx, key, reader, opt)
	return err
}

func (c *client) Download(ctx context.Context, key string, opts ...providers.DownloadOptionFunc) (io.ReadCloser, error) {
	resp, err := c.client.Object.Get(ctx, key, nil)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c *client) Delete(ctx context.Context, key string) error {
	_, err := c.client.Object.Delete(ctx, key)
	return err
}

func (c *client) Exists(ctx context.Context, key string) (bool, error) {
	ok, err := c.client.Object.IsExist(ctx, key)
	return ok, err
}

func (c *client) Stat(ctx context.Context, key string) (*providers.ObjectInfo, error) {
	resp, err := c.client.Object.Head(ctx, key, nil)
	if err != nil {
		return nil, err
	}

	// 从 Header 解析 Last-Modified
	lastModified := time.Time{}
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		lastModified, _ = http.ParseTime(lm)
	}

	return &providers.ObjectInfo{
		Key:          key,
		Size:         resp.ContentLength,
		LastModified: lastModified,
		ETag:         resp.Header.Get("ETag"),
		ContentType:  resp.Header.Get("Content-Type"),
	}, nil
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

	// 根据 Method 决定 HTTP 方法，默认为 GET
	method := http.MethodGet
	if options.Method != "" {
		method = options.Method
	}

	// 生成预签名 URL
	presignedURL, err := c.client.Object.GetPresignedURL(ctx, method, key, c.config.AccessKeyID, c.config.AccessKeySecret, expire, nil)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

func (c *client) AuthorizeDirectUpload(ctx context.Context, req providers.DirectUploadRequest) (*providers.DirectUploadAuthorization, error) {
	mode, err := selectCOSDirectUploadMode(req)
	if err != nil {
		return nil, err
	}

	if mode != providers.DirectUploadModePut {
		return nil, fmt.Errorf("%w: %s", providers.ErrUnsupportedDirectUploadMode, mode)
	}

	expire := c.resolveDirectUploadExpire(req.Expire)
	headers := http.Header{}
	if req.ContentType != "" {
		headers.Set("Content-Type", req.ContentType)
	}
	for key, value := range req.Metadata {
		headers.Set("x-cos-meta-"+key, value)
	}
	if req.Size != nil && req.Size.Exact > 0 {
		headers.Set("Content-Length", fmt.Sprintf("%d", req.Size.Exact))
	}
	if err := applyCOSChecksum(&headers, req.Checksum); err != nil {
		return nil, err
	}

	presignedURL, err := c.client.Object.GetPresignedURL2(ctx, http.MethodPut, req.ObjectKey, expire, &cos.PresignedURLOptions{
		Header: &headers,
	}, true)
	if err != nil {
		return nil, fmt.Errorf("presign put object: %w", err)
	}

	return &providers.DirectUploadAuthorization{
		Provider:  providers.TypeCOS,
		Mode:      providers.DirectUploadModePut,
		ObjectKey: req.ObjectKey,
		URL:       presignedURL.String(),
		Method:    http.MethodPut,
		Headers:   flattenCOSHeaders(headers),
		ExpiresAt: time.Now().Add(expire),
		Constraints: providers.DirectUploadConstraints{
			ContentType: req.ContentType,
			Metadata:    copyCOSStringMap(req.Metadata),
			Size:        copyCOSSize(req.Size),
			Checksum:    copyCOSChecksum(req.Checksum),
		},
	}, nil
}

func (c *client) InitMultipart(ctx context.Context, key string, opts ...providers.UploadOptionFunc) (*providers.MultipartUpload, error) {
	// 处理上传选项
	options := &providers.UploadOption{}
	for _, opt := range opts {
		opt(options)
	}

	// 构建初始化选项
	var initOpt *cos.InitiateMultipartUploadOptions
	if options.ContentType != "" {
		initOpt = &cos.InitiateMultipartUploadOptions{
			ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
				ContentType: options.ContentType,
			},
		}
	}

	// 调用 COS SDK 初始化分片上传
	resp, _, err := c.client.Object.InitiateMultipartUpload(ctx, key, initOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	// 保存 uploadID 到 key 的映射，用于后续操作
	c.uploadKeys[resp.UploadID] = key

	return &providers.MultipartUpload{
		UploadID: resp.UploadID,
		Key:      key,
		Bucket:   c.config.Bucket,
	}, nil
}

func (c *client) UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...providers.UploadOptionFunc) (*providers.PartInfo, error) {
	// 从映射中获取对应的 key
	key, ok := c.uploadKeys[uploadID]
	if !ok {
		return nil, fmt.Errorf("upload ID not found: %s", uploadID)
	}

	// 处理上传选项
	options := &providers.UploadOption{}
	for _, opt := range opts {
		opt(options)
	}

	// 构建上传选项
	uploadOpt := &cos.ObjectUploadPartOptions{}

	resp, err := c.client.Object.UploadPart(ctx, key, uploadID, partNum, reader, uploadOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to upload part %d: %w", partNum, err)
	}

	// 从响应头中获取 ETag
	etag := resp.Header.Get("ETag")

	return &providers.PartInfo{
		PartNumber: partNum,
		ETag:       etag,
	}, nil
}

func (c *client) CompleteMultipart(ctx context.Context, uploadID string, parts []*providers.PartInfo, opts ...providers.UploadOptionFunc) error {
	// 从映射中获取对应的 key
	key, ok := c.uploadKeys[uploadID]
	if !ok {
		return fmt.Errorf("upload ID not found: %s", uploadID)
	}

	// 转换 parts 为 COS SDK 需要的格式
	cosParts := make([]cos.Object, len(parts))
	for i, p := range parts {
		cosParts[i] = cos.Object{
			ETag:       p.ETag,
			PartNumber: p.PartNumber,
		}
	}

	// 构建完成选项，包含 parts 信息
	completeOpt := &cos.CompleteMultipartUploadOptions{
		Parts: cosParts,
	}

	// 调用 COS SDK 完成分片上传
	_, _, err := c.client.Object.CompleteMultipartUpload(ctx, key, uploadID, completeOpt)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	// 清理映射
	delete(c.uploadKeys, uploadID)

	return nil
}

func (c *client) AbortMultipart(ctx context.Context, uploadID string) error {
	// 从映射中获取对应的 key
	key, ok := c.uploadKeys[uploadID]
	if !ok {
		return fmt.Errorf("upload ID not found: %s", uploadID)
	}

	// 调用 COS SDK 中止分片上传
	_, err := c.client.Object.AbortMultipartUpload(ctx, key, uploadID)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	// 清理映射
	delete(c.uploadKeys, uploadID)

	return nil
}

func (c *client) DeleteBatch(ctx context.Context, keys []string) error {
	var objs []cos.Object
	for _, k := range keys {
		objs = append(objs, cos.Object{Key: k})
	}

	opt := &cos.ObjectDeleteMultiOptions{
		Objects: objs,
	}

	_, _, err := c.client.Object.DeleteMulti(ctx, opt)
	return err
}

func selectCOSDirectUploadMode(req providers.DirectUploadRequest) (providers.DirectUploadMode, error) {
	switch req.Mode {
	case "", providers.DirectUploadModeAuto, providers.DirectUploadModePut:
		if req.Size != nil && req.Size.Exact == 0 && (req.Size.Min > 0 || req.Size.Max > 0) {
			return "", fmt.Errorf("%w: size range is not supported by cos put presign", providers.ErrUnsupportedDirectUploadConstraint)
		}
		return providers.DirectUploadModePut, nil
	case providers.DirectUploadModePost:
		return "", fmt.Errorf("%w: post mode is not supported by cos provider", providers.ErrUnsupportedDirectUploadMode)
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

func applyCOSChecksum(headers *http.Header, checksum *providers.DirectUploadChecksum) error {
	if checksum == nil {
		return nil
	}

	switch checksum.Algorithm {
	case providers.DirectUploadChecksumMD5:
		headers.Set("Content-MD5", checksum.Value)
		return nil
	default:
		return fmt.Errorf("%w: checksum algorithm %q", providers.ErrUnsupportedDirectUploadConstraint, checksum.Algorithm)
	}
}

func flattenCOSHeaders(headers http.Header) map[string]string {
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

func copyCOSStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}

	return cloned
}

func copyCOSSize(size *providers.DirectUploadSize) *providers.DirectUploadSize {
	if size == nil {
		return nil
	}

	cloned := *size
	return &cloned
}

func copyCOSChecksum(checksum *providers.DirectUploadChecksum) *providers.DirectUploadChecksum {
	if checksum == nil {
		return nil
	}

	cloned := *checksum
	return &cloned
}
