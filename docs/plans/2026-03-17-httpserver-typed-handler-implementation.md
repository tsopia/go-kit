# HTTPServer Typed Handler Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 收敛 `httpserver` 的 typed handler pipeline，统一默认错误响应，增强校验模型，并补齐高层快捷解码入口。

**Architecture:** 保留 `Handle` / `HandleJSON` 作为主入口，在 `handler.go` 内部整理 decode/validate/error pipeline。新增轻量 `HTTPError`、`ValidationError` 与快捷 handler，避免引入新的框架式 DSL。

**Tech Stack:** Go 1.24、Gin、table-driven tests

---

### Task 1: 默认错误响应模型

**Files:**
- Modify: `httpserver/handler.go`
- Test: `httpserver/handler_test.go`

**Step 1: Write the failing test**

新增用例，断言 decode、validate、unknown error 默认返回 `code/message/details` 结构。

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandleJSONDecodeError|TestHandleJSONValidateError|TestHandleJSONDefaultInternalErrorResponse' -v`
Expected: FAIL，因为当前仍返回 `{"error": "..."}`

**Step 3: Write minimal implementation**

在 `httpserver/handler.go` 中新增 `ErrorResponse`，并重写默认错误渲染。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandleJSONDecodeError|TestHandleJSONValidateError|TestHandleJSONDefaultInternalErrorResponse' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/handler.go httpserver/handler_test.go
git commit -m "feat(httpserver): normalize typed handler error responses"
```

### Task 2: 结构化校验错误与 validator 链

**Files:**
- Modify: `httpserver/handler.go`
- Test: `httpserver/handler_test.go`

**Step 1: Write the failing test**

新增用例，断言：
- `ValidationError` 会输出 `details.fields`
- `WithValidators(...)` 会在请求自校验后继续执行

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandleJSONStructuredValidationError|TestHandleJSONWithValidators' -v`
Expected: FAIL，因为当前没有 `ValidationError` 和 validator 链

**Step 3: Write minimal implementation**

在 `httpserver/handler.go` 中新增：
- `ValidationField`
- `ValidationError`
- `RequestValidator`
- `WithValidators(...)`

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandleJSONStructuredValidationError|TestHandleJSONWithValidators' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/handler.go httpserver/handler_test.go
git commit -m "feat(httpserver): add structured typed handler validation"
```

### Task 3: HTTPError 与快捷 handler

**Files:**
- Modify: `httpserver/handler.go`
- Test: `httpserver/handler_test.go`
- Modify: `httpserver/README.md`

**Step 1: Write the failing test**

新增用例，断言：
- `HTTPError` 按自身状态码和错误码渲染
- `HandleQuery`、`HandleURI`、`HandleQueryURI` 可正确解码

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandleHTTPError|TestHandleQueryShortcut|TestHandleURIShortcut|TestHandleQueryURIShortcut' -v`
Expected: FAIL，因为当前没有这些入口或语义

**Step 3: Write minimal implementation**

在 `httpserver/handler.go` 中新增：
- `HTTPError`
- 默认 HTTP error 渲染
- `HandleQuery`
- `HandleURI`
- `HandleQueryURI`

并同步更新 `README`。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandleHTTPError|TestHandleQueryShortcut|TestHandleURIShortcut|TestHandleQueryURIShortcut' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/handler.go httpserver/handler_test.go httpserver/README.md
git commit -m "feat(httpserver): add typed handler shortcuts and http errors"
```

### Task 4: 全量验证

**Files:**
- Verify only

**Step 1: Run package tests**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/... -v`
Expected: PASS

**Step 2: Run lint**

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
Expected: `0 issues.`

**Step 3: Confirm docs and git diff**

Run: `git diff --stat -- httpserver docs/plans`
Expected: 只包含本轮 typed handler 改动与设计/实现文档

**Step 4: Commit**

```bash
git add docs/plans httpserver
git commit -m "feat(httpserver): refine typed handler pipeline"
```
