package resolver

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/mcstatus-io/mcutil/v4/util"
)

var ErrInvalidAddress = errors.New("Invalid address value")

type SRVRecord struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

// ParseAddress parses address string into host and port.
func ParseAddress(address string, defaultPort uint16) (string, uint16, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", 0, ErrInvalidAddress
	}

	// Address must not contain spaces or control characters
	if strings.ContainsAny(address, " \t\r\n/\\") {
		return "", 0, ErrInvalidAddress
	}

	// Handle bracketed IPv6: [::1] or [::1]:25565
	if strings.HasPrefix(address, "[") {
		endBracket := strings.Index(address, "]")
		if endBracket == -1 {
			return "", 0, ErrInvalidAddress
		}
		host := address[1:endBracket]
		if net.ParseIP(host) == nil {
			return "", 0, ErrInvalidAddress
		}

		rest := address[endBracket+1:]
		if rest == "" {
			return host, defaultPort, nil
		}
		if !strings.HasPrefix(rest, ":") {
			return "", 0, ErrInvalidAddress
		}

		portStr := rest[1:]
		portNum, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil || portNum == 0 {
			return "", 0, ErrInvalidAddress
		}
		return host, uint16(portNum), nil
	}

	// If contains colons, could be host:port or bare IPv6
	colonCount := strings.Count(address, ":")
	if colonCount > 1 {
		// Bare IPv6 without brackets
		if ip := net.ParseIP(address); ip != nil {
			return address, defaultPort, nil
		}
		return "", 0, ErrInvalidAddress
	}

	if colonCount == 1 {
		parts := strings.Split(address, ":")
		host := parts[0]
		portStr := parts[1]
		if host == "" || portStr == "" {
			return "", 0, ErrInvalidAddress
		}
		if !isValidHost(host) {
			return "", 0, ErrInvalidAddress
		}
		portNum, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil || portNum == 0 {
			return "", 0, ErrInvalidAddress
		}
		return host, uint16(portNum), nil
	}

	// Bare hostname or IPv4
	if !isValidHost(address) {
		return "", 0, ErrInvalidAddress
	}

	return address, defaultPort, nil
}

func isValidHost(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return true
	}
	// Check standard domain name characters
	if len(host) > 253 {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		for i, r := range label {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
				return false
			}
			if (r == '-' || r == '_') && (i == 0 || i == len(label)-1) {
				return false
			}
		}
	}
	return true
}

// ResolveIP resolves the IP address of the host. Returns nil if resolution fails.
func ResolveIP(ctx context.Context, host string) *string {
	if ip := net.ParseIP(host); ip != nil {
		s := ip.String()
		return &s
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil
	}

	// Prefer IPv4
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			s := ipv4.String()
			return &s
		}
	}

	s := ips[0].String()
	return &s
}

// ResolveDualStackIPs resolves host into separate lists of IPv6 and IPv4 addresses.
func ResolveDualStackIPs(ctx context.Context, host string) (ipv6 []string, ipv4 []string) {
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			return nil, []string{ip.String()}
		}
		return []string{ip.String()}, nil
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, nil
	}

	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			ipv4 = append(ipv4, v4.String())
		} else {
			ipv6 = append(ipv6, ip.String())
		}
	}
	return ipv6, ipv4
}

// LookupSRV performs SRV lookup for a Minecraft Java server.
func LookupSRV(host string) (*string, *uint16, *SRVRecord) {
	// Only lookup SRV if host is not an IP
	if net.ParseIP(host) != nil {
		return nil, nil, nil
	}

	srv, err := util.LookupSRV(host)
	if err != nil || srv == nil {
		return nil, nil, nil
	}

	target := strings.TrimSuffix(srv.Target, ".")
	port := srv.Port

	record := &SRVRecord{
		Host: target,
		Port: port,
	}

	return &target, &port, record
}
