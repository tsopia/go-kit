package llm

import "testing"

func TestAgentConfigDefaults(t *testing.T) {
	cfg := AgentConfig{}

	if cfg.Model.Config.Protocol != "" {
		t.Fatalf("expected empty model protocol, got %q", cfg.Model.Config.Protocol)
	}
	if cfg.Model.Config.Model != "" {
		t.Fatalf("expected empty model name, got %q", cfg.Model.Config.Model)
	}
	if cfg.Model.Config.Stop != nil {
		t.Fatalf("expected nil model stop list, got %#v", cfg.Model.Config.Stop)
	}
	if cfg.Model.Instance != nil {
		t.Fatalf("expected nil model instance, got %#v", cfg.Model.Instance)
	}
	if cfg.Prompt.System != "" {
		t.Fatalf("expected empty prompt system, got %q", cfg.Prompt.System)
	}
	if cfg.Prompt.PrepareMessages != nil {
		t.Fatal("expected nil prompt prepare hook")
	}
	if cfg.Prompt.RewriteHistory != nil {
		t.Fatal("expected nil prompt rewrite hook")
	}
	if cfg.Tools.Standard != nil {
		t.Fatalf("expected nil standard tools, got %#v", cfg.Tools.Standard)
	}
	if cfg.Tools.Invokable != nil {
		t.Fatalf("expected nil invokable tools, got %#v", cfg.Tools.Invokable)
	}
	if cfg.Tools.MCPServers != nil {
		t.Fatalf("expected nil MCP servers, got %#v", cfg.Tools.MCPServers)
	}
	if cfg.Execution.Mode != "" {
		t.Fatalf("expected zero value mode before normalization, got %q", cfg.Execution.Mode)
	}
	if cfg.Execution.ToolChoice != nil {
		t.Fatalf("expected nil execution tool choice, got %#v", cfg.Execution.ToolChoice)
	}
	if cfg.Execution.MaxRetries != 0 {
		t.Fatalf("expected zero max retries, got %d", cfg.Execution.MaxRetries)
	}
	if cfg.Execution.MaxStep != 0 {
		t.Fatalf("expected zero max step, got %d", cfg.Execution.MaxStep)
	}
	if cfg.Execution.DirectReturnTools != nil {
		t.Fatalf("expected nil direct return tools, got %#v", cfg.Execution.DirectReturnTools)
	}
	if cfg.Streaming.ToolCallChecker != nil {
		t.Fatal("expected nil streaming tool call checker")
	}
	if cfg.Observability.Callbacks != nil {
		t.Fatalf("expected nil callbacks, got %#v", cfg.Observability.Callbacks)
	}
	if cfg.Observability.StructuredLogs != nil {
		t.Fatalf("expected nil structured logs, got %#v", cfg.Observability.StructuredLogs)
	}

	sl := StructuredLogConfig{}
	if sl.Logger != nil {
		t.Fatalf("expected nil structured log logger, got %#v", sl.Logger)
	}
	if sl.LogToolArguments {
		t.Fatal("expected false log tool arguments by default")
	}
	if sl.LogToolResults {
		t.Fatal("expected false log tool results by default")
	}
	if sl.MaxFieldLength != 0 {
		t.Fatalf("expected zero max field length, got %d", sl.MaxFieldLength)
	}
}
