package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/middleware"
	"mcstatus/internal/util"
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

	return util.ParseTimeout(r.URL.Query().Get("timeout"), defaultSec, maxSec)
}

// ParseBool parses a boolean query parameter with fallback default.
func ParseBool(val string, defaultVal bool) bool {
	return util.ParseBool(val, defaultVal)
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
