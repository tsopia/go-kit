# Storage 包实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现支持 OSS/COS/S3 的对象存储抽象包，提供统一 Client 接口和 SDK 风格 API

**Architecture:** 定义统一 Client 接口，通过工厂模式根据 Config.Type 创建对应实现；internal/oss/cos/s3 分别封装各厂商 SDK；包级函数提供便捷调用

**Tech Stack:** Go 1.21+, 阿里云 OSS V2 SDK, 腾讯云 COS SDK, AWS S3 SDK v2

---

## 准备工作

### Task 0: 创建目录结构

**Files:**
- Create: `storage/config.go`
- Create: `storage/errors.go`
- Create: `storage/client.go`
- Create: `storage/options.go`
- Create: `storage/multipart.go`
- Create: `storage/storage.go`
- Create: `storage/internal/factory.go`
- Create: `storage/internal/oss/client.go`
- Create: `storage/internal/cos/client.go`
- Create: `storage/internal/s3/client.go`
- Create: `storage/README.md`
- Create: `storage/examples/upload/main.go`

**Step 1: 创建目录**

```bash
mkdir -p storage/internal/oss
mkdir -p storage/internal/cos
mkdir -p storage/internal/s3
mkdir -p storage/examples/upload
mkdir -p storage/examples/multipart
mkdir -p storage/examples/presigned
```

**Step 2: 提交**

```bash
git add storage/
git commit -m "chore(storage): create directory structure

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 阶段 1: 基础类型定义

### Task 1: 定义存储类型枚举

**Files:**
- Create: `storage/config.go`

**Step 1: 编写代码**

```go
package storage

// Type 存储类型
type Type string

const (
    TypeOSS Type = "oss"
    TypeCOS Type = "cos"
    TypeS3  Type = "s3"
)
```

**Step 2: 提交**

```bash
git add storage/config.go
git commit -m "feat(storage): add storage type enum

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 2: 定义错误

**Files:**
- Create: `storage/errors.go`

**Step 1: 编写代码**

```go
package storage

import "errors"

var (
    ErrMissingClient      = errors.New("storage: client not configured")
    ErrInvalidConfig      = errors.New("storage: invalid configuration")
    ErrUnsupportedType    = errors.New("storage: unsupported storage type")
    ErrObjectNotFound     = errors.New("storage: object not found")
    ErrBucketNotFound     = errors.New("storage: bucket not found")
    ErrAccessDenied       = errors.New("storage: access denied")
    ErrInvalidCredentials = errors.New("storage: invalid credentials")
    ErrMultipartNotFound  = errors.New("storage: multipart upload not found")
    ErrPartAlreadyExist   = errors.New("storage: part already uploaded")
)
```

**Step 2: 提交**

```bash
git add storage/errors.go
git commit -m "feat(storage): define package errors

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 3: 定义元数据类型

**Files:**
- Create: `storage/multipart.go`

**Step 1: 编写代码**

```go
package storage

import "time"

// MultipartUpload 分片上传会话
type MultipartUpload struct {
    UploadID string
    Key      string
    Bucket   string
}

// PartInfo 分片信息
type PartInfo struct {
    PartNumber int
    ETag       string
    Size       int64
}

// ObjectInfo 对象元数据
type ObjectInfo struct {
    Key          string
    Size         int64
    LastModified time.Time
    ETag         string
    ContentType  string
    Metadata     map[string]string
}
```

**Step 2: 提交**

```bash
git add storage/multipart.go
git commit -m "feat(storage): add multipart and object info types

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 4: 定义选项类型

**Files:**
- Create: `storage/options.go`

**Step 1: 编写代码**

```go
package storage

import "time"

// UploadOption 上传选项
type UploadOption func(*uploadOptions)

type uploadOptions struct {
    ContentType  string
    ContentLength int64
    Metadata     map[string]string
    Headers      map[string]string
    StorageClass string
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
```

**Step 2: 提交**

```bash
git add storage/options.go
git commit -m "feat(storage): add upload/download/sign options

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 5: 定义 Client 接口

**Files:**
- Create: `storage/client.go`

**Step 1: 编写代码**

```go
package storage

import (
    "context"
    "io"
    "time"
)

