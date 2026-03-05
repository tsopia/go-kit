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
