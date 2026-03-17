package s3

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/tsopia/go-kit/storage/providers"
)

func TestAuthorizeDirectUploadPut(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)

	auth, err := client.AuthorizeDirectUpload(context.Background(), providers.DirectUploadRequest{
		ObjectKey:   "uploads/a.png",
		Mode:        providers.DirectUploadModePut,
		ContentType: "image/png",
		Metadata: map[string]string{
			"owner": "u1",
		},
		Checksum: &providers.DirectUploadChecksum{
			Algorithm: providers.DirectUploadChecksumMD5,
			Value:     "1B2M2Y8AsgTpgAmY7PhCfg==",
		},
	})
	if err != nil {
		t.Fatalf("AuthorizeDirectUpload() error = %v", err)
	}
	if auth.Mode != providers.DirectUploadModePut {
		t.Fatalf("unexpected mode: %s", auth.Mode)
	}
	if auth.Method != http.MethodPut {
		t.Fatalf("unexpected method: %s", auth.Method)
	}
	if auth.URL == "" {
		t.Fatal("expected signed url")
	}
	if got := getHeader(auth.Headers, "Content-Type"); got != "image/png" {
		t.Fatalf("unexpected content type header: %q", got)
	}
	if got := getHeader(auth.Headers, "Content-Md5"); got != "1B2M2Y8AsgTpgAmY7PhCfg==" {
		t.Fatalf("unexpected content md5 header: %q", got)
	}
	if got := getHeader(auth.Headers, "X-Amz-Meta-Owner"); got != "u1" {
		t.Fatalf("unexpected metadata header: %q", got)
	}
}

func TestAuthorizeDirectUploadAutoUsesPostForSizeRange(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)

	auth, err := client.AuthorizeDirectUpload(context.Background(), providers.DirectUploadRequest{
		ObjectKey:   "uploads/a.png",
		ContentType: "image/png",
		Metadata: map[string]string{
			"owner": "u1",
		},
		Size: &providers.DirectUploadSize{
			Min: 1,
			Max: 1024,
		},
	})
	if err != nil {
		t.Fatalf("AuthorizeDirectUpload() error = %v", err)
	}
	if auth.Mode != providers.DirectUploadModePost {
		t.Fatalf("unexpected mode: %s", auth.Mode)
	}
	if auth.Method != http.MethodPost {
		t.Fatalf("unexpected method: %s", auth.Method)
	}
	if auth.URL == "" {
		t.Fatal("expected post url")
	}
	if auth.FormFields["Content-Type"] != "image/png" {
		t.Fatalf("unexpected content type field: %q", auth.FormFields["Content-Type"])
	}
	if auth.FormFields["x-amz-meta-owner"] != "u1" {
		t.Fatalf("unexpected metadata field: %q", auth.FormFields["x-amz-meta-owner"])
	}
	if auth.FormFields["policy"] == "" {
		t.Fatal("expected post policy")
	}
}

func TestAuthorizeDirectUploadRejectsUnsupportedRangeForPut(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)

	_, err := client.AuthorizeDirectUpload(context.Background(), providers.DirectUploadRequest{
		ObjectKey: "uploads/a.png",
		Mode:      providers.DirectUploadModePut,
		Size: &providers.DirectUploadSize{
			Min: 1,
			Max: 1024,
		},
	})
	if !errors.Is(err, providers.ErrUnsupportedDirectUploadConstraint) {
		t.Fatalf("expected unsupported constraint error, got %v", err)
	}
}

func newTestClient(t *testing.T) *client {
	t.Helper()

	rawClient, err := NewClient(&providers.Config{
		Type:            providers.TypeS3,
		Bucket:          "bucket",
		Region:          "us-east-1",
		AccessKeyID:     "key",
		AccessKeySecret: "secret",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	client, ok := rawClient.(*client)
	if !ok {
		t.Fatalf("unexpected client type %T", rawClient)
	}

	return client
}

func getHeader(headers map[string]string, key string) string {
	for headerKey, value := range headers {
		if strings.EqualFold(headerKey, key) {
			return value
		}
	}
	return ""
}
