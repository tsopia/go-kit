# HTTPServer Streaming And Upload Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 重做 `httpserver` 的 SSE、WebSocket、文件上传传输层抽象，去掉当前不稳定的 API 设计，收敛到协议语义清晰、测试可验证、文档一致的一套新接口。

**Architecture:** 保留 `httpserver` 作为 Gin 之上的轻量传输层，但重做三块边界。SSE 改为显式 `SSEStream` 抽象，统一“写事件 + 访问请求信息 + 控制 heartbeat 生命周期”；WebSocket 改为 `WSSession` 抽象，采用单 data writer、`WriteControl(Ping)`、显式 close/queue 语义，不再暴露裸 `send chan`; 文件上传不再保留 `HandleUpload`，统一回归 `HandleForm` + 通用 handler 选项（如 `WithMaxBodyBytes`）。

**Tech Stack:** Go 1.24、Gin、Gorilla WebSocket、typed handlers、table-driven tests、`go test -race`

---

## 前置检查

- [x] 已完整阅读相关代码文件：`httpserver/server.go`、`httpserver/sse.go`、`httpserver/ws.go`、`httpserver/ws_sender.go`、`httpserver/ws_hub.go`、`httpserver/handler.go`、`httpserver/README.md`、`docs/httpserver.md`
- [x] 已完整阅读相关测试文件：`httpserver/sse_test.go`、`httpserver/ws_test.go`、`httpserver/ws_integration_test.go`、`httpserver/integration_test.go`、`httpserver/handler_test.go`、`httpserver/ws_hub_test.go`、`httpserver/server_test.go`
- [x] 已列出代码现状（每个判断附行号）

## 代码现状

- **确定**：`Server.WS` 当前存在两个独立 goroutine 对同一个 `*websocket.Conn` 执行写操作，分别位于 `httpserver/server.go:307-312` 和 `httpserver/server.go:328-337`，违反 gorilla/websocket 的单 writer 约束。
- **确定**：`HandleUpload` 仍然走 `Handle` 默认 JSON decoder，真实 multipart 解析实际依赖 `HandleForm`，见 `httpserver/handler.go:151-160`、`httpserver/handler.go:248-267`、`httpserver/handler.go:270-277`。
- **确定**：SSE heartbeat 通过等待 heartbeat goroutine 退出收尾，会让“有限流 + heartbeat”在 handler 返回后无法自然结束，见 `httpserver/server.go:158-168`、`httpserver/sse.go:121-145`。
- **确定**：README 中 WebSocket 示例依赖 `ctx.Value("params")` 读取路由参数，但 `ContextFromGin` 只返回 `c.Request.Context()`，见 `httpserver/README.md:669-679`、`httpserver/server.go:612-620`。
- **确定**：上传文档已经分叉；`docs/httpserver.md` 仍把上传放在 `HandleForm`，而 `httpserver/README.md` 已切到 `HandleUpload`，见 `docs/httpserver.md:347-353`、`httpserver/README.md:689-697`。

## 目标 API

### SSE

- `type SSEHandlerFunc func(stream SSEStream)`
- `type SSEStream interface { Context() context.Context; Request() *http.Request; Param(name string) string; Event(name string, data any) error; Data(data any) error; Comment(text string) error }`
- heartbeat 是 `Server.SSE` 内部实现细节，handler 返回时必须主动停止 heartbeat 并结束响应。

### WebSocket

- `type WSHandlerFunc func(session WSSession)`
- `type WSSession interface { Context() context.Context; Request() *http.Request; Param(name string) string; Recv() <-chan WSMessage; Send(msg WSMessage) error; TrySend(msg WSMessage) bool; Close(code int, reason string) error }`
- 删除 `RecvPolicy` / `SendPolicy` 公开配置；保留 `SendBufferSize`、`PingPeriod`、`PongTimeout`、`WriteTimeout`、`ReadIdleTimeout`
- ping 使用 `WriteControl(websocket.PingMessage, ...)`
- `Send` 语义是“消息已入内部发送队列或连接已结束”；`TrySend` 语义是“非阻塞入队”

### Upload

- 删除 `HandleUpload`
- 新推荐写法：`HandleForm(uploadHandler, WithMaxBodyBytes(limit))`
- 若将来需要更细粒度流式上传，再单独设计，不把“禁用 deadline”继续塞进 typed handler API

---

### Task 1: 用失败测试锁定新的上传 contract

**目标**：先把“上传 = `HandleForm` + `WithMaxBodyBytes`，而不是 `HandleUpload`”这个新 contract 锁进测试。

**文件变更**：
- 修改：`httpserver/handler_test.go:724-784`

