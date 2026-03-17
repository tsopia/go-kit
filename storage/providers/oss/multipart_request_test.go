package oss

import (
	"bytes"
	"testing"

	"github.com/tsopia/go-kit/storage/providers"
	providerinternal "github.com/tsopia/go-kit/storage/providers/internal"
)

func TestMultipartRequestBuildersUseMappedKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		build func(*client) (string, string, error)
	}{
		{
			name: "upload part uses stored key",
			build: func(c *client) (string, string, error) {
				req, err := c.buildUploadPartRequest("upload-1", 3, bytes.NewReader([]byte("part")))
				if err != nil {
					return "", "", err
				}
				return stringValue(req.Key), stringValue(req.UploadId), nil
			},
		},
		{
			name: "complete multipart uses stored key",
			build: func(c *client) (string, string, error) {
				req, err := c.buildCompleteMultipartRequest("upload-1", []*providers.PartInfo{
					{PartNumber: 1, ETag: "etag-1"},
				})
				if err != nil {
					return "", "", err
				}
				return stringValue(req.Key), stringValue(req.UploadId), nil
			},
		},
		{
			name: "abort multipart uses stored key",
			build: func(c *client) (string, string, error) {
				req, err := c.buildAbortMultipartRequest("upload-1")
				if err != nil {
					return "", "", err
				}
				return stringValue(req.Key), stringValue(req.UploadId), nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{
				bucket:         "bucket",
				multipartState: providerinternal.NewMultipartState(),
			}
			c.multipartState.Store("upload-1", "objects/a.txt")

			gotKey, gotUploadID, err := tt.build(c)
			if err != nil {
				t.Fatalf("build request error = %v", err)
			}
			if gotKey != "objects/a.txt" {
				t.Fatalf("unexpected key: got %q want %q", gotKey, "objects/a.txt")
			}
			if gotUploadID != "upload-1" {
				t.Fatalf("unexpected upload id: got %q want %q", gotUploadID, "upload-1")
			}
		})
	}
}

func TestMultipartRequestBuildersRejectUnknownUploadID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		build func(*client) error
	}{
		{
			name: "upload part rejects unknown upload id",
			build: func(c *client) error {
				_, err := c.buildUploadPartRequest("missing", 1, bytes.NewReader([]byte("part")))
				return err
			},
		},
		{
			name: "complete multipart rejects unknown upload id",
			build: func(c *client) error {
				_, err := c.buildCompleteMultipartRequest("missing", nil)
				return err
			},
		},
		{
			name: "abort multipart rejects unknown upload id",
			build: func(c *client) error {
				_, err := c.buildAbortMultipartRequest("missing")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{
				bucket:         "bucket",
				multipartState: providerinternal.NewMultipartState(),
			}

			if err := tt.build(c); err == nil {
				t.Fatal("expected error for unknown upload id")
			}
		})
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
