package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mcstatus-io/mcutil/v4/options"
	"github.com/mcstatus-io/mcutil/v4/status"

	"mcstatus/internal/assets"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/middleware"
	"mcstatus/internal/resolver"
)

// getIconAddress extracts the address parameter from PathValue, URL path prefixes, or query string.
func getIconAddress(r *http.Request) string {
	if addr := r.PathValue("address"); addr != "" {
		if unescaped, err := url.PathUnescape(addr); err == nil && unescaped != "" {
			return unescaped
		}
		return addr
	}

	p := r.URL.Path
	for _, prefix := range []string{"/v2/icon", "/icon"} {
		if strings.HasPrefix(p, prefix) {
			p = strings.TrimPrefix(p, prefix)
			p = strings.TrimPrefix(p, "/")
			if idx := strings.Index(p, "/"); idx != -1 {
				p = p[:idx]
			}
			if unescaped, err := url.PathUnescape(p); err == nil && unescaped != "" {
				p = unescaped
			}
			return p
		}
	}

	return r.URL.Query().Get("address")
}

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

		address := getIconAddress(r)
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

		timeout := parseTimeout(r, cfg)
		ttl := 60 * time.Second
		if cfg != nil && cfg.CacheTTL > 0 {
			ttl = cfg.CacheTTL
		}

		cacheKey := fmt.Sprintf("icon:%s", address)

		if c == nil {
			iconBytes := fetchIcon(host, port, timeout)
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			w.Write(iconBytes)
			return
		}

		entry, hit, err := c.GetOrCompute(cacheKey, func() ([]byte, string, error) {
			iconBytes := fetchIcon(host, port, timeout)
			return iconBytes, "image/png", nil
		}, ttl)

		if err != nil {
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			w.Write(assets.DefaultIconBytes)
			return
		}

		w.Header().Set("Content-Type", entry.ContentType)
		middleware.SetCacheHeaders(w, entry.ETag, hit, entry.TimeRemaining())

		if middleware.CheckETag(entry.ETag, r) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(entry.Data)
	}
}