**约束检查**：
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）**：
- [ ] **Step 1: 写失败测试**
  - 将现有上传测试改写为：
    - `HandleForm(..., WithMaxBodyBytes(1024))` 能处理 JSON 与 multipart/form-data
    - 超限时返回 `413`
    - 新增一个真实 multipart 上传测试，请求体里包含 `file` 字段和一个普通 form 字段
- [ ] **Step 2: 运行测试 → 确认失败**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandle(FormMultipartUpload|FormBodyTooLarge)$' -v`
  - Expected: FAIL，因为 `WithMaxBodyBytes` 还不存在，multipart 路径也未被锁定
- [ ] **Step 3: 写最简实现**
  - 此 Task 不改实现，只改测试
- [ ] **Step 4: 运行测试 → 确认失败**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandle(FormMultipartUpload|FormBodyTooLarge)$' -v`
  - Expected: FAIL
- [ ] **Step 5: Commit**
  - `git add httpserver/handler_test.go`
  - `git commit -m "test(httpserver): redefine upload contract around HandleForm"`

### Task 2: 实现通用 body limit 选项并删除 `HandleUpload`

**目标**：把上传大小限制能力收回 typed handler 通用层，移除误导性的 `HandleUpload`。

**文件变更**：
- 修改：`httpserver/handler.go:49-80`
- 修改：`httpserver/handler.go:163-277`
- 修改：`httpserver/handler_test.go:724-784`

**约束检查**：
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）**：
- [ ] **Step 1: 写最简实现**
  - 在 `handlerConfig` 中增加 `beforeDecode []func(*gin.Context) error`
  - 在 `Handle` 中先执行 `beforeDecode`，再执行 `cfg.decoder`
  - 新增 `WithMaxBodyBytes(limit int64) HandlerOption`，内部用 `http.MaxBytesReader`
  - 删除 `HandleUpload`
- [ ] **Step 2: 运行测试 → 确认通过**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandle(FormMultipartUpload|FormBodyTooLarge)$' -v`
  - Expected: PASS
- [ ] **Step 3: 补充回归测试**
  - 运行原有 typed handler 相关测试，确认通用链路未破坏
- [ ] **Step 4: 再次验证**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandle(FormMultipartUpload|FormBodyTooLarge|HandleFormJSONContentType|HandleFormURLEncoded)$' -v`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - `git add httpserver/handler.go httpserver/handler_test.go`
  - `git commit -m "refactor(httpserver): replace HandleUpload with generic body limit option"`

### Task 3: 用失败测试锁定新的 SSE stream contract

**目标**：先把 SSE 的两个新行为锁死：handler 可访问请求信息；heartbeat 不阻塞有限流自然结束。

**文件变更**：
- 修改：`httpserver/sse_test.go:1-290`
- 修改：`httpserver/server_test.go:879-913`
- 修改：`httpserver/integration_test.go:58-110`

**约束检查**：
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）**：
- [ ] **Step 1: 写失败测试**
  - 新增测试覆盖：
    - `SSEHandlerFunc` 新签名可访问 `stream.Request()` 与 `stream.Param("id")`
    - 开启 `WithHeartbeat` 时，handler 发送完有限条事件后返回，响应能自然结束，不必等客户端断开
- [ ] **Step 2: 运行测试 → 确认失败**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(SSEStreamProvidesRequestAndParams|SSEHeartbeatDoesNotBlockFiniteStream)$' -v`
  - Expected: FAIL，因为 `SSEStream` 还不存在，heartbeat 退出语义也不对
- [ ] **Step 3: 写最简实现**
  - 此 Task 不改实现，只改测试
- [ ] **Step 4: 运行测试 → 确认失败**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(SSEStreamProvidesRequestAndParams|SSEHeartbeatDoesNotBlockFiniteStream)$' -v`
  - Expected: FAIL
- [ ] **Step 5: Commit**
  - `git add httpserver/sse_test.go httpserver/server_test.go httpserver/integration_test.go`
  - `git commit -m "test(httpserver): lock new sse stream contract"`

### Task 4: 引入 `SSEStream` 并重写 heartbeat 生命周期

**目标**：把 SSE 抽象重做成“请求信息 + event writer”统一对象，并让 heartbeat 生命周期从属于请求处理，而不是反过来阻塞请求结束。

**文件变更**：
- 修改：`httpserver/sse.go:14-177`
- 修改：`httpserver/server.go:132-170`
- 修改：`httpserver/sse_test.go:1-290`
- 修改：`httpserver/server_test.go:879-913`
- 修改：`httpserver/integration_test.go:58-110`

