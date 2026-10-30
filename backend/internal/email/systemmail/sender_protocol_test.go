package systemmail

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/email/send"
)

// scriptedSMTP is a one-connection SMTP server whose replies can be overridden
// per command, so each failure path of Sender.deliver can be provoked. The
// shared fakesmtp server only speaks the happy path.
type scriptedSMTP struct {
	host string
	port int

	greeting string            // default "220 scripted.test ESMTP"
	ehlo     []string          // extension lines, e.g. "STARTTLS", "AUTH PLAIN"
	reply    map[string]string // verb (EHLO, AUTH, MAIL, RCPT, DATA, EOD, QUIT) -> full reply line

	mu       sync.Mutex
	commands []string
	data     string
}

func startScripted(t *testing.T, s *scriptedSMTP) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	s.host = host
	s.port, _ = strconv.Atoi(portStr)
	if s.greeting == "" {
		s.greeting = "220 scripted.test ESMTP"
	}
	go s.serve(ln)
}

func (s *scriptedSMTP) answer(verb, def string) string {
	if r, ok := s.reply[verb]; ok {
		return r
	}
	return def
}

func (s *scriptedSMTP) serve(ln net.Listener) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	w := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }
	r := bufio.NewReader(conn)

	w(s.greeting)
	if !strings.HasPrefix(s.greeting, "220") {
		return
	}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		s.mu.Lock()
		s.commands = append(s.commands, line)
		s.mu.Unlock()
		up := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(up, "EHLO"):
			lines := append([]string{"scripted.test"}, s.ehlo...)
			for i, l := range lines {
				sep := "-"
				if i == len(lines)-1 {
					sep = " "
				}
				w("250" + sep + l)
			}
		case strings.HasPrefix(up, "STARTTLS"):
			w(s.answer("STARTTLS", "454 TLS not available"))
		case strings.HasPrefix(up, "AUTH"):
			w(s.answer("AUTH", "235 ok"))
		case strings.HasPrefix(up, "MAIL FROM"):
			w(s.answer("MAIL", "250 OK"))
		case strings.HasPrefix(up, "RCPT TO"):
			w(s.answer("RCPT", "250 OK"))
		case up == "DATA":
			reply := s.answer("DATA", "354 go ahead")
			w(reply)
			if !strings.HasPrefix(reply, "354") {
				continue
			}
			var body strings.Builder
			for {
				dl, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dl, "\r\n") == "." {
					break
				}
				body.WriteString(dl)
			}
			s.mu.Lock()
			s.data = body.String()
			s.mu.Unlock()
			w(s.answer("EOD", "250 queued"))
		case up == "QUIT":
			w("221 bye")
			return
		default:
			w("250 OK")
		}
	}
}

func (s *scriptedSMTP) cmds() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...)
}

func (s *scriptedSMTP) seen(prefix string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.commands {
		if strings.HasPrefix(strings.ToUpper(c), prefix) {
			return c, true
		}
	}
	return "", false
}

func (s *scriptedSMTP) sender(user, pass string) *Sender {
	return New(Config{Host: s.host, Port: s.port, Username: user, Password: pass, From: "noreply@zentria.test"})
}

func oneRecipient() Message {
	return Message{To: []send.EmailAddress{{Email: "a@example.test"}}, Subject: "x", BodyText: "hi"}
}

