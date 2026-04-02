package dbmigrate

import (
	"database/sql"
	"strings"
	"testing"
)

func TestCreateMigrate_NilDB(t *testing.T) {
	_, err := createMigrate(Config{
		SourcePath: "migrations",
		DB:         nil,
		DriverName: "mysql",
	})
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
}

func TestCreateMigrate_EmptySourcePath(t *testing.T) {
	db, _ := sql.Open("mysql", "")
	defer db.Close()

	_, err := createMigrate(Config{
		SourcePath: "",
		DB:         db,
		DriverName: "mysql",
	})
	if err == nil {
		t.Fatal("expected error for empty source path")
	}
}

func TestOpenDriver_Unsupported(t *testing.T) {
	db, _ := sql.Open("mysql", "")
	defer db.Close()

	_, err := openDriver(db, "oracle")
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestOpenDriver_MySQL(t *testing.T) {
	db, _ := sql.Open("mysql", "root:invalid@tcp(localhost:3306)/test")
	defer db.Close()

	d, err := openDriver(db, "mysql")
	// WithInstance pings the database, so connection errors are expected
	// without a running MySQL server. The key assertion is that we did NOT
	// get an "unsupported driver" error.
	if err != nil && strings.Contains(err.Error(), "unsupported driver") {
		t.Fatalf("should not return unsupported-driver error: %v", err)
	}
	if err == nil && d == nil {
		t.Fatal("expected non-nil driver when error is nil")
	}
}

func TestOpenDriver_Postgres(t *testing.T) {
	db, _ := sql.Open("postgres", "host=localhost port=5432 user=test dbname=test sslmode=disable")
	defer db.Close()

	d, err := openDriver(db, "postgres")
	// WithInstance pings the database, so connection errors are expected
	// without a running Postgres server. The key assertion is that we did NOT
	// get an "unsupported driver" error.
	if err != nil && strings.Contains(err.Error(), "unsupported driver") {
		t.Fatalf("should not return unsupported-driver error: %v", err)
	}
	if err == nil && d == nil {
		t.Fatal("expected non-nil driver when error is nil")
	}
}
