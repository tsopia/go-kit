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
		filepath.Join("cmd", cfg.Name, "main.go"): mainGoTemplate,
		"go.mod":                                  goModTemplate,
		"configs/config.yaml":                     configYamlTemplate,
		"AGENTS.md":                               agentsMdTemplate,
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
