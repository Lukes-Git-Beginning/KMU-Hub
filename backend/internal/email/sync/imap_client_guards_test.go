package sync

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIMAPClient_NilClientGuards proves every wrapper method that talks to the
// underlying imapclient.Client refuses to run against an unconnected
// IMAPClient (c.client == nil) instead of panicking on a nil pointer
// dereference.
func TestIMAPClient_NilClientGuards(t *testing.T) {
	c := NewIMAPClient(uuid.New(), slog.Default())

	t.Run("Login", func(t *testing.T) {
		err := c.Login("user", "pass")
		assert.ErrorIs(t, err, ErrIMAPConnectionLost)
	})

	t.Run("SelectFolder", func(t *testing.T) {
		status, err := c.SelectFolder("INBOX")
		assert.Nil(t, status)
		assert.ErrorIs(t, err, ErrIMAPConnectionLost)
	})

	t.Run("FetchHeaders", func(t *testing.T) {
		envelopes, err := c.FetchHeaders(imap.UIDSet{})
		assert.Nil(t, envelopes)
		assert.ErrorIs(t, err, ErrIMAPConnectionLost)
	})

	t.Run("FetchBody", func(t *testing.T) {
		body, err := c.FetchBody(42)
		assert.Nil(t, body)
		assert.ErrorIs(t, err, ErrIMAPConnectionLost)
	})

	t.Run("SetFlags", func(t *testing.T) {
		err := c.SetFlags(42, []imap.Flag{imap.FlagSeen}, true)
		assert.ErrorIs(t, err, ErrIMAPConnectionLost)
	})

	t.Run("Idle", func(t *testing.T) {
		err := c.Idle(context.Background())
		assert.ErrorIs(t, err, ErrIMAPConnectionLost)
	})

	t.Run("HasIDLE", func(t *testing.T) {
		assert.False(t, c.HasIDLE())
	})

	t.Run("Noop", func(t *testing.T) {
		err := c.Noop()
		assert.ErrorIs(t, err, ErrIMAPConnectionLost)
	})

	t.Run("ListFolders", func(t *testing.T) {
		folders, err := c.ListFolders()
		assert.Nil(t, folders)
		assert.ErrorIs(t, err, ErrIMAPConnectionLost)
	})

	t.Run("Close", func(t *testing.T) {
		assert.NoError(t, c.Close())
	})
}

// TestConnect_ImmediateCloseWrapsError proves that a server which accepts the
// TCP connection and closes it right away -- no greeting, no TLS ServerHello --
// surfaces as an ErrIMAPConnectionLost-wrapped error rather than hanging.
// TestConnect_HandshakeDeadline (imap_timeout_test.go) already covers the
// "server stays silent" hang case for the STARTTLS branch; this covers the
// distinct "peer closes immediately" failure on both the direct-TLS
// (useSSL/port 993) and STARTTLS (plain) branches of Connect.
func TestConnect_ImmediateCloseWrapsError(t *testing.T) {
	origDial, origHandshake := imapDialTimeout, imapHandshakeDeadline
	imapDialTimeout, imapHandshakeDeadline = 2*time.Second, 500*time.Millisecond
	defer func() { imapDialTimeout, imapHandshakeDeadline = origDial, origHandshake }()

	listenAndCloseImmediately := func(t *testing.T) (string, int) {
		t.Helper()
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = ln.Close() })

		go func() {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}()

		host, portStr, err := net.SplitHostPort(ln.Addr().String())
		require.NoError(t, err)
		port, err := strconv.Atoi(portStr)
		require.NoError(t, err)
		return host, port
	}

	t.Run("SSL branch: TLS handshake fails against a plain closed connection", func(t *testing.T) {
		host, port := listenAndCloseImmediately(t)

		c := NewIMAPClient(uuid.New(), slog.Default())
		done := make(chan error, 1)
		go func() { done <- c.Connect(host, port, true) }()

		select {
		case err := <-done:
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrIMAPConnectionLost)
		case <-time.After(3 * time.Second):
			t.Fatal("Connect did not return promptly after the peer closed the connection")
		}
	})

	t.Run("non-SSL branch: STARTTLS negotiation fails against a plain closed connection", func(t *testing.T) {
		host, port := listenAndCloseImmediately(t)

		c := NewIMAPClient(uuid.New(), slog.Default())
		done := make(chan error, 1)
		go func() { done <- c.Connect(host, port, false) }()

		select {
		case err := <-done:
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrIMAPConnectionLost)
		case <-time.After(3 * time.Second):
			t.Fatal("Connect did not return promptly after the peer closed the connection")
		}
	})
}
