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
	client *cos.Client
	config *providers.Config
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
		client: c,
		config: cfg,
	}, nil
}

func (c *client) Upload(ctx context.Context, key string, reader io.Reader, opts ...providers.UploadOptionFunc) error {
	options := &providers.UploadOption{}
	for _, opt := range opts {
		opt(options)
	}

	opt := &cos.ObjectPutOptions{}
	if options.ContentType != "" {
		opt.ContentType = options.ContentType
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

func (c *client) InitMultipart(ctx context.Context, key string, opts ...providers.UploadOptionFunc) (*providers.MultipartUpload, error) {
	return nil, fmt.Errorf("multipart upload not implemented for cos")
}

func (c *client) UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...providers.UploadOptionFunc) (*providers.PartInfo, error) {
	return nil, fmt.Errorf("multipart upload not implemented for cos")
}

func (c *client) CompleteMultipart(ctx context.Context, uploadID string, parts []*providers.PartInfo, opts ...providers.UploadOptionFunc) error {
	return fmt.Errorf("multipart upload not implemented for cos")
}

func (c *client) AbortMultipart(ctx context.Context, uploadID string) error {
	return fmt.Errorf("multipart upload not implemented for cos")
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
