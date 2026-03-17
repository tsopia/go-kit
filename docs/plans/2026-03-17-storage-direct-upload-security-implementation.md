# Storage Direct Upload Security Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a safe client direct-upload authorization API to `storage` without breaking the existing `SignedURL` contract, including request validation, provider-aware `PUT/POST` authorization, and post-upload object verification.

**Architecture:** Keep the existing `SignedURL` path as a simple compatibility API and add a new `AuthorizeDirectUpload` entry point with explicit request/response types. Put validation and verification logic in the top-level `storage` package, extend provider interfaces only where necessary for authorization generation, and enrich `ObjectInfo` so verification can compare the stored object against the declared constraints.

**Tech Stack:** Go 1.24, existing `storage` package structure, AWS SDK v2 S3 presign client, Tencent COS SDK v5, Alibaba OSS Go SDK v2, table-driven tests, `go test`, `go run ./cmd/gokit list`

---

**References:**
- Read [docs/plans/2026-03-17-storage-direct-upload-security-design.md](/Users/kj/projects/go-kit/docs/plans/2026-03-17-storage-direct-upload-security-design.md) before touching code.
- Follow repository workflow in [AGENTS.md](/Users/kj/projects/go-kit/AGENTS.md): update capability metadata before implementing the new public API.
- Use `@superpowers:test-driven-development` for Tasks 2 through 6.
- Use `@superpowers:verification-before-completion` before claiming the feature is complete.

### Task 1: Update capability metadata and AI-facing docs first

**Files:**
- Modify: `.ai/capabilities.yaml`
- Modify: `cmd/gokit/pkg/gokit/capabilities.yaml`
- Modify: `cmd/gokit/pkg/gokit/capability_test.go`
- Modify: `AGENTS.md`

**Step 1: Update the canonical capability entry**

Extend the existing `storage` capability in [.ai/capabilities.yaml](/Users/kj/projects/go-kit/.ai/capabilities.yaml) with a new scenario that shows secure client direct upload:

```yaml
- name: storage
  description: 对象存储统一封装，支持上传、下载、分片和安全直传授权
  import: github.com/tsopia/go-kit/storage
  scenarios:
    - name: 申请客户端直传授权
      snippet: |
        auth, err := storage.AuthorizeDirectUpload(ctx, storage.DirectUploadRequest{
            ObjectKey:   objectKey,
            ContentType: "image/png",
            Checksum: &storage.DirectUploadChecksum{
                Algorithm: storage.DirectUploadChecksumMD5,
                Value:     checksum,
            },
        })
```

Update `updated_at` to `2026-03-17`.

**Step 2: Keep the embedded fallback in sync**

Add the same `storage` scenario to [cmd/gokit/pkg/gokit/capabilities.yaml](/Users/kj/projects/go-kit/cmd/gokit/pkg/gokit/capabilities.yaml) so `gokit list` stays aligned when the embedded fallback is used.

**Step 3: Add a focused capability test**

Extend [cmd/gokit/pkg/gokit/capability_test.go](/Users/kj/projects/go-kit/cmd/gokit/pkg/gokit/capability_test.go):

```go
func TestGetCapabilityStorage(t *testing.T) {
	t.Parallel()

	capability, err := GetCapability("storage")
	if err != nil {
		t.Fatalf("GetCapability(storage) failed: %v", err)
	}

	if capability.Import != "github.com/tsopia/go-kit/storage" {
		t.Fatalf("unexpected import: %s", capability.Import)
	}

	found := false
	for _, scenario := range capability.Scenarios {
		if scenario.Name == "申请客户端直传授权" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected direct upload scenario")
	}
}
```

**Step 4: Update the quick-reference table**

Update the `storage` row in [AGENTS.md](/Users/kj/projects/go-kit/AGENTS.md) to mention `storage.AuthorizeDirectUpload(...)` alongside the existing upload example.

**Step 5: Verify metadata changes**

Run: `GOCACHE=/tmp/go-build go test ./cmd/gokit/pkg/gokit -run 'Test(GetCapabilityStorage|GetCapabilitySwagger|LoadCapabilities)' -v`

Expected: PASS

Run: `GOCACHE=/tmp/go-build go run ./cmd/gokit list`

Expected: output contains a `storage` row and shows the new direct-upload scenario.

**Step 6: Commit**

