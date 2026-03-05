# Gokit CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a CLI tool (`gokit`) that scaffolds new Go projects using go-kit library with AI-friendly templates.

**Architecture:** A cobra-based CLI with subcommands (new/init/add/update/list) that generates project templates with proper structure, AI configuration files, and go-kit integration.

**Tech Stack:** Go 1.24, Cobra (CLI), text/template (templates), embed (embedding templates), yaml.v3

---

## Prerequisites

- Go 1.24+ installed
- This is a subdirectory project under `cmd/gokit/`
- Module: `github.com/tsopia/go-kit`

---

## Task 1: Initialize CLI Module Structure

**Files:**
- Create: `cmd/gokit/go.mod`
- Create: `cmd/gokit/main.go`

**Step 1: Create module file**

```go
module github.com/tsopia/go-kit/cmd/gokit

go 1.24.0

require (
	github.com/spf13/cobra v1.8.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)
```

**Step 2: Create main.go with basic structure**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gokit",
	Short: "Go-kit project scaffolding tool",
	Long: `gokit is a CLI tool for creating and managing Go projects
that use the go-kit library. It provides project templates,
AI-friendly configuration, and go-kit integration.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 3: Download dependencies**

Run: `cd cmd/gokit && go mod tidy`
Expected: Dependencies downloaded, go.sum created

**Step 4: Test basic CLI**

Run: `cd cmd/gokit && go run main.go --help`
Expected: Shows help text with "Go-kit project scaffolding tool"

**Step 5: Commit**

```bash
git add cmd/gokit/
git commit -m "feat(cli): initialize gokit CLI module structure"
```

---

## Task 2: Create Capability Types and Loader

**Files:**
- Create: `cmd/gokit/pkg/gokit/capability.go`
- Create: `cmd/gokit/pkg/gokit/capability_test.go`

**Step 1: Write the failing test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd cmd/gokit && go test ./pkg/gokit/... -v`
Expected: FAIL - "package not found" or "undefined: LoadCapabilities"

**Step 3: Create capability types and loader**

```go
package gokit

import (
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed capabilities.yaml
capabilitiesFS embed.FS

type Scenario struct {
	Name    string `yaml:"name"`
	Snippet string `yaml:"snippet"`
}

type Capability struct {
	Name         string    `yaml:"name"`
	Description  string    `yaml:"description"`
	Import       string    `yaml:"import"`
	Scenarios    []Scenario `yaml:"scenarios"`
	Dependencies []string  `yaml:"dependencies"`
}

type CapabilityRegistry struct {
	Version      string       `yaml:"version"`
	UpdatedAt    string       `yaml:"updated_at"`
	Capabilities []Capability `yaml:"capabilities"`
}

func LoadCapabilities() ([]Capability, error) {
	data, err := capabilitiesFS.ReadFile("capabilities.yaml")
	if err != nil {
		return nil, fmt.Errorf("read capabilities: %w", err)
	}

	var registry CapabilityRegistry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("parse capabilities: %w", err)
	}

	return registry.Capabilities, nil
}

func GetCapability(name string) (*Capability, error) {
	caps, err := LoadCapabilities()
	if err != nil {
		return nil, err
	}

	for _, c := range caps {
		if c.Name == name {
			return &c, nil
		}
	}

	return nil, fmt.Errorf("capability not found: %s", name)
}
```

**Step 4: Create embedded capabilities.yaml**

Create: `cmd/gokit/pkg/gokit/capabilities.yaml`

```yaml
version: "1.0.0"
updated_at: "2026-03-05"

capabilities:
  - name: kit
    description: 日志和基础工具
    import: github.com/tsopia/go-kit/kit
    scenarios:
      - name: 打印日志
        snippet: kit.Info(ctx, "message", "key", value)
      - name: 调试日志
        snippet: kit.Debug(ctx, "debug info")
    dependencies: []

  - name: database
    description: 数据库连接管理（GORM 封装）
    import: github.com/tsopia/go-kit/database
    scenarios:
      - name: 初始化数据库
        snippet: |
          db, err := database.New(cfg)
          if err != nil {
              return fmt.Errorf("init db: %w", err)
          }
      - name: 获取连接
        snippet: db := database.GetClient()
    dependencies: [kit]

  - name: cfg
    description: 配置管理
    import: github.com/tsopia/go-kit/cfg
    scenarios:
      - name: 加载配置
        snippet: err := cfg.Load("config.yaml", &config)
    dependencies: []
```

**Step 5: Run test to verify it passes**

