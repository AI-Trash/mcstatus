package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mcstatus-io/mcutil/v4/options"
	"github.com/mcstatus-io/mcutil/v4/status"

	"mcstatus/internal/assets"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/resolver"
)

// fetchIcon queries the Java server status fast without full query and returns PNG bytes.
func fetchIcon(host string, port uint16, timeout time.Duration) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	modernOpts := options.StatusModern{
		EnableSRV:         true,
		Timeout:           timeout,
		ProtocolVersion:   -1,
		ReceiveLimitBytes: 1 << 20,
		Ping:              false,
	}

	resp, err := status.Modern(ctx, host, port, modernOpts)
	if err != nil || resp == nil || resp.Favicon == nil {
		return assets.DefaultIconBytes
	}

	favicon := *resp.Favicon
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(favicon, prefix) {
		return assets.DefaultIconBytes
	}

	payload := strings.TrimPrefix(favicon, prefix)
	payload = strings.TrimSpace(payload)
	decoded, decodeErr := base64.StdEncoding.DecodeString(payload)
	if decodeErr != nil {
		decoded, decodeErr = base64.RawStdEncoding.DecodeString(payload)
	}
	if decodeErr != nil || len(decoded) == 0 {
		return assets.DefaultIconBytes
	}

	return decoded
}

// HandleIcon handles GET /v2/icon and /v2/icon/{address} requests.
func HandleIcon(cfg *config.Config, c *cache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method Not Allowed"))
			return
		}

		address := ExtractAddress(r, "/v2/icon", "/icon")
		if address == "" {
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			w.Write(assets.DefaultIconBytes)
			return
		}

		host, port, err := resolver.ParseAddress(address, 25565)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid address value"))
			return
		}

		timeout := ParseTimeout(r, cfg)
		ttl := 60 * time.Second
		if cfg != nil && cfg.CacheTTL > 0 {
			ttl = cfg.CacheTTL
		}

		cacheKey := fmt.Sprintf("icon:%s:%d", strings.ToLower(host), port)
		ServeCached(w, r, c, cacheKey, ttl, func() ([]byte, string, error) {
			iconBytes := fetchIcon(host, port, timeout)
			return iconBytes, "image/png", nil
		})
	}
}
