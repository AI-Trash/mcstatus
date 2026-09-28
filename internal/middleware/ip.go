package middleware

import (
	"net"
	"net/http"
	"strings"
)

var DefaultPrivateRanges = []string{
	"127.0.0.0/8",    // IPv4 Loopback
	"10.0.0.0/8",     // RFC 1918 Class A
	"172.16.0.0/12",  // RFC 1918 Class B
	"192.168.0.0/16", // RFC 1918 Class C
	"100.64.0.0/10",  // RFC 6598 Carrier Grade NAT
	"169.254.0.0/16", // RFC 3927 Link-Local
	"::1/128",        // IPv6 Loopback
	"fc00::/7",       // IPv6 Unique Local Address (ULA)
	"fe80::/10",      // IPv6 Link-Local
}

type IPFilter struct {
	trustAll  bool
	trustNone bool
	subnets   []*net.IPNet
}

func NewIPFilter(cidrs []string) *IPFilter {
	if len(cidrs) == 0 {
		cidrs = DefaultPrivateRanges
	}

	filter := &IPFilter{}
	for _, raw := range cidrs {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if s == "*" {
			filter.trustAll = true
			return filter
		}
		if strings.EqualFold(s, "false") || strings.EqualFold(s, "none") {
			filter.trustNone = true
			return filter
		}

		if !strings.Contains(s, "/") {
			if ip := net.ParseIP(s); ip != nil {
				if ip.To4() != nil {
					s += "/32"
				} else {
					s += "/128"
				}
			}
		}

		_, ipnet, err := net.ParseCIDR(s)
		if err == nil && ipnet != nil {
			filter.subnets = append(filter.subnets, ipnet)
		}
	}

	return filter
}

func (f *IPFilter) IsTrusted(ip net.IP) bool {
	if f == nil || f.trustNone {
		return false
	}
	if f.trustAll {
		return true
	}
	for _, subnet := range f.subnets {
		if subnet.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP extracts the real client IP address.
// If direct peer IP is trusted by filter, headers like CF-Connecting-IP, X-Real-IP,
// and X-Forwarded-For are checked. Otherwise, the direct peer IP is returned to prevent spoofing.
func ClientIP(r *http.Request, filter *IPFilter) string {
	directHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		directHost = r.RemoteAddr
	}
	directHost = strings.TrimSpace(directHost)
	directIP := net.ParseIP(directHost)

	// If direct peer cannot be parsed or is not trusted, use direct peer IP
	if directIP == nil || filter == nil || !filter.IsTrusted(directIP) {
		return directHost
	}

	// 1. Cloudflare header (highest trust when proxy is trusted)
	if cfIP := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cfIP != "" {
		if net.ParseIP(cfIP) != nil {
			return cfIP
		}
	}

	// 2. X-Real-IP header
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		if net.ParseIP(realIP) != nil {
			return realIP
		}
	}

	// 3. X-Forwarded-For header (first valid client IP)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if ip := net.ParseIP(part); ip != nil {
				return part
			}
		}
	}

	return directHost
}
