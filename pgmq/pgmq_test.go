package pgmq

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type testPayload struct {
	ID string `json:"id"`
}

func newSQLiteDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestNewQueueMissingDB(t *testing.T) {
	_, err := NewQueue[testPayload](context.Background(), nil, "orders")
	if err != ErrMissingDB {
		t.Fatalf("expected ErrMissingDB, got %v", err)
	}
}

func TestNewQueueInvalidName(t *testing.T) {
	db := newSQLiteDB(t)
	defer db.Close()

	adapter, err := NewSQLDBAdapter(db)
	if err != nil {
		t.Fatalf("adapter error: %v", err)
	}

	_, err = NewQueue[testPayload](context.Background(), adapter, "invalid-name")
	if err == nil {
		t.Fatal("expected error for invalid queue name")
	}
}

func TestNewQueueExtensionMissing(t *testing.T) {
	db := newSQLiteDB(t)
	defer db.Close()

	_, err := db.Exec("CREATE TABLE pg_extension (extname TEXT)")
	if err != nil {
		t.Fatalf("create pg_extension: %v", err)
	}

	adapter, err := NewSQLDBAdapter(db)
	if err != nil {
		t.Fatalf("adapter error: %v", err)
	}

	_, err = NewQueue[testPayload](context.Background(), adapter, "orders",
		WithCheckExtension(true),
		WithEnsureQueue(false),
	)
	if err != ErrExtensionMissing {
		t.Fatalf("expected ErrExtensionMissing, got %v", err)
	}
}

func TestReadOptionsDefaults(t *testing.T) {
	cfg := QueueConfig{ReadQuantity: 1, Visibility: DefaultVisibility}
	cfg.SetDefaults()

	opts, err := normalizeReadOptions(cfg, ReadOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Quantity != cfg.ReadQuantity {
		t.Fatalf("expected quantity %d, got %d", cfg.ReadQuantity, opts.Quantity)
	}
}

func TestSendNegativeDelay(t *testing.T) {
	db := newSQLiteDB(t)
	defer db.Close()

	adapter, err := NewSQLDBAdapter(db)
	if err != nil {
		t.Fatalf("adapter error: %v", err)
	}

	queue, err := NewQueue[testPayload](context.Background(), adapter, "orders",
		WithCheckExtension(false),
		WithEnsureQueue(false),
	)
	if err != nil {
		t.Fatalf("new queue error: %v", err)
	}

	_, err = queue.Send(context.Background(), testPayload{ID: "1"}, -1)
	if err != ErrInvalidDelay {
		t.Fatalf("expected ErrInvalidDelay, got %v", err)
	}
}
