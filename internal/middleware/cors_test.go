package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSFullCompliance(t *testing.T) {
	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	corsHandler := CORS(dummy)

	// 1. Browser request with Origin (should reflect origin and allow credentials)
	req1 := httptest.NewRequest(http.MethodGet, "/v2/status/java/demo", nil)
	req1.Header.Set("Origin", "https://frontend.example.com")
	rec1 := httptest.NewRecorder()
	corsHandler.ServeHTTP(rec1, req1)

	if got := rec1.Header().Get("Access-Control-Allow-Origin"); got != "https://frontend.example.com" {
		t.Fatalf("expected origin reflection 'https://frontend.example.com', got '%s'", got)
	}
	if got := rec1.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials true, got '%s'", got)
	}

	// 2. Preflight with custom headers and Private Network Access
	req2 := httptest.NewRequest(http.MethodOptions, "/v2/status/java/demo", nil)
	req2.Header.Set("Origin", "http://localhost:3000")
	req2.Header.Set("Access-Control-Request-Method", "GET")
	req2.Header.Set("Access-Control-Request-Headers", "authorization, content-type, sentry-trace")
	req2.Header.Set("Access-Control-Request-Private-Network", "true")
	rec2 := httptest.NewRecorder()
	corsHandler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected preflight 200, got %d", rec2.Code)
	}
	if got := rec2.Header().Get("Access-Control-Allow-Headers"); got != "authorization, content-type, sentry-trace" {
		t.Fatalf("expected mirrored headers, got '%s'", got)
	}
	if got := rec2.Header().Get("Access-Control-Allow-Private-Network"); got != "true" {
		t.Fatalf("expected PNA true, got '%s'", got)
	}

	// 3. Curl / non-browser request without Origin
	req3 := httptest.NewRequest(http.MethodGet, "/v2/status/java/demo", nil)
	rec3 := httptest.NewRecorder()
	corsHandler.ServeHTTP(rec3, req3)

	if got := rec3.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard origin '*', got '%s'", got)
	}
}
