package handlers

import (
	"context"
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
	"mcstatus/internal/util"
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
	if decoded, err := util.DecodeBase64Bytes(favicon); err == nil && len(decoded) > 0 {
		return decoded
	}
	return assets.DefaultIconBytes
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
		format := util.ResolveImageFormat(r)
		if address == "" {
			data, mime, err := util.ConvertImageBytes(assets.DefaultIconBytes, format)
			if err != nil {
				data, mime = assets.DefaultIconBytes, "image/png"
			}
			w.Header().Set("Content-Type", mime)
			w.WriteHeader(http.StatusOK)
			w.Write(data)
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
		cacheKey := fmt.Sprintf("icon:%s:%d:%s", strings.ToLower(host), port, format)
		ServeCached(w, r, c, cacheKey, ttl, func() ([]byte, string, error) {
			iconBytes := fetchIcon(host, port, timeout)
			return util.ConvertImageBytes(iconBytes, format)
		})
	}
}
