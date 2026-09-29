package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mcstatus-io/mcutil/v4/options"
	"github.com/mcstatus-io/mcutil/v4/query"
	"github.com/mcstatus-io/mcutil/v4/status"

	"mcstatus/internal/blocklist"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/motd"
	"mcstatus/internal/resolver"
	"mcstatus/internal/types"
	"mcstatus/internal/util"
)


func pingModernSLP(ctx context.Context, host string, port uint16, targetHost string, targetPort uint16, protocol int32, timeout time.Duration) (map[string]any, error) {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(targetHost, strconv.Itoa(int(targetPort))))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}

	hBuf := &bytes.Buffer{}
	_ = util.WriteVarInt(0x00, hBuf)
	_ = util.WriteVarInt(protocol, hBuf)
	_ = util.WriteVarInt(int32(len(host)), hBuf)
	hBuf.WriteString(host)
	_ = binary.Write(hBuf, binary.BigEndian, port)
	_ = util.WriteVarInt(1, hBuf)

	pktBuf := &bytes.Buffer{}
	_ = util.WriteVarInt(int32(hBuf.Len()), pktBuf)
	pktBuf.Write(hBuf.Bytes())
	if _, err := conn.Write(pktBuf.Bytes()); err != nil {
		return nil, err
	}

	if _, err := conn.Write([]byte{0x01, 0x00}); err != nil {
		return nil, err
	}

	if _, err := util.ReadVarInt(conn); err != nil {
		return nil, err
	}
	if _, err := util.ReadVarInt(conn); err != nil {
		return nil, err
	}
	strLen, err := util.ReadVarInt(conn)
	if err != nil {
		return nil, err
	}
	if strLen <= 0 || strLen > 10*1024*1024 {
		return nil, fmt.Errorf("invalid response length")
	}

	payload := make([]byte, strLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}



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

	// 4. Query status: Modern SLP first (with correct VarInt), then Legacy fallback
	modernRaw, modernErr := pingModernSLP(ctx, host, port, targetHost, targetPort, -1, timeout)
	if modernErr != nil {
		modernOpts := options.StatusModern{
			EnableSRV:         true,
			Timeout:           timeout,
			ProtocolVersion:   -1,
			ReceiveLimitBytes: 1 << 20,
			Ping:              true,
		}
		modernRaw, modernErr = status.ModernRaw(ctx, host, port, modernOpts)
	}

	if modernErr == nil && modernRaw != nil {
		resp.Online = true

		if vMap, ok := modernRaw["version"].(map[string]any); ok {
			protoVal, _ := vMap["protocol"].(float64)
			serverProto := int(protoVal)

			// Re-query with the server's exact reported protocol (with correct VarInt encoding) to eliminate outdated client warnings
			if serverProto > 0 {
				refCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
				if refinedRaw, err := pingModernSLP(refCtx, host, port, targetHost, targetPort, int32(serverProto), 1500*time.Millisecond); err == nil && refinedRaw != nil {
					modernRaw = refinedRaw
					if vMapRefined, ok := modernRaw["version"].(map[string]any); ok {
						vMap = vMapRefined
					}
				}
				cancel()
			}
			nameStr, _ := vMap["name"].(string)
			vFormat := motd.Format(nameStr)
			resp.Version = &types.JavaVersion{
				NameRaw:   vFormat.Raw,
				NameClean: vFormat.Clean,
				NameHTML:  vFormat.HTML,
				Protocol:  serverProto,
			}
		}

		if pMap, ok := modernRaw["players"].(map[string]any); ok {
			onlineVal, _ := pMap["online"].(float64)
			maxVal, _ := pMap["max"].(float64)
			var playersList []types.JavaPlayer
			if sArr, ok := pMap["sample"].([]any); ok {
				for _, p := range sArr {
					if pObj, ok := p.(map[string]any); ok {
						idStr, _ := pObj["id"].(string)
						nameStr, _ := pObj["name"].(string)
						pFormat := motd.Format(nameStr)
						playersList = append(playersList, types.JavaPlayer{
							UUID:      idStr,
							NameRaw:   pFormat.Raw,
							NameClean: pFormat.Clean,
							NameHTML:  pFormat.HTML,
						})
					}
				}
			}
			resp.Players = &types.JavaPlayers{
				Online: int(onlineVal),
				Max:    int(maxVal),
				List:   playersList,
			}
		}

		resp.MOTD = motd.Format(modernRaw["description"])

		if fav, ok := modernRaw["favicon"].(string); ok && fav != "" {
			resp.Icon = &fav
		}

		if mObj, ok := modernRaw["modinfo"].(map[string]any); ok {
			if mList, ok := mObj["modList"].([]any); ok {
				for _, m := range mList {
					if mItem, ok := m.(map[string]any); ok {
						name, _ := mItem["modid"].(string)
						ver, _ := mItem["version"].(string)
						resp.Mods = append(resp.Mods, types.Mod{Name: name, Version: ver})
					}
				}
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