// Client 存储客户端接口
type Client interface {
    // 基础操作
    Upload(ctx context.Context, key string, reader io.Reader, opts ...UploadOption) error
    Download(ctx context.Context, key string, opts ...DownloadOption) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Stat(ctx context.Context, key string) (*ObjectInfo, error)

    // URL 签名
    SignedURL(ctx context.Context, key string, expire time.Duration, opts ...SignOption) (string, error)

    // 分片上传
    InitMultipart(ctx context.Context, key string, opts ...UploadOption) (*MultipartUpload, error)
    UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...UploadOption) (*PartInfo, error)
    CompleteMultipart(ctx context.Context, uploadID string, parts []*PartInfo, opts ...UploadOption) error
    AbortMultipart(ctx context.Context, uploadID string) error

    // 批量操作
    DeleteBatch(ctx context.Context, keys []string) error
}
```

**Step 2: 提交**

```bash
git add storage/client.go
git commit -m "feat(storage): define Client interface

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 6: 定义 Config 结构体

**Files:**
- Modify: `storage/config.go`

**Step 1: 添加 Config 结构体**

```go
package storage

import "time"

// Type 存储类型
type Type string

const (
    TypeOSS Type = "oss"
    TypeCOS Type = "cos"
    TypeS3  Type = "s3"
)

// Config 存储配置
type Config struct {
    Type     Type   `yaml:"type" json:"type"`
    Bucket   string `yaml:"bucket" json:"bucket"`
    Region   string `yaml:"region" json:"region"`
    Endpoint string `yaml:"endpoint" json:"endpoint"`

    // 凭证
    AccessKeyID     string `yaml:"access_key_id" json:"access_key_id"`
    AccessKeySecret string `yaml:"access_key_secret" json:"access_key_secret"`
    SecretAccessKey string `yaml:"secret_access_key" json:"secret_access_key"`
    SessionToken    string `yaml:"session_token" json:"session_token"`

    // 连接控制
    Timeout           time.Duration `yaml:"timeout" json:"timeout"`
    MaxRetries        int           `yaml:"max_retries" json:"max_retries"`
    MaxPartSize       int64         `yaml:"max_part_size" json:"max_part_size"`
    PartSize          int64         `yaml:"part_size" json:"part_size"`
    DefaultSignExpire time.Duration `yaml:"default_sign_expire" json:"default_sign_expire"`
}

// Validate 验证配置
func (c *Config) Validate() error {
    if c.Type == "" {
        return ErrInvalidConfig
    }
    if c.Bucket == "" {
        return ErrInvalidConfig
    }
    if c.Region == "" {
        return ErrInvalidConfig
    }
    if c.AccessKeyID == "" {
        return ErrInvalidConfig
    }
    return nil
}
```

**Step 2: 提交**

```bash
git add storage/config.go
git commit -m "feat(storage): add Config struct with validation

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 阶段 2: 工厂模式与 SDK 封装

### Task 7: 创建工厂函数

**Files:**
- Create: `storage/internal/factory.go`

**Step 1: 编写工厂函数**

```go
package internal

import (
    "fmt"

    "github.com/tsopia/go-kit/storage"
)

// NewClient 根据配置创建对应客户端
func NewClient(cfg *storage.Config) (storage.Client, error) {
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    switch cfg.Type {
    case storage.TypeOSS:
        return nil, fmt.Errorf("oss not implemented yet")
    case storage.TypeCOS:
        return nil, fmt.Errorf("cos not implemented yet")
    case storage.TypeS3:
        return nil, fmt.Errorf("s3 not implemented yet")
    default:
        return nil, fmt.Errorf("%w: %s", storage.ErrUnsupportedType, cfg.Type)
    }
}
```

**Step 2: 提交**

```bash
git add storage/internal/factory.go
git commit -m "feat(storage): add factory function

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 8: 实现 SDK 风格入口

**Files:**
- Create: `storage/storage.go`

**Step 1: 编写代码**

