# Storage Provider Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修复 storage 中 4 个高置信度问题：OSS 分片上传 key 错误、OSS AbortMultipart 缺失、COS uploadKeys 并发不安全、S3/OSS Exists 吞掉真实错误，同时补齐对应回归测试。

**Architecture:** 不扩散到 provider 能力边界或低置信度签名风险，只在 multipart 状态管理和 exists 错误归一化上做最小修复。为 `uploadID -> key` 引入 provider 共享的并发安全 helper，让 S3/COS/OSS 统一复用；`Exists` 统一改为只把 object-not-found 解释为 `false, nil`，其余错误透传。

**Tech Stack:** Go 1.24、现有 `storage/providers/{s3,cos,oss}` 实现、table-driven tests、`go test ./storage/...`

---

### Task 1: 为 multipart 状态管理引入共享并发安全 helper

**Files:**
- Create: `storage/providers/multipart_state.go`
- Create: `storage/providers/multipart_state_test.go`
- Modify: `storage/providers/s3/client.go`
- Modify: `storage/providers/cos/client.go`
- Modify: `storage/providers/oss/client.go`

**Step 1: 写失败测试**

在 `storage/providers/multipart_state_test.go` 中覆盖：
- `Store` 后 `Load` 能取回 key
- `Delete` 后 `Load` 返回不存在
- 并发 `Store/Load/Delete` 不 panic，最终状态正确

**Step 2: 跑测试确认失败**

Run: `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers -run 'TestMultipartState' -v`

Expected: FAIL，因为 helper 还不存在。

**Step 3: 写最小实现**

在 `storage/providers/multipart_state.go` 中添加：
- `type multipartState struct`
- `newMultipartState()`
- `Store(uploadID, key string)`
- `Load(uploadID string) (string, bool)`
- `Delete(uploadID string)`

用 `sync.RWMutex + map[string]string` 实现。

**Step 4: 让三个 provider 复用 helper**

- S3 从自带 `map+mu` 切到共享 helper
- COS 从裸 `map` 切到共享 helper，消除并发风险
- OSS 新增该状态，用于 `InitMultipart/UploadPart/CompleteMultipart/AbortMultipart`

**Step 5: 跑测试确认通过**

Run: `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers -run 'TestMultipartState' -v`

Expected: PASS

### Task 2: 先写失败测试，再修 OSS multipart 和 Exists 语义

**Files:**
- Create: `storage/providers/oss/multipart_state_test.go`
- Create: `storage/providers/s3/exists_test.go`
- Create: `storage/providers/oss/exists_test.go`
- Modify: `storage/providers/oss/client.go`
- Modify: `storage/providers/s3/client.go`

**Step 1: 写失败测试**

在 `storage/providers/oss/multipart_state_test.go` 中覆盖：
- `InitMultipart` 记录的 `uploadID -> key` 能被 `lookup` 到
- `UploadPart/CompleteMultipart/AbortMultipart` 依赖 key 映射，而不是直接把 `uploadID` 当 key

在 `storage/providers/s3/exists_test.go` 和 `storage/providers/oss/exists_test.go` 中覆盖：
- `ErrObjectNotFound` => `Exists == false, err == nil`
- `ErrBucketNotFound` / `ErrAccessDenied` / 其他错误 => `err` 透传

通过小 helper 测，不强依赖真实 SDK 请求。

**Step 2: 跑测试确认失败**

Run:
- `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers/oss -run 'Test(OSSMultipartState|Exists)' -v`
- `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers/s3 -run 'TestExists' -v`

Expected: FAIL

**Step 3: 写最小实现**

在 `storage/providers/oss/client.go` 中：
- `InitMultipart` 记录 `uploadID -> key`
- `UploadPart` 用映射得到真实 key
- `CompleteMultipart` 用映射得到真实 key，完成后删除映射
- `AbortMultipart` 实际调用 OSS API，并在成功后删除映射

在 `storage/providers/s3/client.go` 和 `storage/providers/oss/client.go` 中：
- 提取 `existsFromError(err error) (bool, error)` 风格 helper
- 只把 `ErrObjectNotFound` 解释为对象不存在，其他错误透传

**Step 4: 跑测试确认通过**

Run:
- `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers/oss -run 'Test(OSSMultipartState|Exists)' -v`
- `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers/s3 -run 'TestExists' -v`

Expected: PASS

### Task 3: 为 COS 的 exists/error 路径补测试并补齐 bucket 错误覆盖

**Files:**
- Create: `storage/providers/cos/exists_test.go`
- Modify: `storage/providers/cos/client.go`

**Step 1: 写失败测试**

在 `storage/providers/cos/exists_test.go` 中覆盖：
- `ErrObjectNotFound` => `Exists == false, err == nil`
- `ErrAccessDenied` => 透传
- COS `404 + NoSuchBucket` / `NoSuchBucket` 风格错误 => `ErrBucketNotFound`

**Step 2: 跑测试确认失败**

Run: `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers/cos -run 'Test(Exists|NormalizeStatError)' -v`

Expected: FAIL

**Step 3: 写最小实现**

在 `storage/providers/cos/client.go` 中：
- `Exists` 改成基于归一化错误处理
- `normalizeStatError` 在 `404` 下优先识别 bucket 级错误

**Step 4: 跑测试确认通过**

Run: `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers/cos -run 'Test(Exists|NormalizeStatError)' -v`

Expected: PASS

### Task 4: 做 provider 级回归验证

**Files:**
- Modify: `storage/providers/oss/client.go`
- Modify: `storage/providers/cos/client.go`
- Modify: `storage/providers/s3/client.go`
- Modify: `storage/providers/multipart_state.go`

**Step 1: 跑 storage provider 回归**

Run:
- `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers -v`
- `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers/s3 -v`
- `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers/cos -v`
- `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/providers/oss -v`
- `GOCACHE=/tmp/go-build /usr/local/opt/go@1.24/bin/go test ./storage/...`

Expected: PASS

**Step 2: 提交**

```bash
git add storage/providers
git commit -m "fix(storage): harden provider multipart and exists handling"
```
