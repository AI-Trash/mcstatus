package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"github.com/mcstatus-io/mcutil/v4/options"
	"github.com/mcstatus-io/mcutil/v4/vote"

	"mcstatus/internal/config"
	"mcstatus/internal/middleware"
	"mcstatus/internal/util"
)

// HandleVote handles POST /v2/vote requests.
func HandleVote(cfg *config.Config, filter ...*middleware.IPFilter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method Not Allowed"))
			return
		}

		var jsonMap map[string]any
		if r.Body != nil && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			_ = json.NewDecoder(io.LimitReader(r.Body, 64*1024)).Decode(&jsonMap)
		}

		getVal := func(keys ...string) string {
			for _, k := range keys {
				if v := r.URL.Query().Get(k); v != "" {
					return strings.TrimSpace(v)
				}
				if v := r.FormValue(k); v != "" {
					return strings.TrimSpace(v)
				}
				if jsonMap != nil {
					if val, ok := jsonMap[k]; ok && val != nil {
						s := strings.TrimSpace(fmt.Sprint(val))
						if s != "" {
							return s
						}
					}
				}
			}
			return ""
		}

		host := getVal("host")
		username := getVal("username")
		if host == "" || username == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing required parameter"))
			return
		}

		token := getVal("token")
		publicKey := getVal("publickey", "publicKey")
		if token == "" && publicKey == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing token or publickey"))
			return
		}

		var port uint16 = 8192
		if portStr := getVal("port"); portStr != "" {
			p, err := util.ParseUint16(portStr, 8192)
			if err != nil || p == 0 {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid port value"))
				return
			}
			port = p
		}

		timeout := 5 * time.Second
		if cfg != nil && cfg.DefaultTimeout > 0 {
			timeout = cfg.DefaultTimeout
		}
		if timeoutStr := getVal("timeout"); timeoutStr != "" {
			tVal, err := strconv.ParseFloat(timeoutStr, 64)
			if err != nil || tVal <= 0 {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid timeout value"))
				return
			}
			maxSec := 15.0
			if cfg != nil && cfg.MaxTimeout > 0 {
				maxSec = cfg.MaxTimeout.Seconds()
			}
			if tVal > maxSec {
				tVal = maxSec
			}
			timeout = time.Duration(tVal * float64(time.Second))
		}

		serviceName := getVal("serviceName", "servicename")
		if serviceName == "" {
			serviceName = "mcstatus.io"
		}

		timestamp := time.Now()
		if tsStr := getVal("timestamp"); tsStr != "" {
			parsed, err := time.Parse(time.RFC3339, tsStr)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid timestamp value"))
				return
			}
			timestamp = parsed
		}

		ip := getVal("ip")
		if ip == "" {
			var f *middleware.IPFilter
			if len(filter) > 0 {
				f = filter[0]
			}
			ip = middleware.ClientIP(r, f)
		}

		uuid := getVal("uuid")

		voteOpts := options.Vote{
			Timeout:     timeout,
			Username:    username,
			ServiceName: serviceName,
			Timestamp:   timestamp,
			Token:       token,
			PublicKey:   publicKey,
			IPAddress:   ip,
			UUID:        uuid,
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		if err := vote.SendVote(ctx, host, port, voteOpts); err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The vote was successfully sent to the server"))
	}
}
