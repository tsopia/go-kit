# Stream Inspect 测试实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 编写连接真实 DeepSeek / Qwen API 的测试，观察流式 + thinking 模式下每个 chunk 的返回结构。

**Architecture:** 单个测试文件 `llm/stream_inspect_test.go`，两个测试函数分别测试 DeepSeek 和 Qwen，从环境变量读取 API key，逐 chunk 打印完整字段。

**Tech Stack:** Go, Eino, `llm` 包现有 Agent.Stream()

---

### Task 1: 编写 stream_inspect_test.go

**Files:**
- Create: `llm/stream_inspect_test.go`

- [ ] **Step 1: 编写 DeepSeek 流式 inspect 测试**

```go
func TestInspectDeepSeekStream(t *testing.T) {
    // 从环境变量读取配置，缺失则 Skip
    // 创建 Agent（Conversation 模式，Thinking 开启）
    // 调用 Agent.Stream()
    // 逐 chunk 打印 Role/Content/ContentLen/ToolCalls/ResponseMeta
    // 流结束后打印完整聚合内容
}
```

- [ ] **Step 2: 编写 Qwen 流式 inspect 测试**

类似 Step 1，但使用 QWEN_API_KEY / QWEN_MODEL。

- [ ] **Step 3: Commit**

```bash
git add llm/stream_inspect_test.go
git commit -m "test(llm): add stream inspect tests for deepseek and qwen"
```

### Task 2: 运行测试并观察输出

- [ ] **Step 1: 设置环境变量**

```bash
export DEEPSEEK_API_KEY="..."
export DEEPSEEK_MODEL="deepseek-reasoner"
export QWEN_API_KEY="..."
export QWEN_MODEL="qwen3-235b-a22b"
```

- [ ] **Step 2: 运行 DeepSeek 测试**

```bash
go test -v ./llm -run TestInspectDeepSeekStream -timeout 60s
```

- [ ] **Step 3: 运行 Qwen 测试**

```bash
go test -v ./llm -run TestInspectQwenStream -timeout 60s
```
