package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mcstatus/internal/config"
)

func TestHandleVoteMissingRequiredParameter(t *testing.T) {
	cfg := config.Load()
	handler := HandleVote(cfg)

	// Missing host
	{
		req := httptest.NewRequest(http.MethodPost, "/v2/vote?username=notch&token=abc", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "Missing required parameter" {
			t.Fatalf("expected 'Missing required parameter', got %q", string(body))
		}
	}

	// Missing username
	{
		req := httptest.NewRequest(http.MethodPost, "/v2/vote?host=127.0.0.1&token=abc", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "Missing required parameter" {
			t.Fatalf("expected 'Missing required parameter', got %q", string(body))
		}
	}
}

func TestHandleVoteMissingTokenOrPublicKey(t *testing.T) {
	cfg := config.Load()
	handler := HandleVote(cfg)

	req := httptest.NewRequest(http.MethodPost, "/v2/vote?host=127.0.0.1&username=notch", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Missing token or publickey" {
		t.Fatalf("expected 'Missing token or publickey', got %q", string(body))
	}
}

func TestHandleVoteInvalidPort(t *testing.T) {
	cfg := config.Load()
	handler := HandleVote(cfg)

	cases := []string{"invalid", "0", "99999"}
	for _, port := range cases {
		req := httptest.NewRequest(http.MethodPost, "/v2/vote?host=127.0.0.1&username=notch&token=abc&port="+port, nil)
		w := httptest.NewRecorder()
		handler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected status 400 for port %s, got %d", port, resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "Invalid port value" {
			t.Fatalf("expected 'Invalid port value', got %q", string(body))
		}
	}
}

func TestHandleVoteInvalidTimeout(t *testing.T) {
	cfg := config.Load()
	handler := HandleVote(cfg)

	cases := []string{"invalid", "-1", "0"}
	for _, to := range cases {
		req := httptest.NewRequest(http.MethodPost, "/v2/vote?host=127.0.0.1&username=notch&token=abc&timeout="+to, nil)
		w := httptest.NewRecorder()
		handler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected status 400 for timeout %s, got %d", to, resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "Invalid timeout value" {
			t.Fatalf("expected 'Invalid timeout value', got %q", string(body))
		}
	}
}

func TestHandleVoteInvalidTimestamp(t *testing.T) {
	cfg := config.Load()
	handler := HandleVote(cfg)

	req := httptest.NewRequest(http.MethodPost, "/v2/vote?host=127.0.0.1&username=notch&token=abc&timestamp=invalid-timestamp", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Invalid timestamp value" {
		t.Fatalf("expected 'Invalid timestamp value', got %q", string(body))
	}
}

func TestHandleVoteMethodNotAllowed(t *testing.T) {
	cfg := config.Load()
	handler := HandleVote(cfg)

	req := httptest.NewRequest(http.MethodGet, "/v2/vote?host=127.0.0.1&username=notch&token=abc", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleVoteOfflineServer(t *testing.T) {
	cfg := config.Load()
	cfg.DefaultTimeout = 50 * time.Millisecond
	handler := HandleVote(cfg)

	req := httptest.NewRequest(http.MethodPost, "/v2/vote?host=127.0.0.1&port=59996&username=notch&token=abc&timeout=0.05", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected error status (400 or 500), got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Fatalf("expected non-empty error body")
	}
}

func TestHandleVoteVotifier2MockServer(t *testing.T) {
	// Start mock Votifier 2 TCP server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock listener: %v", err)
	}
	defer ln.Close()

	port := uint16(ln.Addr().(*net.TCPAddr).Port)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// 1. Send handshake
		_, _ = conn.Write([]byte("VOTIFIER 2 testchallenge\n"))

		// 2. Read vote message
		r := bufio.NewReader(conn)
		// Header: 2 bytes magic + 2 bytes length
		header := make([]byte, 4)
		if _, err := io.ReadFull(r, header); err != nil {
			return
		}
		msgLen := int(header[2])<<8 | int(header[3])
		msgBuf := make([]byte, msgLen)
		if _, err := io.ReadFull(r, msgBuf); err != nil {
			return
		}

		// 3. Send response packet
		respJSON, _ := json.Marshal(map[string]string{"status": "ok"})
		_, _ = conn.Write(append(respJSON, '\n'))
	}()

	cfg := config.Load()
	handler := HandleVote(cfg)

	url := fmt.Sprintf("/v2/vote?host=127.0.0.1&port=%d&username=notch&token=secrettoken&timeout=2.0", port)
	req := httptest.NewRequest(http.MethodPost, url, nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d with body: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "The vote was successfully sent to the server" {
		t.Fatalf("expected 'The vote was successfully sent to the server', got %q", string(body))
	}
}

func TestHandleVoteJSONBody(t *testing.T) {
	// Start mock Votifier 2 TCP server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock listener: %v", err)
	}
	defer ln.Close()

	port := uint16(ln.Addr().(*net.TCPAddr).Port)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		_, _ = conn.Write([]byte("VOTIFIER 2 testchallenge\n"))

		r := bufio.NewReader(conn)
		header := make([]byte, 4)
		if _, err := io.ReadFull(r, header); err != nil {
			return
		}
		msgLen := int(header[2])<<8 | int(header[3])
		msgBuf := make([]byte, msgLen)
		if _, err := io.ReadFull(r, msgBuf); err != nil {
			return
		}

		respJSON, _ := json.Marshal(map[string]string{"status": "ok"})
		_, _ = conn.Write(append(respJSON, '\n'))
	}()

	cfg := config.Load()
	handler := HandleVote(cfg)

	payload := map[string]any{
		"host":     "127.0.0.1",
		"port":     port,
		"username": "Steve",
		"token":    "mytoken",
		"timeout":  2.0,
	}
	data, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v2/vote", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d with body: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	if strings.TrimSpace(string(body)) != "The vote was successfully sent to the server" {
		t.Fatalf("expected 'The vote was successfully sent to the server', got %q", string(body))
	}
}
