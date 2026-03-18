package utils

import (
	"context"
	"net"
	"testing"
)

func TestParseIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		wantNil  bool
		is4      bool
		is6      bool
	}{
		{"IPv4", "192.168.1.1", false, true, false},
		{"IPv6", "::1", false, false, true},
		{"IPv6 full", "2001:db8::1", false, false, true},
		{"invalid", "invalid", true, false, false},
		{"empty", "", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseIP(tt.ip)
			if tt.wantNil && got != nil {
				t.Errorf("ParseIP(%q) = %v, want nil", tt.ip, got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("ParseIP(%q) = nil, want non-nil", tt.ip)
			}
			if got != nil {
				if tt.is4 && got.To4() == nil {
					t.Errorf("ParseIP(%q) is not IPv4", tt.ip)
				}
				if tt.is6 && got.To4() != nil {
					t.Errorf("ParseIP(%q) is not IPv6", tt.ip)
				}
			}
		})
	}
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{"valid IPv4 CIDR", "192.168.0.0/16", false},
		{"valid IPv6 CIDR", "::1/128", false},
		{"invalid CIDR", "invalid", true},
		{"invalid format", "192.168.1.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCIDR(tt.cidr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseCIDR(%q) error = nil, wantErr true", tt.cidr)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseCIDR(%q) error = %v, wantErr false", tt.cidr, err)
				return
			}
			if got == nil {
				t.Errorf("ParseCIDR(%q) = nil, want non-nil", tt.cidr)
			}
		})
	}
}

func TestIsIPInCIDR(t *testing.T) {
	cidr, _ := ParseCIDR("192.168.0.0/16")

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"in range", "192.168.1.1", true},
		{"in range 2", "192.168.255.255", true},
		{"out of range", "10.0.0.1", false},
		{"boundary", "192.169.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("ParseIP(%q) returned nil", tt.ip)
			}
			got := IsIPInCIDR(ip, cidr)
			if got != tt.want {
				t.Errorf("IsIPInCIDR(%q, %q) = %v, want %v", tt.ip, cidr.String(), got, tt.want)
			}
		})
	}
}

func TestIsIPInCIDRs(t *testing.T) {
	cidrs := []*net.IPNet{
		mustCIDR("10.0.0.0/8"),
		mustCIDR("192.168.0.0/16"),
	}

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"in first CIDR", "10.1.2.3", true},
		{"in second CIDR", "192.168.1.1", true},
		{"out of all", "172.16.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("ParseIP(%q) returned nil", tt.ip)
			}
			got := IsIPInCIDRs(ip, cidrs)
			if got != tt.want {
				t.Errorf("IsIPInCIDRs(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestWithClientIP(t *testing.T) {
	ctx := context.Background()
	clientIP := "192.168.1.1"

	ctx = WithClientIP(ctx, clientIP)
	got := ClientIPFromContext(ctx)

	if got != clientIP {
		t.Errorf("ClientIPFromContext() = %q, want %q", got, clientIP)
	}
}

func TestClientIPFromContext_NotSet(t *testing.T) {
	ctx := context.Background()
	got := ClientIPFromContext(ctx)

	if got != "" {
		t.Errorf("ClientIPFromContext() = %q, want empty string", got)
	}
}

func TestClientIPFromContext_StringKey(t *testing.T) {
	ctx := context.WithValue(context.Background(), ClientIPKey, "10.0.0.1")
	got := ClientIPFromContext(ctx)

	if got != "10.0.0.1" {
		t.Errorf("ClientIPFromContext() = %q, want %q", got, "10.0.0.1")
	}
}

func mustCIDR(cidr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return ipNet
}