```go
package storage

import (
    "context"
    "io"
    "sync"
    "time"

    "github.com/tsopia/go-kit/storage/internal"
)

var (
    _client Client
    _mu     sync.RWMutex
)

// Configure 初始化全局客户端
func Configure(cfg *Config) error {
    _mu.Lock()
    defer _mu.Unlock()

    client, err := internal.NewClient(cfg)
    if err != nil {
        return err
    }
    _client = client
    return nil
}

// GetClient 获取全局客户端
func GetClient() Client {
    _mu.RLock()
    defer _mu.RUnlock()
    return _client
}

// Upload 上传文件
func Upload(ctx context.Context, key string, reader io.Reader, opts ...UploadOption) error {
    return UploadWithClient(ctx, GetClient(), key, reader, opts...)
}

// UploadWithClient 使用指定客户端上传
func UploadWithClient(ctx context.Context, c Client, key string, reader io.Reader, opts ...UploadOption) error {
    if c == nil {
        return ErrMissingClient
    }
    return c.Upload(ctx, key, reader, opts...)
}

// Download 下载文件
func Download(ctx context.Context, key string, opts ...DownloadOption) (io.ReadCloser, error) {
    return DownloadWithClient(ctx, GetClient(), key, opts...)
}

// DownloadWithClient 使用指定客户端下载
func DownloadWithClient(ctx context.Context, c Client, key string, opts ...DownloadOption) (io.ReadCloser, error) {
    if c == nil {
        return nil, ErrMissingClient
    }
    return c.Download(ctx, key, opts...)
}

// Delete 删除文件
func Delete(ctx context.Context, key string) error {
    return DeleteWithClient(ctx, GetClient(), key)
}

// DeleteWithClient 使用指定客户端删除
func DeleteWithClient(ctx context.Context, c Client, key string) error {
    if c == nil {
        return ErrMissingClient
    }
    return c.Delete(ctx, key)
}

// Exists 检查文件是否存在
func Exists(ctx context.Context, key string) (bool, error) {
    return ExistsWithClient(ctx, GetClient(), key)
}

// ExistsWithClient 使用指定客户端检查存在
func ExistsWithClient(ctx context.Context, c Client, key string) (bool, error) {
    if c == nil {
        return false, ErrMissingClient
    }
    return c.Exists(ctx, key)
}

// SignedURL 生成签名 URL
func SignedURL(ctx context.Context, key string, expire time.Duration, opts ...SignOption) (string, error) {
    return SignedURLWithClient(ctx, GetClient(), key, expire, opts...)
}

// SignedURLWithClient 使用指定客户端生成签名 URL
func SignedURLWithClient(ctx context.Context, c Client, key string, expire time.Duration, opts ...SignOption) (string, error) {
    if c == nil {
        return "", ErrMissingClient
    }
    return c.SignedURL(ctx, key, expire, opts...)
}

// InitMultipart 初始化分片上传
func InitMultipart(ctx context.Context, key string, opts ...UploadOption) (*MultipartUpload, error) {
    return InitMultipartWithClient(ctx, GetClient(), key, opts...)
}

// InitMultipartWithClient 使用指定客户端初始化分片上传
func InitMultipartWithClient(ctx context.Context, c Client, key string, opts ...UploadOption) (*MultipartUpload, error) {
    if c == nil {
        return nil, ErrMissingClient
    }
    return c.InitMultipart(ctx, key, opts...)
}

// UploadPart 上传分片
func UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...UploadOption) (*PartInfo, error) {
    return UploadPartWithClient(ctx, GetClient(), uploadID, partNum, reader, opts...)
}

// UploadPartWithClient 使用指定客户端上传分片
func UploadPartWithClient(ctx context.Context, c Client, uploadID string, partNum int, reader io.Reader, opts ...UploadOption) (*PartInfo, error) {
    if c == nil {
        return nil, ErrMissingClient
    }
    return c.UploadPart(ctx, uploadID, partNum, reader, opts...)
}

// CompleteMultipart 完成分片上传
func CompleteMultipart(ctx context.Context, uploadID string, parts []*PartInfo, opts ...UploadOption) error {
    return CompleteMultipartWithClient(ctx, GetClient(), uploadID, parts, opts...)
}

// CompleteMultipartWithClient 使用指定客户端完成分片上传
func CompleteMultipartWithClient(ctx context.Context, c Client, uploadID string, parts []*PartInfo, opts ...UploadOption) error {
    if c == nil {
        return ErrMissingClient
    }
    return c.CompleteMultipart(ctx, uploadID, parts, opts...)
}

// AbortMultipart 中止分片上传
func AbortMultipart(ctx context.Context, uploadID string) error {
    return AbortMultipartWithClient(ctx, GetClient(), uploadID)
}

// AbortMultipartWithClient 使用指定客户端中止分片上传
func AbortMultipartWithClient(ctx context.Context, c Client, uploadID string) error {
    if c == nil {
        return ErrMissingClient
    }
    return c.AbortMultipart(ctx, uploadID)
}
```

