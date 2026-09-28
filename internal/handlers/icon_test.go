package handlers

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mcstatus/internal/assets"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
)

func TestHandleIconEmptyAddress(t *testing.T) {
	cfg := config.Load()
	c := cache.New(10*time.Second, 1*time.Minute)
	defer c.Close()

	handler := HandleIcon(cfg, c)

	// Case 1: GET /v2/icon
	{
		req := httptest.NewRequest(http.MethodGet, "/v2/icon", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "image/png" {
			t.Fatalf("expected Content-Type image/png, got %s", resp.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(resp.Body)
		if !bytes.Equal(body, assets.DefaultIconBytes) {
			t.Fatalf("expected DefaultIconBytes, got %d bytes", len(body))
		}
	}

	// Case 2: GET /v2/icon/
	{
		req := httptest.NewRequest(http.MethodGet, "/v2/icon/", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "image/png" {
			t.Fatalf("expected Content-Type image/png, got %s", resp.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(resp.Body)
		if !bytes.Equal(body, assets.DefaultIconBytes) {
			t.Fatalf("expected DefaultIconBytes, got %d bytes", len(body))
		}
	}
}

func TestHandleIconInvalidAddress(t *testing.T) {
	cfg := config.Load()
	c := cache.New(10*time.Second, 1*time.Minute)
	defer c.Close()

	handler := HandleIcon(cfg, c)

	req := httptest.NewRequest(http.MethodGet, "/v2/icon/invalid%20host%20with%20spaces", nil)
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

func TestHandleIconOfflineServer(t *testing.T) {
	cfg := config.Load()
	cfg.DefaultTimeout = 50 * time.Millisecond
	c := cache.New(10*time.Second, 1*time.Minute)
	defer c.Close()

	handler := HandleIcon(cfg, c)

	// First request: cache miss, offline server falls back to DefaultIconBytes
	req := httptest.NewRequest(http.MethodGet, "/v2/icon/127.0.0.1:59998?timeout=0.05", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("expected Content-Type image/png, got %s", resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("X-Cache-Hit") != "false" {
		t.Fatalf("expected X-Cache-Hit false, got %s", resp.Header.Get("X-Cache-Hit"))
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, assets.DefaultIconBytes) {
		t.Fatalf("expected DefaultIconBytes for offline server")
	}

	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatalf("expected non-empty ETag")
	}

	// Second request: cache hit
	req2 := httptest.NewRequest(http.MethodGet, "/v2/icon/127.0.0.1:59998?timeout=0.05", nil)
	w2 := httptest.NewRecorder()
	handler(w2, req2)

	resp2 := w2.Result()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 on cache hit, got %d", resp2.StatusCode)
	}
	if resp2.Header.Get("X-Cache-Hit") != "true" {
		t.Fatalf("expected X-Cache-Hit true, got %s", resp2.Header.Get("X-Cache-Hit"))
	}

	// Third request: If-None-Match ETag check -> 304 Not Modified
	req3 := httptest.NewRequest(http.MethodGet, "/v2/icon/127.0.0.1:59998?timeout=0.05", nil)
	req3.Header.Set("If-None-Match", etag)
	w3 := httptest.NewRecorder()
	handler(w3, req3)

	resp3 := w3.Result()
	if resp3.StatusCode != http.StatusNotModified {
		t.Fatalf("expected status 304 Not Modified, got %d", resp3.StatusCode)
	}
}

func TestHandleIconMethodNotAllowed(t *testing.T) {
	cfg := config.Load()
	c := cache.New(10*time.Second, 1*time.Minute)
	defer c.Close()

	handler := HandleIcon(cfg, c)

	req := httptest.NewRequest(http.MethodPost, "/v2/icon", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", resp.StatusCode)
	}
}
