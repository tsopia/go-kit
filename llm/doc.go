// Package llm 提供大模型 Agent 的统一封装。
//
// 路径选择：
//
//   - 新代码请使用 [NewADKAgent]（基于 eino adk.ChatModelAgent，go-kit 主推路径）。
//   - [NewDeepAgent] 用于复杂任务（内置规划/子Agent委派/文件系统/Shell 的预置应用）。
//   - [NewAgent]（基于 eino react.Agent）为 go-kit legacy 路径，仅维护向后兼容、
//     不再演进；新能力只在 ADK 路径实现。
//
// 说明：「legacy」是 go-kit 自身的产品取向（顺应 eino 把核心能力投入 ADK 的方向），
// 并不代表 eino 已废弃 react —— 截至 eino v0.9.8，flow/agent/react 仍在维护。
// 切换成本：仅改函数名，AgentConfig 配置完全兼容。
//
// 三个工厂共享配置层（AgentConfig）和模型工厂（NewModel）：
//
//   - [NewADKAgent]: 基于 eino adk.ChatModelAgent（v0.9 ADK 路径，主推）
//   - [NewDeepAgent]: 基于 eino adk.DeepAgent（内置规划/子Agent委派/文件系统/Shell）
//   - [NewAgent]: 基于 eino react.Agent（legacy，向后兼容）
//
// 核心能力：
//
//   - 多供应商模型路由（OpenAI/Claude/DeepSeek/Gemini/Ark/Ollama/Qwen 等）
//   - 三种执行模式：Conversation（纯对话）、Assistant（工具可选）、
//     Extraction（强制工具调用 + 失败修复重试）
//   - 并发控制（[ConcurrencyConfig.MaxConcurrency]，Agent 调用级信号量，
//     多实例互不影响）
//   - 思考模式统一配置（[ThinkingConfig]，Extraction 模式自动关闭）
//   - MCP 工具集成
//   - 可观测性（Langfuse callback、结构化日志）
//
// Extraction 模式下强制工具调用时会自动关闭思考模式，
// 因为部分模型在 tool_choice=forced 下对 reasoning 输出支持不稳定，
// 且 Extraction 场景只需确定性参数提取，无需额外推理。
package llm
