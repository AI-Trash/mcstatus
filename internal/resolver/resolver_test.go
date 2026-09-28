package resolver

import (
	"context"
	"testing"
	"time"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		input       string
		defaultPort uint16
		wantHost    string
		wantPort    uint16
		wantErr     bool
	}{
		{"demo.mcstatus.io", 25565, "demo.mcstatus.io", 25565, false},
		{"demo.mcstatus.io:25565", 25565, "demo.mcstatus.io", 25565, false},
		{"demo.mcstatus.io:19132", 19132, "demo.mcstatus.io", 19132, false},
		{"127.0.0.1", 25565, "127.0.0.1", 25565, false},
		{"127.0.0.1:25565", 25565, "127.0.0.1", 25565, false},
		{"[::1]", 25565, "::1", 25565, false},
		{"[::1]:25565", 25565, "::1", 25565, false},
		{"::1", 25565, "::1", 25565, false},
		{"", 25565, "", 0, true},
		{"bad host name", 25565, "", 0, true},
		{"host:999999", 25565, "", 0, true},
		{"host:0", 25565, "", 0, true},
		{"host:-1", 25565, "", 0, true},
		{"!invalid", 25565, "", 0, true},
	}

	for _, tt := range tests {
		host, port, err := ParseAddress(tt.input, tt.defaultPort)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseAddress(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr {
			if host != tt.wantHost || port != tt.wantPort {
				t.Errorf("ParseAddress(%q) = (%q, %d), want (%q, %d)", tt.input, host, port, tt.wantHost, tt.wantPort)
			}
		}
	}
}

func TestResolveIPDirect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ip := ResolveIP(ctx, "127.0.0.1")
	if ip == nil || *ip != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1, got %v", ip)
	}

	ip6 := ResolveIP(ctx, "::1")
	if ip6 == nil || *ip6 != "::1" {
		t.Fatalf("expected ::1, got %v", ip6)
	}
}

func TestResolveDualStackIPs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	v6, v4 := ResolveDualStackIPs(ctx, "127.0.0.1")
	if len(v6) != 0 || len(v4) != 1 || v4[0] != "127.0.0.1" {
		t.Fatalf("expected IPv4 only for 127.0.0.1, got v6=%v, v4=%v", v6, v4)
	}

	v6, v4 = ResolveDualStackIPs(ctx, "::1")
	if len(v6) != 1 || v6[0] != "::1" || len(v4) != 0 {
		t.Fatalf("expected IPv6 only for ::1, got v6=%v, v4=%v", v6, v4)
	}
}
