package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/tsopia/go-kit/storage/providers"
)

// client S3 客户端实现
type client struct {
	client *awss3.Client
	bucket string
	config *providers.Config
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

	input := &awss3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   reader,
	}

	if options.ContentType != "" {
		input.ContentType = aws.String(options.ContentType)
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
	return "", fmt.Errorf("signed url not implemented for s3")
}

func (c *client) InitMultipart(ctx context.Context, key string, opts ...providers.UploadOptionFunc) (*providers.MultipartUpload, error) {
	return nil, fmt.Errorf("multipart upload not implemented for s3")
}

func (c *client) UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...providers.UploadOptionFunc) (*providers.PartInfo, error) {
	return nil, fmt.Errorf("multipart upload not implemented for s3")
}

func (c *client) CompleteMultipart(ctx context.Context, uploadID string, parts []*providers.PartInfo, opts ...providers.UploadOptionFunc) error {
	return fmt.Errorf("multipart upload not implemented for s3")
}

func (c *client) AbortMultipart(ctx context.Context, uploadID string) error {
	return fmt.Errorf("multipart upload not implemented for s3")
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
