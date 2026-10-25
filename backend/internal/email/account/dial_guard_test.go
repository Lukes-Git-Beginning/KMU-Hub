package account

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/security/safedial"
)

// internalHosts are the targets a member must not be able to aim the account
// form at: loopback, the RFC 1918 ranges Docker networks live in, IPv6 loopback.
var internalHosts = []string{"127.0.0.1", "10.0.0.5", "172.16.0.9", "192.168.10.2", "::1", "169.254.169.254"}

func TestCreate_RejectsInternalMailHosts(t *testing.T) {
	for _, host := range internalHosts {
		for _, field := range []string{"IMAP", "SMTP"} {
			repo := newMockRepo()
			svc := NewService(repo, &mockEncryptor{})
			in := validInput()
			if field == "IMAP" {
				in.IMAPHost = host
			} else {
				in.SMTPHost = host
			}
			_, err := svc.Create(context.Background(), testTenantID, in)
			assert.ErrorIs(t, err, ErrHostNotAllowed, "%s host %s", field, host)
			assert.Empty(t, repo.accounts, "%s host %s must not be stored", field, host)
		}
	}
}

func TestCreate_AllowlistedInternalHostPasses(t *testing.T) {
	g, err := safedial.New("10.0.0.0/8")
	require.NoError(t, err)
	svc := NewService(newMockRepo(), &mockEncryptor{}).WithDialGuard(g)
	in := validInput()
	in.IMAPHost, in.SMTPHost = "10.0.0.5", "10.0.0.6"
	_, err = svc.Create(context.Background(), testTenantID, in)
	require.NoError(t, err)
}

func TestUpdate_RejectsInternalMailHosts(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, &mockEncryptor{})
	created, err := svc.Create(context.Background(), testTenantID, validInput())
	require.NoError(t, err)

	for _, host := range internalHosts {
		h := host
		_, err := svc.Update(context.Background(), created.ID, testTenantID, UpdateInput{IMAPHost: &h})
		assert.ErrorIs(t, err, ErrHostNotAllowed, "IMAP host %s", host)
		_, err = svc.Update(context.Background(), created.ID, testTenantID, UpdateInput{SMTPHost: &h})
		assert.ErrorIs(t, err, ErrHostNotAllowed, "SMTP host %s", host)
	}
	stored := repo.accounts[created.ID]
	assert.Equal(t, validInput().IMAPHost, stored.IMAPHost)
	assert.Equal(t, validInput().SMTPHost, stored.SMTPHost)
}

// TestConnection must not become a port probe: an internal host is refused
// before any connection, and with the allowlist the same host is really dialled.
func TestTestConnection_DialsThroughGuard(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	accepted := make(chan struct{}, 4)
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
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	in := CreateInput{IMAPHost: "127.0.0.1", IMAPPort: port, Username: "u", Password: "p", UseSSL: true}

	err = NewService(newMockRepo(), &mockEncryptor{}).TestConnection(context.Background(), in)
	assert.ErrorIs(t, err, ErrHostNotAllowed)
	select {
	case <-accepted:
		t.Fatal("TestConnection reached a loopback listener without an allowlist entry")
	case <-time.After(100 * time.Millisecond):
	}

	g, err := safedial.New("127.0.0.1")
	require.NoError(t, err)
	err = NewService(newMockRepo(), &mockEncryptor{}).WithDialGuard(g).TestConnection(context.Background(), in)
	assert.ErrorIs(t, err, ErrConnectionFailed, "allowlisted host is dialled; the peer closes, so the TLS handshake fails")
	select {
	case <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("allowlisted host was never dialled")
	}

	// An internal SMTP host fails the test as well, even though only IMAP is dialled.
	in.SMTPHost = "10.1.1.1"
	err = NewService(newMockRepo(), &mockEncryptor{}).WithDialGuard(g).TestConnection(context.Background(), in)
	assert.ErrorIs(t, err, ErrHostNotAllowed)
}