```bash
git add .ai/capabilities.yaml cmd/gokit/pkg/gokit/capabilities.yaml cmd/gokit/pkg/gokit/capability_test.go AGENTS.md
git commit -m "docs(storage): register direct upload capability"
```

### Task 2: Define the public direct-upload types and wrapper contract

**Files:**
- Create: `storage/direct_upload.go`
- Create: `storage/direct_upload_test.go`
- Modify: `storage/types.go`
- Modify: `storage/storage.go`
- Modify: `storage/errors.go`
- Modify: `storage/providers/types.go`
- Modify: `storage/internal/client.go`

**Step 1: Write the failing top-level tests**

Add table-driven tests in [storage/direct_upload_test.go](/Users/kj/projects/go-kit/storage/direct_upload_test.go):

```go
func TestAuthorizeDirectUploadWithClientRequiresClient(t *testing.T) {
	t.Parallel()

	_, err := AuthorizeDirectUploadWithClient(context.Background(), nil, DirectUploadRequest{
		ObjectKey: "uploads/a.png",
	})
	if err != ErrMissingClient {
		t.Fatalf("expected ErrMissingClient, got %v", err)
	}
}

func TestVerifyDirectUploadObjectWithClientRequiresClient(t *testing.T) {
	t.Parallel()

	_, err := VerifyDirectUploadObjectWithClient(context.Background(), nil, DirectUploadVerificationRequest{
		ObjectKey: "uploads/a.png",
	})
	if err != ErrMissingClient {
		t.Fatalf("expected ErrMissingClient, got %v", err)
	}
}

func TestNormalizeDirectUploadRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     DirectUploadRequest
		wantErr bool
	}{
		{
			name: "default mode becomes auto",
			req:  DirectUploadRequest{ObjectKey: "uploads/a.png"},
		},
		{
			name: "exact size rejects range",
			req: DirectUploadRequest{
				ObjectKey: "uploads/a.png",
				Size: &DirectUploadSize{Exact: 10, Min: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeDirectUploadRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeDirectUploadRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./storage -run 'Test(AuthorizeDirectUploadWithClientRequiresClient|VerifyDirectUploadObjectWithClientRequiresClient|NormalizeDirectUploadRequest)' -v`

Expected: FAIL because the direct-upload types and functions do not exist yet.

**Step 3: Add the public types and wrapper surface**

Create [storage/direct_upload.go](/Users/kj/projects/go-kit/storage/direct_upload.go) and add the top-level API surface:

```go
type DirectUploadMode string

const (
	DirectUploadModeAuto DirectUploadMode = "auto"
	DirectUploadModePut  DirectUploadMode = "put"
	DirectUploadModePost DirectUploadMode = "post"
)

func AuthorizeDirectUpload(ctx context.Context, req DirectUploadRequest) (*DirectUploadAuthorization, error) {
	return AuthorizeDirectUploadWithClient(ctx, GetClient(), req)
}

func AuthorizeDirectUploadWithClient(ctx context.Context, c Client, req DirectUploadRequest) (*DirectUploadAuthorization, error) {
	if c == nil {
		return nil, ErrMissingClient
	}

	normalized, err := normalizeDirectUploadRequest(req)
	if err != nil {
		return nil, err
	}

	return c.AuthorizeDirectUpload(ctx, normalized)
}
```

