# Storage 包设计文档

## 概述

为 go-kit 增加统一的存储抽象层，支持阿里云 OSS、腾讯云 COS、AWS S3 三大对象存储服务，通过配置自动切换底层实现。

## 目标

- 提供统一的 `Client` 接口，屏蔽底层存储差异
- 支持通过配置 `Type` 字段自动选择实现
- 遵循 go-kit SDK 风格（Configure + GetClient + 包级函数）
- 覆盖常用对象存储操作：上传、下载、删除、预签名 URL、分片上传

## 非目标

- 支持所有厂商的高级特性（取交集，保持通用性）
- 本地文件系统存储（后续可扩展）
- 跨存储桶复制、生命周期管理等运维操作

## 架构设计

### 目录结构

```
storage/
├── config.go          # 配置结构体和校验
├── storage.go         # SDK 风格入口：Configure/GetClient/高层函数
├── errors.go          # 包级错误定义
├── client.go          # Client interface 定义
├── options.go         # 上传/下载等操作的选项
├── multipart.go       # 分片上传相关类型
├── internal/
│   ├── factory.go     # 工厂函数 newClient
│   ├── oss/           # 阿里云 OSS V2 实现
│   ├── cos/           # 腾讯云 COS 实现
│   └── s3/            # AWS S3 实现
├── README.md          # 使用文档
└── examples/          # 使用示例
    ├── upload/
    ├── multipart/
    └── presigned/
```

### 设计原则

1. **统一抽象**：定义 `Client` 接口，屏蔽底层差异
2. **配置驱动**：通过 `Type` 字段自动选择实现
3. **SDK 封装**：包级函数 + 全局默认客户端，与 database/pgmq 风格一致
4. **内部实现隔离**：各厂商实现在 `internal/` 下，对外不可见

## 数据结构与接口

### 存储类型枚举

```go
type Type string

const (
    TypeOSS Type = "oss"
    TypeCOS Type = "cos"
    TypeS3  Type = "s3"
)
```

### 配置结构体

```go
type Config struct {
    Type     Type   `yaml:"type" json:"type"`
    Bucket   string `yaml:"bucket" json:"bucket"`
    Region   string `yaml:"region" json:"region"`     // 如 cn-hangzhou, ap-beijing, us-east-1
    Endpoint string `yaml:"endpoint" json:"endpoint"` // 可选，用于内网/自定义端点

    // 凭证（支持 AK/SK 或临时凭证）
    AccessKeyID     string `yaml:"access_key_id" json:"access_key_id"`
    AccessKeySecret string `yaml:"access_key_secret" json:"access_key_secret"` // OSS/COS 用
    SecretAccessKey string `yaml:"secret_access_key" json:"secret_access_key"` // S3 用
    SessionToken    string `yaml:"session_token" json:"session_token"`         // 临时凭证

    // 连接控制
    Timeout           time.Duration `yaml:"timeout" json:"timeout"`                       // 默认 30s
    MaxRetries        int           `yaml:"max_retries" json:"max_retries"`               // 默认 3
    MaxPartSize       int64         `yaml:"max_part_size" json:"max_part_size"`           // 分片大小，默认 100MB
    PartSize          int64         `yaml:"part_size" json:"part_size"`                   // 分片大小，默认 5MB
    DefaultSignExpire time.Duration `yaml:"default_sign_expire" json:"default_sign_expire"` // URL签名默认过期时间，默认 15m
}
```

### Client 接口

```go
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

### 选项类型

```go
// UploadOption 上传选项
type UploadOption func(*uploadOptions)

func WithContentType(ct string) UploadOption
func WithMetadata(key, value string) UploadOption
func WithStorageClass(class string) UploadOption

// DownloadOption 下载选项
type DownloadOption func(*downloadOptions)

func WithRange(start, end int64) DownloadOption

// SignOption 签名选项
type SignOption func(*signOptions)

func WithExpire(d time.Duration) SignOption
func WithMethod(method string) SignOption
```

### 元数据类型

```go
type MultipartUpload struct {
    UploadID string
    Key      string
    Bucket   string
}

type PartInfo struct {
    PartNumber int
    ETag       string
    Size       int64
}

type ObjectInfo struct {
    Key          string
    Size         int64
    LastModified time.Time
    ETag         string
    ContentType  string
    Metadata     map[string]string
}
```

## SDK 风格 API

### 包级函数

```go
// Configure 初始化全局客户端
func Configure(cfg *Config) error

// GetClient 获取全局客户端
func GetClient() Client

// 高层函数（省略时默认用全局 client）
func Upload(ctx context.Context, key string, reader io.Reader, opts ...UploadOption) error
func UploadWithClient(ctx context.Context, c Client, key string, reader io.Reader, opts ...UploadOption) error

func Download(ctx context.Context, key string, opts ...DownloadOption) (io.ReadCloser, error)
func DownloadWithClient(ctx context.Context, c Client, key string, opts ...DownloadOption) (io.ReadCloser, error)

