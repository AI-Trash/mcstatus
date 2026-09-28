package cache

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheSetAndGet(t *testing.T) {
	c := New(time.Minute, 0)
	defer c.Close()

	data := []byte(`{"online":true}`)
	entry := c.Set("test-key", data, "application/json", time.Minute)

	if entry.ETag == "" {
		t.Fatalf("expected non-empty ETag")
	}

	got, ok := c.Get("test-key")
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if string(got.Data) != string(data) {
		t.Fatalf("expected %s, got %s", string(data), string(got.Data))
	}
	if got.TimeRemaining() <= 0 {
		t.Fatalf("expected positive time remaining")
	}
}

func TestCacheExpiry(t *testing.T) {
	c := New(50*time.Millisecond, 10*time.Millisecond)
	defer c.Close()

	c.Set("short-lived", []byte("ok"), "text/plain", 50*time.Millisecond)
	if _, ok := c.Get("short-lived"); !ok {
		t.Fatalf("expected hit before expiry")
	}

	time.Sleep(70 * time.Millisecond)
	if _, ok := c.Get("short-lived"); ok {
		t.Fatalf("expected miss after expiry")
	}
}

func TestCacheSingleFlight(t *testing.T) {
	c := New(time.Minute, 0)
	defer c.Close()

	var counter int64
	var wg sync.WaitGroup
	concurrent := 20

	for range concurrent {
		wg.Add(1)
		go func() {
			defer wg.Done()
			entry, _, err := c.GetOrCompute("shared-key", func() ([]byte, string, error) {
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt64(&counter, 1)
				return []byte("computed"), "text/plain", nil
			}, time.Minute)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if string(entry.Data) != "computed" {
				t.Errorf("unexpected data: %s", string(entry.Data))
			}
		}()
	}

	wg.Wait()

	if counter != 1 {
		t.Fatalf("expected compute called exactly 1 time, got %d", counter)
	}
}
