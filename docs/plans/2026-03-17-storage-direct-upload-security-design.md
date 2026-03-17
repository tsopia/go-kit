# Storage Direct Upload Security Design

## 概述

为 `storage` 增加一套面向客户端直传的安全授权能力，目标不是替代现有 `SignedURL`，而是在保留现有简单签名能力的前提下，补充一套能表达上传约束、provider 能力差异和上传后校验的统一抽象。

本设计聚焦单对象客户端直传，不覆盖 multipart 安全直传、bucket 事件配置、上传状态机或业务层权限判断。

## 背景

当前 `storage` 已支持：

- 普通上传、下载、删除、对象信息查询
- `SignedURL`
- 分片上传

但当前预签名能力过于薄：

- `SignedURL` 仅返回 `string`
- 签名选项仅有 `Method`
- 无法表达必带 headers、form fields、size/checksum/metadata 等约束
- 无法提供上传完成后的事实校验能力

这意味着当前接口不足以支撑“客户端申请上传 A，实际上传 B”这一类安全约束场景。

## 目标

- 提供清晰可读的客户端直传授权 API
- 保留现有 `SignedURL`，不破坏已有使用方
- 允许统一表达 `PUT presign` 与 `POST policy`
- 允许表达固定 `ObjectKey`、`ContentType`、`Metadata`、`Size`、`Checksum`
- 为上传完成后的对象事实校验提供 helper
- 对 provider 能力差异采用显式失败，不静默放宽安全约束

## 非目标

- 不在 `storage` 内实现业务权限判断
- 不在 `storage` 内保存上传意图或状态机
- 不在 `storage` 内生成业务语义的 `ObjectKey`
- 不处理 bucket callback / event / notification 配置
- 不在第一版覆盖 multipart 安全直传
- 不把 provider 的所有高级能力一次性抽象进公共 API

## 设计原则

1. `storage` 只提供技术能力，不替业务做权限或流程决策
2. 保持现有 API 兼容，新能力走新入口
3. `auto` 必须是可预测规则，不是黑盒
4. 可以降级模式，不能降级安全约束
5. provider 无法满足约束时必须显式报错

## 总体方案

保留现有：

```go
func SignedURL(ctx context.Context, key string, expire time.Duration, opts ...SignOptionFunc) (string, error)
```

新增面向客户端直传的主入口：

```go
func AuthorizeDirectUpload(ctx context.Context, req DirectUploadRequest) (*DirectUploadAuthorization, error)
func AuthorizeDirectUploadWithClient(ctx context.Context, c Client, req DirectUploadRequest) (*DirectUploadAuthorization, error)
```

新增上传完成后的对象事实校验：

```go
func VerifyDirectUploadObject(ctx context.Context, req DirectUploadVerificationRequest) (*DirectUploadVerificationResult, error)
func VerifyDirectUploadObjectWithClient(ctx context.Context, c Client, req DirectUploadVerificationRequest) (*DirectUploadVerificationResult, error)
```

## API 草图

### DirectUploadRequest

```go
type DirectUploadRequest struct {
    ObjectKey   string
    Expire      time.Duration
    ContentType string
    Metadata    map[string]string
    Size        *DirectUploadSize
    Checksum    *DirectUploadChecksum
    Mode        DirectUploadMode
}
```

### DirectUploadSize

```go
type DirectUploadSize struct {
    Exact int64
    Min   int64
    Max   int64
}
```

### DirectUploadChecksum

```go
type DirectUploadChecksum struct {
    Algorithm DirectUploadChecksumAlgorithm
    Value     string
}

type DirectUploadChecksumAlgorithm string

const (
    DirectUploadChecksumMD5    DirectUploadChecksumAlgorithm = "md5"
    DirectUploadChecksumSHA256 DirectUploadChecksumAlgorithm = "sha256"
)
```

### DirectUploadMode

```go
type DirectUploadMode string

const (
    DirectUploadModeAuto DirectUploadMode = "auto"
    DirectUploadModePut  DirectUploadMode = "put"
    DirectUploadModePost DirectUploadMode = "post"
)
```

### DirectUploadAuthorization

```go
type DirectUploadAuthorization struct {
    Provider    string
    Mode        DirectUploadMode
    ObjectKey   string
    URL         string
    Method      string
    Headers     map[string]string
    FormFields  map[string]string
    ExpiresAt   time.Time
    Constraints DirectUploadConstraints
}

type DirectUploadConstraints struct {
    ContentType string
    Metadata    map[string]string
    Size        *DirectUploadSize
    Checksum    *DirectUploadChecksum
}
```

### DirectUploadVerificationRequest

```go
type DirectUploadVerificationRequest struct {
    ObjectKey   string
    ContentType string
    Metadata    map[string]string
    Size        *DirectUploadSize
    Checksum    *DirectUploadChecksum
}
```

### DirectUploadVerificationResult

```go
type DirectUploadVerificationResult struct {
    Exists     bool
    Matched    bool
    Mismatches []DirectUploadMismatch
    Object     *ObjectInfo
}

type DirectUploadMismatch struct {
    Field    string
    Expected string
    Actual   string
}
```

## 默认过期时间

新接口沿用现有签名过期时间规则：

1. `req.Expire > 0` 时使用请求值
2. `req.Expire == 0` 时使用 `cfg.DefaultSignExpire`
3. 配置未设置时回退到 `storage.DefaultSignExpire`