Run: `cd cmd/gokit && go test ./pkg/gokit/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add cmd/gokit/pkg/gokit/
git commit -m "feat(cli): add capability types and loader"
```

---

## Task 3: Create Project Template Types

**Files:**
- Create: `cmd/gokit/pkg/template/template.go`
- Create: `cmd/gokit/pkg/template/template_test.go`

**Step 1: Write the failing test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd cmd/gokit && go test ./pkg/template/... -v`
Expected: FAIL - "undefined: Render"

**Step 3: Create template engine**

```go
package template

import (
	"bytes"
	"fmt"
	"text/template"
)

type Data struct {
	Name         string
	Module       string
	GoKitModule  string
	GoKitVersion string
	Features     []string
}

func Render(name, tmpl string, data Data) (string, error) {
	t := template.New(name)
	t, err := t.Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd cmd/gokit && go test ./pkg/template/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/gokit/pkg/template/
git commit -m "feat(cli): add template rendering engine"
```

---

## Task 4: Create Scaffold Package

**Files:**
- Create: `cmd/gokit/pkg/scaffold/project.go`
- Create: `cmd/gokit/pkg/scaffold/project_test.go`

**Step 1: Write the failing test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd cmd/gokit && go test ./pkg/scaffold/... -v`
Expected: FAIL - "undefined: CreateProject"

**Step 3: Create scaffold package**

```go
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tsopia/go-kit/cmd/gokit/pkg/template"
)

type Config struct {
	Name         string
	Module       string
	GoKitModule  string
	GoKitVersion string
	Template     string
	OutputDir    string
}

func CreateProject(cfg Config) error {
	// Create directory structure
	dirs := []string{
		filepath.Join(cfg.OutputDir, "cmd", cfg.Name),
		filepath.Join(cfg.OutputDir, "internal", "handler"),
		filepath.Join(cfg.OutputDir, "internal", "service"),
		filepath.Join(cfg.OutputDir, "internal", "config"),
		filepath.Join(cfg.OutputDir, "configs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	// Generate files
	files := map[string]string{
		"cmd/main.go":        mainGoTemplate,
		"go.mod":            goModTemplate,
		"configs/config.yaml": configYamlTemplate,
		"AGENTS.md":         agentsMdTemplate,
	}

	data := template.Data{
		Name:         cfg.Name,
		Module:       cfg.Module,
		GoKitModule:  cfg.GoKitModule,
		GoKitVersion: cfg.GoKitVersion,
		Features:     []string{},
	}

	for path, tmpl := range files {
		fullPath := filepath.Join(cfg.OutputDir, path)
		content, err := template.Render(filepath.Base(path), tmpl, data)
		if err != nil {
			return fmt.Errorf("render %s: %w", path, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	return nil
}

const mainGoTemplate = `package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"{{.GoKitModule}}/kit"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	kit.Info(ctx, "Starting {{.Name}} service")

	// TODO: Initialize your service here

	// Wait for shutdown signal
	<-sigChan
	kit.Info(ctx, "Shutting down {{.Name}} service")
}
`

const goModTemplate = `module {{.Module}}

go 1.24.0

require {{.GoKitModule}} {{.GoKitVersion}}
`

const configYamlTemplate = `# {{.Name}} configuration
app:
  name: {{.Name}}
  env: development

# Add your configuration here
`

const agentsMdTemplate = `# {{.Name}} AI 开发指南

本项目使用 [go-kit]({{.GoKitModule}}) 提供基础能力。

## 常用能力