func Delete(ctx context.Context, key string) error
func DeleteWithClient(ctx context.Context, c Client, key string) error

func Exists(ctx context.Context, key string) (bool, error)
func ExistsWithClient(ctx context.Context, c Client, key string) (bool, error)

func SignedURL(ctx context.Context, key string, expire time.Duration, opts ...SignOption) (string, error)
func SignedURLWithClient(ctx context.Context, c Client, key string, expire time.Duration, opts ...SignOption) (string, error)

// 分片上传
func InitMultipart(ctx context.Context, key string, opts ...UploadOption) (*MultipartUpload, error)
func UploadPart(ctx context.Context, uploadID string, partNum int, reader io.Reader, opts ...UploadOption) (*PartInfo, error)
func CompleteMultipart(ctx context.Context, uploadID string, parts []*PartInfo, opts ...UploadOption) error
func AbortMultipart(ctx context.Context, uploadID string) error
```

### 使用示例

```go
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

// 简单上传
data := strings.NewReader("hello world")
err := storage.Upload(ctx, "test/hello.txt", data,
    storage.WithContentType("text/plain"),
    storage.WithMetadata("author", "alice"),
)

// 生成临时访问链接（30分钟）
url, err := storage.SignedURL(ctx, "avatar.jpg", 30*time.Minute)

// 分片上传大文件
upload, err := storage.InitMultipart(ctx, "large.zip")
part1, err := storage.UploadPart(ctx, upload.UploadID, 1, bytes.NewReader(chunk1))
part2, err := storage.UploadPart(ctx, upload.UploadID, 2, bytes.NewReader(chunk2))
err = storage.CompleteMultipart(ctx, upload.UploadID, []*storage.PartInfo{part1, part2})
```

## 错误处理

### 包级错误定义

```go
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

## 内部实现

### 工厂函数

```go
// internal/factory.go
func newClient(cfg *Config) (Client, error) {
    switch cfg.Type {
    case TypeOSS:
        return oss.NewClient(cfg)
    case TypeCOS:
        return cos.NewClient(cfg)
    case TypeS3:
        return s3.NewClient(cfg)
    default:
        return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, cfg.Type)
    }
}
```

### OSS 实现示例

```go
// internal/oss/client.go
package oss

import "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"

type client struct {
    client *oss.Client
    bucket string
    config *storage.Config
}

func NewClient(cfg *storage.Config) (storage.Client, error) {
    c := oss.NewClient(
        oss.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret),
        ),
        oss.WithRegion(cfg.Region),
    )

    return &client{
        client: c,
        bucket: cfg.Bucket,
        config: cfg,
    }, nil
}

func (c *client) Upload(ctx context.Context, key string, reader io.Reader, opts ...storage.UploadOption) error {
    options := parseUploadOptions(opts...)

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
```

### 各厂商适配要点

| 功能 | OSS | COS | S3 |
|------|-----|-----|-----|
| **SDK** | alibabacloud-oss-go-sdk-v2 | cos-go-sdk-v5 | aws-sdk-go-v2 |
| **上传** | PutObject | PutObject | PutObject |
| **签名URL** | Presign | GetPresignedURL | PresignGetObject |
| **分片初始化** | InitiateMultipartUpload | InitiateMultipartUpload | CreateMultipartUpload |
| **特殊处理** | 无 | 需处理 Region 格式 | 需处理 Endpoint/Region |

## 依赖

```go
// go.mod 添加
require (
    github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss v2.x.x
    github.com/tencentyun/cos-go-sdk-v5 v0.x.x
    github.com/aws/aws-sdk-go-v2/service/s3 v1.x.x
    github.com/aws/aws-sdk-go-v2/credentials v1.x.x
)
```

## 测试策略

### 单元测试

- Table-driven tests 覆盖所有公共函数
- Mock Client 实现用于隔离测试

### 集成测试

- 通过环境变量 `STORAGE_INTEGRATION=1` 控制
- 需要真实云账号凭证

## AI 能力清单更新

新增 storage 包后，更新 `.ai/capabilities.yaml`：

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

## 新包检查清单

- [ ] 创建 `doc.go` 包含 AI 使用提示
- [ ] 创建 `.ai-snippet.md` 描述使用场景
- [ ] 更新 `.ai/capabilities.yaml` 添加能力定义
- [ ] 提供 `README.md` 说明角色、依赖、初始化方式
- [ ] 覆盖关键路径的测试用例
- [ ] 遵循 SDK 封装规范（Configure + Helper 模式）

## 实现计划

1. **阶段 1**：基础接口 + OSS 实现
   - config.go、errors.go、client.go、options.go
   - storage.go（SDK 封装）
   - internal/factory.go、internal/oss/

2. **阶段 2**：补充实现
   - internal/cos/
   - internal/s3/

3. **阶段 3**：文档与测试
   - README.md
   - examples/
   - 单元测试

4. **阶段 4**：AI 集成
   - doc.go
   - .ai-snippet.md
   - 更新 .ai/capabilities.yaml
