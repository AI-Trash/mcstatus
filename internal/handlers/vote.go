package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mcstatus-io/mcutil/v4/options"
	"github.com/mcstatus-io/mcutil/v4/vote"

	"mcstatus/internal/config"
)

type voteRequestBody struct {
	Host        string   `json:"host"`
	Port        *uint16  `json:"port"`
	Timeout     *float64 `json:"timeout"`
	Username    string   `json:"username"`
	UUID        string   `json:"uuid"`
	ServiceName string   `json:"serviceName"`
	Timestamp   string   `json:"timestamp"`
	Token       string   `json:"token"`
	PublicKey   string   `json:"publickey"`
	IP          string   `json:"ip"`
}

// HandleVote handles POST /v2/vote requests.
func HandleVote(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method Not Allowed"))
			return
		}

		var jsonBody voteRequestBody
		if r.Body != nil && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			_ = json.NewDecoder(io.LimitReader(r.Body, 64*1024)).Decode(&jsonBody)
		}

		// 1. host
		host := r.URL.Query().Get("host")
		if host == "" {
			host = r.FormValue("host")
		}
		if host == "" {
			host = jsonBody.Host
		}
		host = strings.TrimSpace(host)

		// 2. username
		username := r.URL.Query().Get("username")
		if username == "" {
			username = r.FormValue("username")
		}
		if username == "" {
			username = jsonBody.Username
		}
		username = strings.TrimSpace(username)

		if host == "" || username == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing required parameter"))
			return
		}

		// 3. token and publickey
		token := r.URL.Query().Get("token")
		if token == "" {
			token = r.FormValue("token")
		}
		if token == "" {
			token = jsonBody.Token
		}
		token = strings.TrimSpace(token)

		publicKey := r.URL.Query().Get("publickey")
		if publicKey == "" {
			publicKey = r.FormValue("publickey")
		}
		if publicKey == "" {
			publicKey = jsonBody.PublicKey
		}
		publicKey = strings.TrimSpace(publicKey)

		if token == "" && publicKey == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing token or publickey"))
			return
		}

		// 4. port (default 8192)
		var port uint16 = 8192
		portStr := r.URL.Query().Get("port")
		if portStr == "" {
			portStr = r.FormValue("port")
		}
		portStr = strings.TrimSpace(portStr)
		if portStr != "" {
			p, err := strconv.ParseUint(portStr, 10, 16)
			if err != nil || p == 0 {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid port value"))
				return
			}
			port = uint16(p)
		} else if jsonBody.Port != nil && *jsonBody.Port > 0 {
			port = *jsonBody.Port
		}

		// 5. timeout (float64 seconds, default 5.0)
		timeoutSec := 5.0
		if cfg != nil && cfg.DefaultTimeout > 0 {
			timeoutSec = cfg.DefaultTimeout.Seconds()
		}
		timeoutDuration := time.Duration(timeoutSec * float64(time.Second))

		timeoutStr := r.URL.Query().Get("timeout")
		if timeoutStr == "" {
			timeoutStr = r.FormValue("timeout")
		}
		timeoutStr = strings.TrimSpace(timeoutStr)
		if timeoutStr != "" {
			t, err := strconv.ParseFloat(timeoutStr, 64)
			if err != nil || t <= 0 {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid timeout value"))
				return
			}
			timeoutDuration = time.Duration(t * float64(time.Second))
		} else if jsonBody.Timeout != nil && *jsonBody.Timeout > 0 {
			timeoutDuration = time.Duration(*jsonBody.Timeout * float64(time.Second))
		}

		if cfg != nil && cfg.MaxTimeout > 0 && timeoutDuration > cfg.MaxTimeout {
			timeoutDuration = cfg.MaxTimeout
		}

		// 6. uuid (optional)
		uuid := r.URL.Query().Get("uuid")
		if uuid == "" {
			uuid = r.FormValue("uuid")
		}
		if uuid == "" {
			uuid = jsonBody.UUID
		}
		uuid = strings.TrimSpace(uuid)

		// 7. serviceName (default "mcstatus.io")
		serviceName := r.URL.Query().Get("serviceName")
		if serviceName == "" {
			serviceName = r.FormValue("serviceName")
		}
		if serviceName == "" {
			serviceName = jsonBody.ServiceName
		}
		serviceName = strings.TrimSpace(serviceName)
		if serviceName == "" {
			serviceName = "mcstatus.io"
		}

		// 8. timestamp (RFC3339 string, default now)
		timestamp := time.Now()
		timestampStr := r.URL.Query().Get("timestamp")
		if timestampStr == "" {
			timestampStr = r.FormValue("timestamp")
		}
		if timestampStr == "" {
			timestampStr = jsonBody.Timestamp
		}
		timestampStr = strings.TrimSpace(timestampStr)
		if timestampStr != "" {
			t, err := time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid timestamp value"))
				return
			}
			timestamp = t
		}

		// 9. ip (optional, default remote IP)
		ip := r.URL.Query().Get("ip")
		if ip == "" {
			ip = r.FormValue("ip")
		}
		if ip == "" {
			ip = jsonBody.IP
		}
		ip = strings.TrimSpace(ip)
		if ip == "" {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.Split(xff, ",")
				ip = strings.TrimSpace(parts[0])
			} else if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
				ip = strings.TrimSpace(xrip)
			} else {
				h, _, err := net.SplitHostPort(r.RemoteAddr)
				if err == nil {
					ip = h
				} else {
					ip = r.RemoteAddr
				}
			}
		}
		if ip == "" {
			ip = "127.0.0.1"
		}

		// 10. Send vote
		ctx, cancel := context.WithTimeout(r.Context(), timeoutDuration)
		defer cancel()

		opts := options.Vote{
			PublicKey:   publicKey,
			ServiceName: serviceName,
			Username:    username,
			Token:       token,
			UUID:        uuid,
			IPAddress:   ip,
			Timestamp:   timestamp,
			Timeout:     timeoutDuration,
		}

		err := vote.SendVote(ctx, host, port, opts)
		if err != nil {
			statusCode := http.StatusBadRequest
			if errors.Is(err, context.Canceled) {
				statusCode = http.StatusInternalServerError
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(statusCode)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The vote was successfully sent to the server"))
	}
}