**约束检查**：
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）**：
- [ ] **Step 1: 写最简实现**
  - 定义 `SSEStream` 接口和内部 `sseStream` 实现
  - `Server.SSE` 负责构建 `context.WithCancel`
  - heartbeat 改为 `startHeartbeat(ctx) (stop func())`
  - handler 返回时调用 `stop()`，等待 heartbeat goroutine 退出，但不要求客户端先断开
  - `stream.Param(name)` 直接从 `gin.Context.Param(name)` 读取
- [ ] **Step 2: 运行测试 → 确认通过**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(SSEStreamProvidesRequestAndParams|SSEHeartbeatDoesNotBlockFiniteStream|SSESender_|Server_SSE)' -v`
  - Expected: PASS
- [ ] **Step 3: 跑一次 race 相关增量验证**
  - Run: `GOCACHE=/tmp/go-build go test -race ./httpserver -run 'TestSSE|TestServer_SSE' -count=1 -v`
  - Expected: PASS
- [ ] **Step 4: Commit**
  - `git add httpserver/sse.go httpserver/server.go httpserver/sse_test.go httpserver/server_test.go httpserver/integration_test.go`
  - `git commit -m "refactor(httpserver): redesign sse stream lifecycle"`

### Task 5: 用失败测试锁定新的 WebSocket session contract

**目标**：先通过测试锁定新的公开 API，避免继续沿用 `recv/send chan` 语义。

**文件变更**：
- 修改：`httpserver/ws_test.go:1-459`
- 修改：`httpserver/ws_hub_test.go:1-100`
- 修改：`httpserver/ws_integration_test.go:14-171`

**约束检查**：
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）**：
- [ ] **Step 1: 写失败测试**
  - 将 handler 签名切到 `func(session WSSession)`
  - 新增测试覆盖：
    - `session.Param("id")` 和 `session.Request()` 可用
    - `session.Send` 与 `session.TrySend` 语义明确
    - `WSHub` 改为基于 `TrySend` 广播，而不是 `chan<- WSMessage`
- [ ] **Step 2: 运行测试 → 确认失败**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(WSSessionContract|WSHub|Server_WS)' -v`
  - Expected: FAIL，因为公开类型和 hub 契约都还没重做
- [ ] **Step 3: 写最简实现**
  - 此 Task 不改实现，只改测试
- [ ] **Step 4: 运行测试 → 确认失败**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(WSSessionContract|WSHub|Server_WS)' -v`
  - Expected: FAIL
- [ ] **Step 5: Commit**
  - `git add httpserver/ws_test.go httpserver/ws_hub_test.go httpserver/ws_integration_test.go`
  - `git commit -m "test(httpserver): lock websocket session contract"`

### Task 6: 引入 `WSSession`、收敛公开配置并改造 `WSHub`

**目标**：先把公共类型层做对，再动运行时。

**文件变更**：
- 修改：`httpserver/ws.go:11-130`
- 创建：`httpserver/ws_session.go`
- 修改：`httpserver/ws_hub.go:7-66`
- 修改：`httpserver/ws_test.go:1-459`
- 修改：`httpserver/ws_hub_test.go:1-100`

**约束检查**：
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）**：
- [ ] **Step 1: 写最简实现**
  - 在 `ws.go` 定义：
    - `WSSession` 接口
    - 新的 `WSHandlerFunc`
    - `WSRouteConfig` 仅保留 `SendBufferSize`、`PingPeriod`、`PongTimeout`、`WriteTimeout`、`ReadIdleTimeout`
  - 在 `ws_hub.go` 中把房间成员类型改为 `interface{ TrySend(WSMessage) bool }`
  - 删除 `RecvPolicy`、`SendPolicy`、`WithRecvBuffer`、旧 `WithSendBuffer(size, policy)`，改为 `WithWSSendBuffer(size int)`
- [ ] **Step 2: 运行测试 → 确认部分通过**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(WSConfig_|WSSessionContract|WSHub)' -v`
  - Expected: 公开类型与 hub 相关测试 PASS；运行时集成测试仍可能 FAIL
- [ ] **Step 3: 补齐最小桩实现**
  - 在 `ws_session.go` 提供最小可编译的 `wsSession` 结构，先满足编译
- [ ] **Step 4: 再次验证**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(WSConfig_|WSSessionContract|WSHub|Server_WS)' -v`
  - Expected: PASS 或仅剩 runtime 路径失败
- [ ] **Step 5: Commit**
  - `git add httpserver/ws.go httpserver/ws_session.go httpserver/ws_hub.go httpserver/ws_test.go httpserver/ws_hub_test.go`
  - `git commit -m "refactor(httpserver): introduce websocket session api"`

### Task 7: 重写 `Server.WS` 运行时为单 writer 架构

**目标**：彻底去掉当前多 goroutine 直写 conn 的问题，让 WS 生命周期和协议语义一致。