// Each SMTP stage that the server can refuse must surface as an error naming
// that stage, never as a silent success.
func TestSender_Send_ServerRefusals(t *testing.T) {
	tests := []struct {
		name     string
		srv      *scriptedSMTP
		user     string
		wantErr  string
		wantSeen string // command the server must have received before the refusal
	}{
		{
			name:    "greeting refused",
			srv:     &scriptedSMTP{greeting: "554 no service"},
			wantErr: "smtp client",
		},
		{
			name:    "starttls offered but refused",
			srv:     &scriptedSMTP{ehlo: []string{"STARTTLS"}},
			wantErr: "smtp starttls",
		},
		{
			name:     "auth rejected",
			srv:      &scriptedSMTP{ehlo: []string{"AUTH PLAIN"}, reply: map[string]string{"AUTH": "535 bad credentials"}},
			user:     "sys",
			wantErr:  "smtp auth",
			wantSeen: "AUTH PLAIN",
		},
		{
			name:     "sender rejected",
			srv:      &scriptedSMTP{reply: map[string]string{"MAIL": "550 sender not allowed"}},
			wantErr:  "smtp mail from",
			wantSeen: "MAIL FROM",
		},
		{
			name:     "recipient rejected",
			srv:      &scriptedSMTP{reply: map[string]string{"RCPT": "550 no such user"}},
			wantErr:  "smtp rcpt a@example.test",
			wantSeen: "RCPT TO",
		},
		{
			name:     "data refused",
			srv:      &scriptedSMTP{reply: map[string]string{"DATA": "554 transaction failed"}},
			wantErr:  "smtp data",
			wantSeen: "DATA",
		},
		{
			name:    "message rejected after end of data",
			srv:     &scriptedSMTP{reply: map[string]string{"EOD": "554 spam suspected"}},
			wantErr: "smtp close data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startScripted(t, tt.srv)
			err := tt.srv.sender(tt.user, "pw").Send(context.Background(), oneRecipient())
			if err == nil {
				t.Fatal("Send returned nil although the server refused")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), "systemmail: send to [a@example.test]") {
				t.Errorf("error = %q, want the recipient list in the wrap", err)
			}
			if tt.wantSeen != "" {
				if _, ok := tt.srv.seen(tt.wantSeen); !ok {
					t.Errorf("server never received %q, commands=%v", tt.wantSeen, tt.srv.cmds())
				}
			}
		})
	}
}

// A refused recipient must stop the exchange before DATA: no body goes out for
// a message the server already declined.
func TestSender_Send_RejectedRecipientSendsNoData(t *testing.T) {
	srv := &scriptedSMTP{reply: map[string]string{"RCPT": "550 no such user"}}
	startScripted(t, srv)
	_ = srv.sender("", "").Send(context.Background(), oneRecipient())
	if _, ok := srv.seen("DATA"); ok {
		t.Error("DATA was sent after the recipient was refused")
	}
}

// With a username the sender must authenticate with PLAIN before MAIL FROM;
// without one it must not attempt AUTH at all.
func TestSender_Send_Auth(t *testing.T) {
	t.Run("credentials are sent", func(t *testing.T) {
		srv := &scriptedSMTP{ehlo: []string{"AUTH PLAIN"}}
		startScripted(t, srv)
		if err := srv.sender("sys-user", "s3cret").Send(context.Background(), oneRecipient()); err != nil {
			t.Fatalf("Send: %v", err)
		}
		line, ok := srv.seen("AUTH PLAIN")
		if !ok {
			t.Fatalf("no AUTH PLAIN received, commands=%v", srv.cmds())
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "AUTH PLAIN "))
		if err != nil {
			t.Fatalf("decode auth payload: %v", err)
		}
		if string(raw) != "\x00sys-user\x00s3cret" {
			t.Errorf("auth payload = %q, want identity-less PLAIN with the configured credentials", raw)
		}
	})
	t.Run("no username means no AUTH", func(t *testing.T) {
		srv := &scriptedSMTP{ehlo: []string{"AUTH PLAIN"}}
		startScripted(t, srv)
		if err := srv.sender("", "").Send(context.Background(), oneRecipient()); err != nil {
			t.Fatalf("Send: %v", err)
		}
		if _, ok := srv.seen("AUTH"); ok {
			t.Error("AUTH attempted without a configured username")
		}
	})
}

