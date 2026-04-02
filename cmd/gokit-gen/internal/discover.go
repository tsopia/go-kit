package internal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tsopia/go-kit/database"
)

// DiscoverFromSource 从 Go 源码中发现 database.Config 字面量。
// 只提取 Driver/Host/Port/Username/Password/Database 六个字段。
// 只支持直接字面量赋值，不支持变量引用、函数调用或表达式拼接。
func DiscoverFromSource(workDir string) (*database.Config, error) {
	patterns := []string{
		filepath.Join(workDir, "main.go"),
		filepath.Join(workDir, "cmd", "*", "main.go"),
	}

	var configs []*database.Config

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, file := range matches {
			cfg, err := parseGoFile(file)
			if err != nil {
				continue
			}
			configs = append(configs, cfg)
		}
	}

	switch len(configs) {
	case 0:
		return nil, fmt.Errorf("未发现数据库配置")
	case 1:
		return configs[0], nil
	default:
		return nil, fmt.Errorf("发现多个候选配置，请显式指定 --dsn 或 --config")
	}
}

func parseGoFile(filename string) (*database.Config, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}

	var cfg database.Config
	found := false

	ast.Inspect(node, func(n ast.Node) bool {
		composit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		if isDatabaseConfig(composit.Type) {
			found = true
			extractFields(composit, &cfg)
		}

		return true
	})

	if !found {
		return nil, fmt.Errorf("no database.Config found")
	}

	return &cfg, nil
}

func isDatabaseConfig(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		if t.Sel.Name != "Config" {
			return false
		}
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == "database"
		}
	case *ast.StarExpr:
		return isDatabaseConfig(t.X)
	}
	return false
}

func extractFields(composit *ast.CompositeLit, cfg *database.Config) {
	for _, elt := range composit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "Driver":
			cfg.Driver = stringVal(kv.Value)
		case "Host":
			cfg.Host = stringVal(kv.Value)
		case "Port":
			cfg.Port = intVal(kv.Value)
		case "Username":
			cfg.Username = stringVal(kv.Value)
		case "Password":
			cfg.Password = stringVal(kv.Value)
		case "Database":
			cfg.Database = stringVal(kv.Value)
		}
	}
}

func stringVal(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		return strings.Trim(lit.Value, `"`)
	}
	return ""
}

func intVal(expr ast.Expr) int {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.INT {
		if v, err := strconv.Atoi(lit.Value); err == nil {
			return v
		}
	}
	return 0
}
