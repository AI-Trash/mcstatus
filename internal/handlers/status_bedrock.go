package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mcstatus-io/mcutil/v4/options"
	"github.com/mcstatus-io/mcutil/v4/response"
	"github.com/mcstatus-io/mcutil/v4/status"
	"mcstatus/internal/blocklist"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/resolver"
	"mcstatus/internal/types"
)

// FetchBedrockStatus executes status lookup against a Bedrock server.
func FetchBedrockStatus(ctx context.Context, cfg *config.Config, bl *blocklist.BlockList, host string, port uint16, timeout time.Duration) (*types.BedrockStatusResponse, error) {
	startTime := time.Now()
	var expiresAt time.Time
	if cfg != nil && cfg.CacheTTL > 0 {
		expiresAt = startTime.Add(cfg.CacheTTL)
	} else {
		expiresAt = startTime.Add(60 * time.Second)
	}

	resp := &types.BedrockStatusResponse{
		Online:      false,
		Host:        host,
		Port:        port,
		EULABlocked: false,
		RetrievedAt: startTime.UnixMilli(),
		ExpiresAt:   expiresAt.UnixMilli(),
	}

	// 1. Check Mojang blocklist
	if bl != nil {
		resp.EULABlocked = bl.IsBlocked(host)
	}

	// 2. Query Bedrock status using RFC 8305 Happy Eyeballs
	bedrockResp, winningIP, err := queryBedrockHappyEyeballs(ctx, host, port, timeout)
	if winningIP != nil {
		resp.IPAddress = winningIP
	} else {
		resp.IPAddress = resolver.ResolveIP(ctx, host)
	}
	if err == nil && bedrockResp != nil {
		resp.Online = true

		if bedrockResp.Version != nil || bedrockResp.ProtocolVersion != nil {
			resp.Version = &types.BedrockVersion{
				Name:     bedrockResp.Version,
				Protocol: bedrockResp.ProtocolVersion,
			}
		}

		if bedrockResp.OnlinePlayers != nil || bedrockResp.MaxPlayers != nil {
			resp.Players = &types.BedrockPlayers{
				Online: bedrockResp.OnlinePlayers,
				Max:    bedrockResp.MaxPlayers,
			}
		}

		if bedrockResp.MOTD != nil {
			resp.MOTD = &types.FormattedString{
				Raw:   bedrockResp.MOTD.Raw,
				Clean: bedrockResp.MOTD.Clean,
				HTML:  bedrockResp.MOTD.HTML,
			}
		}

		resp.Gamemode = bedrockResp.Gamemode
		if bedrockResp.ServerID != nil {
			resp.ServerID = bedrockResp.ServerID
		} else if bedrockResp.ServerGUID != 0 {
			guidStr := strconv.FormatUint(uint64(bedrockResp.ServerGUID), 10)
			resp.ServerID = &guidStr
		}
		resp.Edition = bedrockResp.Edition
	}

	return resp, nil
}

// GetBedrockStatus retrieves Bedrock status using cache if available.
func GetBedrockStatus(ctx context.Context, cfg *config.Config, c *cache.Cache, bl *blocklist.BlockList, host string, port uint16, timeout time.Duration) (*types.BedrockStatusResponse, bool, error) {
	if c == nil {
		resp, err := FetchBedrockStatus(ctx, cfg, bl, host, port, timeout)
		return resp, false, err
	}

	ttl := 60 * time.Second
	if cfg != nil && cfg.CacheTTL > 0 {
		ttl = cfg.CacheTTL
	}

	cacheKey := fmt.Sprintf("bedrock:%s:%d", strings.ToLower(host), port)
	entry, hit, err := c.GetOrCompute(cacheKey, func() ([]byte, string, error) {
		fetchCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		resp, fetchErr := FetchBedrockStatus(fetchCtx, cfg, bl, host, port, timeout)
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

	var resp types.BedrockStatusResponse
	if err := json.Unmarshal(entry.Data, &resp); err != nil {
		return nil, false, err
	}

	return &resp, hit, nil
}

// HandleBedrockStatus handles HTTP requests for Bedrock server status.
func HandleBedrockStatus(cfg *config.Config, c *cache.Cache, bl *blocklist.BlockList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := ExtractAddress(r, "/v2/status/bedrock/", "/status/bedrock/")
		host, port, err := resolver.ParseAddress(address, 19132)
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

		cacheKey := fmt.Sprintf("bedrock:%s:%d", strings.ToLower(host), port)
		ServeCached(w, r, c, cacheKey, ttl, func() ([]byte, string, error) {
			fetchCtx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			resp, fetchErr := FetchBedrockStatus(fetchCtx, cfg, bl, host, port, timeout)
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

type bedrockQueryResult struct {
	resp *response.StatusBedrock
	ip   string
	err  error
}

func queryBedrockSingle(ctx context.Context, target string, port uint16, timeout time.Duration) (*response.StatusBedrock, error) {
	opts := options.StatusBedrock{
		Timeout:           timeout,
		ReceiveLimitBytes: 1 << 20,
	}
	return status.Bedrock(ctx, target, port, opts)
}

// queryBedrockHappyEyeballs implements RFC 8305 Happy Eyeballs for UDP / Bedrock RakNet ping.
func queryBedrockHappyEyeballs(ctx context.Context, host string, port uint16, timeout time.Duration) (*response.StatusBedrock, *string, error) {
	ipv6List, ipv4List := resolver.ResolveDualStackIPs(ctx, host)

	// If single stack or already IP
	if len(ipv6List) == 0 && len(ipv4List) == 0 {
		r, err := queryBedrockSingle(ctx, host, port, timeout)
		return r, nil, err
	}
	if len(ipv6List) == 0 {
		target := ipv4List[0]
		r, err := queryBedrockSingle(ctx, target, port, timeout)
		return r, &target, err
	}
	if len(ipv4List) == 0 {
		target := ipv6List[0]
		r, err := queryBedrockSingle(ctx, target, port, timeout)
		return r, &target, err
	}

	// Dual-stack: IPv6 preferred with 250ms Connection Attempt Delay (RFC 8305)
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan bedrockQueryResult, 2)

	// 1. Launch IPv6 probe first
	go func() {
		r, err := queryBedrockSingle(childCtx, ipv6List[0], port, timeout)
		results <- bedrockQueryResult{resp: r, ip: ipv6List[0], err: err}
	}()

	timer := time.NewTimer(250 * time.Millisecond)
	defer timer.Stop()

	var ipv4Launched bool
	var lastErr error

	for range 2 {
		if !ipv4Launched {
			select {
			case res := <-results:
				if res.err == nil && res.resp != nil {
					cancel()
					return res.resp, &res.ip, nil
				}
				lastErr = res.err
				// IPv6 failed early, launch IPv4 immediately without waiting
				ipv4Launched = true
				go func() {
					r, err := queryBedrockSingle(childCtx, ipv4List[0], port, timeout)
					results <- bedrockQueryResult{resp: r, ip: ipv4List[0], err: err}
				}()
			case <-timer.C:
				// 250ms fallback delay elapsed, launch IPv4 concurrently
				ipv4Launched = true
				go func() {
					r, err := queryBedrockSingle(childCtx, ipv4List[0], port, timeout)
					results <- bedrockQueryResult{resp: r, ip: ipv4List[0], err: err}
				}()
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		} else {
			select {
			case res := <-results:
				if res.err == nil && res.resp != nil {
					cancel()
					return res.resp, &res.ip, nil
				}
				lastErr = res.err
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		}
	}

	return nil, nil, lastErr
}
