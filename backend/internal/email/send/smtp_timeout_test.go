package send

import (
	"net"
	"testing"
	"time"
)

// TestSendSMTPStartTLS_ExchangeDeadline proves the SMTP exchange deadline bounds
// a server that accepts the TCP connection but never sends its greeting. Without
// the deadline, smtp.NewClient would block reading the greeting forever and leak
// the calling goroutine.
func TestSendSMTPStartTLS_ExchangeDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// Accept the connection and hold it open without ever writing the 220
	// greeting, simulating a stalled SMTP server.
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 1)
		_, _ = c.Read(buf) // unblocks once the client closes after its deadline
	}()

	// Shrink the timeouts so the test is fast and deterministic.
	origDial, origExchange := smtpDialTimeout, smtpExchangeDeadline
	smtpDialTimeout, smtpExchangeDeadline = 2*time.Second, 200*time.Millisecond
	defer func() { smtpDialTimeout, smtpExchangeDeadline = origDial, origExchange }()

	s := &Service{}
	done := make(chan error, 1)
	go func() {
		done <- s.sendSMTPStartTLS(ln.Addr().String(), nil, "127.0.0.1", "from@example.test", []string{"to@example.test"}, []byte("body"))
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error from the stalled SMTP server, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("sendSMTPStartTLS did not return within the exchange deadline — timeout not enforced")
	}
}
