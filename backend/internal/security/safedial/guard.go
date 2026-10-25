// Package safedial guards outbound TCP connections to hosts a tenant chose
// outside of HTTP — today the IMAP and SMTP servers of a user's mail account.
//
// A mail account form that dials whatever host:port it is handed is a port
// scanner into the server's own network: Postgres, Redis and MinIO sit next to
// the email service, and "connection refused" versus "TLS handshake failed"
// tells an open port from a closed one. safedial applies the address rules of
// security/safehttp to those connections.
//
// There are two layers. Validate resolves a host before it is stored or tested,
// so a private host fails early with a clear error. The boundary is the dialer's
// Control hook: it judges the address the kernel is about to connect to, after
// resolution, so a name that resolves to a public address on the first lookup
// and to 127.0.0.1 on the second (DNS rebinding) is still refused.
//
// Self-hosted customers run their mail server inside their own network. The
// allowlist (EMAIL_DIAL_ALLOWLIST) exempts it: hostnames, IP addresses or CIDR
// ranges, comma-separated. A hostname entry skips the address check for that
// name entirely; an address or range entry lets that address through for any
// host that resolves to it.
package safedial

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"syscall"
	"time"

	"github.com/kmuhub/kmuhub/internal/security/safehttp"
)

// ErrBlockedAddress is returned when a host resolves to, or a connection would
// reach, an address that is neither public nor allowlisted. Use errors.Is: it
// arrives wrapped, from Validate directly and from a dial inside a *net.OpError.
var ErrBlockedAddress = safehttp.ErrBlockedAddress

// Guard holds the allowlist. A nil *Guard is valid and allows public addresses
// only, so a caller that was never wired with one fails closed.
type Guard struct {
	hosts    map[string]struct{}
	prefixes []netip.Prefix
	// lookup replaces DNS resolution in tests; nil means net.DefaultResolver.
	lookup func(ctx context.Context, host string) ([]netip.Addr, error)
}

// New parses a comma-separated allowlist of hostnames, IP addresses and CIDR
// ranges. An empty list yields a guard that allows public addresses only.
func New(allowlist string) (*Guard, error) {
	g := &Guard{hosts: map[string]struct{}{}}
	for _, raw := range strings.Split(allowlist, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if p, err := netip.ParsePrefix(entry); err == nil {
			g.prefixes = append(g.prefixes, p.Masked())
			continue
		}
		if a, err := netip.ParseAddr(entry); err == nil {
			a = a.Unmap()
			g.prefixes = append(g.prefixes, netip.PrefixFrom(a, a.BitLen()))
			continue
		}
		if !validHostname(entry) {
			return nil, fmt.Errorf("safedial: invalid allowlist entry %q (want hostname, IP or CIDR)", entry)
		}
		g.hosts[normalizeHost(entry)] = struct{}{}
	}
	return g, nil
}

// Validate checks host before it is stored or dialled. An IP literal is judged
// directly; a name is resolved and every address it yields must pass. A name
// that does not resolve passes: the dial will fail on its own, and the Control
// hook — not this check — is what keeps a later resolution honest.
func (g *Guard) Validate(ctx context.Context, host string) error {
	host = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(host), "["), "]")
	if g.hostAllowed(host) {
		return nil
	}
	if a, err := netip.ParseAddr(host); err == nil {
		return g.checkAddr(a)
	}
	addrs, err := g.resolve(ctx, host)
	if err != nil {
		return nil
	}
	for _, a := range addrs {
		if err := g.checkAddr(a); err != nil {
			return err
		}
	}
	return nil
}

// Dialer returns a dialer for a connection to host. Unless host is an
// allowlisted name, its Control hook refuses every address that is neither
// public nor inside an allowlisted range.
func (g *Guard) Dialer(host string, timeout time.Duration) *net.Dialer {
	d := &net.Dialer{Timeout: timeout}
	if !g.hostAllowed(host) {
		d.Control = g.control
	}
	return d
}

func (g *Guard) control(network, address string, _ syscall.RawConn) error {
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return fmt.Errorf("%w: network %q", ErrBlockedAddress, network)
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: cannot parse %q", ErrBlockedAddress, address)
	}
	a, err := netip.ParseAddr(host)
	if err != nil {
		// Control receives a resolved literal; anything else means an
		// assumption broke, so refuse rather than guess.
		return fmt.Errorf("%w: %q is not a literal address", ErrBlockedAddress, host)
	}
	return g.checkAddr(a)
}

func (g *Guard) checkAddr(a netip.Addr) error {
	a = a.Unmap()
	if g != nil {
		for _, p := range g.prefixes {
			if p.Contains(a) {
				return nil
			}
		}
	}
	return safehttp.CheckAddr(a)
}

func (g *Guard) hostAllowed(host string) bool {
	if g == nil {
		return false
	}
	_, ok := g.hosts[normalizeHost(host)]
	return ok
}

func (g *Guard) resolve(ctx context.Context, host string) ([]netip.Addr, error) {
	if g != nil && g.lookup != nil {
		return g.lookup(ctx, host)
	}
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
}

func normalizeHost(h string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(h), "."))
}

func validHostname(h string) bool {
	for _, r := range h {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.':
		default:
			return false
		}
	}
	return true
}
