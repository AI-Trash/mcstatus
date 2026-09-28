package resolver

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/mcstatus-io/mcutil/v4/util"
)

var ErrInvalidAddress = errors.New("Invalid address value")

type SRVRecord struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

// ParseAddress parses address string into host and port using mcutil/v4/util.ParseAddress.
func ParseAddress(address string, defaultPort uint16) (string, uint16, error) {
	address = strings.TrimSpace(address)
	if address == "" || strings.ContainsAny(address, " \t\r\n/\\!@#$%^&*()=+~`\"'<>?,;") {
		return "", 0, ErrInvalidAddress
	}

	// Bare IPv6 check e.g. ::1
	if ip := net.ParseIP(address); ip != nil {
		return ip.String(), defaultPort, nil
	}

	host, port, err := util.ParseAddress(address)
	if err != nil || host == "" {
		return "", 0, ErrInvalidAddress
	}

	// Strip brackets for IPv6 e.g. [::1] -> ::1
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}

	finalPort := defaultPort
	if port != nil {
		if *port == 0 {
			return "", 0, ErrInvalidAddress
		}
		finalPort = *port
	}

	return host, finalPort, nil
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
