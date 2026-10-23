package sync

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	protoUser = "agent@example.test"
	protoPass = "s3cret"
)

const protoMessage = "From: Alice <alice@example.test>\r\n" +
	"To: Bob <bob@example.test>\r\n" +
	"Subject: Quarterly numbers\r\n" +
	"Message-ID: <q3@example.test>\r\n" +
	"In-Reply-To: <q2@example.test>\r\n" +
	"Date: Tue, 06 Oct 2026 09:30:00 +0000\r\n" +
	"\r\n" +
	"Hello Bob,\r\nnumbers attached.\r\n"

// connectTo wires an IMAPClient to addr over plain TCP. Connect() itself is
// not used because it demands TLS or STARTTLS; the protocol client is built
// the same way Connect does for the direct path.
func connectTo(t *testing.T, addr string) *IMAPClient {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	require.NoError(t, err)
	client, err := newIMAPProtocolClient(conn, true)
	require.NoError(t, err)
	c := NewIMAPClient(uuid.New(), slog.Default())
	c.client = client
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// startMemIMAP runs go-imap's in-memory server (real wire protocol, real
// go-imap parsing) with one user whose INBOX holds one message and which also
// has a "Sent" mailbox. It returns the server address and the user.
func startMemIMAP(t *testing.T) (string, *imapmemserver.User) {
	t.Helper()

	mem := imapmemserver.New()
	user := imapmemserver.NewUser(protoUser, protoPass)
	require.NoError(t, user.Create("INBOX", nil))
	require.NoError(t, user.Create("Sent", nil))
	_, err := user.Append("INBOX", strings.NewReader(protoMessage), &imap.AppendOptions{})
	require.NoError(t, err)
	mem.AddUser(user)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}},
		InsecureAuth: true,
	})
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	return ln.Addr().String(), user
}

// startRejectingIMAP runs a minimal raw server that greets (advertising IDLE)
// and answers every command with the given tagged status ("NO" or "BAD").
func startRejectingIMAP(t *testing.T, status string) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = conn.Write([]byte("* OK [CAPABILITY IMAP4rev1 IDLE] ready\r\n"))
				sc := bufio.NewScanner(conn)
				for sc.Scan() {
					tag, _, _ := strings.Cut(sc.Text(), " ")
					if _, err := conn.Write([]byte(tag + " " + status + " rejected by test server\r\n")); err != nil {
						return
					}
				}
			}()
		}
	}()

	return ln.Addr().String()
}

func loggedIn(t *testing.T, addr string) *IMAPClient {
	t.Helper()
	c := connectTo(t, addr)
	require.NoError(t, c.Login(protoUser, protoPass))
	return c
}

func uidSetOf(uids ...uint32) imap.UIDSet {
	set := imap.UIDSet{}
	for _, u := range uids {
		set.AddNum(imap.UID(u))
	}
	return set
}

func TestIMAPClient_Protocol_Login(t *testing.T) {
	addr, _ := startMemIMAP(t)

	t.Run("success", func(t *testing.T) {
		c := connectTo(t, addr)
		require.NoError(t, c.Login(protoUser, protoPass))
	})

	t.Run("wrong password is a NO from the server", func(t *testing.T) {
		c := connectTo(t, addr)
		err := c.Login(protoUser, "wrong")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "IMAP login failed")
		var imapErr *imap.Error
		require.ErrorAs(t, err, &imapErr)
		assert.Equal(t, imap.StatusResponseTypeNo, imapErr.Type)
	})
}

func TestIMAPClient_Protocol_SelectFolder(t *testing.T) {
	addr, _ := startMemIMAP(t)
	c := loggedIn(t, addr)

	st, err := c.SelectFolder("INBOX")
	require.NoError(t, err)
	assert.EqualValues(t, 1, st.NumMessages)
	assert.NotZero(t, st.UIDValidity)
	assert.EqualValues(t, 2, st.UIDNext)

	st, err = c.SelectFolder("Missing")
	require.Error(t, err)
	assert.Nil(t, st)
	assert.Contains(t, err.Error(), `IMAP select "Missing" failed`)
	var imapErr *imap.Error
	require.ErrorAs(t, err, &imapErr)
	assert.Equal(t, imap.StatusResponseTypeNo, imapErr.Type)
}

