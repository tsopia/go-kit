package storage

import (
	"context"
	"io"
	"testing"
	"time"
)

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
		want    DirectUploadRequest
		wantErr bool
	}{
		{
			name: "default mode becomes auto",
			req:  DirectUploadRequest{ObjectKey: "uploads/a.png"},
			want: DirectUploadRequest{
				ObjectKey: "uploads/a.png",
				Mode:      DirectUploadModeAuto,
			},
		},
		{
			name: "content type and metadata are trimmed",
			req: DirectUploadRequest{
				ObjectKey:   "uploads/a.png",
				ContentType: " image/png ",
				Metadata: map[string]string{
					" owner ": " alice ",
				},
			},
			want: DirectUploadRequest{
				ObjectKey:   "uploads/a.png",
				ContentType: "image/png",
				Metadata: map[string]string{
					"owner": "alice",
				},
				Mode: DirectUploadModeAuto,
			},
		},
		{
			name: "exact size rejects range",
			req: DirectUploadRequest{
				ObjectKey: "uploads/a.png",
				Size: &DirectUploadSize{
					Exact: 10,
					Min:   1,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeDirectUploadRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeDirectUploadRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if got.ObjectKey != tt.want.ObjectKey {
				t.Fatalf("unexpected object key: got %q want %q", got.ObjectKey, tt.want.ObjectKey)
			}
			if got.ContentType != tt.want.ContentType {
				t.Fatalf("unexpected content type: got %q want %q", got.ContentType, tt.want.ContentType)
			}
			if got.Mode != tt.want.Mode {
				t.Fatalf("unexpected mode: got %q want %q", got.Mode, tt.want.Mode)
			}
			if len(got.Metadata) != len(tt.want.Metadata) {
				t.Fatalf("unexpected metadata length: got %d want %d", len(got.Metadata), len(tt.want.Metadata))
			}
			for key, wantValue := range tt.want.Metadata {
				if got.Metadata[key] != wantValue {
					t.Fatalf("unexpected metadata value for %q: got %q want %q", key, got.Metadata[key], wantValue)
				}
			}
		})
	}
}

func TestAuthorizeDirectUploadWithClientNormalizesMetadataForAuthorizer(t *testing.T) {
	t.Parallel()

	client := &fakeDirectUploadClient{
		authorizeResult: &DirectUploadAuthorization{},
	}

	_, err := AuthorizeDirectUploadWithClient(context.Background(), client, DirectUploadRequest{
		ObjectKey: "uploads/a.png",
		Metadata: map[string]string{
			" owner ": " alice ",
		},
	})
	if err != nil {
		t.Fatalf("AuthorizeDirectUploadWithClient() error = %v", err)
	}

	if got := client.lastAuthorizeRequest.Metadata["owner"]; got != "alice" {
		t.Fatalf("unexpected normalized metadata value: %q", got)
	}
}

func TestVerifyDirectUploadObject(t *testing.T) {
	t.Parallel()

	client := &fakeDirectUploadClient{
		statResult: &ObjectInfo{
			Key:         "uploads/a.png",
			Size:        12,
			ContentType: "image/png",
			Metadata: map[string]string{
				"owner": "u1",
			},
		},
	}

	result, err := VerifyDirectUploadObjectWithClient(context.Background(), client, DirectUploadVerificationRequest{
		ObjectKey:   "uploads/a.png",
		ContentType: "image/png",
		Metadata: map[string]string{
			"owner": "u1",
		},
		Size: &DirectUploadSize{Exact: 12},
	})
	if err != nil {
		t.Fatalf("VerifyDirectUploadObjectWithClient() error = %v", err)
	}
	if !result.Exists || !result.Matched {
		t.Fatalf("expected matched result, got %+v", result)
	}
	if len(result.Mismatches) != 0 {
		t.Fatalf("expected no mismatches, got %+v", result.Mismatches)
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
	if !result.Exists {
		t.Fatal("expected object to exist")
	}
	if result.Matched {
		t.Fatal("expected mismatches")
	}
	if len(result.Mismatches) != 2 {
		t.Fatalf("unexpected mismatch count: %d", len(result.Mismatches))
	}
}

type fakeDirectUploadClient struct {
	statResult           *ObjectInfo
	statErr              error
	authorizeResult      *DirectUploadAuthorization
	authorizeErr         error
	lastAuthorizeRequest DirectUploadRequest
}

func (f *fakeDirectUploadClient) Upload(context.Context, string, io.Reader, ...UploadOptionFunc) error {
	return nil
}

func (f *fakeDirectUploadClient) Download(context.Context, string, ...DownloadOptionFunc) (io.ReadCloser, error) {
	return nil, nil
}

func (f *fakeDirectUploadClient) Delete(context.Context, string) error {
	return nil
}

func (f *fakeDirectUploadClient) Exists(context.Context, string) (bool, error) {
	return f.statResult != nil, nil
}

func (f *fakeDirectUploadClient) Stat(context.Context, string) (*ObjectInfo, error) {
	return f.statResult, f.statErr
}

func (f *fakeDirectUploadClient) SignedURL(context.Context, string, time.Duration, ...SignOptionFunc) (string, error) {
	return "", nil
}

func (f *fakeDirectUploadClient) AuthorizeDirectUpload(_ context.Context, req DirectUploadRequest) (*DirectUploadAuthorization, error) {
	f.lastAuthorizeRequest = req
	return f.authorizeResult, f.authorizeErr
}

func (f *fakeDirectUploadClient) InitMultipart(context.Context, string, ...UploadOptionFunc) (*MultipartUpload, error) {
	return nil, nil
}

func (f *fakeDirectUploadClient) UploadPart(context.Context, string, int, io.Reader, ...UploadOptionFunc) (*PartInfo, error) {
	return nil, nil
}

func (f *fakeDirectUploadClient) CompleteMultipart(context.Context, string, []*PartInfo, ...UploadOptionFunc) error {
	return nil
}

func (f *fakeDirectUploadClient) AbortMultipart(context.Context, string) error {
	return nil
}

func (f *fakeDirectUploadClient) DeleteBatch(context.Context, []string) error {
	return nil
}
