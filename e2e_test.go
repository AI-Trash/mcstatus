package main

import (
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mcstatus/internal/config"
	"mcstatus/internal/server"
	"mcstatus/internal/types"
)

func setupTestServer() *httptest.Server {
	cfg := &config.Config{
		Host:                 "127.0.0.1",
		Port:                 0,
		CacheTTL:             60 * time.Second,
		DefaultTimeout:       100 * time.Millisecond,
		MaxTimeout:           500 * time.Millisecond,
		MojangBlockedRefresh: 0,
	}
	srv := server.New(cfg)
	return httptest.NewServer(srv.Handler())
}

func TestE2ERootAndHealth(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// 1. Test /
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("failed to GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// 2. Test /health
	hResp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("failed to GET /health: %v", err)
	}
	defer hResp.Body.Close()
	if hResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", hResp.StatusCode)
	}
}

func TestE2EJavaStatusOffline(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// Request an offline local port
	url := ts.URL + "/v2/status/java/127.0.0.1:9999?timeout=0.1"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET java status error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var status types.JavaStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v, body: %s", err, string(body))
	}

	if status.Online {
		t.Fatalf("expected server to be offline")
	}
	if status.Host != "127.0.0.1" || status.Port != 9999 {
		t.Fatalf("expected 127.0.0.1:9999, got %s:%d", status.Host, status.Port)
	}
	if status.RetrievedAt == 0 || status.ExpiresAt == 0 {
		t.Fatalf("expected timestamps to be non-zero")
	}

	// Verify headers for Cloudflare and Cache
	if resp.Header.Get("X-Cache-Hit") != "false" {
		t.Fatalf("expected cache miss on first request")
	}
	if !strings.Contains(resp.Header.Get("Cache-Control"), "public, max-age=") {
		t.Fatalf("expected Cache-Control header, got %s", resp.Header.Get("Cache-Control"))
	}
	if resp.Header.Get("CDN-Cache-Control") == "" {
		t.Fatalf("expected CDN-Cache-Control header")
	}
	if resp.Header.Get("Cloudflare-CDN-Cache-Control") == "" {
		t.Fatalf("expected Cloudflare-CDN-Cache-Control header")
	}
	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header")
	}

	// 2nd request should hit cache
	resp2, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET java status 2 error: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.Header.Get("X-Cache-Hit") != "true" {
		t.Fatalf("expected cache hit on second request")
	}

	// 3rd request with If-None-Match should return 304 Not Modified
	req3, _ := http.NewRequest(http.MethodGet, url, nil)
	req3.Header.Set("If-None-Match", etag)
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("conditional GET error: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusNotModified {
		t.Fatalf("expected 304 Not Modified, got %d", resp3.StatusCode)
	}

	// 4th request with HEAD
	respHead, err := http.Head(url)
	if err != nil {
		t.Fatalf("HEAD request error: %v", err)
	}
	defer respHead.Body.Close()
	if respHead.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for HEAD, got %d", respHead.StatusCode)
	}
	headBody, _ := io.ReadAll(respHead.Body)
	if len(headBody) > 0 {
		t.Fatalf("expected empty body for HEAD, got %d bytes", len(headBody))
	}
}

func TestE2EBedrockStatusOffline(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	url := ts.URL + "/v2/status/bedrock/127.0.0.1:19133?timeout=0.1"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET bedrock status error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var status types.BedrockStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v, body: %s", err, string(body))
	}

	if status.Online {
		t.Fatalf("expected bedrock server to be offline")
	}
	if status.Host != "127.0.0.1" || status.Port != 19133 {
		t.Fatalf("expected 127.0.0.1:19133, got %s:%d", status.Host, status.Port)
	}
}

func TestE2EInvalidAddress(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	urls := []string{
		ts.URL + "/v2/status/java/invalid!address",
		ts.URL + "/v2/status/bedrock/invalid!address",
		ts.URL + "/v2/icon/invalid!address",
		ts.URL + "/v2/widget/java/invalid!address",
		ts.URL + "/v2/widget/bedrock/invalid!address",
	}

	for _, u := range urls {
		resp, err := http.Get(u)
		if err != nil {
			t.Fatalf("request %s error: %v", u, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", u, resp.StatusCode)
		}
		if string(body) != "Invalid address value" {
			t.Fatalf("expected 'Invalid address value', got %s", string(body))
		}
	}
}

func TestE2EIconEndpoints(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// Default icon
	resp, err := http.Get(ts.URL + "/v2/icon")
	if err != nil {
		t.Fatalf("GET /v2/icon error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("expected image/png, got %s", resp.Header.Get("Content-Type"))
	}
	img, err := png.Decode(resp.Body)
	if err != nil {
		t.Fatalf("expected valid PNG: %v", err)
	}
	if img.Bounds().Dx() != 64 || img.Bounds().Dy() != 64 {
		t.Fatalf("expected 64x64 icon, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestE2EWidgetEndpoints(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// Java widget for offline server
	jUrl := ts.URL + "/v2/widget/java/127.0.0.1:9999?timeout=0.1"
	jResp, err := http.Get(jUrl)
	if err != nil {
		t.Fatalf("GET java widget error: %v", err)
	}
	defer jResp.Body.Close()

	if jResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", jResp.StatusCode)
	}
	if jResp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("expected image/png, got %s", jResp.Header.Get("Content-Type"))
	}
	jImg, err := png.Decode(jResp.Body)
	if err != nil {
		t.Fatalf("expected valid PNG: %v", err)
	}
	if jImg.Bounds().Dx() != 860 || jImg.Bounds().Dy() != 240 {
		t.Fatalf("expected 860x240 widget, got %dx%d", jImg.Bounds().Dx(), jImg.Bounds().Dy())
	}

	// Bedrock widget for offline server
	bUrl := ts.URL + "/v2/widget/bedrock/127.0.0.1:19133?timeout=0.1"
	bResp, err := http.Get(bUrl)
	if err != nil {
		t.Fatalf("GET bedrock widget error: %v", err)
	}
	defer bResp.Body.Close()

	if bResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", bResp.StatusCode)
	}
	bImg, err := png.Decode(bResp.Body)
	if err != nil {
		t.Fatalf("expected valid PNG: %v", err)
	}
	if bImg.Bounds().Dx() != 860 || bImg.Bounds().Dy() != 240 {
		t.Fatalf("expected 860x240 widget, got %dx%d", bImg.Bounds().Dx(), bImg.Bounds().Dy())
	}
}

func TestE2EVoteMissingParams(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/v2/vote", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /v2/vote error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing params, got %d", resp.StatusCode)
	}
}
