package cos

import (
	"net/http"
	"testing"

	cossdk "github.com/tencentyun/cos-go-sdk-v5"
)

func TestBuildObjectInfoIncludesMetadataAndChecksums(t *testing.T) {
	t.Parallel()

	resp := &cossdk.Response{
		Response: &http.Response{
			Header: http.Header{
				"Content-Type":         []string{"image/png"},
				"ETag":                 []string{"\"etag\""},
				"Last-Modified":        []string{"Mon, 02 Jan 2006 15:04:05 GMT"},
				"X-Cos-Meta-Owner":     []string{"u1"},
				"X-Cos-Hash-Crc64Ecma": []string{"crc64-value"},
			},
			ContentLength: 12,
		},
	}

	info := buildObjectInfo("uploads/a.png", resp)
	if got := info.Metadata["owner"]; got != "u1" {
		t.Fatalf("unexpected metadata value: %q", got)
	}
	if got := info.Checksums["crc64ecma"]; got != "crc64-value" {
		t.Fatalf("unexpected crc64 checksum: %q", got)
	}
}
