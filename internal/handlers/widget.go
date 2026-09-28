package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"strconv"
	"strings"

	"mcstatus/internal/assets"
	"mcstatus/internal/blocklist"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/middleware"
	"mcstatus/internal/resolver"
	"mcstatus/internal/widget"
)

// parseWidgetBool parses boolean query parameters with fallback default value.
func parseWidgetBool(val string, defaultVal bool) bool {
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}

// decodeBase64Icon decodes a base64-encoded PNG data URI or raw base64 string.
// If decoding fails or iconStr is nil/empty, it returns assets.DefaultIcon.
func decodeBase64Icon(iconStr *string) image.Image {
	if iconStr == nil || *iconStr == "" {
		return assets.DefaultIcon
	}
	s := *iconStr
	if idx := strings.Index(s, ","); idx != -1 {
		s = s[idx+1:]
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return assets.DefaultIcon
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return assets.DefaultIcon
	}
	return img
}

// HandleJavaWidget handles HTTP requests to generate widget image for Java Minecraft servers.
func HandleJavaWidget(cfg *config.Config, c *cache.Cache, bl *blocklist.BlockList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := extractAddress(r, "/v2/widget/java/", "/widget/java/")
		address = strings.TrimSpace(address)
		if address == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid address value"))
			return
		}

		host, port, err := resolver.ParseAddress(address, 25565)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid address value"))
			return
		}

		dark := parseWidgetBool(r.URL.Query().Get("dark"), true)
		rounded := parseWidgetBool(r.URL.Query().Get("rounded"), true)
		transparent := parseWidgetBool(r.URL.Query().Get("transparent"), false)
		timeout := parseTimeout(r, cfg)

		resp, hit, _ := GetJavaStatus(r.Context(), cfg, c, bl, host, port, true, timeout)

		widgetData := &widget.WidgetData{
			Online:      false,
			Host:        host,
			Port:        port,
			Edition:     "Java Edition",
			Icon:        assets.DefaultIcon,
			Dark:        dark,
			Rounded:     rounded,
			Transparent: transparent,
		}

		if resp != nil {
			widgetData.Online = resp.Online
			if resp.Online {
				if resp.Version != nil {
					widgetData.Version = resp.Version.NameClean
				}
				if resp.Players != nil {
					widgetData.PlayersOnline = resp.Players.Online
					widgetData.PlayersMax = resp.Players.Max
				}
				if resp.MOTD != nil {
					widgetData.MOTD = resp.MOTD.Clean
				}
				widgetData.Icon = decodeBase64Icon(resp.Icon)
			}
		}

		pngBytes, err := widget.Render(widgetData)
		if err != nil {
			http.Error(w, "Failed to render widget", http.StatusInternalServerError)
			return
		}

		etag := fmt.Sprintf(`"%x"`, sha256.Sum256(pngBytes))

		w.Header().Set("Content-Type", "image/png")
		ttl := 60
		if cfg != nil && cfg.CacheTTL > 0 {
			ttl = int(cfg.CacheTTL.Seconds())
		}
		middleware.SetCacheHeaders(w, etag, hit, ttl)

		if middleware.CheckETag(etag, r) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(pngBytes)
	}
}

// HandleBedrockWidget handles HTTP requests to generate widget image for Bedrock Minecraft servers.
func HandleBedrockWidget(cfg *config.Config, c *cache.Cache, bl *blocklist.BlockList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := extractAddress(r, "/v2/widget/bedrock/", "/widget/bedrock/")
		address = strings.TrimSpace(address)
		if address == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid address value"))
			return
		}

		host, port, err := resolver.ParseAddress(address, 19132)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid address value"))
			return
		}

		dark := parseWidgetBool(r.URL.Query().Get("dark"), true)
		rounded := parseWidgetBool(r.URL.Query().Get("rounded"), true)
		transparent := parseWidgetBool(r.URL.Query().Get("transparent"), false)
		timeout := parseTimeout(r, cfg)

		resp, hit, _ := GetBedrockStatus(r.Context(), cfg, c, bl, host, port, timeout)

		widgetData := &widget.WidgetData{
			Online:      false,
			Host:        host,
			Port:        port,
			Edition:     "Bedrock Edition",
			Icon:        assets.DefaultIcon,
			Dark:        dark,
			Rounded:     rounded,
			Transparent: transparent,
		}

		if resp != nil && resp.Online {
			widgetData.Online = true
			if resp.Version != nil && resp.Version.Name != nil {
				widgetData.Version = *resp.Version.Name
			}
			if resp.Players != nil {
				if resp.Players.Online != nil {
					widgetData.PlayersOnline = int(*resp.Players.Online)
				}
				if resp.Players.Max != nil {
					widgetData.PlayersMax = int(*resp.Players.Max)
				}
			}
			if resp.MOTD != nil {
				widgetData.MOTD = resp.MOTD.Clean
			}
		}

		pngBytes, err := widget.Render(widgetData)
		if err != nil {
			http.Error(w, "Failed to render widget", http.StatusInternalServerError)
			return
		}

		etag := fmt.Sprintf(`"%x"`, sha256.Sum256(pngBytes))

		w.Header().Set("Content-Type", "image/png")
		ttl := 60
		if cfg != nil && cfg.CacheTTL > 0 {
			ttl = int(cfg.CacheTTL.Seconds())
		}
		middleware.SetCacheHeaders(w, etag, hit, ttl)

		if middleware.CheckETag(etag, r) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(pngBytes)
	}
}
