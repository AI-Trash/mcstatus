package util_test

import (
	"testing"
	"time"

	"mcstatus/internal/util"
)

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		def      bool
		expected bool
	}{
		{"true", false, true},
		{"1", false, true},
		{"false", true, false},
		{"0", true, false},
		{"", true, true},
		{"", false, false},
		{"invalid", true, true},
		{"invalid", false, false},
	}

	for _, tc := range tests {
		if got := util.ParseBool(tc.input, tc.def); got != tc.expected {
			t.Errorf("ParseBool(%q, %v) = %v; want %v", tc.input, tc.def, got, tc.expected)
		}
	}
}

func TestParseInt(t *testing.T) {
	if got := util.ParseInt("42", 10); got != 42 {
		t.Errorf("want 42, got %d", got)
	}
	if got := util.ParseInt("", 10); got != 10 {
		t.Errorf("want 10, got %d", got)
	}
	if got := util.ParseInt("invalid", 10); got != 10 {
		t.Errorf("want 10, got %d", got)
	}
}

func TestParseUint16(t *testing.T) {
	u, err := util.ParseUint16("25565", 19132)
	if err != nil || u != 25565 {
		t.Errorf("want 25565, got %d, err %v", u, err)
	}

	u, err = util.ParseUint16("", 19132)
	if err != nil || u != 19132 {
		t.Errorf("want 19132, got %d, err %v", u, err)
	}

	_, err = util.ParseUint16("999999", 19132)
	if err == nil {
		t.Error("expected error for 999999 overflow")
	}
}

func TestParseTimeout(t *testing.T) {
	// Normal
	if d := util.ParseTimeout("3.5", 5.0, 15.0); d != 3500*time.Millisecond {
		t.Errorf("want 3.5s, got %v", d)
	}
	// Clamped to max
	if d := util.ParseTimeout("20.0", 5.0, 15.0); d != 15*time.Second {
		t.Errorf("want 15s, got %v", d)
	}
	// Empty default
	if d := util.ParseTimeout("", 5.0, 15.0); d != 5*time.Second {
		t.Errorf("want 5s, got %v", d)
	}
	// Invalid negative
	if d := util.ParseTimeout("-1", 5.0, 15.0); d != 5*time.Second {
		t.Errorf("want 5s, got %v", d)
	}
}
