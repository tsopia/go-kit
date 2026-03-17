// Package storage 提供对象存储统一封装。
//
// 支持阿里云 OSS、腾讯云 COS、AWS S3，通过配置自动切换。
//
// 基本使用：
//
//	storage.Configure(&storage.Config{
//		Type:            storage.TypeOSS,
//		Bucket:          "my-bucket",
//		Region:          "cn-hangzhou",
//		AccessKeyID:     "...",
//		AccessKeySecret: "...",
//	})
//
//	if err := storage.Upload(ctx, "file.txt", reader); err != nil {
//		return err
//	}
//
// SignedURL 适合基础签名访问；如果是客户端直传且需要约束 Content-Type、
// checksum、metadata 等条件，应使用 AuthorizeDirectUpload，并在上传完成后
// 调用 VerifyDirectUploadObject 做对象事实校验。
//
// 注意：上传后 checksum 校验是否可用取决于 provider 能否从对象元信息回读
// 对应算法。
//
// 更多信息请参考 README.md。
package storage
