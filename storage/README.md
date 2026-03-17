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
    storage.WithMetadata("source", "web"),
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
