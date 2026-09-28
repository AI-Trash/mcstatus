package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPFromTrustedProxy(t *testing.T) {
	filter := NewIPFilter([]string{"127.0.0.1/32", "10.0.0.0/8"})

	// 1. CF-Connecting-IP from trusted proxy
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	req1.Header.Set("CF-Connecting-IP", "198.51.100.1")
	if ip := ClientIP(req1, filter); ip != "198.51.100.1" {
		t.Fatalf("expected 198.51.100.1 from CF-Connecting-IP, got %s", ip)
	}

	// 2. X-Real-IP from trusted proxy
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.1.2.3:54321"
	req2.Header.Set("X-Real-IP", "198.51.100.2")
	if ip := ClientIP(req2, filter); ip != "198.51.100.2" {
		t.Fatalf("expected 198.51.100.2 from X-Real-IP, got %s", ip)
	}

	// 3. X-Forwarded-For from trusted proxy
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "127.0.0.1:12345"
	req3.Header.Set("X-Forwarded-For", "203.0.113.195, 10.0.0.1")
	if ip := ClientIP(req3, filter); ip != "203.0.113.195" {
		t.Fatalf("expected 203.0.113.195 from X-Forwarded-For, got %s", ip)
	}

	// 4. Untrusted public peer trying to spoof CF-Connecting-IP
	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	req4.RemoteAddr = "198.51.100.99:12345"
	req4.Header.Set("CF-Connecting-IP", "1.1.1.1")
	req4.Header.Set("X-Forwarded-For", "1.1.1.1")
	if ip := ClientIP(req4, filter); ip != "198.51.100.99" {
		t.Fatalf("expected untrusted peer IP 198.51.100.99, got %s", ip)
	}
}

func TestDefaultPrivateRangesTrust(t *testing.T) {
	filter := NewIPFilter(nil) // defaults to private ranges

	testCases := []struct {
		remoteAddr string
		headerIP   string
		expected   string
	}{
		{"127.0.0.1:1234", "1.2.3.4", "1.2.3.4"},
		{"192.168.1.10:1234", "1.2.3.4", "1.2.3.4"},
		{"172.20.0.5:1234", "1.2.3.4", "1.2.3.4"},
		{"10.0.5.1:1234", "1.2.3.4", "1.2.3.4"},
		{"[::1]:1234", "1.2.3.4", "1.2.3.4"},
		{"8.8.8.8:1234", "1.2.3.4", "8.8.8.8"}, // Public IP cannot spoof
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = tc.remoteAddr
		req.Header.Set("CF-Connecting-IP", tc.headerIP)
		got := ClientIP(req, filter)
		if got != tc.expected {
			t.Errorf("RemoteAddr=%s, got %s, expected %s", tc.remoteAddr, got, tc.expected)
		}
	}
}
