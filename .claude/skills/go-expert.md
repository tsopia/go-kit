---
name: go-expert
description: 用于将散乱的逻辑重构成高性能、可测试、符合 Go 惯用法的工具包或模块
---

当触发此技能时，请按照以下步骤处理代码：

1. **接口抽象 (Interface Extraction)**：
   - 分析核心逻辑，将其提取为最小化的接口（Interface Segregation）。
   - 确保外部依赖（如数据库、API）通过接口注入，而非在函数内硬编码初始化。

2. **封装优化 (Encapsulation)**：
   - 检查是否有可以设为私有的字段或方法（减少暴露面积）。
   - 将通用逻辑移动到 `pkg/`，项目私有逻辑移动到 `internal/`。

3. **健壮性增强**：
   - 检查所有切片（Slice）和 Map 的初始化，如果已知长度，请使用 `make(type, 0, capacity)` 以优化内存。
   - 检查是否存在 Goroutine 泄露风险，确保所有的 Channel 都有关闭机制或 Context 退出机制。

4. **自动化文档**：
   - 为所有导出的函数编写符合 `go doc` 规范的注释（以函数名开头）。
   - 如果是一个包，生成一个简单的 `example_test.go` 展示如何使用。

5. **QA 验证**：
   - 自动生成一个覆盖核心路径的 Table-driven unit test。