package util

import (
	"strconv"
	"strings"
	"time"
)

// ParseBool parses a boolean query parameter with fallback default.
func ParseBool(val string, defaultVal bool) bool {
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}

// ParseInt parses an integer string with a fallback default.
func ParseInt(val string, defaultVal int) int {
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

// ParseUint16 parses a port/uint16 string.
func ParseUint16(val string, defaultVal uint16) (uint16, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal, nil
	}
	u, err := strconv.ParseUint(val, 10, 16)
	if err != nil {
		return defaultVal, err
	}
	return uint16(u), nil
}

// ParseTimeout extracts timeout in seconds from string, clamping to [0, maxSec].
func ParseTimeout(val string, defaultSec, maxSec float64) time.Duration {
	val = strings.TrimSpace(val)
	if val == "" {
		return time.Duration(defaultSec * float64(time.Second))
	}
	sec, err := strconv.ParseFloat(val, 64)
	if err != nil || sec < 0 {
		sec = defaultSec
	}
	if maxSec > 0 && sec > maxSec {
		sec = maxSec
	}
	return time.Duration(sec * float64(time.Second))
}
