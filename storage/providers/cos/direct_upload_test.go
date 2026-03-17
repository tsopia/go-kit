package cos

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
	if got := getHeader(auth.Headers, "X-Cos-Meta-Owner"); got != "u1" {
		t.Fatalf("unexpected metadata header: %q", got)
	}
}

func TestAuthorizeDirectUploadRejectsUnsupportedRange(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)

	_, err := client.AuthorizeDirectUpload(context.Background(), providers.DirectUploadRequest{
		ObjectKey: "uploads/a.png",
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
		Type:            providers.TypeCOS,
		Bucket:          "bucket",
		Region:          "ap-shanghai",
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
