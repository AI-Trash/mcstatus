package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mcstatus/internal/config"
)

func TestServerHealthAndRoot(t *testing.T) {
	cfg := &config.Config{
		Host:           "127.0.0.1",
		Port:           0,
		CacheTTL:       time.Minute,
		DefaultTimeout: 2 * time.Second,
		MaxTimeout:     5 * time.Second,
	}
	srv := New(cfg)
	handler := srv.Handler()

	// 1. Test /health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected /health to return 200, got %d", rec.Code)
	}

	var healthResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if healthResp["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", healthResp["status"])
	}

	// 2. Test CORS headers
	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin *, got %s", origin)
	}

	// 3. Test root endpoint
	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRec := httptest.NewRecorder()
	handler.ServeHTTP(rootRec, rootReq)

	if rootRec.Code != http.StatusOK {
		t.Fatalf("expected / to return 200, got %d", rootRec.Code)
	}

	var rootResp map[string]any
	if err := json.Unmarshal(rootRec.Body.Bytes(), &rootResp); err != nil {
		t.Fatalf("failed to decode root response: %v", err)
	}
	if rootResp["version"] == nil || rootResp["version"] == "" {
		t.Fatalf("expected non-empty version in root response, got %v", rootResp["version"])
	}
}

func TestServerCORSPreflight(t *testing.T) {
	cfg := &config.Config{
		Host:     "127.0.0.1",
		Port:     0,
		CacheTTL: time.Minute,
	}
	srv := New(cfg)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodOptions, "/v2/status/java/demo.mcstatus.io", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected OPTIONS preflight to return 200, got %d", rec.Code)
	}
	if methods := rec.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Fatalf("expected non-empty Access-Control-Allow-Methods")
	}
}

func TestServerDefaultIcon(t *testing.T) {
	cfg := &config.Config{
		Host:     "127.0.0.1",
		Port:     0,
		CacheTTL: time.Minute,
	}
	srv := New(cfg)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/v2/icon", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected /v2/icon to return 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("expected Content-Type image/png, got %s", ct)
	}
	if rec.Body.Len() != 3143 {
		t.Fatalf("expected default icon length 3143, got %d", rec.Body.Len())
	}
}
