package safehttp_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/security/safehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// get issues a GET through the given client. Every address in these tests is an
// IP literal, so nothing here depends on DNS or on reaching the network: the
// dial filter refuses before connect() is ever called.
func get(t *testing.T, c *http.Client, rawURL string) error {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	resp, err := c.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	return err
}

func TestNew_BlocksNonPublicAddresses(t *testing.T) {
	client := safehttp.New()

	cases := []struct {
		name string
		url  string
	}{
		{"loopback v4", "http://127.0.0.1:8080/"},
		{"loopback v4 alias", "http://127.99.1.2/"},
		{"loopback v6", "http://[::1]:8080/"},
		{"private 10/8", "http://10.0.0.1/"},
		{"private 172.16/12", "http://172.20.3.4/"},
		{"private 192.168/16", "http://192.168.1.1/"},
		{"link-local / cloud metadata", "http://169.254.169.254/latest/meta-data/"},
		{"unique local v6", "http://[fc00::1]/"},
		{"link-local v6", "http://[fe80::1]/"},
		{"unspecified", "http://0.0.0.0/"},
		{"cgnat", "http://100.64.0.1/"},
		{"reserved 240/4", "http://240.0.0.1/"},
		{"broadcast", "http://255.255.255.255/"},
		{"multicast", "http://224.0.0.1/"},
		{"ipv4-mapped loopback", "http://[::ffff:127.0.0.1]/"},
		{"ipv4-mapped private", "http://[::ffff:10.0.0.1]/"},
		{"nat64", "http://[64:ff9b::a00:1]/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := get(t, client, tc.url)
			require.Error(t, err, "connection to %s must not be attempted", tc.url)
			assert.ErrorIs(t, err, safehttp.ErrBlockedAddress)
		})
	}
}

// The production constructor takes no options, so this is the behaviour every
// caller in cmd/ gets. If AllowLoopback ever leaked into it, this fails.
func TestNew_LoopbackBlockedWithoutOption(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := get(t, safehttp.New(), srv.URL)
	require.Error(t, err)
	assert.ErrorIs(t, err, safehttp.ErrBlockedAddress)
}

func TestNew_AllowsPublicAddressAndLoopbackUnderTest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := safehttp.New(safehttp.AllowLoopback())
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "ok", string(body))
}

// A redirect is a second chance to reach the private network, and it is the
// one an SSRF payload actually uses: the first hop looks harmless.
func TestNew_BlocksRedirectIntoPrivateRange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	defer srv.Close()

	err := get(t, safehttp.New(safehttp.AllowLoopback()), srv.URL)
	require.Error(t, err)
	assert.ErrorIs(t, err, safehttp.ErrBlockedAddress)
}

func TestNew_BlocksRedirectToUnsupportedScheme(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "file:///etc/passwd", http.StatusFound)
	}))
	defer srv.Close()

	err := get(t, safehttp.New(safehttp.AllowLoopback()), srv.URL)
	require.Error(t, err)
	assert.ErrorIs(t, err, safehttp.ErrUnsupportedScheme)
}

func TestNew_StopsRedirectLoop(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
	}))
	defer srv.Close()

	err := get(t, safehttp.New(safehttp.AllowLoopback()), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped after")
	assert.LessOrEqual(t, hits, safehttp.DefaultMaxRedirects+1)
}

// Go strips Authorization and Cookie across hosts by itself; a tenant-set
// X-Api-Key is the one it would forward, which is the leak this covers.
func TestNew_DropsCustomHeadersOnCrossHostRedirect(t *testing.T) {
	var got http.Header
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer origin.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, origin.URL, nil)
	require.NoError(t, err)
	req.Header.Set("X-Api-Key", "s3cret")
	req.Header.Set("Authorization", "Bearer s3cret")
	req.Header.Set("User-Agent", "Cosmi-Automation/1.0")

	resp, err := safehttp.New(safehttp.AllowLoopback()).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	require.NotNil(t, got)
	assert.Empty(t, got.Get("X-Api-Key"), "custom header must not survive a cross-host redirect")
	assert.Empty(t, got.Get("Authorization"))
	assert.Equal(t, "Cosmi-Automation/1.0", got.Get("User-Agent"), "harmless headers stay")
}

func TestNew_SameHostRedirectKeepsHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/final" {
			got = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/final", http.StatusFound)
	}))
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	require.NoError(t, err)
	req.Header.Set("X-Api-Key", "s3cret")

	resp, err := safehttp.New(safehttp.AllowLoopback()).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	require.NotNil(t, got)
	assert.Equal(t, "s3cret", got.Get("X-Api-Key"))
}

func TestNew_TimeoutApplies(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	defer close(release)

	client := safehttp.New(safehttp.AllowLoopback(), safehttp.WithTimeout(150*time.Millisecond))

	start := time.Now()
	err := get(t, client, srv.URL)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Less(t, elapsed, 5*time.Second, "the client timeout must fire, not the test framework")
}

func TestNew_ResponseBodyCap(t *testing.T) {
	const limit = 1000

	t.Run("body at the cap is readable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("a", limit)))
		}))
		defer srv.Close()

		client := safehttp.New(safehttp.AllowLoopback(), safehttp.WithMaxResponseBytes(limit))
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close() //nolint:errcheck

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "a body of exactly the cap must not be an error")
		assert.Len(t, body, limit)
	})

	t.Run("one byte over the cap fails", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("a", limit+1)))
		}))
		defer srv.Close()

		client := safehttp.New(safehttp.AllowLoopback(), safehttp.WithMaxResponseBytes(limit))
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close() //nolint:errcheck

		_, err = io.ReadAll(resp.Body)
		require.Error(t, err)
		assert.ErrorIs(t, err, safehttp.ErrResponseTooLarge)
	})
}

func TestCheckURL(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr error
	}{
		{"https ok", "https://example.com/hook", nil},
		{"http ok", "http://example.com/hook", nil},
		{"ftp rejected", "ftp://example.com/x", safehttp.ErrUnsupportedScheme},
		{"file rejected", "file:///etc/passwd", safehttp.ErrUnsupportedScheme},
		{"no host rejected", "http:///path", safehttp.ErrUnsupportedScheme},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := url.Parse(tc.raw)
			require.NoError(t, err)
			err = safehttp.CheckURL(u)
			if tc.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}

	t.Run("credentials in the URL are rejected", func(t *testing.T) {
		u, err := url.Parse("https://user:pass@example.com/hook")
		require.NoError(t, err)
		err = safehttp.CheckURL(u)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "credentials")
	})

	t.Run("nil is rejected", func(t *testing.T) {
		assert.Error(t, safehttp.CheckURL(nil))
	})
}

// A public address must stay reachable — the guard is worthless if it also
// blocks the endpoints tenants legitimately configure. No packet is sent: the
// address is parsed and judged, which is exactly what the dial hook does.
func TestNew_PublicAddressPassesTheFilter(t *testing.T) {
	for _, raw := range []string{"1.1.1.1", "93.184.216.34", "2606:4700:4700::1111"} {
		addr, err := netip.ParseAddr(raw)
		require.NoError(t, err)
		assert.True(t, addr.IsGlobalUnicast() && !addr.IsPrivate(), "%s should be treated as public", raw)
	}

	// End to end: a public literal gets past the filter and fails later, in the
	// dial itself, not with ErrBlockedAddress.
	client := safehttp.New(safehttp.WithTimeout(300 * time.Millisecond))
	err := get(t, client, "http://192.0.2.1:9/") // TEST-NET-1, routable-looking, never answers
	if err != nil {
		assert.False(t, errors.Is(err, safehttp.ErrBlockedAddress),
			"a public address must fail on connect, not on the filter: %v", err)
	}
}