// The wire message must carry an RFC 2047 encoded subject (umlauts survive),
// the configured From with the default display name, and every To header.
func TestSender_Send_HeadersAndUmlautSubject(t *testing.T) {
	srv := &scriptedSMTP{}
	startScripted(t, srv)
	msg := Message{
		To: []send.EmailAddress{
			{Name: "Jürgen Müller", Email: "j@example.test"},
			{Email: "k@example.test"},
		},
		Subject:  "Einladung zu Cosmi: Grüße aus Zürich",
		BodyText: "Willkommen",
	}
	if err := srv.sender("", "").Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	srv.mu.Lock()
	data := srv.data
	srv.mu.Unlock()
	m, err := mail.ReadMessage(strings.NewReader(data))
	if err != nil {
		t.Fatalf("parse wire message: %v", err)
	}

	rawSubject := m.Header.Get("Subject")
	if !strings.HasPrefix(rawSubject, "=?utf-8?") {
		t.Errorf("subject %q is not RFC 2047 encoded", rawSubject)
	}
	decoded, err := (&mime.WordDecoder{}).DecodeHeader(rawSubject)
	if err != nil || decoded != msg.Subject {
		t.Errorf("decoded subject = %q (err %v), want %q", decoded, err, msg.Subject)
	}

	from, err := m.Header.AddressList("From")
	if err != nil || len(from) != 1 || from[0].Address != "noreply@zentria.test" || from[0].Name != "Cosmi" {
		t.Errorf("From = %v (err %v), want Cosmi <noreply@zentria.test>", from, err)
	}
	to, err := m.Header.AddressList("To")
	if err != nil || len(to) != 2 || to[0].Name != "Jürgen Müller" || to[1].Address != "k@example.test" {
		t.Errorf("To = %v (err %v), want both recipients with decoded name", to, err)
	}
}

func TestSender_FromNameAndAccessors(t *testing.T) {
	s := New(Config{Host: "smtp.example.test", From: "noreply@zentria.test", FromName: "Zentria"})
	if got := s.From(); got != "noreply@zentria.test" {
		t.Errorf("From() = %q, want noreply@zentria.test", got)
	}
	if s.cfg.FromName != "Zentria" {
		t.Errorf("FromName = %q, an explicit name must not be replaced by the default", s.cfg.FromName)
	}
	if got := New(Config{}).cfg.FromName; got != "Cosmi" {
		t.Errorf("default FromName = %q, want Cosmi", got)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("disk gone") }

// A MIME build failure must be reported before any connection is opened.
func TestSender_Send_MIMEBuildError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	accepted := make(chan struct{}, 1)
	go func() {
		if c, err := ln.Accept(); err == nil {
			accepted <- struct{}{}
			_ = c.Close()
		}
	}()
	defer ln.Close()
	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	s := New(Config{Host: host, Port: port, From: "noreply@zentria.test"})
	msg := oneRecipient()
	msg.Attachments = []send.AttachmentData{{Filename: "a.bin", ContentType: "application/octet-stream", Data: failingReader{}}}
	err = s.Send(context.Background(), msg)
	if err == nil || !strings.Contains(err.Error(), "build mime message") {
		t.Fatalf("Send error = %v, want a build mime message error", err)
	}
	select {
	case <-accepted:
		t.Error("a connection was opened although the message could not be built")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSender_Send_CanceledContextDoesNotDial(t *testing.T) {
	srv := &scriptedSMTP{}
	startScripted(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := srv.sender("", "").Send(ctx, oneRecipient())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Send error = %v, want context.Canceled", err)
	}
	if c := srv.cmds(); len(c) != 0 {
		t.Errorf("server saw commands %v for a canceled send", c)
	}
}

func TestSender_Send_DialFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	_ = ln.Close() // nothing listens on the port any more

	s := New(Config{Host: host, Port: port, From: "noreply@zentria.test"})
	err = s.Send(context.Background(), oneRecipient())
	if err == nil || !strings.Contains(err.Error(), "smtp dial") {
		t.Fatalf("Send error = %v, want an smtp dial error", err)
	}
}

// A caller deadline shorter than the package's exchange deadline must win: the
// stalled server is abandoned when ctx expires, not after exchangeDeadline.
func TestSender_Send_ContextDeadlineBeatsExchangeDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		if c, err := ln.Accept(); err == nil {
			defer c.Close()
			_, _ = io.Copy(io.Discard, c) // accept, never greet
		}
	}()
	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	origDial, origExchange := dialTimeout, exchangeDeadline
	dialTimeout, exchangeDeadline = 2*time.Second, 30*time.Second
	defer func() { dialTimeout, exchangeDeadline = origDial, origExchange }()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	s := New(Config{Host: host, Port: port, From: "noreply@zentria.test"})

	done := make(chan error, 1)
	go func() { done <- s.Send(ctx, oneRecipient()) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error from the stalled server")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Send ignored the context deadline and waited for the exchange deadline")
	}
}
