package egresspolicy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"testing"
)

func TestSafeDialerSelectsPublicAddressFromMixedDNSResults(t *testing.T) {
	dialer := &SafeDialer{
		LookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{
				netip.MustParseAddr("10.20.30.40"),
				netip.MustParseAddr("142.250.72.14"),
			}, nil
		},
	}

	got, err := dialer.ResolvePublicAddress(context.Background(), "tcp", "www.google.com:443")
	if err != nil {
		t.Fatalf("ResolvePublicAddress() error = %v", err)
	}
	if got != "142.250.72.14:443" {
		t.Fatalf("ResolvePublicAddress() = %q, want public address", got)
	}
}

func TestSafeDialerRejectsPrivateAndSpecialUseTargets(t *testing.T) {
	tests := []string{
		"127.0.0.1:80",
		"10.0.0.1:443",
		"169.254.169.254:80",
		"100.64.0.1:80",
		"192.0.2.1:443",
		"[::1]:80",
		"[fd00::1]:443",
		"[2001:db8::1]:443",
	}
	dialer := NewSafeDialer()
	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			_, err := dialer.ResolvePublicAddress(context.Background(), "tcp", target)
			if !errors.Is(err, ErrUnsafeTarget) {
				t.Fatalf("ResolvePublicAddress(%q) error = %v, want ErrUnsafeTarget", target, err)
			}
		})
	}
}

func TestSafeDialerRejectsHostnameWhenEveryResolutionIsUnsafe(t *testing.T) {
	dialer := &SafeDialer{
		LookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{
				netip.MustParseAddr("10.20.30.40"),
				netip.MustParseAddr("fd00::10"),
			}, nil
		},
	}

	_, err := dialer.ResolvePublicAddress(context.Background(), "tcp", "internal.example:443")
	if !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("ResolvePublicAddress() error = %v, want ErrUnsafeTarget", err)
	}
}

func TestSafeDialerFallsBackAcrossPinnedPublicAddresses(t *testing.T) {
	attempts := []string{}
	dialer := &SafeDialer{
		LookupNetIP: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{
				netip.MustParseAddr("142.250.72.14"),
				netip.MustParseAddr("142.250.72.15"),
			}, nil
		},
		DialContextFunc: func(_ context.Context, _, address string) (net.Conn, error) {
			attempts = append(attempts, address)
			if len(attempts) == 1 {
				return nil, fmt.Errorf("simulated first-address failure")
			}
			client, server := net.Pipe()
			_ = server.Close()
			return client, nil
		},
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", "www.google.com:443")
	if err != nil {
		t.Fatalf("DialContext() error = %v", err)
	}
	_ = conn.Close()
	if len(attempts) != 2 || attempts[0] != "142.250.72.14:443" || attempts[1] != "142.250.72.15:443" {
		t.Fatalf("unexpected pinned dial attempts: %#v", attempts)
	}
}
