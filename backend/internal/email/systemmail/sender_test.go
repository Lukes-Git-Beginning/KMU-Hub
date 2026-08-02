package systemmail

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/email/send"
	"github.com/kmuhub/kmuhub/internal/testutil/fakesmtp"
)

// A configured sender must actually put the message on the wire: envelope
// sender, every recipient, and the attachment inside the DATA payload.
func TestSender_Send_DeliversWithAttachment(t *testing.T) {
	srv := fakesmtp.Start(t)
	s := New(Config{Host: srv.Host, Port: srv.Port, From: "noreply@zentria.test"})

	payload := []byte("id,value\n1,42\n")
	err := s.Send(context.Background(), Message{
		To: []send.EmailAddress{
			{Name: "Erste Empfaengerin", Email: "a@example.test"},
			{Email: "b@example.test"},
		},
		Subject:  "[Cosmi] Umsatzbericht",
		BodyText: "Anbei der Bericht.",
		BodyHTML: "<p>Anbei der Bericht.</p>",
		Attachments: []send.AttachmentData{{
			Filename:    "umsatzbericht.csv",
			ContentType: "text/csv; charset=utf-8",
			Data:        bytes.NewReader(payload),
			Size:        int64(len(payload)),
		}},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	sess := srv.WaitSession(t, 5*time.Second)
	if sess.MailFrom != "noreply@zentria.test" {
		t.Errorf("MAIL FROM = %q, want noreply@zentria.test", sess.MailFrom)
	}
	if len(sess.RcptTo) != 2 || sess.RcptTo[0] != "a@example.test" || sess.RcptTo[1] != "b@example.test" {
		t.Errorf("RCPT TO = %v, want both recipients", sess.RcptTo)
	}
	if !strings.Contains(sess.Data, "umsatzbericht.csv") {
		t.Error("DATA payload does not mention the attachment filename")
	}
	if !strings.Contains(sess.Data, "Umsatzbericht") {
		t.Error("DATA payload does not carry the subject")
	}
}

// An unconfigured sender must fail loudly. Reporting success without a host is
// how a scheduled report silently stops being delivered.
func TestSender_Send_NotConfigured(t *testing.T) {
	s := New(Config{})
	if s.Configured() {
		t.Fatal("Configured() = true for empty host")
	}
	err := s.Send(context.Background(), Message{
		To:      []send.EmailAddress{{Email: "a@example.test"}},
		Subject: "x",
	})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Send error = %v, want ErrNotConfigured", err)
	}
}

func TestSender_Send_NoRecipients(t *testing.T) {
	s := New(Config{Host: "localhost", Port: 25, From: "noreply@zentria.test"})
	if err := s.Send(context.Background(), Message{Subject: "x"}); err == nil {
		t.Fatal("expected an error for a message without recipients")
	}
}

// A server that accepts the connection but never greets must not hang the
// caller: the exchange deadline has to break the read.
func TestSender_Send_StalledServerHitsDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 1)
		_, _ = c.Read(buf)
	}()

	origDial, origExchange := dialTimeout, exchangeDeadline
	dialTimeout, exchangeDeadline = 2*time.Second, 200*time.Millisecond
	defer func() { dialTimeout, exchangeDeadline = origDial, origExchange }()

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port := 0
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	s := New(Config{Host: host, Port: port, From: "noreply@zentria.test"})
	done := make(chan error, 1)
	go func() {
		done <- s.Send(context.Background(), Message{
			To:      []send.EmailAddress{{Email: "a@example.test"}},
			Subject: "x",
		})
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error from the stalled SMTP server, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Send did not return within the exchange deadline")
	}
}
