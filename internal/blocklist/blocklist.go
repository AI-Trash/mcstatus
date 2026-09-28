package blocklist

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultMojangBlockedServersURL = "https://sessionserver.mojang.com/blockedservers"

type BlockList struct {
	mu      sync.RWMutex
	hashes  map[string]struct{}
	client  *http.Client
	url     string
	stopCh  chan struct{}
}

func New(refreshInterval time.Duration) *BlockList {
	bl := &BlockList{
		hashes: make(map[string]struct{}),
		client: &http.Client{Timeout: 5 * time.Second},
		url:    defaultMojangBlockedServersURL,
		stopCh: make(chan struct{}),
	}

	// Fetch asynchronously so startup is immediate and non-blocking
	go func() {
		bl.refresh()
		if refreshInterval > 0 {
			bl.refreshLoop(refreshInterval)
		}
	}()

	return bl
}

func (bl *BlockList) Close() {
	select {
	case <-bl.stopCh:
	default:
		close(bl.stopCh)
	}
}

func (bl *BlockList) refreshLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-bl.stopCh:
			return
		case <-ticker.C:
			bl.refresh()
		}
	}
}

func (bl *BlockList) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bl.url, nil)
	if err != nil {
		slog.Debug("failed to create request for mojang blocked servers", "error", err)
		return
	}

	resp, err := bl.client.Do(req)
	if err != nil {
		slog.Debug("failed to fetch mojang blocked servers", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("mojang blocked servers returned non-200", "status", resp.StatusCode)
		return
	}

	newHashes := make(map[string]struct{})
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			newHashes[strings.ToLower(line)] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Debug("error reading mojang blocked servers", "error", err)
		return
	}

	bl.mu.Lock()
	bl.hashes = newHashes
	bl.mu.Unlock()

	slog.Info("loaded mojang blocked servers list", "count", len(newHashes))
}

// IsBlocked checks whether the given host is in Mojang's blocked servers list.
func (bl *BlockList) IsBlocked(host string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	if len(bl.hashes) == 0 {
		return false
	}

	host = strings.TrimSuffix(strings.ToLower(host), ".")
	patterns := generatePatterns(host)

	for _, pattern := range patterns {
		h := sha1.Sum([]byte(pattern))
		hashStr := hex.EncodeToString(h[:])
		if _, ok := bl.hashes[hashStr]; ok {
			return true
		}
	}

	return false
}

// SetHashes sets hash set directly, mainly for testing.
func (bl *BlockList) SetHashes(hashes []string) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.hashes = make(map[string]struct{}, len(hashes))
	for _, h := range hashes {
		bl.hashes[strings.ToLower(h)] = struct{}{}
	}
}

func generatePatterns(host string) []string {
	var patterns []string
	patterns = append(patterns, host)

	ip := net.ParseIP(host)
	if ip != nil {
		// IPv4 pattern handling
		parts := strings.Split(host, ".")
		if len(parts) == 4 {
			patterns = append(patterns,
				parts[0]+"."+parts[1]+"."+parts[2]+".*",
				parts[0]+"."+parts[1]+".*",
				parts[0]+".*",
				"*",
			)
		}
		return patterns
	}

	// Domain pattern handling
	parts := strings.Split(host, ".")
	if len(parts) > 1 {
		for i := 1; i < len(parts); i++ {
			patterns = append(patterns, "*."+strings.Join(parts[i:], "."))
		}
	}

	return patterns
}
