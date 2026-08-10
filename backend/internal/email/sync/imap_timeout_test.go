package sync

import (
	"errors"
	"log/slog"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestConnect_HandshakeDeadline proves the handshake deadline bounds a server
// that accepts the TCP connection but never speaks (no greeting, no STARTTLS
// response). Without it, imapclient.DialStartTLS's synchronous STARTTLS
// negotiation blocks forever and leaks the sync worker's goroutine.
func TestConnect_HandshakeDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// Accept the connection and hold it open without ever writing the IMAP
	// greeting, simulating a stalled mailbox server.
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 4096)
		for {
			if _, err := c.Read(buf); err != nil {
				return
			}
		}
	}()

	// Shrink the timeouts so the test is fast and deterministic.
	origDial, origHandshake := imapDialTimeout, imapHandshakeDeadline
	imapDialTimeout, imapHandshakeDeadline = 2*time.Second, 200*time.Millisecond
	defer func() { imapDialTimeout, imapHandshakeDeadline = origDial, origHandshake }()

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	c := NewIMAPClient(uuid.New(), slog.Default())
	done := make(chan error, 1)
	go func() {
		done <- c.Connect(host, port, false) // false -> STARTTLS path
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error from the stalled IMAP server, got nil")
		}
		if !errors.Is(err, ErrIMAPConnectionLost) {
			t.Fatalf("expected ErrIMAPConnectionLost, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Connect did not return within the handshake deadline — timeout not enforced")
	}
}
