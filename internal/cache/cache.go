package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type Entry struct {
	Data        []byte
	ContentType string
	ETag        string
	RetrievedAt time.Time
	ExpiresAt   time.Time
}

func (e *Entry) TimeRemaining() int {
	rem := time.Until(e.ExpiresAt).Seconds()
	if rem < 0 {
		return 0
	}
	return int(rem)
}

type Cache struct {
	mu         sync.RWMutex
	items      map[string]*Entry
	group      singleflight.Group
	defaultTTL time.Duration
	stopCh     chan struct{}
}

func New(defaultTTL time.Duration, cleanupInterval time.Duration) *Cache {
	c := &Cache{
		items:      make(map[string]*Entry),
		defaultTTL: defaultTTL,
		stopCh:     make(chan struct{}),
	}
	if cleanupInterval > 0 {
		go c.cleanupLoop(cleanupInterval)
	}
	return c
}

func (c *Cache) Close() {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
}

func (c *Cache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case now := <-ticker.C:
			c.cleanup(now)
		}
	}
}

func (c *Cache) cleanup(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.items {
		if now.After(v.ExpiresAt) {
			delete(c.items, k)
		}
	}
}

func (c *Cache) Get(key string) (*Entry, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry, true
}

func (c *Cache) Set(key string, data []byte, contentType string, ttl time.Duration) *Entry {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}
	now := time.Now()
	expiresAt := now.Add(ttl)

	sum := sha256.Sum256(data)
	etag := fmt.Sprintf(`"%d-%s"`, len(data), hex.EncodeToString(sum[:8]))

	entry := &Entry{
		Data:        data,
		ContentType: contentType,
		ETag:        etag,
		RetrievedAt: now,
		ExpiresAt:   expiresAt,
	}

	c.mu.Lock()
	c.items[key] = entry
	c.mu.Unlock()

	return entry
}

// GetOrCompute retrieves from cache if valid, otherwise computes the value single-flighted across concurrent requests.
// Returns entry, hit (true if served from cache, false if fresh), error.
func (c *Cache) GetOrCompute(key string, compute func() ([]byte, string, error), ttl time.Duration) (*Entry, bool, error) {
	if entry, ok := c.Get(key); ok {
		return entry, true, nil
	}

	// Use singleflight to prevent thundering herd
	v, err, _ := c.group.Do(key, func() (any, error) {
		// Double check under singleflight execution
		if entry, ok := c.Get(key); ok {
			return entry, nil
		}
		data, contentType, err := compute()
		if err != nil {
			return nil, err
		}
		entry := c.Set(key, data, contentType, ttl)
		return entry, nil
	})

	if err != nil {
		return nil, false, err
	}

	entry, ok := v.(*Entry)
	if !ok {
		return nil, false, fmt.Errorf("cache: unexpected value type in singleflight")
	}

	return entry, false, nil
}