Implementation rules:
- Put validation helpers in this file instead of bloating `storage.go`.
- Add new exported error values in [storage/errors.go](/Users/kj/projects/go-kit/storage/errors.go), for example `ErrInvalidDirectUploadRequest` and `ErrUnsupportedDirectUploadConstraint`.
- Re-export the new provider types from [storage/types.go](/Users/kj/projects/go-kit/storage/types.go) and [storage/internal/client.go](/Users/kj/projects/go-kit/storage/internal/client.go).
- Extend the provider `Client` interface in [storage/providers/types.go](/Users/kj/projects/go-kit/storage/providers/types.go) with `AuthorizeDirectUpload`.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./storage -run 'Test(AuthorizeDirectUploadWithClientRequiresClient|VerifyDirectUploadObjectWithClientRequiresClient|NormalizeDirectUploadRequest)' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add storage/direct_upload.go storage/direct_upload_test.go storage/types.go storage/storage.go storage/errors.go storage/providers/types.go storage/internal/client.go
git commit -m "feat(storage): add direct upload package surface"
```

### Task 3: Implement request normalization, upload verification, and metadata-aware object info

**Files:**
- Modify: `storage/direct_upload.go`
- Modify: `storage/direct_upload_test.go`
- Modify: `storage/providers/types.go`
- Modify: `storage/storage_test.go`

**Step 1: Extend the failing tests for verification behavior**

Add focused tests around verification mismatches:

```go
func TestVerifyDirectUploadObject(t *testing.T) {
	t.Parallel()

	client := &fakeDirectUploadClient{
		statResult: &ObjectInfo{
			Key:         "uploads/a.png",
			Size:        12,
			ContentType: "image/png",
			Metadata: map[string]string{
				"x-owner": "u1",
			},
		},
	}

	result, err := VerifyDirectUploadObjectWithClient(context.Background(), client, DirectUploadVerificationRequest{
		ObjectKey:   "uploads/a.png",
		ContentType: "image/png",
		Metadata:    map[string]string{"x-owner": "u1"},
		Size:        &DirectUploadSize{Exact: 12},
	})
	if err != nil {
		t.Fatalf("VerifyDirectUploadObjectWithClient() error = %v", err)
	}
	if !result.Exists || !result.Matched {
		t.Fatalf("expected matched result, got %+v", result)
	}
}

func TestVerifyDirectUploadObjectMismatch(t *testing.T) {
	t.Parallel()

	client := &fakeDirectUploadClient{
		statResult: &ObjectInfo{
			Key:         "uploads/a.png",
			Size:        10,
			ContentType: "image/jpeg",
		},
	}

	result, err := VerifyDirectUploadObjectWithClient(context.Background(), client, DirectUploadVerificationRequest{
		ObjectKey:   "uploads/a.png",
		ContentType: "image/png",
		Size:        &DirectUploadSize{Exact: 12},
	})
	if err != nil {
		t.Fatalf("VerifyDirectUploadObjectWithClient() error = %v", err)
	}
	if result.Matched {
		t.Fatal("expected mismatches")
	}
	if len(result.Mismatches) != 2 {
		t.Fatalf("unexpected mismatch count: %d", len(result.Mismatches))
	}
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./storage -run 'TestVerifyDirectUploadObject' -v`

Expected: FAIL because `ObjectInfo` does not expose metadata yet and verification logic is missing.

**Step 3: Implement the verification layer**

Extend the public and provider-facing `ObjectInfo` types:

```go
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
	Metadata     map[string]string
	Checksums    map[string]string
}
```

Implement `VerifyDirectUploadObjectWithClient` in [storage/direct_upload.go](/Users/kj/projects/go-kit/storage/direct_upload.go):

```go
func VerifyDirectUploadObjectWithClient(ctx context.Context, c Client, req DirectUploadVerificationRequest) (*DirectUploadVerificationResult, error) {
	if c == nil {
		return nil, ErrMissingClient
	}

	info, err := c.Stat(ctx, req.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("stat object: %w", err)
	}

	result := &DirectUploadVerificationResult{
		Exists:  info != nil,
		Matched: true,
		Object:  info,
	}

	appendMismatch := func(field, expected, actual string) {
		result.Matched = false
		result.Mismatches = append(result.Mismatches, DirectUploadMismatch{
			Field:    field,
			Expected: expected,
			Actual:   actual,
		})
	}

	// compare content type, size, metadata, optional checksum
	return result, nil
}
```

Implementation rules:
- Treat “object missing” as `Exists=false`, `Matched=false`, not as a hard error when the provider returns a not-found condition that can be recognized.
- Compare metadata by exact key/value equality after the request normalization rules have been applied.
- If checksum verification is requested but the provider cannot expose a checksum field, return a structured unsupported error rather than silently ignoring it.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./storage -run 'TestVerifyDirectUploadObject' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add storage/direct_upload.go storage/direct_upload_test.go storage/providers/types.go storage/storage_test.go
git commit -m "feat(storage): add direct upload verification"
```

### Task 4: Add metadata support to the shared upload model and fix README drift

**Files:**
- Modify: `storage/providers/types.go`
- Modify: `storage/types.go`
- Modify: `storage/providers/s3/client.go`
- Modify: `storage/providers/cos/client.go`
- Modify: `storage/providers/oss/client.go`
- Modify: `storage/README.md`

