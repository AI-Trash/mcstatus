package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// CORS adds standard CORS headers to all responses and handles OPTIONS preflights.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "HEAD,OPTIONS,GET,POST,PUT,DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "ETag, X-Cache-Hit, X-Cache-Time-Remaining")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Recovery catches any panics and logs them cleanly.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec, "url", r.URL.String())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Logging logs HTTP requests with timing, status code, and real client IP.
func Logging(filter *IPFilter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			clientIP := ClientIP(r, filter)
			slog.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration", duration.String(),
				"remote", clientIP,
			)
		})
	}
}

// CheckETag checks If-None-Match header and returns 304 Not Modified if matched.
func CheckETag(etag string, r *http.Request) bool {
	ifNoneMatch := r.Header.Get("If-None-Match")
	if ifNoneMatch == "" {
		return false
	}
	for _, match := range strings.Split(ifNoneMatch, ",") {
		match = strings.TrimSpace(match)
		if match == "*" || match == etag || strings.Trim(match, `W/`) == strings.Trim(etag, `W/`) {
			return true
		}
	}
	return false
}

// SetCacheHeaders sets standard caching headers compatible with Cloudflare CDN and in-memory cache metadata.
func SetCacheHeaders(w http.ResponseWriter, etag string, hit bool, remaining int) {
	if remaining < 0 {
		remaining = 0
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("X-Cache-Hit", strconv.FormatBool(hit))
	w.Header().Set("X-Cache-Time-Remaining", strconv.Itoa(remaining))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", remaining))
	w.Header().Set("CDN-Cache-Control", fmt.Sprintf("public, max-age=%d", remaining))
	w.Header().Set("Cloudflare-CDN-Cache-Control", fmt.Sprintf("public, max-age=%d", remaining))
	w.Header().Set("Vary", "Accept-Encoding")
}
