// Package safehttp builds outbound HTTP clients for requests whose target a
// tenant chose: automation actions, form webhooks, anything where a URL from
// the database decides where the server connects.
//
// The server sits inside a private network next to Postgres, Redis, MinIO, the
// gRPC services and — on a cloud host — a metadata endpoint that hands out
// credentials to anyone who asks. A plain http.Client would happily reach all
// of them, which turns "let the tenant configure a webhook" into a read
// primitive on our own infrastructure (SSRF).
//
// A client from New reaches public unicast addresses only. The check runs in
// the dialer's Control hook, after DNS resolution and immediately before
// connect, so a hostname that resolves to a public address on the first lookup
// and to 127.0.0.1 on the second (DNS rebinding) is caught on the connection
// that matters rather than on the name. For the same reason every redirect hop
// is re-checked: each one opens its own connection.
package safehttp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"syscall"
	"testing"
	"time"
)

// ErrBlockedAddress is returned when a request would leave the public internet
// and enter a private, loopback, link-local or otherwise non-routable range.
// It arrives wrapped in a *url.Error, so use errors.Is.
var ErrBlockedAddress = errors.New("safehttp: destination address is not a public address")

// ErrResponseTooLarge is returned by a response body Read once the peer has
// sent more than the client's size cap. It exists so a hostile or broken
// endpoint cannot exhaust memory by answering a small request with a huge body.
var ErrResponseTooLarge = errors.New("safehttp: response body exceeds size limit")

// ErrUnsupportedScheme is returned for any URL that is not http or https —
// including a redirect into file:// or gopher://.
var ErrUnsupportedScheme = errors.New("safehttp: only http and https are allowed")

const (
	// DefaultTimeout bounds a whole request including redirects and body read.
	// It stays under the automation engine's 10s per-action budget so a slow
	// endpoint surfaces as our own error rather than as a cancelled context.
	DefaultTimeout = 8 * time.Second

	// DefaultMaxResponseBytes caps how much of a response we read.
	DefaultMaxResponseBytes int64 = 1 << 20 // 1 MiB

	// DefaultMaxRedirects caps redirect chains. Go's own default is 10.
	DefaultMaxRedirects = 5

	dialTimeout = 5 * time.Second
)

// blockedPrefixes covers the ranges netip's own predicates do not.
//
// IsGlobalUnicast already rejects loopback, link-local (which is where the
// 169.254.169.254 metadata endpoint lives), multicast and the unspecified
// address; IsPrivate rejects 10/8, 172.16/12, 192.168/16 and fc00::/7.
var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), // CGNAT — a hoster's internal fabric
	netip.MustParsePrefix("192.0.0.0/24"),  // IETF protocol assignments
	netip.MustParsePrefix("198.18.0.0/15"), // benchmarking
	netip.MustParsePrefix("240.0.0.0/4"),   // reserved, includes 255.255.255.255
	netip.MustParsePrefix("64:ff9b::/96"),  // NAT64 — embeds an IPv4 destination
	netip.MustParsePrefix("2002::/16"),     // 6to4 — likewise
}

type options struct {
	timeout          time.Duration
	maxResponseBytes int64
	maxRedirects     int
	allowLoopback    bool
}

// Option adjusts a client built by New.
type Option func(*options)

// WithTimeout replaces DefaultTimeout. Values <= 0 are ignored.
func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		if d > 0 {
			o.timeout = d
		}
	}
}

// WithMaxResponseBytes replaces DefaultMaxResponseBytes. Values <= 0 are ignored.
func WithMaxResponseBytes(n int64) Option {
	return func(o *options) {
		if n > 0 {
			o.maxResponseBytes = n
		}
	}
}

// AllowLoopback lets requests reach 127.0.0.0/8 and ::1 so that a test can talk
// to an httptest server.
//
// It is inert outside `go test`: the flag is only honoured when
// testing.Testing() reports true. A production path therefore cannot switch the
// address filter off, neither deliberately nor by copying a line out of a test.
func AllowLoopback() Option {
	return func(o *options) {
		o.allowLoopback = testing.Testing()
	}
}

// New returns an http.Client that can only reach public addresses.
//
// The guarantees ride on the returned client itself — dial filter, redirect
// filter, timeout and body cap are all part of it — so callers get them by
// using it as an ordinary *http.Client and cannot opt out of one while keeping
// the others.
func New(opts ...Option) *http.Client {
	o := options{
		timeout:          DefaultTimeout,
		maxResponseBytes: DefaultMaxResponseBytes,
		maxRedirects:     DefaultMaxRedirects,
	}
	for _, apply := range opts {
		apply(&o)
	}

	dialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
		Control:   controlFunc(o.allowLoopback),
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   dialTimeout,
		ResponseHeaderTimeout: o.timeout,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   2,
		// A tenant URL is not a reason to trust a proxy from the process
		// environment, and honouring one would route around the dial filter.
		Proxy: nil,
	}

	return &http.Client{
		Timeout:       o.timeout,
		Transport:     &limitTransport{base: transport, max: o.maxResponseBytes},
		CheckRedirect: checkRedirect(o.maxRedirects),
	}
}