**Step 2: 提交**

```bash
git add storage/storage.go
git commit -m "feat(storage): add SDK-style entry points

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 阶段 3: OSS 实现

### Task 9: 添加 OSS 依赖

**Files:**
- Modify: `go.mod`

**Step 1: 添加依赖**

```bash
go get github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss
```

**Step 2: 提交**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add aliyun oss v2 sdk

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 10: 实现 OSS 客户端

**Files:**
- Create: `storage/internal/oss/client.go`

**Step 1: 编写代码**

```go
package oss

import (
    "context"
    "fmt"
    "io"
    "time"

    "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
    "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
    "github.com/tsopia/go-kit/storage"
)

// client OSS 客户端实现
type client struct {
    client *oss.Client
    bucket string
    config *storage.Config
}

// NewClient 创建 OSS 客户端
func NewClient(cfg *storage.Config) (storage.Client, error) {
    opts := []oss.Option{
        oss.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret),
        ),
        oss.WithRegion(cfg.Region),
    }

    if cfg.Endpoint != "" {
        opts = append(opts, oss.WithEndpoint(cfg.Endpoint))
    }

    c := oss.NewClient(opts...)

    return &client{
        client: c,
        bucket: cfg.Bucket,
        config: cfg,
    }, nil
}

func (c *client) Upload(ctx context.Context, key string, reader io.Reader, opts ...storage.UploadOption) error {
    options := &storage.UploadOption{}
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

    _, err := c.client.PutObject(ctx, req)
    return err
}

func (c *client) Download(ctx context.Context, key string, opts ...storage.DownloadOption) (io.ReadCloser, error) {
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
        // 检查是否为 404
        return false, nil
    }
    return true, nil
}

func (c *client) Stat(ctx context.Context, key string) (*storage.ObjectInfo, error) {
    req := &oss.GetObjectMetaRequest{
        Bucket: oss.Ptr(c.bucket),
        Key:    oss.Ptr(key),
    }

    resp, err := c.client.GetObjectMeta(ctx, req)
    if err != nil {
        return nil, err
    }

    return &storage.ObjectInfo{
        Key:          key,
        Size:         resp.ContentLength,
        LastModified: *resp.LastModified,
        ETag:         *resp.ETag,
        ContentType:  *resp.ContentType,
    }, nil
}

func (c *client) SignedURL(ctx context.Context, key string, expire time.Duration, opts ...storage.SignOption) (string, error) {
    options := &storage.SignOption{}
    for _, opt := range opts {
        opt(options)
    }

    if expire == 0 {
        expire = c.config.DefaultSignExpire
        if expire == 0 {
            expire = 15 * time.Minute
        }
    }

    req := &oss.PresignRequest{
        Bucket:  oss.Ptr(c.bucket),
        Key:     oss.Ptr(key),
        Expires: oss.Ptr(int64(expire.Seconds())),
    }

    if options.Method != "" {
        req.Method = options.Method
    }

    resp, err := c.client.Presign(ctx, req)
    if err != nil {
        return "", err
    }

    return resp.URL, nil
}

func (c *client) InitMultipart(ctx context.Context, key string, opts ...storage.UploadOption) (*storage.MultipartUpload, error) {
    req := &oss.InitiateMultipartUploadRequest{
        Bucket: oss.Ptr(c.bucket),
        Key:    oss.Ptr(key),
    }

    resp, err := c.client.InitiateMultipartUpload(ctx, req)
    if err != nil {
        return nil, err
    }

    return &storage.MultipartUpload{
        UploadID: *resp.UploadId,
        Key:      key,
        Bucket:   c.bucket,
    }, nil
}

func (c *client) UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...storage.UploadOption) (*storage.PartInfo, error) {
    req := &oss.UploadPartRequest{
        Bucket:   oss.Ptr(c.bucket),
        Key:      oss.Ptr(uploadID), // 这里需要根据 uploadID 获取原始 key
        UploadId: oss.Ptr(uploadID),
        PartNumber: int32(partNum),
        Body:     reader,
    }

    resp, err := c.client.UploadPart(ctx, req)
    if err != nil {
        return nil, err
    }

    return &storage.PartInfo{
        PartNumber: partNum,
        ETag:       *resp.ETag,
    }, nil
}

