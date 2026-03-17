package s3

import (
	"testing"
	"time"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestBuildObjectInfoIncludesMetadataAndChecksums(t *testing.T) {
	t.Parallel()

	lastModified := time.Unix(1700000000, 0).UTC()
	contentLength := int64(12)
	info := buildObjectInfo("uploads/a.png", &awss3.HeadObjectOutput{
		ContentLength:     &contentLength,
		LastModified:      &lastModified,
		ETag:              stringPtr("\"etag\""),
		ContentType:       stringPtr("image/png"),
		Metadata:          map[string]string{"owner": "u1"},
		ChecksumSHA256:    stringPtr("sha256-value"),
		ChecksumCRC32:     stringPtr("crc32-value"),
		ChecksumCRC32C:    stringPtr("crc32c-value"),
		ChecksumSHA1:      stringPtr("sha1-value"),
		ChecksumCRC64NVME: stringPtr("crc64-value"),
	})

	if got := info.Metadata["owner"]; got != "u1" {
		t.Fatalf("unexpected metadata value: %q", got)
	}
	if got := info.Checksums["sha256"]; got != "sha256-value" {
		t.Fatalf("unexpected sha256 checksum: %q", got)
	}
	if got := info.Checksums["crc32"]; got != "crc32-value" {
		t.Fatalf("unexpected crc32 checksum: %q", got)
	}
}

func stringPtr(value string) *string {
	return &value
}
