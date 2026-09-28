package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mcstatus-io/mcutil/v4/options"
	"github.com/mcstatus-io/mcutil/v4/query"
	"github.com/mcstatus-io/mcutil/v4/status"

	"mcstatus/internal/blocklist"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/resolver"
	"mcstatus/internal/types"
)



// parsePluginsAndSoftware parses the software name and plugins from query GS4 response data.
func parsePluginsAndSoftware(data map[string]string) (*string, []types.Plugin) {
	if data == nil {
		return nil, nil
	}

	var software *string
	if sm, ok := data["server_mod"]; ok && strings.TrimSpace(sm) != "" {
		s := strings.TrimSpace(sm)
		software = &s
	}

	rawPlugins, ok := data["plugins"]
	if !ok || strings.TrimSpace(rawPlugins) == "" {
		return software, nil
	}

	rawPlugins = strings.TrimSpace(rawPlugins)
	var pluginListStr string

	if strings.Contains(rawPlugins, ":") {
		parts := strings.SplitN(rawPlugins, ":", 2)
		sw := strings.TrimSpace(parts[0])
		if sw != "" && software == nil {
			software = &sw
		}
		if len(parts) > 1 {
			pluginListStr = strings.TrimSpace(parts[1])
		}
	} else {
		pluginListStr = rawPlugins
	}

	if pluginListStr == "" {
		return software, nil
	}

	var rawEntries []string
	if strings.Contains(pluginListStr, ";") {
		rawEntries = strings.Split(pluginListStr, ";")
	} else {
		rawEntries = strings.Split(pluginListStr, ",")
	}

	plugins := make([]types.Plugin, 0, len(rawEntries))
	for _, entry := range rawEntries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.SplitN(entry, " ", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}

		var version *string
		if len(parts) > 1 {
			v := strings.TrimSpace(parts[1])
			if v != "" {
				version = &v
			}
		}

		plugins = append(plugins, types.Plugin{
			Name:    name,
			Version: version,
		})
	}

	return software, plugins
}

// FetchJavaStatus executes status and optional query lookups against a Java server.
func FetchJavaStatus(ctx context.Context, cfg *config.Config, bl *blocklist.BlockList, host string, port uint16, queryEnabled bool, timeout time.Duration) (*types.JavaStatusResponse, error) {
	startTime := time.Now()
	var expiresAt time.Time
	if cfg != nil && cfg.CacheTTL > 0 {
		expiresAt = startTime.Add(cfg.CacheTTL)
	} else {
		expiresAt = startTime.Add(60 * time.Second)
	}

	resp := &types.JavaStatusResponse{
		Online:      false,
		Host:        host,
		Port:        port,
		EULABlocked: false,
		RetrievedAt: startTime.UnixMilli(),
		ExpiresAt:   expiresAt.UnixMilli(),
		SRVRecord:   nil,
	}

	// 1. Check Mojang blocklist
	if bl != nil {
		resp.EULABlocked = bl.IsBlocked(host)
	}

	// 2. Resolve SRV record
	targetHost := host
	targetPort := port
	if port == 25565 {
		srvHost, srvPort, record := resolver.LookupSRV(host)
		if record != nil {
			resp.SRVRecord = &types.SRVRecord{
				Host: record.Host,
				Port: record.Port,
			}
			if srvHost != nil {
				targetHost = *srvHost
			}
			if srvPort != nil {
				targetPort = *srvPort
			}
		}
	}

	if bl != nil && !resp.EULABlocked && targetHost != host {
		if bl.IsBlocked(targetHost) {
			resp.EULABlocked = true
		}
	}

	// 3. Resolve IP address
	ipAddress := resolver.ResolveIP(ctx, targetHost)
	if ipAddress == nil && targetHost != host {
		ipAddress = resolver.ResolveIP(ctx, host)
	}
	resp.IPAddress = ipAddress

	// 4. Query status: Modern first, then Legacy fallback
	modernOpts := options.StatusModern{
		EnableSRV:         true,
		Timeout:           timeout,
		ProtocolVersion:   -1,
		ReceiveLimitBytes: 1 << 20,
		Ping:              true,
	}

	modernResp, modernErr := status.Modern(ctx, host, port, modernOpts)
	if modernErr == nil && modernResp != nil {
		resp.Online = true
		resp.Version = &types.JavaVersion{
			NameRaw:   modernResp.Version.Name.Raw,
			NameClean: modernResp.Version.Name.Clean,
			NameHTML:  modernResp.Version.Name.HTML,
			Protocol:  int(modernResp.Version.Protocol),
		}

		var onlineCount int
		var maxCount int
		if modernResp.Players.Online != nil {
			onlineCount = int(*modernResp.Players.Online)
		}
		if modernResp.Players.Max != nil {
			maxCount = int(*modernResp.Players.Max)
		}
		playersList := make([]types.JavaPlayer, 0, len(modernResp.Players.Sample))
		for _, p := range modernResp.Players.Sample {
			playersList = append(playersList, types.JavaPlayer{
				UUID:      p.ID,
				NameRaw:   p.Name.Raw,
				NameClean: p.Name.Clean,
				NameHTML:  p.Name.HTML,
			})
		}
		resp.Players = &types.JavaPlayers{
			Online: onlineCount,
			Max:    maxCount,
			List:   playersList,
		}

		resp.MOTD = &types.FormattedString{
			Raw:   modernResp.MOTD.Raw,
			Clean: modernResp.MOTD.Clean,
			HTML:  modernResp.MOTD.HTML,
		}

		resp.Icon = modernResp.Favicon

		if modernResp.Mods != nil && len(modernResp.Mods.List) > 0 {
			resp.Mods = make([]types.Mod, 0, len(modernResp.Mods.List))
			for _, m := range modernResp.Mods.List {
				resp.Mods = append(resp.Mods, types.Mod{
					Name:    m.ID,
					Version: m.Version,
				})
			}
		}
	} else {
		legacyOpts := options.StatusLegacy{
			EnableSRV:         true,
			Timeout:           timeout,
			ProtocolVersion:   -1,
			ReceiveLimitBytes: 1 << 20,
		}
		legacyResp, legacyErr := status.Legacy(ctx, host, port, legacyOpts)
		if legacyErr == nil && legacyResp != nil {
			resp.Online = true
			if legacyResp.Version != nil {
				resp.Version = &types.JavaVersion{
					NameRaw:   legacyResp.Version.Name.Raw,
					NameClean: legacyResp.Version.Name.Clean,
					NameHTML:  legacyResp.Version.Name.HTML,
					Protocol:  int(legacyResp.Version.Protocol),
				}
			}
			resp.Players = &types.JavaPlayers{
				Online: int(legacyResp.Players.Online),
				Max:    int(legacyResp.Players.Max),
				List:   []types.JavaPlayer{},
			}
			resp.MOTD = &types.FormattedString{
				Raw:   legacyResp.MOTD.Raw,
				Clean: legacyResp.MOTD.Clean,
				HTML:  legacyResp.MOTD.HTML,
			}
		}
	}

	// 5. Query protocol if online and query=true
	if resp.Online && queryEnabled {
		elapsed := time.Since(startTime)
		remainingTimeout := timeout - elapsed
		if remainingTimeout > 0 {
			queryTimeout := 1 * time.Second
			if remainingTimeout < queryTimeout {
				queryTimeout = remainingTimeout
			}

			queryCtx, queryCancel := context.WithTimeout(ctx, queryTimeout)
			defer queryCancel()

			queryOpts := options.Query{
				Timeout:           queryTimeout,
				ReceiveLimitBytes: 1 << 20,
			}

			queryResp, queryErr := query.Full(queryCtx, targetHost, targetPort, queryOpts)
			if queryErr == nil && queryResp != nil && queryResp.Data != nil {
				software, plugins := parsePluginsAndSoftware(queryResp.Data)
				if software != nil {
					resp.Software = software
				}
				if len(plugins) > 0 {
					resp.Plugins = plugins
				}
			}
		}
	}

	return resp, nil
}

