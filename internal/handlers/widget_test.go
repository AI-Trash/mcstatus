package handlers

import (
	"bytes"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mcstatus/internal/assets"
	"mcstatus/internal/blocklist"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
)

func TestParseWidgetBool(t *testing.T) {
	tests := []struct {
		val        string
		defaultVal bool
		expected   bool
	}{
		{"", true, true},
		{"", false, false},
		{"true", false, true},
		{"1", false, true},
		{"false", true, false},
		{"0", true, false},
		{"invalid", true, true},
		{"invalid", false, false},
	}

	for _, tt := range tests {
		got := ParseBool(tt.val, tt.defaultVal)
		if got != tt.expected {
			t.Errorf("ParseBool(%q, %v) = %v; want %v", tt.val, tt.defaultVal, got, tt.expected)
		}
	}
}

func TestDecodeBase64Icon(t *testing.T) {
	// Nil
	if img := decodeBase64Icon(nil); img == nil {
		t.Fatal("decodeBase64Icon(nil) returned nil")
	}

	// Empty
	empty := ""
	if img := decodeBase64Icon(&empty); img == nil {
		t.Fatal("decodeBase64Icon(\"\") returned nil")
	}

	// Invalid base64
	invalid := "not-valid-base64!!!"
	if img := decodeBase64Icon(&invalid); img != assets.DefaultIcon {
		t.Fatal("decodeBase64Icon(invalid) did not return DefaultIcon")
	}

	// Valid base64 PNG
	importBase64 := "data:image/png;base64," + "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	if img := decodeBase64Icon(&importBase64); img == nil {
		t.Fatal("decodeBase64Icon(valid) returned nil")
	}
}

func TestHandleJavaWidget(t *testing.T) {
	cfg := &config.Config{
		DefaultTimeout: 100 * time.Millisecond,
		MaxTimeout:     500 * time.Millisecond,
		CacheTTL:       60 * time.Second,
	}
	c := cache.New(60*time.Second, 10*time.Minute)
	defer c.Close()
	bl := blocklist.New(1 * time.Hour)
	defer bl.Close()

	handler := HandleJavaWidget(cfg, c, bl)

	t.Run("Valid address (offline server)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/widget/java/127.0.0.1:25599?dark=false&rounded=false&timeout=0.1", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
			t.Fatalf("expected Content-Type image/png, got %s", ct)
		}

		etag := resp.Header.Get("ETag")
		if etag == "" {
			t.Fatal("expected ETag header to be set")
		}

		// Verify PNG can be decoded
		body := w.Body.Bytes()
		img, err := png.Decode(bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed to decode returned PNG: %v", err)
		}

		if img.Bounds().Dx() != 860 || img.Bounds().Dy() != 240 {
			t.Fatalf("unexpected dimensions: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
		}
	})

	t.Run("ETag 304 Not Modified", func(t *testing.T) {
		req1 := httptest.NewRequest(http.MethodGet, "/v2/widget/java/127.0.0.1:25599?timeout=0.1", nil)
		w1 := httptest.NewRecorder()
		handler(w1, req1)

		etag := w1.Header().Get("ETag")
		if etag == "" {
			t.Fatal("expected ETag header")
		}

		req2 := httptest.NewRequest(http.MethodGet, "/v2/widget/java/127.0.0.1:25599?timeout=0.1", nil)
		req2.Header.Set("If-None-Match", etag)
		w2 := httptest.NewRecorder()
		handler(w2, req2)

		if w2.Code != http.StatusNotModified {
			t.Fatalf("expected 304 Not Modified, got %d", w2.Code)
		}
	})

	t.Run("HEAD request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/v2/widget/java/127.0.0.1:25599?timeout=0.1", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
		if w.Body.Len() != 0 {
			t.Fatalf("expected empty body for HEAD request, got %d bytes", w.Body.Len())
		}
	})

	t.Run("Invalid address", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/widget/java/invalid:address:too:many:colons", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
		if body := w.Body.String(); body != "Invalid address value" {
			t.Fatalf("expected 'Invalid address value', got %q", body)
		}
	})

	t.Run("Empty address", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/widget/java/", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})
}

func TestHandleBedrockWidget(t *testing.T) {
	cfg := &config.Config{
		DefaultTimeout: 100 * time.Millisecond,
		MaxTimeout:     500 * time.Millisecond,
		CacheTTL:       60 * time.Second,
	}
	c := cache.New(60*time.Second, 10*time.Minute)
	defer c.Close()
	bl := blocklist.New(1 * time.Hour)
	defer bl.Close()

	handler := HandleBedrockWidget(cfg, c, bl)

	t.Run("Valid address (offline Bedrock server)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/widget/bedrock/127.0.0.1:19199?dark=true&rounded=true&transparent=true&timeout=0.1", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
			t.Fatalf("expected Content-Type image/png, got %s", ct)
		}

		body := w.Body.Bytes()
		img, err := png.Decode(bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed to decode returned PNG: %v", err)
		}

		if img.Bounds().Dx() != 860 || img.Bounds().Dy() != 240 {
			t.Fatalf("unexpected dimensions: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
		}
	})

	t.Run("Invalid address", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/widget/bedrock/invalid::host:123", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
		if body := w.Body.String(); body != "Invalid address value" {
			t.Fatalf("expected 'Invalid address value', got %q", body)
		}
	})

	t.Run("ETag 304 Not Modified", func(t *testing.T) {
		req1 := httptest.NewRequest(http.MethodGet, "/v2/widget/bedrock/127.0.0.1:19199?timeout=0.1", nil)
		w1 := httptest.NewRecorder()
		handler(w1, req1)

		etag := w1.Header().Get("ETag")
		if etag == "" {
			t.Fatal("expected ETag header")
		}

		req2 := httptest.NewRequest(http.MethodGet, "/v2/widget/bedrock/127.0.0.1:19199?timeout=0.1", nil)
		req2.Header.Set("If-None-Match", etag)
		w2 := httptest.NewRecorder()
		handler(w2, req2)

		if w2.Code != http.StatusNotModified {
			t.Fatalf("expected 304 Not Modified, got %d", w2.Code)
		}
	})
}
