package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInferDriver(t *testing.T) {
	tests := []struct {
		dsn    string
		expect string
	}{
		{"root:pass@tcp(127.0.0.1:3306)/demo", "mysql"},
		{"postgres://user:pass@localhost:5432/demo", "postgres"},
		{"postgresql://user:pass@localhost:5432/demo", "postgres"},
		{"unknown://something", ""},
	}

	for _, tt := range tests {
		t.Run(tt.dsn, func(t *testing.T) {
			got := inferDriver(tt.dsn)
			if got != tt.expect {
				t.Errorf("inferDriver(%q) = %q, want %q", tt.dsn, got, tt.expect)
			}
		})
	}
}

func TestLoadFromDSN_MySQL(t *testing.T) {
	cfg, err := loadFromDSN("root:P@ss:123@tcp(127.0.0.1:3306)/demo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Driver != "mysql" {
		t.Errorf("driver = %q, want mysql", cfg.Driver)
	}
	if cfg.Username != "root" {
		t.Errorf("username = %q, want root", cfg.Username)
	}
	if cfg.Password != "P@ss:123" {
		t.Errorf("password = %q, want P@ss:123", cfg.Password)
	}
	if cfg.Database != "demo" {
		t.Errorf("database = %q, want demo", cfg.Database)
	}
}

func TestLoadFromDSN_Postgres(t *testing.T) {
	cfg, err := loadFromDSN("postgres://admin:secret@db.example.com:5432/mydb?sslmode=require", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Driver != "postgres" {
		t.Errorf("driver = %q, want postgres", cfg.Driver)
	}
	if cfg.Username != "admin" {
		t.Errorf("username = %q, want admin", cfg.Username)
	}
	if cfg.Password != "secret" {
		t.Errorf("password = %q, want secret", cfg.Password)
	}
	if cfg.Database != "mydb" {
		t.Errorf("database = %q, want mydb", cfg.Database)
	}
	if cfg.SSLMode != "require" {
		t.Errorf("sslmode = %q, want require", cfg.SSLMode)
	}
}

func TestLoadFromDSN_DriverMismatch(t *testing.T) {
	_, err := loadFromDSN("root:pass@tcp(127.0.0.1:3306)/demo", "postgres")
	if err == nil {
		t.Fatal("expected error for driver mismatch")
	}
}

func TestLoadFromDSN_UnknownFormat(t *testing.T) {
	_, err := loadFromDSN("something-random", "")
	if err == nil {
		t.Fatal("expected error for unknown DSN format")
	}
}

func TestDiscoverFromSource_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := DiscoverFromSource(tmpDir)
	if err == nil {
		t.Fatal("expected error when no source files found")
	}
}

func TestDiscoverFromSource_SingleConfig(t *testing.T) {
	tmpDir := t.TempDir()
	mainGo := `package main
import "github.com/tsopia/go-kit/database"
func main() {
	db, _ := database.New(&database.Config{
		Driver:   "mysql",
		Host:     "127.0.0.1",
		Port:     3306,
		Username: "root",
		Password: "pass",
		Database: "demo",
	})
	_ = db
}`
	err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := DiscoverFromSource(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Driver != "mysql" {
		t.Errorf("driver = %q, want mysql", cfg.Driver)
	}
	if cfg.Database != "demo" {
		t.Errorf("database = %q, want demo", cfg.Database)
	}
}

func TestDiscoverFromSource_MultipleConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "cmd", "server"), 0755)

	mainGo := `package main
import "github.com/tsopia/go-kit/database"
func main() {
	db, _ := database.New(&database.Config{Driver: "mysql", Host: "localhost", Port: 3306, Username: "root", Password: "a", Database: "db1"})
	_ = db
}`
	cmdGo := `package main
import "github.com/tsopia/go-kit/database"
func main() {
	db, _ := database.New(&database.Config{Driver: "postgres", Host: "localhost", Port: 5432, Username: "admin", Password: "b", Database: "db2"})
	_ = db
}`
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	os.WriteFile(filepath.Join(tmpDir, "cmd", "server", "main.go"), []byte(cmdGo), 0644)

	_, err := DiscoverFromSource(tmpDir)
	if err == nil {
		t.Fatal("expected error for multiple configs")
	}
}
