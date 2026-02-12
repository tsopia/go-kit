# Go Project Standards

## 🛠 Build & Test Commands
- **Install Dependencies**: `go mod tidy`
- **Build**: `go build ./...`
- **Unit Test**: `go test -v ./...`
- **Test with Coverage**: `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`
- **Lint**: `golangci-lint run`

## 📋 Go Coding Standards
- **Errors**:
  - 必须显式处理所有 `error`（禁止使用 `_` 忽略）。
  - 使用 `fmt.Errorf("context: %w", err)` 包装错误以保留堆栈信息。
- **Naming**:
  - 导出变量/函数使用 `PascalCase`，内部使用 `camelCase`。
  - 缩写词全大写（如 `JSON`, `API`, `ID`）。
- **Receivers**:
  - 除非有极特殊的理由（如不可变小结构体），否则一致使用指针接收者 `(s *Service)`。
- **Concurrency**:
  - 必须支持 `context.Context` 传播和超时控制。
  - 优先使用 Channel 实现逻辑流，使用 Mutex 保护简单的状态。

## 🧪 QA & Testing
- **Pattern**: 强制使用 **Table-driven tests** (表格驱动测试)。
- **Location**: 测试文件必须位于被测代码同级目录，命名为 `*_test.go`。
- **Mocking**: 优先通过 Interface 抽象进行 Mock，而非使用猴子补全（Monkey Patching）。