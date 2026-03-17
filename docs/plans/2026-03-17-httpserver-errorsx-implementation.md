# HTTPServer ErrorsX Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 新增 `httpserver/integration/errorsx`，把 `errors` 包的业务错误稳定映射为 typed handler 可复用的 HTTP 响应。

**Architecture:** 保持 `httpserver` core 与 `errors` core 解耦，在 integration 层新增桥接包，提供 `Response(err)` 与 `Mapper()` 两个最小 API。实现按 TDD 推进，先锁定映射规则，再补文档与能力清单。

**Tech Stack:** Go 1.24、Gin、table-driven tests

---

### Task 1: 新包骨架与映射测试

**Files:**
- Create: `httpserver/integration/errorsx/mapper_test.go`
- Create: `httpserver/integration/errorsx/doc.go`
- Create: `httpserver/integration/errorsx/README.md`

**Step 1: Write the failing test**

编写 table-driven tests，覆盖：
- `errors.InvalidParam.New(...)`
- `errors.NotFound.New(...)`
- 普通 `error`
- `Mapper()` 挂到 `httpserver.HandleJSON(...)`

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/integration/errorsx -v`
Expected: FAIL，因为包和 API 尚不存在

**Step 3: Write minimal implementation**

新增包骨架与测试占位文档。

**Step 4: Run test to verify it still fails for missing implementation details**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/integration/errorsx -v`
Expected: FAIL，开始进入实现阶段

**Step 5: Commit**

```bash
git add httpserver/integration/errorsx docs/plans
git commit -m "test(httpserver): add errorsx integration tests"
```

### Task 2: ResponseBody 与映射实现

**Files:**
- Create: `httpserver/integration/errorsx/mapper.go`
- Test: `httpserver/integration/errorsx/mapper_test.go`

**Step 1: Write the failing test**

锁定：
- `Response(err)` 返回正确的 status/code/name/message
- 普通错误走 `INTERNAL_ERROR`

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/integration/errorsx -run 'TestResponse' -v`
Expected: FAIL

**Step 3: Write minimal implementation**

实现：
- `ResponseBody`
- `Response(err)`
- `Mapper()`

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/integration/errorsx -run 'TestResponse|TestMapper' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/integration/errorsx
git commit -m "feat(httpserver): add errorsx integration mapper"
```

### Task 3: 文档与能力清单联动

**Files:**
- Modify: `AGENTS.md`
- Modify: `httpserver/README.md`
- Modify: `.ai/capabilities.yaml`
- Create: `httpserver/integration/errorsx/README.md`
- Create: `httpserver/integration/errorsx/doc.go`

**Step 1: Update docs**

补充：
- `AGENTS.md` 能力速查
- `httpserver/README.md` 的 integration 说明
- `errorsx` 子包 README 与 doc.go

**Step 2: Validate YAML and package listing**

Run: `ruby -e 'require "yaml"; YAML.load_file(".ai/capabilities.yaml")'`
Expected: PASS

Run: `cd cmd/gokit && GOCACHE=/tmp/go-build go run . list`
Expected: 输出包含 `errorsx`

**Step 3: Commit**

```bash
git add AGENTS.md .ai/capabilities.yaml httpserver/README.md httpserver/integration/errorsx
git commit -m "docs(httpserver): document errorsx integration"
```

### Task 4: 全量验证

**Files:**
- Verify only

**Step 1: Run package tests**

Run: `cd /Users/kj/projects/go-kit && GOCACHE=/tmp/go-build go test ./httpserver/... -v`
Expected: PASS

**Step 2: Run lint**

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
Expected: `0 issues.`

**Step 3: Confirm diff scope**

Run: `git diff --stat -- AGENTS.md .ai/capabilities.yaml httpserver docs/plans`
Expected: 只包含本轮 errorsx 集成与前序 typed handler 改动

**Step 4: Commit**

```bash
git add AGENTS.md .ai/capabilities.yaml httpserver docs/plans
git commit -m "feat(httpserver): add errors package integration"
```
