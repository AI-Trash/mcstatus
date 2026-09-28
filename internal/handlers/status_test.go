package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mcstatus/internal/blocklist"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/types"
)

func TestParsePluginsAndSoftware(t *testing.T) {
	tests := []struct {
		name         string
		data         map[string]string
		expectedSw   string
		expectedPlen int
	}{
		{
			name: "Standard with software prefix and plugins",
			data: map[string]string{
				"plugins": "Paper 1.20.4: Essentials 2.20.1; WorldEdit 7.2.15; Vault",
			},
			expectedSw:   "Paper 1.20.4",
			expectedPlen: 3,
		},
		{
			name: "Software only in server_mod",
			data: map[string]string{
				"server_mod": "Fabric",
				"plugins":    "Sodium 0.5.8; Lithium 0.11.2",
			},
			expectedSw:   "Fabric",
			expectedPlen: 2,
		},
		{
			name: "Software with no plugins",
			data: map[string]string{
				"plugins": "CraftBukkit:",
			},
			expectedSw:   "CraftBukkit",
			expectedPlen: 0,
		},
		{
			name:         "Nil data",
			data:         nil,
			expectedSw:   "",
			expectedPlen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw, plugins := parsePluginsAndSoftware(tt.data)
			if tt.expectedSw == "" && sw != nil {
				t.Fatalf("expected nil software, got %v", *sw)
			}
			if tt.expectedSw != "" {
				if sw == nil || *sw != tt.expectedSw {
					t.Fatalf("expected software %q, got %v", tt.expectedSw, sw)
				}
			}
			if len(plugins) != tt.expectedPlen {
				t.Fatalf("expected %d plugins, got %d", tt.expectedPlen, len(plugins))
			}
		})
	}
}

func TestJavaStatusInvalidAddress(t *testing.T) {
	cfg := config.Load()
	c := cache.New(10*time.Second, 1*time.Minute)
	defer c.Close()
	bl := blocklist.New(1 * time.Hour)
	defer bl.Close()

	handler := HandleJavaStatus(cfg, c, bl)

	req := httptest.NewRequest(http.MethodGet, "/status/java/invalid%20host%20with%20spaces", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Invalid address value" {
		t.Fatalf("expected 'Invalid address value', got %q", string(body))
	}
}

func TestJavaStatusOffline(t *testing.T) {
	cfg := config.Load()
	cfg.DefaultTimeout = 50 * time.Millisecond
	c := cache.New(10*time.Second, 1*time.Minute)
	defer c.Close()
	bl := blocklist.New(1 * time.Hour)
	defer bl.Close()

	handler := HandleJavaStatus(cfg, c, bl)

	// Use an unroutable/closed port on localhost
	req := httptest.NewRequest(http.MethodGet, "/status/java/127.0.0.1:59999?timeout=0.05", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json, got %s", resp.Header.Get("Content-Type"))
	}

	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}

	var statusResp types.JavaStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if statusResp.Online {
		t.Fatal("expected server to be offline")
	}
	if statusResp.Host != "127.0.0.1" {
		t.Fatalf("expected host 127.0.0.1, got %s", statusResp.Host)
	}
	if statusResp.Port != 59999 {
		t.Fatalf("expected port 59999, got %d", statusResp.Port)
	}
	if statusResp.RetrievedAt <= 0 || statusResp.ExpiresAt <= 0 {
		t.Fatal("expected non-zero retrieved_at and expires_at")
	}

	// Test cache hit and ETag
	req2 := httptest.NewRequest(http.MethodGet, "/status/java/127.0.0.1:59999?timeout=0.05", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()

	handler(w2, req2)
	resp2 := w2.Result()
	if resp2.StatusCode != http.StatusNotModified {
		t.Fatalf("expected 304 Not Modified, got %d", resp2.StatusCode)
	}
}

func TestBedrockStatusInvalidAddress(t *testing.T) {
	cfg := config.Load()
	c := cache.New(10*time.Second, 1*time.Minute)
	defer c.Close()
	bl := blocklist.New(1 * time.Hour)
	defer bl.Close()

	handler := HandleBedrockStatus(cfg, c, bl)

	req := httptest.NewRequest(http.MethodGet, "/status/bedrock/invalid%20host%20with%20spaces", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Invalid address value" {
		t.Fatalf("expected 'Invalid address value', got %q", string(body))
	}
}

func TestBedrockStatusOffline(t *testing.T) {
	cfg := config.Load()
	cfg.DefaultTimeout = 50 * time.Millisecond
	c := cache.New(10*time.Second, 1*time.Minute)
	defer c.Close()
	bl := blocklist.New(1 * time.Hour)
	defer bl.Close()

	handler := HandleBedrockStatus(cfg, c, bl)

	// Use an unroutable/closed port on localhost
	req := httptest.NewRequest(http.MethodGet, "/status/bedrock/127.0.0.1:59999?timeout=0.05", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json, got %s", resp.Header.Get("Content-Type"))
	}

	var statusResp types.BedrockStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if statusResp.Online {
		t.Fatal("expected server to be offline")
	}
	if statusResp.Host != "127.0.0.1" {
		t.Fatalf("expected host 127.0.0.1, got %s", statusResp.Host)
	}
	if statusResp.Port != 59999 {
		t.Fatalf("expected port 59999, got %d", statusResp.Port)
	}
	if statusResp.RetrievedAt <= 0 || statusResp.ExpiresAt <= 0 {
		t.Fatal("expected non-zero retrieved_at and expires_at")
	}
}
