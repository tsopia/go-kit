package cos

import (
	"errors"
	"net/http"
	"testing"

	cosapi "github.com/tencentyun/cos-go-sdk-v5"
	"github.com/tsopia/go-kit/storage/providers"
)

func TestNormalizeStatError(t *testing.T) {
	t.Parallel()

	otherErr := errors.New("boom")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "object not found",
			err: &cosapi.ErrorResponse{
				Code:     "NoSuchKey",
				Response: &http.Response{StatusCode: http.StatusNotFound},
			},
			want: providers.ErrObjectNotFound,
		},
		{
			name: "bucket not found",
			err: &cosapi.ErrorResponse{
				Code:     "NoSuchBucket",
				Response: &http.Response{StatusCode: http.StatusNotFound},
			},
			want: providers.ErrBucketNotFound,
		},
		{
			name: "access denied",
			err: &cosapi.ErrorResponse{
				Code:     "AccessDenied",
				Response: &http.Response{StatusCode: http.StatusForbidden},
			},
			want: providers.ErrAccessDenied,
		},
		{
			name: "other error",
			err:  otherErr,
			want: otherErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeStatError(tt.err)
			if !errors.Is(got, tt.want) {
				t.Fatalf("unexpected error: got %v want %v", got, tt.want)
			}
		})
	}
}