### 日志
使用 go-kit 的日志工具：
` + "```go\nkit.Info(ctx, \"message\", \"key\", value)\n```" + `

## 参考

- [go-kit AI 指南]({{.GoKitModule}}/blob/main/.ai/GUIDE.md)
`
```

**Step 4: Run test to verify it passes**

Run: `cd cmd/gokit && go test ./pkg/scaffold/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/gokit/pkg/scaffold/
git commit -m "feat(cli): add project scaffold package"
```

---

## Task 5: Implement 'new' Command

**Files:**
- Create: `cmd/gokit/cmd/new.go`

**Step 1: Create new command**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tsopia/go-kit/cmd/gokit/pkg/scaffold"
)

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new go-kit project",
	Long:  `Create a new Go project with go-kit integration and AI-friendly configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		templateType, _ := cmd.Flags().GetString("template")
		module, _ := cmd.Flags().GetString("module")
		output, _ := cmd.Flags().GetString("output")

		// Default module name if not specified
		if module == "" {
			module = fmt.Sprintf("github.com/yourcompany/%s", name)
		}

		// Default output if not specified
		if output == "" {
			output = name
		}

		cfg := scaffold.Config{
			Name:         name,
			Module:       module,
			GoKitModule:  "github.com/tsopia/go-kit",
			GoKitVersion: "v0.0.0",
			Template:     templateType,
			OutputDir:    output,
		}

		fmt.Printf("Creating new %s project: %s\n", templateType, name)
		fmt.Printf("Module: %s\n", module)
		fmt.Printf("Output: %s\n", output)

		if err := scaffold.CreateProject(cfg); err != nil {
			return fmt.Errorf("create project: %w", err)
		}

		fmt.Printf("✓ Project created successfully!\n")
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  cd %s\n", output)
		fmt.Printf("  go mod tidy\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringP("template", "t", "api", "Project template (api|worker|cron|library)")
	newCmd.Flags().StringP("module", "m", "", "Go module name (default: github.com/yourcompany/<name>)")
	newCmd.Flags().StringP("output", "o", "", "Output directory (default: ./<name>)")
}
```

**Step 2: Update main.go to import cmd package**

Modify: `cmd/gokit/main.go`

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	_ "github.com/tsopia/go-kit/cmd/gokit/cmd" // Import for side effects
)

var rootCmd = &cobra.Command{
	Use:   "gokit",
	Short: "Go-kit project scaffolding tool",
	Long: `gokit is a CLI tool for creating and managing Go projects
that use the go-kit library. It provides project templates,
AI-friendly configuration, and go-kit integration.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 3: Refactor to use correct pattern**

Better approach - create cmd/root.go:

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gokit",
	Short: "Go-kit project scaffolding tool",
	Long: `gokit is a CLI tool for creating and managing Go projects
that use the go-kit library. It provides project templates,
AI-friendly configuration, and go-kit integration.`,
}

func Execute() error {
	return rootCmd.Execute()
}
```

Then update main.go:

```go
package main

import (
	"fmt"
	"os"

	"github.com/tsopia/go-kit/cmd/gokit/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 4: Test new command**

Run: `cd cmd/gokit && go run main.go new test-api`
Expected: Creates directory `cmd/gokit/test-api/` with project structure

**Step 5: Commit**

```bash
git add cmd/gokit/
git commit -m "feat(cli): implement 'new' command for creating projects"
```

---

## Task 6: Implement 'list' Command

**Files:**
- Create: `cmd/gokit/cmd/list.go`

**Step 1: Create list command**

```go
package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tsopia/go-kit/cmd/gokit/pkg/gokit"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available go-kit capabilities",
	Long:  `List all capabilities provided by go-kit with their descriptions and usage examples.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		caps, err := gokit.LoadCapabilities()
		if err != nil {
			return fmt.Errorf("load capabilities: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tIMPORT")

		for _, c := range caps {
			fmt.Fprintf(w, "%s\t%s\t%s\n", c.Name, c.Description, c.Import)
		}

		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
```

**Step 2: Test list command**

Run: `cd cmd/gokit && go run main.go list`
Expected: Table showing kit, database, cfg capabilities

**Step 3: Commit**

```bash
git add cmd/gokit/cmd/list.go
git commit -m "feat(cli): implement 'list' command"
```

---

## Task 7: Build and Test Complete CLI

**Step 1: Build binary**

Run: `cd cmd/gokit && go build -o gokit main.go`
Expected: Binary created at `cmd/gokit/gokit`

**Step 2: Test full workflow**

```bash
cd cmd/gokit
./gokit --help
./gokit list
./gokit new demo-api --module=github.com/example/demo-api
```

Expected: All commands work, demo-api project created with proper structure

**Step 3: Clean up test project**

Run: `rm -rf cmd/gokit/demo-api cmd/gokit/gokit`

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat(cli): complete basic gokit CLI implementation"
```

---

## Post-Implementation

### Remaining Tasks (Future PRs)

1. **'init' command** - Add go-kit to existing projects
2. **'add' command** - Add specific capabilities to projects
3. **'update' command** - Update AI guides and templates
4. **More templates** - worker, cron, library templates
5. **Template embedding** - Use go:embed for all templates
6. **Dependency resolution** - Auto-add dependent capabilities

### Testing Checklist

- [ ] Unit tests for all packages
- [ ] Integration test creating full project
- [ ] Test project builds successfully
- [ ] Test project can import go-kit packages

### Documentation

- [ ] Add README.md to cmd/gokit/
- [ ] Document each command
- [ ] Add usage examples
