package safedial

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"
)

// stubResolver maps names to addresses the way a container network would:
// "postgres" is a Docker service name on a bridge network.
func stubResolver(ctx context.Context, host string) ([]netip.Addr, error) {
	switch host {
	case "postgres":
		return []netip.Addr{netip.MustParseAddr("172.18.0.5")}, nil
	case "mail.example.com":
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	case "split.example.com":
		return []netip.Addr{netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("10.0.0.7")}, nil
	case "mail.intern":
		return []netip.Addr{netip.MustParseAddr("10.20.0.4")}, nil
	}
	return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
}

func guardWith(t *testing.T, allowlist string) *Guard {
	t.Helper()
	g, err := New(allowlist)
	if err != nil {
		t.Fatalf("New(%q): %v", allowlist, err)
	}
	g.lookup = stubResolver
	return g
}

func TestValidate_RejectsInternalAddresses(t *testing.T) {
	g := guardWith(t, "")
	for _, host := range []string{
		"127.0.0.1",
		"10.0.0.1",
		"10.255.1.2",
		"172.16.0.1",
		"172.31.255.254",
		"192.168.1.10",
		"169.254.169.254",
		"0.0.0.0",
		"::1",
		"[::1]",
		"::ffff:127.0.0.1",
		"fd00::1",
		"postgres",
		"split.example.com",
	} {
		if err := g.Validate(context.Background(), host); !errors.Is(err, ErrBlockedAddress) {
			t.Errorf("Validate(%q) = %v, want ErrBlockedAddress", host, err)
		}
	}
}

func TestValidate_AllowsPublicAndAllowlisted(t *testing.T) {
	g := guardWith(t, "10.0.0.0/8, mail.intern., 192.168.1.10, Postgres")
	for _, host := range []string{
		"93.184.216.34",
		"mail.example.com",
		"10.1.2.3",            // CIDR entry
		"192.168.1.10",        // single-address entry
		"mail.intern",         // hostname entry (trailing dot in the list)
		"MAIL.INTERN",         // hostnames compare case-insensitively
		"postgres",            // a hostname entry exempts even a Docker name
		"does-not-exist.test", // unresolvable: the dial fails on its own
	} {
		if err := g.Validate(context.Background(), host); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", host, err)
		}
	}
	// An address next to an allowlisted single address stays blocked.
	if err := g.Validate(context.Background(), "192.168.1.11"); !errors.Is(err, ErrBlockedAddress) {
		t.Errorf("Validate(192.168.1.11) = %v, want ErrBlockedAddress", err)
	}
}

func TestNilGuard_AllowsPublicOnly(t *testing.T) {
	var g *Guard
	if err := g.Validate(context.Background(), "127.0.0.1"); !errors.Is(err, ErrBlockedAddress) {
		t.Fatalf("nil guard Validate(127.0.0.1) = %v, want ErrBlockedAddress", err)
	}
	if err := g.Validate(context.Background(), "93.184.216.34"); err != nil {
		t.Fatalf("nil guard Validate(public) = %v, want nil", err)
	}
}

func TestNew_RejectsMalformedEntries(t *testing.T) {
	for _, list := range []string{"10.0.0.0/33", "mail server", "host:25", "http://mail.intern"} {
		if _, err := New(list); err == nil {
			t.Errorf("New(%q) = nil error, want rejection", list)
		}
	}
}

// The dial-time check is the boundary: it runs on the address actually
// connected to, so it is exercised against a real listener rather than the
// resolver stub.
func TestDialer_BlocksLoopbackUnlessAllowlisted(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	addr := ln.Addr().String()
	_, port, _ := net.SplitHostPort(addr)

	cases := []struct {
		name      string
		allowlist string
		host      string
		wantBlock bool
	}{
		{"nil allowlist", "", "127.0.0.1", true},
		{"hostname resolving to loopback", "", "localhost", true},
		{"other range allowlisted", "10.0.0.0/8", "127.0.0.1", true},
		{"address allowlisted", "127.0.0.1", "127.0.0.1", false},
		{"range allowlisted", "127.0.0.0/8", "127.0.0.1", false},
		{"hostname allowlisted", "localhost", "localhost", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g, err := New(tc.allowlist)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			conn, err := g.Dialer(tc.host, 2*time.Second).Dial("tcp4", net.JoinHostPort(tc.host, port))
			if conn != nil {
				_ = conn.Close()
			}
			if tc.wantBlock && !errors.Is(err, ErrBlockedAddress) {
				t.Fatalf("dial %s = %v, want ErrBlockedAddress", tc.host, err)
			}
			if !tc.wantBlock && err != nil {
				t.Fatalf("dial %s = %v, want success", tc.host, err)
			}
		})
	}

	var nilGuard *Guard
	if _, err := nilGuard.Dialer("127.0.0.1", time.Second).Dial("tcp", addr); !errors.Is(err, ErrBlockedAddress) {
		t.Fatalf("nil guard dial = %v, want ErrBlockedAddress", err)
	}
}