// controlFunc rejects a connection after the name has been resolved and before
// the socket is connected. That ordering is the whole point: it sees the
// address the kernel will actually talk to, not the one a previous lookup
// returned.
func controlFunc(allowLoopback bool) func(network, address string, _ syscall.RawConn) error {
	return func(network, address string, _ syscall.RawConn) error {
		switch network {
		case "tcp", "tcp4", "tcp6":
		default:
			return fmt.Errorf("%w: network %q", ErrBlockedAddress, network)
		}

		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("%w: cannot parse %q", ErrBlockedAddress, address)
		}
		addr, err := netip.ParseAddr(host)
		if err != nil {
			// Control is documented to receive a resolved literal. Anything
			// else means an assumption of ours broke, so refuse rather than
			// guess.
			return fmt.Errorf("%w: %q is not a literal address", ErrBlockedAddress, host)
		}
		return checkAddr(addr, allowLoopback)
	}
}

func checkAddr(addr netip.Addr, allowLoopback bool) error {
	if addr.Is4In6() {
		// ::ffff:127.0.0.1 must be judged as 127.0.0.1, not as an IPv6 address
		// that happens to pass the v6 predicates.
		addr = addr.Unmap()
	}

	if allowLoopback && addr.IsLoopback() {
		return nil
	}

	if !addr.IsGlobalUnicast() || addr.IsPrivate() {
		return fmt.Errorf("%w: %s", ErrBlockedAddress, addr)
	}
	for _, p := range blockedPrefixes {
		if p.Contains(addr) {
			return fmt.Errorf("%w: %s", ErrBlockedAddress, addr)
		}
	}
	return nil
}

// CheckURL validates a target before a request is built, so a bad URL fails
// with a precise message instead of a dial error. It is not the security
// boundary — that is the dial filter, because a hostname says nothing about
// the address it will resolve to.
func CheckURL(u *url.URL) error {
	if u == nil {
		return fmt.Errorf("%w: empty URL", ErrUnsupportedScheme)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: got %q", ErrUnsupportedScheme, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("%w: URL has no host", ErrUnsupportedScheme)
	}
	if u.User != nil {
		// Credentials in the URL would be written to every log line that
		// records the target, and forwarded on a redirect.
		return errors.New("safehttp: credentials in the URL are not allowed")
	}
	return nil
}

// checkRedirect bounds the chain and re-validates each hop. The address filter
// already applies to the new connection; what is added here is the scheme check
// and the removal of caller-set headers when the host changes — Go strips
// Authorization and Cookie by itself, but an X-Api-Key a tenant configured
// would otherwise be handed to whatever host the redirect names.
func checkRedirect(maxRedirects int) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("safehttp: stopped after %d redirects", maxRedirects)
		}
		if err := CheckURL(req.URL); err != nil {
			return err
		}
		if len(via) > 0 && req.URL.Host != via[0].URL.Host {
			for name := range req.Header {
				if !safeOnRedirect(name) {
					req.Header.Del(name)
				}
			}
		}
		return nil
	}
}

func safeOnRedirect(name string) bool {
	switch http.CanonicalHeaderKey(name) {
	case "Accept", "Accept-Encoding", "Accept-Language", "Content-Type", "User-Agent", "Referer":
		return true
	default:
		return false
	}
}

// limitTransport caps every response body, so no caller can forget to.
type limitTransport struct {
	base http.RoundTripper
	max  int64
}

func (t *limitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Body = &limitedBody{rc: resp.Body, max: t.max}
	return resp, nil
}

// limitedBody fails the read instead of truncating it. A silently short body
// would reach a JSON decoder as a parse error somewhere far from the cause.
type limitedBody struct {
	rc   io.ReadCloser
	max  int64
	read int64
}

func (b *limitedBody) Read(p []byte) (int, error) {
	if b.read > b.max {
		return 0, ErrResponseTooLarge
	}
	// One byte of headroom past the cap is what tells a body of exactly max
	// bytes apart from one that is too long; without it the legal case would
	// fail on its last read.
	if room := b.max - b.read + 1; int64(len(p)) > room {
		p = p[:room]
	}
	n, err := b.rc.Read(p)
	b.read += int64(n)
	if b.read > b.max {
		return 0, ErrResponseTooLarge
	}
	return n, err
}

func (b *limitedBody) Close() error { return b.rc.Close() }
