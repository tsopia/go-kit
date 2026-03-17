package oss

import (
	"testing"
	"time"

	osssdk "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

func TestBuildObjectInfoIncludesMetadataAndChecksums(t *testing.T) {
	t.Parallel()

	lastModified := time.Unix(1700000000, 0).UTC()
	info := buildObjectInfo("uploads/a.png", &osssdk.HeadObjectResult{
		ContentLength: 12,
		LastModified:  &lastModified,
		ETag:          stringPtr("\"etag\""),
		ContentType:   stringPtr("image/png"),
		ContentMD5:    stringPtr("md5-value"),
		Metadata:      map[string]string{"owner": "u1"},
	})

	if got := info.Metadata["owner"]; got != "u1" {
		t.Fatalf("unexpected metadata value: %q", got)
	}
	if got := info.Checksums["md5"]; got != "md5-value" {
		t.Fatalf("unexpected md5 checksum: %q", got)
	}
}

func stringPtr(value string) *string {
	return &value
}
