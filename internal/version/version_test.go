package version

import (
	"testing"
)

func TestVersionDefault(t *testing.T) {
	v := Get()
	if v == "" {
		t.Fatalf("expected non-empty version")
	}
}

func TestVersionCustom(t *testing.T) {
	orig := Version
	defer func() { Version = orig }()

	Version = "abc1234"
	if got := Get(); got != "abc1234" {
		t.Fatalf("expected abc1234, got %s", got)
	}
}