这保持与现有 `SignedURL` 行为一致。

## Mode 选择规则

### 显式模式

- `Mode=put`：只尝试 `PUT presign`
- `Mode=post`：只尝试 `POST policy`

### Auto 模式

`Mode=auto` 采用固定规则：

- 存在 `Size.Min` / `Size.Max` 这类区间约束时，优先 `POST`
- 只有固定 `ContentType`、固定 `Checksum`、固定 `Metadata` 时，优先 `PUT`
- provider 不支持首选模式时，可以尝试另一模式
- 如果两种模式都不能满足约束，则直接报错

关键原则：

- 允许降级模式
- 不允许降级安全约束

例如请求声明了 checksum 或精确 size，而当前 provider + mode 无法保证该约束，则必须失败，不能返回更弱的授权结果。

## 请求校验规则

`AuthorizeDirectUpload` 在生成任何授权前必须执行参数校验和标准化。

### 通用规则

- `ObjectKey` 必填
- `Expire < 0` 非法
- `Mode` 只能是 `auto` / `put` / `post`
- `ContentType` 如果给定，必须是单个明确值
- `Metadata` 中 key/value 不能为空
- `Checksum` 如果出现，`Algorithm` 和 `Value` 都必须给定

### Size 规则

- `Size == nil`：不限制
- `Exact > 0` 时：
  - `Min == 0`
  - `Max == 0`
- `Exact == 0` 时：
  - `Min`、`Max` 不能为负
  - 两者若都大于 0，则 `Min <= Max`
  - `Min == 0 && Max == 0` 视为无效输入

### 标准化规则

- `Mode=""` 视为 `auto`
- `ContentType` 去首尾空格
- `Metadata` key/value 去首尾空格
- `Checksum.Algorithm` 转小写
- `ObjectKey` 保持原样，不在库内擅自变更

## 上传后校验

`VerifyDirectUploadObject` 只负责对象事实校验，不负责业务授权判断。

应校验：

- 对象是否存在
- `ObjectKey` 是否匹配
- `ContentType` 是否匹配
- `Metadata` 是否匹配
- `Size` 是否匹配
- provider 可稳定提供 checksum 时，checksum 是否匹配

返回结果应区分：

- 对象不存在
- 对象存在但不匹配
- 对象存在且匹配

建议 `error` 只用于系统错误，约束不匹配通过结构化 `Mismatches` 返回。

## ObjectInfo 扩展

为支持 `VerifyDirectUploadObject`，需要扩展 `ObjectInfo`：

```go
type ObjectInfo struct {
    Key          string
    Size         int64
    LastModified time.Time
    ETag         string
    ContentType  string
    Metadata     map[string]string
    Checksums    map[string]string
}
```

若某 provider 当前无法返回 metadata 或 checksum，应返回可用子集，不伪造数据。

## 错误语义

建议增加清晰错误类别，至少覆盖：

- `direct upload mode not supported by provider`
- `checksum constraint not supported for mode/provider`
- `size range constraint not supported for mode/provider`
- `requested constraints cannot be satisfied`
- `direct upload request is invalid`

这些错误应帮助调用方判断是请求本身非法，还是 provider 能力不足。

## 与现有 SignedURL 的兼容策略

- 保留现有 `SignedURL`
- 不立刻废弃
- 文档重新定位为“基础签名能力”
- 新增文档说明：`SignedURL` 不承诺提供完整的客户端直传安全约束

这样可以保持兼容，同时把“简单签名 URL”和“安全直传授权协议”清晰分层。

## Provider 能力实现原则

- 公共层只承诺“授权结果满足声明约束”
- provider 层自行决定映射到 `PUT presign` 还是 `POST policy`
- provider 无法满足约束时必须显式失败
- 不暴露过多 provider-specific 原始参数到公共 API

第一版以“单对象直传”能力为核心，不把 multipart、安全回调或 bucket 事件通知纳入同一设计。

## 能力边界

### storage 负责

- 直传授权
- 请求校验与标准化
- provider 模式选择
- 返回客户端真实上传载荷
- 上传后的对象事实校验

### storage 不负责

- 上传权限判断
- `UploadIntent` 持久化
- 业务语义 `ObjectKey` 生成
- 上传完成确认状态机
- bucket callback / event / notification 配置
- 审核、转码、病毒扫描等业务后处理

### 业务层负责

- 创建上传意图
- 生成最终 `ObjectKey`
- 决定约束内容
- 发放授权给客户端
- 上传完成后调用确认接口
- 决定对象是否正式可用
- 清理超时和脏对象

## 第一版范围

### 包含

- 保留 `SignedURL`
- 新增 `AuthorizeDirectUpload`
- 新增 `VerifyDirectUploadObject`
- 扩展 `ObjectInfo`
- provider 内部实现 `PUT/POST` 约束映射
- 明确 provider 能力不足时的失败行为

### 不包含

- multipart 安全直传
- callback / event 统一抽象
- bucket 级事件配置
- 临时凭证直传
- 上传工作流状态机

## 推荐落地方向

按本设计推进时，`storage` 应保持“提供技术能力，不替业务做流程决策”的边界。业务层通过 `UploadIntent` 或等价机制保存授权上下文，再调用 `storage` 生成直传授权并在上传后进行确认。
