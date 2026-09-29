package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"strings"
	"time"

	"mcstatus/internal/assets"
	"mcstatus/internal/blocklist"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/resolver"
	"mcstatus/internal/widget"
)

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

// parseWidgetBool is an alias for ParseBool for backward compatibility.
func parseWidgetBool(val string, defaultVal bool) bool {
	return ParseBool(val, defaultVal)
}

// HandleJavaWidget handles HTTP requests to generate widget image for Java Minecraft servers.
func HandleJavaWidget(cfg *config.Config, c *cache.Cache, bl *blocklist.BlockList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := ExtractAddress(r, "/v2/widget/java/", "/widget/java/")
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

		dark := ParseBool(r.URL.Query().Get("dark"), true)
		rounded := ParseBool(r.URL.Query().Get("rounded"), true)
		transparent := ParseBool(r.URL.Query().Get("transparent"), false)
		style := strings.TrimSpace(r.URL.Query().Get("style"))
		format, _ := ResolveImageFormat(r)
		hideIcon := !ParseBool(r.URL.Query().Get("icon"), true)
		showAddress := ParseBool(r.URL.Query().Get("address"), true)
		if r.URL.Query().Has("title") {
			showAddress = ParseBool(r.URL.Query().Get("title"), showAddress)
		}
		hideAddress := !showAddress
		timeout := ParseTimeout(r, cfg)

		ttl := 60 * time.Second
		if cfg != nil && cfg.CacheTTL > 0 {
			ttl = cfg.CacheTTL
		}

		cacheKey := fmt.Sprintf("widget:java:%s:%d:%t:%t:%t:%s:%s:%t:%t", strings.ToLower(host), port, dark, rounded, transparent, style, format, hideIcon, hideAddress)
		ServeCached(w, r, c, cacheKey, ttl, func() ([]byte, string, error) {
			resp, _, _ := GetJavaStatus(r.Context(), cfg, c, bl, host, port, true, timeout)

			widgetData := &widget.WidgetData{
				Style:       style,
				Format:      format,
				Online:      false,
				Host:        host,
				Port:        port,
				Edition:     "Java Edition",
				Icon:        assets.DefaultIcon,
				Dark:        dark,
				Rounded:     rounded,
				Transparent: transparent,
				HideIcon:    hideIcon,
				HideAddress: hideAddress,
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
						widgetData.MOTDRaw = resp.MOTD.Raw
					}
					widgetData.Icon = decodeBase64Icon(resp.Icon)
				}
			}

			return widget.RenderFormatted(widgetData)
		})
	}
}

// HandleBedrockWidget handles HTTP requests to generate widget image for Bedrock Minecraft servers.
func HandleBedrockWidget(cfg *config.Config, c *cache.Cache, bl *blocklist.BlockList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := ExtractAddress(r, "/v2/widget/bedrock/", "/widget/bedrock/")
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

		dark := ParseBool(r.URL.Query().Get("dark"), true)
		rounded := ParseBool(r.URL.Query().Get("rounded"), true)
		transparent := ParseBool(r.URL.Query().Get("transparent"), false)
		style := strings.TrimSpace(r.URL.Query().Get("style"))
		format, _ := ResolveImageFormat(r)
		hideIcon := !ParseBool(r.URL.Query().Get("icon"), true)
		showAddress := ParseBool(r.URL.Query().Get("address"), true)
		if r.URL.Query().Has("title") {
			showAddress = ParseBool(r.URL.Query().Get("title"), showAddress)
		}
		hideAddress := !showAddress
		timeout := ParseTimeout(r, cfg)

		ttl := 60 * time.Second
		if cfg != nil && cfg.CacheTTL > 0 {
			ttl = cfg.CacheTTL
		}

		cacheKey := fmt.Sprintf("widget:bedrock:%s:%d:%t:%t:%t:%s:%s:%t:%t", strings.ToLower(host), port, dark, rounded, transparent, style, format, hideIcon, hideAddress)
		ServeCached(w, r, c, cacheKey, ttl, func() ([]byte, string, error) {
			resp, _, _ := GetBedrockStatus(r.Context(), cfg, c, bl, host, port, timeout)

			widgetData := &widget.WidgetData{
				Style:       style,
				Format:      format,
				Online:      false,
				Host:        host,
				Port:        port,
				Edition:     "Bedrock Edition",
				Icon:        assets.DefaultIcon,
				Dark:        dark,
				Rounded:     rounded,
				Transparent: transparent,
				HideIcon:    hideIcon,
				HideAddress: hideAddress,
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
					widgetData.MOTDRaw = resp.MOTD.Raw
				}
			}

			return widget.RenderFormatted(widgetData)
		})
	}
}
