package llm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/compose"
)

func TestApplyToolDefenses_MapsAllFields(t *testing.T) {
	called := false
	cfg := ToolsConfig{
		Aliases: map[string][]string{"search": {"find", "query_tool"}},
		UnknownHandler: func(_ context.Context, name, _ string) (string, error) {
			return "unknown: " + name, nil
		},
		ArgumentsFixer: func(_ context.Context, _, args string) (string, error) {
			called = true
			return args, nil
		},
		ErrorToText: boolPtr(true),
	}

	var tnc compose.ToolsNodeConfig
	applyToolDefenses(&tnc, cfg)

	alias, ok := tnc.ToolAliases["search"]
	if !ok || len(alias.NameAliases) != 2 || alias.NameAliases[0] != "find" {
		t.Errorf("aliases not mapped: %+v", tnc.ToolAliases)
	}
	if tnc.UnknownToolsHandler == nil {
		t.Error("UnknownToolsHandler not set")
	}
	if tnc.ToolArgumentsHandler == nil {
		t.Error("ToolArgumentsHandler not set")
	} else {
		_, _ = tnc.ToolArgumentsHandler(context.Background(), "t", "{}")
		if !called {
			t.Error("ArgumentsFixer not wired through")
		}
	}
	if len(tnc.ToolCallMiddlewares) != 1 {
		t.Errorf("ErrorToText middleware not appended, got %d middlewares", len(tnc.ToolCallMiddlewares))
	}
}

func TestApplyToolDefenses_NoConfig_NoChange(t *testing.T) {
	var tnc compose.ToolsNodeConfig
	applyToolDefenses(&tnc, ToolsConfig{})
	if tnc.ToolAliases != nil || tnc.UnknownToolsHandler != nil ||
		tnc.ToolArgumentsHandler != nil || len(tnc.ToolCallMiddlewares) != 0 {
		t.Error("empty ToolsConfig should leave ToolsNodeConfig untouched")
	}
}

func TestApplyToolDefenses_ErrorToTextDisabledByDefault(t *testing.T) {
	var tnc compose.ToolsNodeConfig
	applyToolDefenses(&tnc, ToolsConfig{}) // ErrorToText nil
	if len(tnc.ToolCallMiddlewares) != 0 {
		t.Error("ErrorToText must be off by default (nil)")
	}
	var tnc2 compose.ToolsNodeConfig
	applyToolDefenses(&tnc2, ToolsConfig{ErrorToText: boolPtr(false)})
	if len(tnc2.ToolCallMiddlewares) != 0 {
		t.Error("ErrorToText=false must not append middleware")
	}
}

func TestErrorToTextMiddleware_ConvertsErrorToResult(t *testing.T) {
	mw := errorToTextMiddleware()
	endpoint := mw.Invokable(func(_ context.Context, _ *compose.ToolInput) (*compose.ToolOutput, error) {
		return nil, errors.New("boom")
	})

	out, err := endpoint(context.Background(), &compose.ToolInput{Name: "t", Arguments: "{}"})
	if err != nil {
		t.Fatalf("expected nil error (converted to text), got %v", err)
	}
	if out == nil || !strings.Contains(out.Result, "boom") {
		t.Fatalf("expected result text containing the error, got %+v", out)
	}
}

func TestErrorToTextMiddleware_PassThroughOnSuccess(t *testing.T) {
	mw := errorToTextMiddleware()
	endpoint := mw.Invokable(func(_ context.Context, _ *compose.ToolInput) (*compose.ToolOutput, error) {
		return &compose.ToolOutput{Result: "ok"}, nil
	})
	out, err := endpoint(context.Background(), &compose.ToolInput{Name: "t"})
	if err != nil || out == nil || out.Result != "ok" {
		t.Fatalf("success path should pass through unchanged, got out=%+v err=%v", out, err)
	}
}

func boolPtr(b bool) *bool { return &b }
