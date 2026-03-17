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
	"time"

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
	url, err := storage.SignedURL(ctx, "test/hello.txt", 30*time.Minute)
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
| AccessKeySecret | string | 是 | 访问密钥 |
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

`SignedURL` 适合基础签名访问或低约束上传场景；如果需要对象存储校验
`Content-Type`、checksum、metadata 等条件，应该使用下面的“安全直传授权”。

```go
url, err := storage.SignedURL(ctx, "key", 30*time.Minute)
```

### 安全直传授权

```go
auth, err := storage.AuthorizeDirectUpload(ctx, storage.DirectUploadRequest{
	ObjectKey:   objectKey,
	ContentType: "image/png",
	Metadata: map[string]string{
		"owner": userID,
	},
	Checksum: &storage.DirectUploadChecksum{
		Algorithm: storage.DirectUploadChecksumMD5,
		Value:     checksum,
	},
})
if err != nil {
	return err
}

// 客户端按 auth.Method / auth.URL / auth.Headers / auth.FormFields 发起上传。
```

### 上传后校验

`VerifyDirectUploadObject` 会校验对象是否存在、`Content-Type`、metadata 和 size。
checksum 校验依赖 provider 能否从对象元信息回读对应算法:
- OSS 可回读 `Content-MD5`
- S3 仅在对象保存了对应 checksum 时可校验
- COS 当前只暴露 `CRC64`，不直接回读 `MD5/SHA256`

```go
result, err := storage.VerifyDirectUploadObject(ctx, storage.DirectUploadVerificationRequest{
	ObjectKey:   objectKey,
	ContentType: "image/png",
	Metadata: map[string]string{
		"owner": userID,
	},
})
if err != nil {
	return err
}
if !result.Matched {
	return fmt.Errorf("uploaded object does not match authorization")
}
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
