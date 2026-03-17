package internal

import (
	"errors"
	"testing"

	"github.com/tsopia/go-kit/storage/providers"
)

func TestExistsFromError(t *testing.T) {
	t.Parallel()

	otherErr := errors.New("boom")

	tests := []struct {
		name       string
		err        error
		wantExists bool
		wantErr    error
	}{
		{
			name:       "nil error means exists",
			wantExists: true,
		},
		{
			name:       "object not found means missing object",
			err:        providers.ErrObjectNotFound,
			wantExists: false,
		},
		{
			name:    "bucket not found is returned",
			err:     providers.ErrBucketNotFound,
			wantErr: providers.ErrBucketNotFound,
		},
		{
			name:    "access denied is returned",
			err:     providers.ErrAccessDenied,
			wantErr: providers.ErrAccessDenied,
		},
		{
			name:    "other error is returned",
			err:     otherErr,
			wantErr: otherErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExists, gotErr := ExistsFromError(tt.err)
			if gotExists != tt.wantExists {
				t.Fatalf("unexpected exists result: got %v want %v", gotExists, tt.wantExists)
			}
			if !errors.Is(gotErr, tt.wantErr) {
				t.Fatalf("unexpected error: got %v want %v", gotErr, tt.wantErr)
			}
		})
	}
}
