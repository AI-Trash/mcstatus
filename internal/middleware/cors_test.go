package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSStandardWildcard(t *testing.T) {
	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	corsHandler := CORS(dummy)

	// 1. Regular GET request
	req1 := httptest.NewRequest(http.MethodGet, "/v2/status/java/demo", nil)
	rec1 := httptest.NewRecorder()
	corsHandler.ServeHTTP(rec1, req1)

	if got := rec1.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard origin '*', got '%s'", got)
	}

	// 2. Preflight OPTIONS request
	req2 := httptest.NewRequest(http.MethodOptions, "/v2/status/java/demo", nil)
	req2.Header.Set("Access-Control-Request-Method", "GET")
	rec2 := httptest.NewRecorder()
	corsHandler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected preflight 200, got %d", rec2.Code)
	}
	if got := rec2.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected '*', got '%s'", got)
	}
	if got := rec2.Header().Get("Access-Control-Allow-Headers"); got != "*" {
		t.Fatalf("expected '*', got '%s'", got)
	}
	if got := rec2.Header().Get("Access-Control-Allow-Private-Network"); got != "true" {
		t.Fatalf("expected PNA true, got '%s'", got)
	}
}
