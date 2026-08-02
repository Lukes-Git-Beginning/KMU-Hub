// Package fakesmtp provides a minimal in-process SMTP server for tests that
// need to prove a mail actually went on the wire -- envelope sender, every
// recipient, and the DATA payload including attachments. It speaks just enough
// of the protocol for net/smtp's plain path: greeting, EHLO, MAIL, RCPT, DATA,
// QUIT. It advertises neither STARTTLS nor AUTH.
package fakesmtp

import (
	"bufio"
	"bytes"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Session is one recorded SMTP conversation.
type Session struct {
	MailFrom string
	RcptTo   []string
	Data     string
}

// Server is a listening fake SMTP server. It records a single session and
// publishes it on Sessions once the client sends QUIT.
type Server struct {
	Host     string
	Port     int
	Sessions <-chan Session
}

// Start listens on a free loopback port and serves one connection. The
// listener is closed when the test ends.
func Start(t *testing.T) *Server {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("fakesmtp: listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("fakesmtp: split host port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("fakesmtp: parse port: %v", err)
	}

	sessions := make(chan Session, 1)
	go serve(ln, sessions)

	return &Server{Host: host, Port: port, Sessions: sessions}
}

// WaitSession blocks until a session was recorded or the timeout elapses.
func (s *Server) WaitSession(t *testing.T, timeout time.Duration) Session {
	t.Helper()
	select {
	case sess := <-s.Sessions:
		return sess
	case <-time.After(timeout):
		t.Fatal("fakesmtp: no SMTP session recorded — nothing was delivered")
		return Session{}
	}
}

func serve(ln net.Listener, out chan<- Session) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	var sess Session
	r := bufio.NewReader(conn)
	write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }

	write("220 fake.test ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			write("250-fake.test")
			write("250 SIZE 10485760")
		case strings.HasPrefix(cmd, "MAIL FROM"):
			sess.MailFrom = extractAddr(line)
			write("250 OK")
		case strings.HasPrefix(cmd, "RCPT TO"):
			sess.RcptTo = append(sess.RcptTo, extractAddr(line))
			write("250 OK")
		case cmd == "DATA":
			write("354 End data with <CR><LF>.<CR><LF>")
			var body bytes.Buffer
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
			sess.Data = body.String()
			write("250 OK queued")
		case cmd == "QUIT":
			write("221 Bye")
			out <- sess
			return
		default:
			write("250 OK")
		}
	}
}

func extractAddr(line string) string {
	start := strings.Index(line, "<")
	end := strings.LastIndex(line, ">")
	if start < 0 || end <= start {
		return strings.TrimSpace(line)
	}
	return line[start+1 : end]
}
