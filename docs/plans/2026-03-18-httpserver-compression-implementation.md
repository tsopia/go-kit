# HTTPServer Compression Middleware Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `httpserver/middleware` 增加一个只支持 `gzip` 的响应压缩中间件，提供稳定的默认压缩规则和可配置跳过逻辑。

**Architecture:** 在 `httpserver/middleware` 中新增 `compression.go` 与对应测试，采用“先缓冲响应、后判断是否压缩”的实现方式，避免第一版误压流式响应、SSE 或不适合压缩的内容。压缩能力保持在 middleware 层，不进入 `httpserver` core 默认行为。

**Tech Stack:** Go 1.24、Gin、`compress/gzip`、table-driven tests

---

### Task 1: 能力清单和文档入口

**Files:**
- Modify: `.ai/capabilities.yaml`
- Modify: `AGENTS.md`
- Modify: `httpserver/middleware/doc.go`
- Modify: `httpserver/middleware/README.md`
- Modify: `httpserver/README.md`

**Step 1: Update capability metadata first**

在 `.ai/capabilities.yaml` 中补充 `middleware` 的压缩场景，在 `AGENTS.md` 与 `README` 中增加 `Compression` 的简要说明。

**Step 2: Verify YAML syntax**

Run: `ruby -e 'require "yaml"; YAML.load_file(".ai/capabilities.yaml")'`
Expected: exit 0

**Step 3: Commit**

```bash
git add .ai/capabilities.yaml AGENTS.md httpserver/middleware/doc.go httpserver/middleware/README.md httpserver/README.md
git commit -m "docs(middleware): describe compression capability"
```

### Task 2: 摘要压缩行为

**Files:**
- Create: `httpserver/middleware/compression.go`
- Test: `httpserver/middleware/compression_test.go`

**Step 1: Write the failing test**

新增 table-driven tests，断言：

- 客户端未声明 `gzip` 时不压缩
- 客户端声明 `gzip` 且响应超过阈值时压缩
- 解压后的响应内容和原始内容一致

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestCompressionNegotiation|TestCompressionRoundTrip' -v`
Expected: FAIL，因为当前没有 `Compression`

**Step 3: Write minimal implementation**

在 `httpserver/middleware/compression.go` 中新增：

- `CompressionConfig`
- `Compression(...)`
- gzip 协商与缓冲响应 writer

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestCompressionNegotiation|TestCompressionRoundTrip' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/compression.go httpserver/middleware/compression_test.go
git commit -m "feat(middleware): add gzip compression middleware"
```

### Task 3: 默认跳过规则

**Files:**
- Modify: `httpserver/middleware/compression.go`
- Test: `httpserver/middleware/compression_test.go`

**Step 1: Write the failing test**

新增 tests，断言：

- 小响应低于阈值时不压缩
- `HEAD` / `204` / `304` 不压缩
- `text/event-stream` 不压缩
- 已有 `Content-Encoding` 不重复压缩
- `image/*`、`application/pdf` 等默认跳过

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestCompressionSkipsSmallBodies|TestCompressionSkipsStatuses|TestCompressionSkipsContentTypes' -v`
Expected: FAIL，因为当前跳过规则尚未完整实现

**Step 3: Write minimal implementation**

在 `compression.go` 中补充：

- 默认 `MinSizeBytes`
- 默认允许/排除的 `Content-Type`
- 状态码、SSE、已有 `Content-Encoding`、upgrade 跳过逻辑

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestCompressionSkipsSmallBodies|TestCompressionSkipsStatuses|TestCompressionSkipsContentTypes' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/compression.go httpserver/middleware/compression_test.go
git commit -m "feat(middleware): add compression skip rules"
```

### Task 4: 可配置跳过逻辑

**Files:**
- Modify: `httpserver/middleware/compression.go`
- Test: `httpserver/middleware/compression_test.go`

**Step 1: Write the failing test**

新增 tests，断言：

- `ShouldCompress(...)` 可强制跳过
- 自定义 `AllowedContentTypes`
- 自定义 `ExcludedContentTypes`

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestCompressionShouldCompress|TestCompressionCustomContentTypes' -v`
Expected: FAIL，因为当前配置覆盖能力未完整实现

**Step 3: Write minimal implementation**

在 `compression.go` 中补充配置覆盖逻辑和帮助函数。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestCompressionShouldCompress|TestCompressionCustomContentTypes' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/compression.go httpserver/middleware/compression_test.go
git commit -m "feat(middleware): support configurable compression rules"
```

### Task 5: 全量验证

**Files:**
- Verify only

**Step 1: Run package tests**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -v`
Expected: PASS

**Step 2: Run httpserver package tests**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/... -v`
Expected: PASS

**Step 3: Run lint**

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
Expected: `0 issues.`

**Step 4: Verify capability listing**

Run:

```bash
cd cmd/gokit && GOCACHE=/tmp/go-build go build -o /tmp/gokit .
cd /Users/kj/projects/go-kit && /tmp/gokit list
```

Expected: `middleware` 能力描述体现压缩能力

**Step 5: Run full repository tests**

Run: `GOCACHE=/tmp/go-build go test ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add docs/plans .ai/capabilities.yaml AGENTS.md httpserver
git commit -m "feat(middleware): add gzip compression middleware"
```