**Step 1: Write the failing metadata tests**

Add targeted provider-agnostic tests in [storage/direct_upload_test.go](/Users/kj/projects/go-kit/storage/direct_upload_test.go) that assert metadata survives normalization and is handed to provider methods through the request object. Add one focused README-alignment test if useful, but do not overfit to doc text.

Also add a tiny unit test for the new option helper in [storage/storage_test.go](/Users/kj/projects/go-kit/storage/storage_test.go):

```go
func TestWithMetadata(t *testing.T) {
	t.Parallel()

	opt := UploadOption{}
	WithMetadata("author", "alice")(&opt)

	if got := opt.Metadata["author"]; got != "alice" {
		t.Fatalf("unexpected metadata value: %q", got)
	}
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./storage -run 'TestWithMetadata' -v`

Expected: FAIL because `UploadOption` does not carry metadata yet.

**Step 3: Implement metadata support**

Extend `UploadOption` and the helper surface:

```go
type UploadOption struct {
	ContentType string
	Metadata    map[string]string
}

func WithMetadata(key, value string) UploadOptionFunc {
	return func(o *UploadOption) {
		if o.Metadata == nil {
			o.Metadata = make(map[string]string)
		}
		o.Metadata[key] = value
	}
}
```

Apply metadata in each provider upload path and refresh [storage/README.md](/Users/kj/projects/go-kit/storage/README.md) so the documented examples are now real.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./storage -run 'TestWithMetadata' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add storage/providers/types.go storage/types.go storage/providers/s3/client.go storage/providers/cos/client.go storage/providers/oss/client.go storage/README.md
git commit -m "feat(storage): add metadata upload support"
```

### Task 5: Implement S3 direct-upload authorization with provider capability checks

**Files:**
- Create: `storage/providers/s3/direct_upload_test.go`
- Modify: `storage/providers/s3/client.go`
- Modify: `storage/providers/types.go`

**Step 1: Write the failing S3 tests**

Add table-driven tests in [storage/providers/s3/direct_upload_test.go](/Users/kj/projects/go-kit/storage/providers/s3/direct_upload_test.go):

```go
func TestAuthorizeDirectUploadPut(t *testing.T) {
	t.Parallel()

	client, err := NewClient(&providers.Config{
		Type:            providers.TypeS3,
		Bucket:          "bucket",
		Region:          "us-east-1",
		AccessKeyID:     "key",
		AccessKeySecret: "secret",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	auth, err := client.AuthorizeDirectUpload(context.Background(), providers.DirectUploadRequest{
		ObjectKey:   "uploads/a.png",
		Mode:        providers.DirectUploadModePut,
		ContentType: "image/png",
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
}

func TestAuthorizeDirectUploadRejectsUnsupportedRangeForPut(t *testing.T) {
	t.Parallel()

	// expect an unsupported constraint error when Min/Max forces POST but Mode=put
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./storage/providers/s3 -run 'TestAuthorizeDirectUpload' -v`

Expected: FAIL because the S3 client does not implement `AuthorizeDirectUpload`.

**Step 3: Implement the S3 authorization mapping**

In [storage/providers/s3/client.go](/Users/kj/projects/go-kit/storage/providers/s3/client.go), add:

```go
func (c *client) AuthorizeDirectUpload(ctx context.Context, req providers.DirectUploadRequest) (*providers.DirectUploadAuthorization, error) {
	mode, err := selectS3DirectUploadMode(req)
	if err != nil {
		return nil, err
	}

	switch mode {
	case providers.DirectUploadModePut:
		return c.authorizePutDirectUpload(ctx, req)
	case providers.DirectUploadModePost:
		return c.authorizePostDirectUpload(ctx, req)
	default:
		return nil, fmt.Errorf("select direct upload mode: %w", providers.ErrUnsupportedDirectUploadMode)
	}
}
```

Implementation rules:
- Reuse the existing presign client for `PUT`.
- Include `ContentType`, checksum headers, and metadata headers in the signed request when required.
- Use POST policy when a size range forces it.
- Return provider-specific unsupported errors instead of silently weakening the request.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./storage/providers/s3 -run 'TestAuthorizeDirectUpload' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add storage/providers/s3/client.go storage/providers/s3/direct_upload_test.go storage/providers/types.go
git commit -m "feat(storage): add s3 direct upload authorization"
```

### Task 6: Implement COS and OSS direct-upload authorization

**Files:**
- Create: `storage/providers/cos/direct_upload_test.go`
- Create: `storage/providers/oss/direct_upload_test.go`
- Modify: `storage/providers/cos/client.go`
- Modify: `storage/providers/oss/client.go`

**Step 1: Write the failing COS and OSS tests**

Add one focused `PUT` authorization test and one unsupported-constraint test for each provider.

For OSS include a regression test that proves upload authorization is no longer built from `GetObjectRequest`:

```go
func TestAuthorizeDirectUploadDoesNotUseGetObjectSigning(t *testing.T) {
	t.Parallel()

	// assert returned method is PUT or POST and that the signed output is not a read-only GetObject flow
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./storage/providers/cos ./storage/providers/oss -run 'TestAuthorizeDirectUpload' -v`

Expected: FAIL because neither client implements the new contract yet.

**Step 3: Implement COS authorization**

In [storage/providers/cos/client.go](/Users/kj/projects/go-kit/storage/providers/cos/client.go):
- support `PUT` presign with signed headers when possible
- support `POST policy` where COS exposes the required condition set
- return explicit unsupported errors where the provider cannot enforce the requested constraint set

**Step 4: Implement OSS authorization**

In [storage/providers/oss/client.go](/Users/kj/projects/go-kit/storage/providers/oss/client.go):
- add real upload authorization support
- map `PUT` or `POST` according to the selected mode
- stop using `GetObjectRequest` for upload authorization

**Step 5: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./storage/providers/cos ./storage/providers/oss -run 'TestAuthorizeDirectUpload' -v`

Expected: PASS

**Step 6: Commit**

```bash
git add storage/providers/cos/client.go storage/providers/cos/direct_upload_test.go storage/providers/oss/client.go storage/providers/oss/direct_upload_test.go
git commit -m "feat(storage): add cos and oss direct upload authorization"
```

### Task 7: Document the new API, align package docs, and run full verification

**Files:**
- Modify: `storage/README.md`
- Modify: `storage/doc.go`
- Modify: `storage/storage_test.go`
- Modify: `storage/direct_upload_test.go`

**Step 1: Update README and package docs**

Refresh [storage/README.md](/Users/kj/projects/go-kit/storage/README.md) and [storage/doc.go](/Users/kj/projects/go-kit/storage/doc.go):
- keep `SignedURL` documented as a basic signing API
- add a new “安全直传授权” section showing `AuthorizeDirectUpload`
- add a note that strong client direct upload should use `AuthorizeDirectUpload` rather than `SignedURL`
- describe the upload-after-authorization verification step

Use a concrete example:

```go
auth, err := storage.AuthorizeDirectUpload(ctx, storage.DirectUploadRequest{
	ObjectKey:   objectKey,
	ContentType: "image/png",
	Checksum: &storage.DirectUploadChecksum{
		Algorithm: storage.DirectUploadChecksumMD5,
		Value:     checksum,
	},
})
```

**Step 2: Add final integration-style package tests**

Extend [storage/direct_upload_test.go](/Users/kj/projects/go-kit/storage/direct_upload_test.go) with table-driven tests that cover:
- auto-mode selection inputs
- invalid checksum inputs
- invalid size combinations
- verification mismatch reporting

**Step 3: Run focused package verification**

Run:
- `GOCACHE=/tmp/go-build go test ./storage -v`
- `GOCACHE=/tmp/go-build go test ./storage/providers/s3 -v`
- `GOCACHE=/tmp/go-build go test ./storage/providers/cos -v`
- `GOCACHE=/tmp/go-build go test ./storage/providers/oss -v`

Expected: PASS

**Step 4: Run repository-level verification**

Run:
- `GOCACHE=/tmp/go-build go test ./storage/...`
- `GOCACHE=/tmp/go-build go test ./cmd/gokit/pkg/gokit -v`
- `GOCACHE=/tmp/go-build go run ./cmd/gokit list`

Expected:
- all `storage` package tests PASS
- capability tests PASS
- `gokit list` shows the updated `storage` capability

**Step 5: Commit**

```bash
git add storage/README.md storage/doc.go storage/storage_test.go storage/direct_upload_test.go
git commit -m "docs(storage): document direct upload authorization"
```
