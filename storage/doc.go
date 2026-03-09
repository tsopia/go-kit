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
