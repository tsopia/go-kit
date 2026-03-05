package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Name:         "test-api",
		Module:       "github.com/example/test-api",
		GoKitModule:  "github.com/tsopia/go-kit",
		GoKitVersion: "v1.0.0",
		Template:     "api",
		OutputDir:    tmpDir,
	}

	err := CreateProject(cfg)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Check that main.go was created
	mainFile := filepath.Join(tmpDir, "cmd", "test-api", "main.go")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		t.Errorf("Expected main.go to exist at %s", mainFile)
	}
}