**文件变更**：
- 修改：`httpserver/server.go:172-355`
- 修改：`httpserver/ws_session.go`
- 修改：`httpserver/ws_integration_test.go:14-171`
- 修改：`httpserver/ws_test.go:148-459`
- 保留或删除：`httpserver/ws_sender.go`（若 `wsSession` 已完全吸收其职责，则删除）

**约束检查**：
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）**：
- [ ] **Step 1: 写失败测试**
  - 新增/改写运行时测试锁定：
    - ping 走 `WriteControl`，不再与 data frame writer 冲突
    - `ReadIdleTimeout` 在每次收到消息或 pong 后重置
    - handler 返回时，writer goroutine 能完成 close/退出，不遗留 goroutine
    - `TrySend` 在队列满时返回 `false`，不会阻塞广播
- [ ] **Step 2: 运行测试 → 确认失败**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestWS_(ReadIdleTimeout|TrySend|PongTimeout|GoroutineCleanup|Integration_Echo)$' -v`
  - Expected: FAIL，因为 runtime 仍是旧架构
- [ ] **Step 3: 写最简实现**
  - 在 `Server.WS` 中：
    - upgrade 后创建 `sessionCtx`
    - 启动一个 read pump，唯一负责 `ReadMessage`
    - 启动一个 data write pump，唯一负责 `WriteMessage`
    - 独立 ping loop 只调用 `WriteControl`
    - `session.Close` 负责发送 close frame 并触发 cancel
  - `ReadIdleTimeout` 语义改为“读空闲超时”，每次 inbound message / pong 均刷新时间戳
- [ ] **Step 4: 运行测试 → 确认通过**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestWS_(ReadIdleTimeout|TrySend|PongTimeout|GoroutineCleanup|Integration_Echo|SendPolicy_Block|SendPolicy_DropNewest|SendPolicy_DropOldest|SendPolicy_Disconnect)$' -v`
  - Expected: PASS；同时删除或重写旧 policy 测试名称
- [ ] **Step 5: Commit**
  - `git add httpserver/server.go httpserver/ws_session.go httpserver/ws_integration_test.go httpserver/ws_test.go httpserver/ws_sender.go`
  - `git commit -m "refactor(httpserver): rebuild websocket runtime around single writer session"`

### Task 8: 更新文档与示例，清理旧术语

**目标**：让 README、设计文档、示例代码与新 API 完全一致，不再留下旧抽象入口。

**文件变更**：
- 修改：`httpserver/README.md:580-697`
- 修改：`docs/httpserver.md:336-372`
- 可选修改：`httpserver/doc.go`

**约束检查**：
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）**：
- [ ] **Step 1: 先改文档**
  - README 中：
    - SSE 示例改成 `func(stream httpserver.SSEStream)`
    - WS 示例改成 `func(session httpserver.WSSession)`
    - 上传示例改成 `HandleForm(..., WithMaxBodyBytes(...))`
    - 删除 `SendPolicy`、`RecvPolicy`、`HandleUpload`
  - `docs/httpserver.md` 同步同样的 contract
- [ ] **Step 2: 运行文档相关测试/示例验证**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(Server_SSE|Server_WS|HandleForm)' -v`
  - Expected: PASS
- [ ] **Step 3: Commit**
  - `git add httpserver/README.md docs/httpserver.md httpserver/doc.go`
  - `git commit -m "docs(httpserver): align streaming and upload api documentation"`

### Task 9: 验证闭环

**目标**：在不跑全仓不必要路径的前提下，拿到这次重构的完整证据链。

**文件变更**：
- Verify only

**约束检查**：
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）**：
- [ ] **Step 1: 包级测试**
  - Run: `GOCACHE=/tmp/go-build go test ./httpserver/... -v`
  - Expected: PASS
- [ ] **Step 2: race 验证**
  - Run: `GOCACHE=/tmp/go-build go test -race ./httpserver -run 'TestSSE|TestWS' -count=1 -v`
  - Expected: PASS
- [ ] **Step 3: 构建验证**
  - Run: `GOCACHE=/tmp/go-build go build ./httpserver/...`
  - Expected: PASS
- [ ] **Step 4: lint**
  - Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
  - Expected: `0 issues.`
- [ ] **Step 5: 可选全仓测试**
  - Run: `GOCACHE=/tmp/go-build go test ./...`
  - Expected: 若命中已知 `sonic` 基线问题，则在交付说明中明确为仓库基线，不归因于本次改动
- [ ] **Step 6: Commit**
  - `git add httpserver docs/httpserver.md docs/plans`
  - `git commit -m "refactor(httpserver): redesign streaming and upload contracts"`
