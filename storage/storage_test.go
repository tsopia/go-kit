package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/tsopia/go-kit/storage/providers"
)

func TestConfigure(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *providers.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &providers.Config{
				Type:            providers.TypeOSS,
				Bucket:          "test-bucket",
				Region:          "cn-hangzhou",
				AccessKeyID:     "test-key",
				AccessKeySecret: "test-secret",
			},
			wantErr: false,
		},
		{
			name:    "missing type",
			cfg:     &providers.Config{},
			wantErr: true,
		},
		{
			name: "missing bucket",
			cfg: &providers.Config{
				Type:   providers.TypeOSS,
				Region: "cn-hangzhou",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Configure(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Configure() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUploadWithClient(t *testing.T) {
	ctx := context.Background()
	err := UploadWithClient(ctx, nil, "test.txt", strings.NewReader("hello"))
	if err != ErrMissingClient {
		t.Errorf("UploadWithClient() error = %v, want %v", err, ErrMissingClient)
	}
}

func TestDownloadWithClient(t *testing.T) {
	ctx := context.Background()
	_, err := DownloadWithClient(ctx, nil, "test.txt")
	if err != ErrMissingClient {
		t.Errorf("DownloadWithClient() error = %v, want %v", err, ErrMissingClient)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *providers.Config
		wantErr bool
	}{
		{
			name:    "empty config",
			cfg:     &providers.Config{},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &providers.Config{
				Type:            providers.TypeOSS,
				Bucket:          "bucket",
				Region:          "region",
				AccessKeyID:     "key",
				AccessKeySecret: "secret",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
