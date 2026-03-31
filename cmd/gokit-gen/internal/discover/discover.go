// Package discover provides configuration auto-discovery from project files.
package discover

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Config holds database connection configuration.
type Config struct {
	Driver   string
	Host     string
	Port     int
	Database string
	Username string
	Password string
	DSN      string // Raw DSN if provided
}

// FromProject attempts to discover database configuration from project files.
// It looks for:
// 1. Go code with database.Config initialization
// 2. .env files with DATABASE_URL
// 3. docker-compose.yml with database service
func FromProject() (*Config, error) {
	// Try to find config from Go source files
	cfg, err := fromGoSource()
	if err == nil {
		return cfg, nil
	}

	// Try .env file
	cfg, err = fromEnvFile()
	if err == nil {
		return cfg, nil
	}

	// Try docker-compose
	cfg, err = fromDockerCompose()
	if err == nil {
		return cfg, nil
	}

	return nil, fmt.Errorf("could not discover database configuration from project files")
}

func fromGoSource() (*Config, error) {
	// Look for main.go or cmd/*/main.go
	patterns := []string{
		"main.go",
		"cmd/*/main.go",
		"cmd/server/main.go",
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, file := range matches {
			cfg, err := parseGoFile(file)
			if err == nil {
				return cfg, nil
			}
		}
	}

	return nil, fmt.Errorf("no database config found in Go source")
}

func parseGoFile(filename string) (*Config, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Look for database.Config composite literal
	var cfg Config
	found := false

	ast.Inspect(node, func(n ast.Node) bool {
		composit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Check if it's database.Config or &database.Config
		switch t := composit.Type.(type) {
		case *ast.SelectorExpr:
			if isDatabaseConfigSelector(t) {
				found = true
				extractConfigFields(composit, &cfg)
			}
		case *ast.StarExpr:
			if sel, ok := t.X.(*ast.SelectorExpr); ok && isDatabaseConfigSelector(sel) {
				found = true
				extractConfigFields(composit, &cfg)
			}
		}

		return true
	})

	if !found {
		return nil, fmt.Errorf("no database.Config found")
	}

	return &cfg, nil
}

func isDatabaseConfigSelector(sel *ast.SelectorExpr) bool {
	if sel.Sel.Name != "Config" {
		return false
	}
	// Check if the package name is "database"
	if ident, ok := sel.X.(*ast.Ident); ok {
		return ident.Name == "database"
	}
	return false
}

func extractConfigFields(composit *ast.CompositeLit, cfg *Config) {
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
			cfg.Driver = extractStringValue(kv.Value)
		case "Host":
			cfg.Host = extractStringValue(kv.Value)
		case "Port":
			cfg.Port = extractIntValue(kv.Value)
		case "Database":
			cfg.Database = extractStringValue(kv.Value)
		case "Username":
			cfg.Username = extractStringValue(kv.Value)
		case "Password":
			cfg.Password = extractStringValue(kv.Value)
		}
	}
}

func extractStringValue(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		return strings.Trim(v.Value, `"`)
	case *ast.Ident:
		return v.Name
	}
	return ""
}

func extractIntValue(expr ast.Expr) int {
	switch v := expr.(type) {
	case *ast.BasicLit:
		var val int
		fmt.Sscanf(v.Value, "%d", &val)
		return val
	}
	return 0
}

func fromEnvFile() (*Config, error) {
	files := []string{".env", ".env.local", ".env.development"}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		content := string(data)
		lines := strings.Split(content, "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "DATABASE_URL=") || strings.HasPrefix(line, "DSN=") {
				dsn := strings.TrimPrefix(line, "DATABASE_URL=")
				dsn = strings.TrimPrefix(dsn, "DSN=")
				dsn = strings.Trim(dsn, `"'`)
				return ParseDSN(dsn)
			}
		}
	}

	return nil, fmt.Errorf("no DATABASE_URL found in .env files")
}

func fromDockerCompose() (*Config, error) {
	files := []string{"docker-compose.yml", "docker-compose.yaml"}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		content := string(data)
		// Simple string matching for mysql/postgres service
		if strings.Contains(content, "mysql") || strings.Contains(content, "postgres") {
			// Extract connection info from environment variables in compose file
			// This is a simplified version - in production, use proper YAML parsing
			return nil, fmt.Errorf("docker-compose parsing not fully implemented")
		}
	}

	return nil, fmt.Errorf("no database service found in docker-compose")
}

func ParseDSN(dsn string) (*Config, error) {
	// Try to parse MySQL DSN: user:pass@tcp(host:port)/dbname
	if strings.Contains(dsn, "tcp(") {
		return parseMySQLDSN(dsn)
	}

	// Try to parse PostgreSQL DSN: postgres://user:pass@host:port/dbname
	if strings.HasPrefix(dsn, "postgres://") {
		return parsePostgresDSN(dsn)
	}

	return nil, fmt.Errorf("unsupported DSN format")
}

func parseMySQLDSN(dsn string) (*Config, error) {
	cfg := &Config{Driver: "mysql", Port: 3306}

	// Simple parsing - in production, use proper DSN parser
	parts := strings.Split(dsn, "@tcp(")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid MySQL DSN")
	}

	// user:pass
	auth := strings.Split(parts[0], ":")
	if len(auth) >= 2 {
		cfg.Username = auth[0]
		cfg.Password = auth[1]
	}

	// host:port)/dbname
	addrParts := strings.Split(parts[1], ")/")
	if len(addrParts) >= 1 {
		hostPort := strings.Split(addrParts[0], ":")
		cfg.Host = hostPort[0]
		if len(hostPort) > 1 {
			fmt.Sscanf(hostPort[1], "%d", &cfg.Port)
		}
	}

	if len(addrParts) > 1 {
		// Handle query parameters: dbname?parseTime=true
		dbName := addrParts[1]
		if idx := strings.Index(dbName, "?"); idx != -1 {
			dbName = dbName[:idx]
		}
		cfg.Database = dbName
	}

	return cfg, nil
}

func parsePostgresDSN(dsn string) (*Config, error) {
	cfg := &Config{Driver: "postgres", Port: 5432}

	// postgres://user:pass@host:port/dbname
	dsn = strings.TrimPrefix(dsn, "postgres://")

	parts := strings.Split(dsn, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid PostgreSQL DSN")
	}

	// user:pass
	auth := strings.Split(parts[0], ":")
	if len(auth) >= 2 {
		cfg.Username = auth[0]
		cfg.Password = auth[1]
	}

	// host:port/dbname
	addrParts := strings.Split(parts[1], "/")
	if len(addrParts) >= 1 {
		hostPort := strings.Split(addrParts[0], ":")
		cfg.Host = hostPort[0]
		if len(hostPort) > 1 {
			fmt.Sscanf(hostPort[1], "%d", &cfg.Port)
		}
	}

	if len(addrParts) > 1 {
		// Handle query parameters: dbname?sslmode=require
		dbName := addrParts[1]
		if idx := strings.Index(dbName, "?"); idx != -1 {
			dbName = dbName[:idx]
		}
		cfg.Database = dbName
	}

	return cfg, nil
}
