package blocklist

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

func TestBlockListDirectMatch(t *testing.T) {
	bl := New(0)
	defer bl.Close()

	target := "badserver.example.com"
	h := sha1.Sum([]byte(target))
	hashStr := hex.EncodeToString(h[:])

	bl.SetHashes([]string{hashStr})

	if !bl.IsBlocked(target) {
		t.Fatalf("expected %s to be blocked", target)
	}

	if bl.IsBlocked("goodserver.example.com") {
		t.Fatalf("expected goodserver not to be blocked")
	}
}

func TestBlockListWildcardMatch(t *testing.T) {
	bl := New(0)
	defer bl.Close()

	wildcard := "*.example.com"
	h := sha1.Sum([]byte(wildcard))
	hashStr := hex.EncodeToString(h[:])

	bl.SetHashes([]string{hashStr})

	if !bl.IsBlocked("play.example.com") {
		t.Fatalf("expected play.example.com to match wildcard *.example.com")
	}
	if !bl.IsBlocked("sub.play.example.com") {
		t.Fatalf("expected sub.play.example.com to match wildcard *.example.com")
	}
	if bl.IsBlocked("example.net") {
		t.Fatalf("expected example.net not to match")
	}
}

func TestBlockListIPWildcardMatch(t *testing.T) {
	bl := New(0)
	defer bl.Close()

	wildcardIP := "192.168.1.*"
	h := sha1.Sum([]byte(wildcardIP))
	hashStr := hex.EncodeToString(h[:])

	bl.SetHashes([]string{hashStr})

	if !bl.IsBlocked("192.168.1.50") {
		t.Fatalf("expected 192.168.1.50 to match 192.168.1.*")
	}
	if bl.IsBlocked("192.168.2.50") {
		t.Fatalf("expected 192.168.2.50 not to match")
	}
}