func TestIMAPClient_Protocol_FetchHeaders(t *testing.T) {
	addr, _ := startMemIMAP(t)
	c := loggedIn(t, addr)
	_, err := c.SelectFolder("INBOX")
	require.NoError(t, err)

	envs, err := c.FetchHeaders(uidSetOf(1))
	require.NoError(t, err)
	require.Len(t, envs, 1)
	env := envs[0]
	assert.EqualValues(t, 1, env.UID)
	assert.Equal(t, "Quarterly numbers", env.Subject)
	assert.Equal(t, "q3@example.test", env.MessageID) // go-imap strips the angle brackets
	assert.Equal(t, []string{"q2@example.test"}, env.InReplyTo)
	assert.Equal(t, "2026-10-06T09:30:00Z", env.Date)
	assert.EqualValues(t, len(protoMessage), env.Size)
	require.Len(t, env.From, 1)
	assert.Equal(t, "alice", env.From[0].Mailbox)
	assert.Equal(t, "example.test", env.From[0].Host)
	assert.Equal(t, "Alice", env.From[0].Name)
	require.Len(t, env.To, 1)
	assert.Equal(t, "bob", env.To[0].Mailbox)

	t.Run("unknown UID yields no envelopes", func(t *testing.T) {
		envs, err := c.FetchHeaders(uidSetOf(99))
		require.NoError(t, err)
		assert.Empty(t, envs)
	})

	t.Run("no mailbox selected is rejected by the server", func(t *testing.T) {
		fresh := loggedIn(t, addr)
		envs, err := fresh.FetchHeaders(uidSetOf(1))
		require.Error(t, err)
		assert.Nil(t, envs)
		assert.Contains(t, err.Error(), "IMAP fetch headers failed")
	})
}

func TestIMAPClient_Protocol_FetchBody(t *testing.T) {
	addr, _ := startMemIMAP(t)
	c := loggedIn(t, addr)
	_, err := c.SelectFolder("INBOX")
	require.NoError(t, err)

	body, err := c.FetchBody(1)
	require.NoError(t, err)
	assert.EqualValues(t, 1, body.UID)
	assert.Equal(t, protoMessage, string(body.Literal))

	t.Run("unknown UID", func(t *testing.T) {
		body, err := c.FetchBody(99)
		assert.Nil(t, body)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "message UID 99 not found")
	})

	t.Run("no mailbox selected is rejected by the server", func(t *testing.T) {
		fresh := loggedIn(t, addr)
		body, err := fresh.FetchBody(1)
		assert.Nil(t, body)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "IMAP fetch body failed")
	})
}

func TestIMAPClient_Protocol_SetFlags(t *testing.T) {
	addr, _ := startMemIMAP(t)
	c := loggedIn(t, addr)
	_, err := c.SelectFolder("INBOX")
	require.NoError(t, err)

	flagsOf := func() []imap.Flag {
		envs, err := c.FetchHeaders(uidSetOf(1))
		require.NoError(t, err)
		require.Len(t, envs, 1)
		return envs[0].Flags
	}

	assert.NotContains(t, flagsOf(), imap.FlagSeen)

	require.NoError(t, c.SetFlags(1, []imap.Flag{imap.FlagSeen, imap.FlagFlagged}, true))
	assert.Contains(t, flagsOf(), imap.FlagSeen)
	assert.Contains(t, flagsOf(), imap.FlagFlagged)

	require.NoError(t, c.SetFlags(1, []imap.Flag{imap.FlagSeen}, false))
	got := flagsOf()
	assert.NotContains(t, got, imap.FlagSeen)
	assert.Contains(t, got, imap.FlagFlagged)

	t.Run("no mailbox selected is rejected by the server", func(t *testing.T) {
		fresh := loggedIn(t, addr)
		err := fresh.SetFlags(1, []imap.Flag{imap.FlagSeen}, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "IMAP store flags failed")
	})
}