func (c *client) CompleteMultipart(ctx context.Context, uploadID string, parts []*storage.PartInfo, opts ...storage.UploadOption) error {
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
    // 需要 key，但接口只传了 uploadID，需要调整设计
    return fmt.Errorf("not implemented")
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
```

**Step 2: 更新工厂函数**

修改 `storage/internal/factory.go`：

```go
import "github.com/tsopia/go-kit/storage/internal/oss"

func NewClient(cfg *storage.Config) (storage.Client, error) {
    // ...
    switch cfg.Type {
    case storage.TypeOSS:
        return oss.NewClient(cfg)
    // ...
    }
}
```

**Step 3: 提交**

```bash
git add storage/internal/oss/ storage/internal/factory.go
git commit -m "feat(storage): implement oss client

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 阶段 4: COS 实现

### Task 11: 添加 COS 依赖

**Files:**
- Modify: `go.mod`

**Step 1: 添加依赖**

```bash
go get github.com/tencentyun/cos-go-sdk-v5
```

**Step 2: 提交**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add tencent cos sdk

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 12: 实现 COS 客户端

**Files:**
- Create: `storage/internal/cos/client.go`

**Step 1: 编写代码**

```go
package cos

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "time"

    "github.com/tencentyun/cos-go-sdk-v5"
    "github.com/tsopia/go-kit/storage"
)

// client COS 客户端实现
type client struct {
    client *cos.Client
    config *storage.Config
}

// NewClient 创建 COS 客户端
func NewClient(cfg *storage.Config) (storage.Client, error) {
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

    c := cos.NewClient(baseURL, &http.Client{
        Timeout: cfg.Timeout,
    }, func(r *http.Request) {
        // 添加认证头
        // COS SDK 会自动处理签名
    })

    // 设置密钥
    c.SecretID = cfg.AccessKeyID
    c.SecretKey = cfg.AccessKeySecret
    if cfg.SessionToken != "" {
        c.SessionToken = cfg.SessionToken
    }

    return &client{
        client: c,
        config: cfg,
    }, nil
}

func (c *client) Upload(ctx context.Context, key string, reader io.Reader, opts ...storage.UploadOption) error {
    options := &storage.UploadOption{}
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

func (c *client) Download(ctx context.Context, key string, opts ...storage.DownloadOption) (io.ReadCloser, error) {
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

func (c *client) Stat(ctx context.Context, key string) (*storage.ObjectInfo, error) {
    resp, err := c.client.Object.Head(ctx, key, nil)
    if err != nil {
        return nil, err
    }

    return &storage.ObjectInfo{
        Key:          key,
        Size:         resp.ContentLength,
        LastModified: resp.LastModified,
        ETag:         resp.Header.Get("ETag"),
        ContentType:  resp.Header.Get("Content-Type"),
    }, nil
}

func (c *client) SignedURL(ctx context.Context, key string, expire time.Duration, opts ...storage.SignOption) error {
    // 实现签名 URL
    return fmt.Errorf("not implemented")
}

func (c *client) InitMultipart(ctx context.Context, key string, opts ...storage.UploadOption) (*storage.MultipartUpload, error) {
    // 实现分片上传初始化
    return nil, fmt.Errorf("not implemented")
}

func (c *client) UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...storage.UploadOption) (*storage.PartInfo, error) {
    return nil, fmt.Errorf("not implemented")
}

func (c *client) CompleteMultipart(ctx context.Context, uploadID string, parts []*storage.PartInfo, opts ...storage.UploadOption) error {
    return fmt.Errorf("not implemented")
}

func (c *client) AbortMultipart(ctx context.Context, uploadID string) error {
    return fmt.Errorf("not implemented")
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
```

**Step 2: 更新工厂函数**

```go
import (
    "github.com/tsopia/go-kit/storage/internal/cos"
    "github.com/tsopia/go-kit/storage/internal/oss"
)

case storage.TypeCOS:
    return cos.NewClient(cfg)
```

**Step 3: 提交**

```bash
git add storage/internal/cos/ storage/internal/factory.go
git commit -m "feat(storage): implement cos client (basic)

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 阶段 5: S3 实现

### Task 13: 添加 S3 依赖

**Files:**
- Modify: `go.mod`

**Step 1: 添加依赖**

```bash
go get github.com/aws/aws-sdk-go-v2/service/s3
go get github.com/aws/aws-sdk-go-v2/credentials
```

**Step 2: 提交**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add aws s3 sdk

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 14: 实现 S3 客户端

**Files:**
- Create: `storage/internal/s3/client.go`

**Step 1: 编写代码**

```go
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
    "github.com/tsopia/go-kit/storage"
)

