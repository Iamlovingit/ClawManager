package egresspolicy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

var ErrUnsafeTarget = errors.New("target resolves to a private, internal, or special-use address")

type LookupNetIPFunc func(ctx context.Context, network, host string) ([]netip.Addr, error)

// SafeDialer resolves a hostname once, rejects every unsafe result, and then
// dials a selected public IP directly. This prevents DNS rebinding between
// policy evaluation and the outbound connection.
type SafeDialer struct {
	LookupNetIP     LookupNetIPFunc
	DialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)
	Dialer          net.Dialer
}

func NewSafeDialer() *SafeDialer {
	resolver := net.DefaultResolver
	return &SafeDialer{
		LookupNetIP: resolver.LookupNetIP,
		Dialer: net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	}
}

func (d *SafeDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	resolved, err := d.ResolvePublicAddresses(ctx, network, address)
	if err != nil {
		return nil, err
	}
	dial := d.DialContextFunc
	if dial == nil {
		dial = d.Dialer.DialContext
	}
	var errs []error
	for _, target := range resolved {
		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		conn, dialErr := dial(attemptCtx, network, target)
		cancel()
		if dialErr == nil {
			return conn, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", target, dialErr))
	}
	return nil, fmt.Errorf("dial public target %q: %w", address, errors.Join(errs...))
}

func (d *SafeDialer) ResolvePublicAddress(ctx context.Context, network, address string) (string, error) {
	addresses, err := d.ResolvePublicAddresses(ctx, network, address)
	if err != nil {
		return "", err
	}
	return addresses[0], nil
}

func (d *SafeDialer) ResolvePublicAddresses(ctx context.Context, network, address string) ([]string, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return nil, fmt.Errorf("invalid target address %q: %w", address, err)
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil || port == "0" {
		return nil, fmt.Errorf("invalid target port %q", port)
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return nil, fmt.Errorf("target host is required")
	}

	if literal, err := netip.ParseAddr(host); err == nil {
		literal = literal.Unmap()
		if !IsPublicAddress(literal) {
			return nil, fmt.Errorf("%w: %s", ErrUnsafeTarget, literal)
		}
		return []string{net.JoinHostPort(literal.String(), port)}, nil
	}

	lookup := d.LookupNetIP
	if lookup == nil {
		lookup = net.DefaultResolver.LookupNetIP
	}
	addresses, err := lookup(ctx, lookupNetwork(network), host)
	if err != nil {
		return nil, fmt.Errorf("resolve target %q: %w", host, err)
	}
	publicV4 := make([]string, 0, len(addresses))
	publicV6 := make([]string, 0, len(addresses))
	seen := map[netip.Addr]struct{}{}
	for _, address := range addresses {
		address = address.Unmap()
		if !IsPublicAddress(address) {
			continue
		}
		if _, ok := seen[address]; ok {
			continue
		}
		seen[address] = struct{}{}
		target := net.JoinHostPort(address.String(), port)
		if address.Is4() {
			publicV4 = append(publicV4, target)
		} else {
			publicV6 = append(publicV6, target)
		}
	}
	public := append(publicV4, publicV6...)
	if len(public) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrUnsafeTarget, host)
	}
	return public, nil
}

func lookupNetwork(network string) string {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "tcp4":
		return "ip4"
	case "tcp6":
		return "ip6"
	default:
		return "ip"
	}
}

var specialUsePrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2001:2::/48"),
}

func IsPublicAddress(address netip.Addr) bool {
	if !address.IsValid() {
		return false
	}
	address = address.Unmap()
	if !address.IsGlobalUnicast() ||
		address.IsPrivate() ||
		address.IsLoopback() ||
		address.IsLinkLocalUnicast() ||
		address.IsLinkLocalMulticast() ||
		address.IsMulticast() ||
		address.IsUnspecified() {
		return false
	}
	for _, prefix := range specialUsePrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}
