package s3

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/tsopia/go-kit/storage/providers"
)

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
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	resp, err := c.client.HeadObject(ctx, input)
	if err != nil {
		return nil, err
	}

	return &providers.ObjectInfo{
		Key:          key,
		Size:         *resp.ContentLength,
		LastModified: *resp.LastModified,
		ETag:         *resp.ETag,
		ContentType:  *resp.ContentType,
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
