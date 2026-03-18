# HTTPServer Access Log Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `httpserver/middleware` 增加分层访问日志能力，提供默认摘要日志、可选 payload 调试日志，以及 JSON/form/multipart 的结构化脱敏。

**Architecture:** 在 `httpserver/middleware` 中新增 `access_log.go` 与相关测试，通过单个 Gin middleware 产出 `access_log` 与可选 `payload_log` 两类事件。脱敏与 multipart 解析放在包内 helper 中完成，不依赖具体 logger 实现。

**Tech Stack:** Go 1.24、Gin、mime/multipart、table-driven tests

---

### Task 1: 能力清单与对外文档入口

**Files:**
- Modify: `.ai/capabilities.yaml`
- Modify: `AGENTS.md`
- Modify: `httpserver/middleware/doc.go`
- Modify: `httpserver/middleware/README.md`

**Step 1: Write the failing documentation expectation**

确认当前 `middleware` 能力描述中尚未包含 `AccessLog` 场景与载荷日志说明。

**Step 2: Update capability metadata first**

在 `.ai/capabilities.yaml` 中补充 `middleware` 的新使用场景，在 `AGENTS.md` 和 `README` 中增加 `AccessLog` 简要说明。

**Step 3: Verify YAML syntax**

Run: `ruby -e 'require "yaml"; YAML.load_file(".ai/capabilities.yaml")'`
Expected: exit 0

**Step 4: Commit**

```bash
git add .ai/capabilities.yaml AGENTS.md httpserver/middleware/doc.go httpserver/middleware/README.md
git commit -m "docs(middleware): describe access log capability"
```

### Task 2: 摘要 access log 中间件

**Files:**
- Create: `httpserver/middleware/access_log.go`
- Test: `httpserver/middleware/access_log_test.go`

**Step 1: Write the failing test**

新增 table-driven tests，断言：

- 默认输出一条 `access_log`
- 包含 `method/path/route/status/latency_ms/request_id/trace_id`
- `5xx` 输出 `error` 级别

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestAccessLogSummary|TestAccessLogErrorLevel' -v`
Expected: FAIL，因为当前没有 `AccessLog`

**Step 3: Write minimal implementation**

在 `httpserver/middleware/access_log.go` 中新增：

- `LoggerFunc`
- `AccessLogConfig`
- `AccessLogOption`
- `AccessLog(...) gin.HandlerFunc`

先实现摘要日志和默认级别逻辑。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestAccessLogSummary|TestAccessLogErrorLevel' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/access_log.go httpserver/middleware/access_log_test.go
git commit -m "feat(middleware): add summary access log"
```

### Task 3: payload capture 与基础脱敏

**Files:**
- Modify: `httpserver/middleware/access_log.go`
- Test: `httpserver/middleware/access_log_test.go`

**Step 1: Write the failing test**

新增 tests，断言：

- `CapturePayload` 关闭时不输出 `payload_log`
- 开启后输出 `payload_log`
- JSON 与 form body 中敏感字段被脱敏
- 超过 `MaxBodyBytes` 时标记截断

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestAccessLogPayloadJSON|TestAccessLogPayloadForm|TestAccessLogPayloadTruncated' -v`
Expected: FAIL，因为当前没有 payload capture 和脱敏逻辑

**Step 3: Write minimal implementation**

在 `access_log.go` 中补充：

- payload 捕获逻辑
- `RedactionConfig`
- 默认敏感字段规则
- JSON / form 脱敏 helper

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestAccessLogPayloadJSON|TestAccessLogPayloadForm|TestAccessLogPayloadTruncated' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/access_log.go httpserver/middleware/access_log_test.go
git commit -m "feat(middleware): add payload capture and redaction"
```

### Task 4: multipart 结构化捕获

**Files:**
- Modify: `httpserver/middleware/access_log.go`
- Test: `httpserver/middleware/access_log_test.go`

**Step 1: Write the failing test**

新增 tests，断言：

- `metadata_only` 仅记录字段名与文件元数据
- `form_fields_only` 记录脱敏后的文本字段
- `selected_parts` 仅记录白名单文本 part
- 文件内容不会出现在日志中

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestAccessLogMultipartMetadataOnly|TestAccessLogMultipartFormFieldsOnly|TestAccessLogMultipartSelectedParts' -v`
Expected: FAIL，因为当前没有 multipart-aware capture

**Step 3: Write minimal implementation**

在 `access_log.go` 中补充：

- `MultipartCaptureMode`
- `MultipartConfig`
- multipart 文本字段解析
- 文件 part 元数据提取

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestAccessLogMultipartMetadataOnly|TestAccessLogMultipartFormFieldsOnly|TestAccessLogMultipartSelectedParts' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/access_log.go httpserver/middleware/access_log_test.go
git commit -m "feat(middleware): add multipart access log capture"
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

Expected: 能看到更新后的 `middleware` 场景描述

**Step 5: Commit**

```bash
git add docs/plans .ai/capabilities.yaml AGENTS.md httpserver/middleware
git commit -m "feat(middleware): add access log middleware"
```
