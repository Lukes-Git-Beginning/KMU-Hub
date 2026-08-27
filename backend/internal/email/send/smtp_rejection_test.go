package send

import (
	"context"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAccountProvider returns a fixed set of credentials, used to point Send
// at the fake SMTP server started for these tests.
type stubAccountProvider struct {
	creds *Credentials
}

func (s *stubAccountProvider) GetDecryptedCredentials(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*Credentials, error) {
	return s.creds, nil
}

// smtpRejectStage identifies which SMTP command the fake server below answers
// with a permanent 5xx error, simulating a real mail server bouncing a send.
type smtpRejectStage int

const (
	rejectAtMail smtpRejectStage = iota
	rejectAtRcpt
	rejectAtData
)

// fakeRejectingSMTPServer completes the SMTP handshake normally and then
// rejects the request at the given stage with a permanent 5xx response.
func fakeRejectingSMTPServer(t *testing.T, stage smtpRejectStage) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tp := textproto.NewConn(conn)
		_ = tp.PrintfLine("220 fake.smtp ready")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			verb := ""
			if fields := strings.Fields(line); len(fields) > 0 {
				verb = strings.ToUpper(fields[0])
			}
			switch verb {
			case "EHLO", "HELO":
				_ = tp.PrintfLine("250 fake.smtp")
			case "MAIL":
				if stage == rejectAtMail {
					_ = tp.PrintfLine("550 5.1.0 sender rejected")
					continue
				}
				_ = tp.PrintfLine("250 ok")
			case "RCPT":
				if stage == rejectAtRcpt {
					_ = tp.PrintfLine("550 5.1.1 user unknown")
					continue
				}
				_ = tp.PrintfLine("250 ok")
			case "DATA":
				if stage == rejectAtData {
					_ = tp.PrintfLine("550 5.7.1 message rejected")
					continue
				}
				_ = tp.PrintfLine("354 go ahead")
				for {
					dline, derr := tp.ReadLine()
					if derr != nil || dline == "." {
						break
					}
				}
				_ = tp.PrintfLine("250 ok")
			case "QUIT":
				_ = tp.PrintfLine("221 bye")
				return
			default:
				_ = tp.PrintfLine("250 ok")
			}
		}
	}()
	return ln.Addr().String()
}

// TestSend_SMTPRejection_WrapsErrSendFailed proves that a permanent rejection
// at any stage of the SMTP envelope (MAIL, RCPT, DATA) surfaces through Send
// as ErrSendFailed and that no local copy is stored — a rejected send is not
// silently treated as delivered.
func TestSend_SMTPRejection_WrapsErrSendFailed(t *testing.T) {
	tests := []struct {
		name  string
		stage smtpRejectStage
	}{
		{"rejected at MAIL FROM", rejectAtMail},
		{"rejected at RCPT TO", rejectAtRcpt},
		{"rejected at DATA", rejectAtData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := fakeRejectingSMTPServer(t, tt.stage)
			host, portStr, err := net.SplitHostPort(addr)
			require.NoError(t, err)
			port, err := strconv.Atoi(portStr)
			require.NoError(t, err)

			creator := &mockMessageCreator{}
			provider := &stubAccountProvider{creds: &Credentials{SMTPHost: host, SMTPPort: port}}
			svc := NewService(provider, creator, nil)

			input := SendInput{
				AccountID: uuid.New(),
				From:      EmailAddress{Email: "from@example.com"},
				To:        []EmailAddress{{Email: "to@example.com"}},
				Subject:   "Test",
			}

			_, err = svc.Send(context.Background(), input)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrSendFailed)
			assert.Empty(t, creator.messages, "message must not be stored locally when SMTP rejects the send")
		})
	}
}