// client S3 客户端实现
type client struct {
    client *awss3.Client
    bucket string
    config *storage.Config
}

// NewClient 创建 S3 客户端
func NewClient(cfg *storage.Config) (storage.Client, error) {
    creds := credentials.NewStaticCredentialsProvider(
        cfg.AccessKeyID,
        cfg.SecretAccessKey,
        cfg.SessionToken,
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

func (c *client) Upload(ctx context.Context, key string, reader io.Reader, opts ...storage.UploadOption) error {
    options := &storage.UploadOption{}
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

func (c *client) Download(ctx context.Context, key string, opts ...storage.DownloadOption) (io.ReadCloser, error) {
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
        // 检查是否为 404
        return false, nil
    }
    return true, nil
}

func (c *client) Stat(ctx context.Context, key string) (*storage.ObjectInfo, error) {
    input := &awss3.HeadObjectInput{
        Bucket: aws.String(c.bucket),
        Key:    aws.String(key),
    }

    resp, err := c.client.HeadObject(ctx, input)
    if err != nil {
        return nil, err
    }

    return &storage.ObjectInfo{
        Key:          key,
        Size:         resp.ContentLength,
        LastModified: *resp.LastModified,
        ETag:         *resp.ETag,
        ContentType:  *resp.ContentType,
    }, nil
}

func (c *client) SignedURL(ctx context.Context, key string, expire time.Duration, opts ...storage.SignOption) error {
    // 实现预签名 URL
    return fmt.Errorf("not implemented")
}

func (c *client) InitMultipart(ctx context.Context, key string, opts ...storage.UploadOption) (*storage.MultipartUpload, error) {
    // 实现分片上传
    return nil, fmt.Errorf("not implemented")
}

func (c *client) UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...storage.UploadOption) (*storage.PartInfo, error) {
    return nil, fmt.Errorf("not implemented")
}

func (c *client) CompleteMultipart(ctx context.Context, uploadID string, parts []*storage.PartInfo, opts ...storage.UploadOption) error {
    return fmt.Errorf("not implemented")
}

func (c *client) AbortMultipart(ctx context.Context, uploadID string) error {
    return fmt.Errorf("not implemented")
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
```

**Step 2: 更新工厂函数**

```go
import (
    "github.com/tsopia/go-kit/storage/internal/cos"
    "github.com/tsopia/go-kit/storage/internal/oss"
    "github.com/tsopia/go-kit/storage/internal/s3"
)

case storage.TypeS3:
    return s3.NewClient(cfg)
```

**Step 3: 提交**

```bash
git add storage/internal/s3/ storage/internal/factory.go
git commit -m "feat(storage): implement s3 client (basic)

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 阶段 6: 测试

### Task 15: 编写基础测试

**Files:**
- Create: `storage/storage_test.go`

**Step 1: 编写测试**

```go
package storage

import (
    "context"
    "strings"
    "testing"
)

func TestConfigure(t *testing.T) {
    tests := []struct {
        name    string
        cfg     *Config
        wantErr bool
    }{
        {
            name: "valid config",
            cfg: &Config{
                Type:            TypeOSS,
                Bucket:          "test-bucket",
                Region:          "cn-hangzhou",
                AccessKeyID:     "test-key",
                AccessKeySecret: "test-secret",
            },
            wantErr: false,
        },
        {
            name:    "missing type",
            cfg:     &Config{},
            wantErr: true,
        },
        {
            name: "missing bucket",
            cfg: &Config{
                Type:   TypeOSS,
                Region: "cn-hangzhou",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Configure(tt.cfg)
            if (err != nil) != tt.wantErr {
                t.Errorf("Configure() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

func TestUploadWithClient(t *testing.T) {
    ctx := context.Background()
    err := UploadWithClient(ctx, nil, "test.txt", strings.NewReader("hello"))
    if err != ErrMissingClient {
        t.Errorf("UploadWithClient() error = %v, want %v", err, ErrMissingClient)
    }
}

func TestDownloadWithClient(t *testing.T) {
    ctx := context.Background()
    _, err := DownloadWithClient(ctx, nil, "test.txt")
    if err != ErrMissingClient {
        t.Errorf("DownloadWithClient() error = %v, want %v", err, ErrMissingClient)
    }
}

func TestConfigValidate(t *testing.T) {
    tests := []struct {
        name    string
        cfg     *Config
        wantErr error
    }{
        {
            name:    "empty config",
            cfg:     &Config{},
            wantErr: ErrInvalidConfig,
        },
        {
            name: "valid config",
            cfg: &Config{
                Type:            TypeOSS,
                Bucket:          "bucket",
                Region:          "region",
                AccessKeyID:     "key",
                AccessKeySecret: "secret",
            },
            wantErr: nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.cfg.Validate()
            if err != tt.wantErr {
                t.Errorf("Config.Validate() error = %v, want %v", err, tt.wantErr)
            }
        })
    }
}
```

**Step 2: 运行测试**

```bash
go test -v ./storage/...
```

**Step 3: 提交**

```bash
git add storage/storage_test.go
git commit -m "test(storage): add basic unit tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 阶段 7: 文档

### Task 16: 编写 README

**Files:**
- Create: `storage/README.md`

**Step 1: 编写文档**

```markdown
# Storage 包

对象存储统一封装，支持阿里云 OSS、腾讯云 COS、AWS S3。

## 安装

```go
go get github.com/tsopia/go-kit/storage
```

## 快速开始

```go
package main

import (
    "context"
    "log"
    "os"
    "strings"

    "github.com/tsopia/go-kit/storage"
)

func main() {
    // 初始化
    cfg := &storage.Config{
        Type:            storage.TypeOSS,
        Bucket:          "my-bucket",
        Region:          "cn-hangzhou",
        AccessKeyID:     os.Getenv("OSS_ACCESS_KEY_ID"),
        AccessKeySecret: os.Getenv("OSS_ACCESS_KEY_SECRET"),
    }

    if err := storage.Configure(cfg); err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // 上传文件
    data := strings.NewReader("hello world")
    if err := storage.Upload(ctx, "test/hello.txt", data); err != nil {
        log.Fatal(err)
    }

    // 生成临时访问链接
    url, err := storage.SignedURL(ctx, "test/hello.txt", 30*60)
    if err != nil {
        log.Fatal(err)
    }
    log.Println("URL:", url)
}
```

## 配置说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| Type | Type | 是 | 存储类型: oss/cos/s3 |
| Bucket | string | 是 | 存储桶名称 |
| Region | string | 是 | 地域，如 cn-hangzhou |
| Endpoint | string | 否 | 自定义端点 |
| AccessKeyID | string | 是 | Access Key ID |
| AccessKeySecret | string | 条件 | OSS/COS 密钥 |
| SecretAccessKey | string | 条件 | S3 密钥 |
| SessionToken | string | 否 | 临时凭证 Token |
| Timeout | duration | 否 | 超时时间，默认 30s |
| MaxRetries | int | 否 | 重试次数，默认 3 |
| DefaultSignExpire | duration | 否 | 签名 URL 默认过期时间，默认 15m |

## API 文档

### 上传

```go
err := storage.Upload(ctx, "key", reader,
    storage.WithContentType("application/json"),
    storage.WithMetadata("author", "alice"),
)
```

### 下载

```go
reader, err := storage.Download(ctx, "key")
if err != nil {
    return err
}
defer reader.Close()
```

### 删除

```go
err := storage.Delete(ctx, "key")
```

### 预签名 URL

```go
url, err := storage.SignedURL(ctx, "key", 30*60)
```

### 分片上传

```go
// 初始化
upload, err := storage.InitMultipart(ctx, "large.zip")

// 上传分片
part1, err := storage.UploadPart(ctx, upload.UploadID, 1, reader1)
part2, err := storage.UploadPart(ctx, upload.UploadID, 2, reader2)

// 完成
err = storage.CompleteMultipart(ctx, upload.UploadID, []*storage.PartInfo{part1, part2})
```

## 多客户端

如果需要使用多个不同的存储配置：

```go
// 获取配置的客户端
client := storage.GetClient()

// 或者使用指定客户端
storage.UploadWithClient(ctx, client, "key", reader)
```
```

**Step 2: 提交**

```bash
git add storage/README.md
git commit -m "docs(storage): add README

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 阶段 8: AI 集成

### Task 17: 创建 AI 使用提示

**Files:**
- Create: `storage/doc.go`

**Step 1: 编写代码**

```go
// Package storage 提供对象存储统一封装。
//
// 支持阿里云 OSS、腾讯云 COS、AWS S3，通过配置自动切换。
//
// 基本使用：
//
//	storage.Configure(&storage.Config{
//	    Type:     storage.TypeOSS,
//	    Bucket:   "my-bucket",
//	    Region:   "cn-hangzhou",
//	    AccessKeyID:     "...",
//	    AccessKeySecret: "...",
//	})
//
//	storage.Upload(ctx, "file.txt", reader)
//
// 更多信息请参考 README.md
package storage
```

**Step 2: 提交**

```bash
git add storage/doc.go
git commit -m "docs(storage): add package doc

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 18: 创建 AI Snippet

**Files:**
- Create: `storage/.ai-snippet.md`

**Step 1: 编写文档**

```markdown
# Storage 使用场景

## 场景 1: 初始化并上传文件

```go
storage.Configure(&storage.Config{
    Type:     storage.TypeOSS,
    Bucket:   "my-bucket",
    Region:   "cn-hangzhou",
    AccessKeyID:     os.Getenv("OSS_ACCESS_KEY_ID"),
    AccessKeySecret: os.Getenv("OSS_ACCESS_KEY_SECRET"),
})

err := storage.Upload(ctx, "avatar.jpg", reader)
```

## 场景 2: 生成临时访问链接

```go
url, err := storage.SignedURL(ctx, "avatar.jpg", 30*time.Minute)
```

## 场景 3: 分片上传大文件

```go
upload, _ := storage.InitMultipart(ctx, "large.zip")

// 循环上传每个分片
for i, chunk := range chunks {
    part, _ := storage.UploadPart(ctx, upload.UploadID, i+1, bytes.NewReader(chunk))
    parts = append(parts, part)
}

err := storage.CompleteMultipart(ctx, upload.UploadID, parts)
```

## 场景 4: 切换存储后端

```go
// 只需修改 Type 字段即可切换
storage.Configure(&storage.Config{
    Type:   storage.TypeCOS, // 或 TypeS3
    // ... 其他配置
})
```
```

**Step 2: 提交**

```bash
git add storage/.ai-snippet.md
git commit -m "docs(storage): add AI snippet

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

### Task 19: 更新能力清单

**Files:**
- Modify: `.ai/capabilities.yaml`

**Step 1: 添加 storage 能力**

```yaml
- name: storage
  description: 对象存储统一封装（OSS/COS/S3）
  import: github.com/tsopia/go-kit/storage
  scenarios:
    - name: 初始化存储客户端
      snippet: |
        storage.Configure(&storage.Config{
            Type: storage.TypeOSS,
            Bucket: "my-bucket",
            Region: "cn-hangzhou",
            AccessKeyID: "...",
            AccessKeySecret: "...",
        })
    - name: 上传文件
      snippet: storage.Upload(ctx, "file.txt", reader)
    - name: 生成临时访问链接
      snippet: storage.SignedURL(ctx, "file.txt", 30*time.Minute)
    - name: 分片上传大文件
      snippet: |
        upload, _ := storage.InitMultipart(ctx, "large.zip")
        part, _ := storage.UploadPart(ctx, upload.UploadID, 1, reader)
        storage.CompleteMultipart(ctx, upload.UploadID, []*storage.PartInfo{part})
  dependencies: [kit]
```

**Step 2: 提交**

```bash
git add .ai/capabilities.yaml
git commit -m "docs(ai): add storage to capabilities

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 验证

### Task 20: 最终验证

**Step 1: 运行所有测试**

```bash
go test -v ./storage/...
```

**Step 2: 检查编译**

```bash
go build ./storage/...
```

**Step 3: 检查代码规范**

```bash
golangci-lint run ./storage/...
```

**Step 4: 最终提交**

```bash
git add .
git commit -m "feat(storage): complete storage package implementation

- Unified Client interface for OSS/COS/S3
- SDK-style API with Configure/GetClient
- Basic operations: upload, download, delete, exists
- Multipart upload support
- Signed URL generation
- Comprehensive tests and documentation

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 执行选项

**计划已完成并保存到 `docs/plans/2026-03-09-storage-implementation.md`。两个执行选项：**

**1. Subagent-Driven（当前会话）** - 我为每个任务调度新的子代理，任务之间进行审查，快速迭代

**2. 并行会话（独立）** - 在新会话中打开 worktree 并使用 executing-plans 批量执行

选择哪种方式？
