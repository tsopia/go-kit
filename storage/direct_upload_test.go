package storage

import (
	"context"
	"testing"
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