// GetJavaStatus retrieves Java status using cache if available.
func GetJavaStatus(ctx context.Context, cfg *config.Config, c *cache.Cache, bl *blocklist.BlockList, host string, port uint16, queryEnabled bool, timeout time.Duration) (*types.JavaStatusResponse, bool, error) {
	if c == nil {
		resp, err := FetchJavaStatus(ctx, cfg, bl, host, port, queryEnabled, timeout)
		return resp, false, err
	}

	ttl := 60 * time.Second
	if cfg != nil && cfg.CacheTTL > 0 {
		ttl = cfg.CacheTTL
	}

	cacheKey := fmt.Sprintf("java:%s:%d:%t", strings.ToLower(host), port, queryEnabled)
	entry, hit, err := c.GetOrCompute(cacheKey, func() ([]byte, string, error) {
		fetchCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		resp, fetchErr := FetchJavaStatus(fetchCtx, cfg, bl, host, port, queryEnabled, timeout)
		if fetchErr != nil {
			return nil, "", fetchErr
		}

		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			return nil, "", marshalErr
		}

		return data, "application/json", nil
	}, ttl)

	if err != nil {
		return nil, false, err
	}

	var resp types.JavaStatusResponse
	if err := json.Unmarshal(entry.Data, &resp); err != nil {
		return nil, false, err
	}

	return &resp, hit, nil
}

// HandleJavaStatus handles HTTP requests for Java server status.
func HandleJavaStatus(cfg *config.Config, c *cache.Cache, bl *blocklist.BlockList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := ExtractAddress(r, "/v2/status/java/", "/status/java/")
		host, port, err := resolver.ParseAddress(address, 25565)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid address value"))
			return
		}

		queryEnabled := ParseBool(r.URL.Query().Get("query"), true)
		timeout := ParseTimeout(r, cfg)

		ttl := 60 * time.Second
		if cfg != nil && cfg.CacheTTL > 0 {
			ttl = cfg.CacheTTL
		}

		cacheKey := fmt.Sprintf("java:%s:%d:%t", strings.ToLower(host), port, queryEnabled)
		ServeCached(w, r, c, cacheKey, ttl, func() ([]byte, string, error) {
			fetchCtx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			resp, fetchErr := FetchJavaStatus(fetchCtx, cfg, bl, host, port, queryEnabled, timeout)
			if fetchErr != nil {
				return nil, "", fetchErr
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				return nil, "", marshalErr
			}

			return data, "application/json", nil
		})
	}
}
