package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/middleware"
)

// ExtractAddress retrieves the address parameter from PathValue, URL path prefixes, or query string.
func ExtractAddress(r *http.Request, defaultPrefixes ...string) string {
	if addr := r.PathValue("address"); addr != "" {
		if unescaped, err := url.PathUnescape(addr); err == nil && unescaped != "" {
			return unescaped
		}
		return addr
	}

	p := r.URL.Path
	for _, prefix := range defaultPrefixes {
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
		if idx := strings.Index(p, prefix); idx != -1 {
			p = p[idx+len(prefix):]
			p = strings.Trim(p, "/")
			if idx := strings.Index(p, "/"); idx != -1 {
				p = p[:idx]
			}
			if unescaped, err := url.PathUnescape(p); err == nil && unescaped != "" {
				p = unescaped
			}
			if p != "" {
				return p
			}
		}
	}

	return r.URL.Query().Get("address")
}

// ParseTimeout extracts timeout in seconds from query params, clamping to [0, maxTimeout].
func ParseTimeout(r *http.Request, cfg *config.Config) time.Duration {
	defaultSec := 5.0
	maxSec := 15.0
	if cfg != nil {
		if cfg.DefaultTimeout > 0 {
			defaultSec = cfg.DefaultTimeout.Seconds()
		}
		if cfg.MaxTimeout > 0 {
			maxSec = cfg.MaxTimeout.Seconds()
		}
	}

	timeoutSec := defaultSec
	if tStr := r.URL.Query().Get("timeout"); tStr != "" {
		if tVal, err := strconv.ParseFloat(tStr, 64); err == nil && tVal > 0 {
			timeoutSec = tVal
		}
	}

	if timeoutSec > maxSec {
		timeoutSec = maxSec
	}

	return time.Duration(timeoutSec * float64(time.Second))
}

// ParseBool parses a boolean query parameter with fallback default.
func ParseBool(val string, defaultVal bool) bool {
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}

// ResolveImageFormat returns the format ("avif" or "png") and mime type based on the query parameter.
// Defaults to PNG to prevent Cloudflare CDN cache poisoning on URLs without explicit format parameters.
func ResolveImageFormat(r *http.Request) (string, string) {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "avif" {
		return "avif", "image/avif"
	}
	return "png", "image/png"
}

// ServeCached executes or retrieves cached response, injecting standard Cloudflare CDN headers, ETag 304, and HEAD handling.
func ServeCached(w http.ResponseWriter, r *http.Request, c *cache.Cache, cacheKey string, ttl time.Duration, compute func() ([]byte, string, error)) {
	if c == nil {
		data, contentType, err := compute()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", contentType)
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	entry, hit, err := c.GetOrCompute(cacheKey, compute, ttl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", entry.ContentType)
	if strings.HasPrefix(entry.ContentType, "image/") {
		w.Header().Add("Vary", "Accept")
	}
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
