package internal

import (
	"fmt"
	"sync"
	"testing"
)

func TestMultipartStateLifecycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		uploadID string
		key      string
	}{
		{
			name:     "stores and loads key",
			uploadID: "upload-1",
			key:      "objects/a.txt",
		},
		{
			name:     "stores independent upload id",
			uploadID: "upload-2",
			key:      "objects/b.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewMultipartState()

			state.Store(tt.uploadID, tt.key)

			got, ok := state.Load(tt.uploadID)
			if !ok {
				t.Fatal("expected uploadID to exist")
			}
			if got != tt.key {
				t.Fatalf("unexpected key: got %q want %q", got, tt.key)
			}

			state.Delete(tt.uploadID)

			if _, ok := state.Load(tt.uploadID); ok {
				t.Fatal("expected uploadID to be deleted")
			}
		})
	}
}

func TestMultipartStateConcurrentAccess(t *testing.T) {
	t.Parallel()

	state := NewMultipartState()

	const workers = 32
	errCh := make(chan error, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			uploadID := fmt.Sprintf("upload-%d", i)
			key := fmt.Sprintf("objects/%d.txt", i)

			state.Store(uploadID, key)

			got, ok := state.Load(uploadID)
			if !ok {
				errCh <- fmt.Errorf("uploadID %q missing", uploadID)
				return
			}
			if got != key {
				errCh <- fmt.Errorf("unexpected key for %q: got %q want %q", uploadID, got, key)
				return
			}

			state.Delete(uploadID)

			if _, ok := state.Load(uploadID); ok {
				errCh <- fmt.Errorf("uploadID %q still exists after delete", uploadID)
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}
