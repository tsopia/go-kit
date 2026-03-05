package template

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	data := Data{
		Name:         "test-service",
		Module:       "github.com/example/test-service",
		GoKitModule:  "github.com/tsopia/go-kit",
		GoKitVersion: "v1.0.0",
		Features:     []string{"kit"},
	}

	tmpl := "package {{.Name}}"
	result, err := Render("test", tmpl, data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "package test-service") {
		t.Errorf("Expected 'package test-service', got: %s", result)
	}
}
