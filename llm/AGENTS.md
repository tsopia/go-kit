# Agent Instructions (Codex)

## Goal
Implement a Go package that wraps CloudWeGo Eino chat models with:
- protocol-based routing (OPENAI_COMPAT / CLAUDE_COMPAT) with optional BaseURL
- a tool-call loop with policies (optional / required-one / required-exact)
- tool execution with argument validation and structured error feedback
- support for local tools + MCP tools (MCP client lifecycle is owned by caller)

## Repo layout
Create a new package under:
- ./llm

Suggested files (you may adjust, but keep public API stable):
- llm/config.go          (ModelConfig + enums + policies)
- llm/factory.go         (NewToolCallingModel routing)
- llm/toolloop.go        (RunToolCallLoop + state machine)
- llm/feedback.go        (feedback templates)
- llm/validation.go      (arg validation + issues normalization)
- llm/result.go          (RunResult + StopReason)
- llm/*_test.go          (unit tests)

## Constraints
- Follow the public API names and semantics defined in PLANS.md exactly.
- Use real CloudWeGo Eino APIs where possible.
- Do NOT manage MCP client lifecycle. Assume caller already initialized MCP clients and provides MCP-derived tools (via mcp.GetTools).

## Commands to run
- gofmt ./...
- go test ./...

## Notes
- Prefer deterministic, testable code.
- Add unit tests with fakes/mocks for the model and tools.
