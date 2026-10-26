package sync

import (
	"log/slog"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/security/safedial"
)

// loopbackGuard allowlists 127.0.0.1 so a test can reach its fake IMAP server
// the way a self-hosted customer allowlists an internal mail server.
func loopbackGuard(t *testing.T) *safedial.Guard {
	t.Helper()
	g, err := safedial.New("127.0.0.1")
	require.NoError(t, err)
	return g
}

// TestConnect_RefusesInternalAddressWithoutAllowlist proves the sync dial runs
// through the guard: a listener on loopback is never reached, on either branch,
// unless the address is allowlisted.
func TestConnect_RefusesInternalAddressWithoutAllowlist(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	accepted := make(chan struct{}, 2)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			accepted <- struct{}{}
			_ = c.Close()
		}
	}()
	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	for _, ssl := range []bool{true, false} {
		c := NewIMAPClient(uuid.New(), slog.Default())
		err := c.Connect(host, port, ssl)
		assert.ErrorIs(t, err, ErrIMAPConnectionLost, "ssl=%v", ssl)
		assert.ErrorIs(t, err, safedial.ErrBlockedAddress, "ssl=%v", ssl)
	}
	select {
	case <-accepted:
		t.Fatal("the guard let a connection to loopback through")
	case <-time.After(100 * time.Millisecond):
	}
}