func TestIMAPClient_Protocol_HasIDLEAndNoop(t *testing.T) {
	addr, _ := startMemIMAP(t)
	c := connectTo(t, addr)

	// The server only advertises IDLE once the session is authenticated.
	assert.False(t, c.HasIDLE())
	require.NoError(t, c.Login(protoUser, protoPass))
	assert.True(t, c.HasIDLE())

	require.NoError(t, c.Noop())
}

func TestIMAPClient_Protocol_ListFolders(t *testing.T) {
	addr, _ := startMemIMAP(t)
	c := loggedIn(t, addr)

	folders, err := c.ListFolders()
	require.NoError(t, err)
	names := map[string]*FolderInfo{}
	for _, f := range folders {
		names[f.Name] = f
	}
	require.Contains(t, names, "INBOX")
	require.Contains(t, names, "Sent")
	assert.Equal(t, '/', names["INBOX"].Delimiter)
}

func TestIMAPClient_Protocol_IdleRunsUntilContextEnds(t *testing.T) {
	addr, user := startMemIMAP(t)
	c := loggedIn(t, addr)
	_, err := c.SelectFolder("INBOX")
	require.NoError(t, err)

	// A message arriving while the client idles must not break IDLE.
	go func() {
		time.Sleep(100 * time.Millisecond)
		_, _ = user.Append("INBOX", strings.NewReader(protoMessage), &imap.AppendOptions{})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	err = c.Idle(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// The connection is still usable after IDLE ended.
	require.NoError(t, c.Noop())
}

func TestIMAPClient_Protocol_Close(t *testing.T) {
	addr, _ := startMemIMAP(t)
	c := loggedIn(t, addr)

	require.NoError(t, c.Close())
	assert.Nil(t, c.client)
	assert.NoError(t, c.Close(), "second Close is a no-op")
	assert.ErrorIs(t, c.Noop(), ErrIMAPConnectionLost)
}

// TestIMAPClient_Protocol_ServerRejects drives every wrapper method against a
// server that answers each command with a tagged NO or BAD and checks the
// error is wrapped with the method's context instead of being swallowed.
func TestIMAPClient_Protocol_ServerRejects(t *testing.T) {
	for _, status := range []string{"NO", "BAD"} {
		t.Run(status, func(t *testing.T) {
			addr := startRejectingIMAP(t, status)
			want := imap.StatusResponseTypeNo
			if status == "BAD" {
				want = imap.StatusResponseTypeBad
			}

			calls := map[string]func(c *IMAPClient) error{
				"Login": func(c *IMAPClient) error { return c.Login("u", "p") },
				"SelectFolder": func(c *IMAPClient) error {
					st, err := c.SelectFolder("INBOX")
					assert.Nil(t, st)
					return err
				},
				"FetchHeaders": func(c *IMAPClient) error {
					envs, err := c.FetchHeaders(uidSetOf(1))
					assert.Nil(t, envs)
					return err
				},
				"FetchBody": func(c *IMAPClient) error {
					b, err := c.FetchBody(1)
					assert.Nil(t, b)
					return err
				},
				"SetFlags": func(c *IMAPClient) error {
					return c.SetFlags(1, []imap.Flag{imap.FlagSeen}, true)
				},
				"Noop": func(c *IMAPClient) error { return c.Noop() },
				"ListFolders": func(c *IMAPClient) error {
					f, err := c.ListFolders()
					assert.Nil(t, f)
					return err
				},
				"Idle": func(c *IMAPClient) error {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					return c.Idle(ctx)
				},
			}

			for name, call := range calls {
				t.Run(name, func(t *testing.T) {
					c := connectTo(t, addr)
					err := call(c)
					require.Error(t, err)
					assert.NotErrorIs(t, err, ErrIMAPConnectionLost)
					assert.NotErrorIs(t, err, context.DeadlineExceeded)
					var imapErr *imap.Error
					if errors.As(err, &imapErr) {
						assert.Equal(t, want, imapErr.Type)
					}
				})
			}
		})
	}
}
