package gokit

import (
	"testing"
)

func TestLoadCapabilities(t *testing.T) {
	caps, err := LoadCapabilities()
	if err != nil {
		t.Fatalf("LoadCapabilities failed: %v", err)
	}

	if len(caps) == 0 {
		t.Error("Expected at least one capability")
	}

	// Check first capability has required fields
	if caps[0].Name == "" {
		t.Error("Capability name should not be empty")
	}
}

func TestGetCapabilitySwagger(t *testing.T) {
	t.Parallel()

	capability, err := GetCapability("swagger")
	if err != nil {
		t.Fatalf("GetCapability(swagger) failed: %v", err)
	}

	if capability.Import != "github.com/tsopia/go-kit/httpserver/swagger" {
		t.Fatalf("unexpected import: %s", capability.Import)
	}

	if len(capability.Scenarios) == 0 {
		t.Fatal("expected swagger scenarios")
	}
}
