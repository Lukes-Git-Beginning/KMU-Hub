package sync

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"
)

// FolderStatus holds the status of an IMAP folder.
type FolderStatus struct {
	NumMessages uint32
	UIDValidity uint32
	UIDNext     uint32
}

// MessageEnvelope holds header-level information about a message fetched from IMAP.
type MessageEnvelope struct {
	UID       uint32
	Flags     []imap.Flag
	Date      string // RFC 5322 date from envelope
	Subject   string
	From      []imap.Address
	To        []imap.Address
	CC        []imap.Address
	BCC       []imap.Address
	MessageID string
	InReplyTo []string
	Size      int64
}

// MessageBody holds the full message data.
type MessageBody struct {
	UID     uint32
	Literal []byte
}

// FolderInfo holds metadata about an IMAP folder.
type FolderInfo struct {
	Name       string
	Delimiter  rune
	Attributes []imap.MailboxAttr
}

// IMAPClient wraps go-imap v2's imapclient.Client with a higher-level interface.
type IMAPClient struct {
	client    *imapclient.Client
	logger    *slog.Logger
	accountID uuid.UUID
}

// NewIMAPClient creates a new IMAP client wrapper.
func NewIMAPClient(accountID uuid.UUID, logger *slog.Logger) *IMAPClient {
	return &IMAPClient{
		logger:    logger.With("account_id", accountID),
		accountID: accountID,
	}
}

// Connect establishes a connection to the IMAP server.
func (c *IMAPClient) Connect(host string, port int, useSSL bool) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	var client *imapclient.Client
	var err error

	if useSSL || port == 993 {
		client, err = imapclient.DialTLS(addr, nil)
	} else {
		client, err = imapclient.DialStartTLS(addr, nil)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrIMAPConnectionLost, err)
	}

	c.client = client
	c.logger.Info("IMAP connected", "host", host, "port", port, "ssl", useSSL)
	return nil
}

// Login authenticates with the IMAP server.
func (c *IMAPClient) Login(username, password string) error {
	if c.client == nil {
		return ErrIMAPConnectionLost
	}

	if err := c.client.Login(username, password).Wait(); err != nil {
		return fmt.Errorf("IMAP login failed: %w", err)
	}

	c.logger.Info("IMAP authenticated", "username", username)
	return nil
}

// SelectFolder selects an IMAP mailbox and returns its status.
func (c *IMAPClient) SelectFolder(name string) (*FolderStatus, error) {
	if c.client == nil {
		return nil, ErrIMAPConnectionLost
	}

	data, err := c.client.Select(name, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("IMAP select %q failed: %w", name, err)
	}

	return &FolderStatus{
		NumMessages: data.NumMessages,
		UIDValidity: data.UIDValidity,
		UIDNext:     uint32(data.UIDNext),
	}, nil
}

// FetchHeaders fetches envelope data for a set of UIDs.
func (c *IMAPClient) FetchHeaders(uidSet imap.UIDSet) ([]*MessageEnvelope, error) {
	if c.client == nil {
		return nil, ErrIMAPConnectionLost
	}

	fetchOptions := &imap.FetchOptions{
		Flags:      true,
		Envelope:   true,
		UID:        true,
		RFC822Size: true,
	}

	messages, err := c.client.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("IMAP fetch headers failed: %w", err)
	}

	var envelopes []*MessageEnvelope
	for _, msg := range messages {
		env := &MessageEnvelope{
			UID:   uint32(msg.UID),
			Flags: msg.Flags,
			Size:  int64(msg.RFC822Size),
		}

		if msg.Envelope != nil {
			env.Date = msg.Envelope.Date.Format("2006-01-02T15:04:05Z07:00")
			env.Subject = msg.Envelope.Subject
			env.From = msg.Envelope.From
			env.To = msg.Envelope.To
			env.CC = msg.Envelope.Cc
			env.BCC = msg.Envelope.Bcc
			env.MessageID = msg.Envelope.MessageID
			env.InReplyTo = msg.Envelope.InReplyTo
		}

		envelopes = append(envelopes, env)
	}

	return envelopes, nil
}

// FetchBody fetches the full MIME body for a single message by UID.
func (c *IMAPClient) FetchBody(uid uint32) (*MessageBody, error) {
	if c.client == nil {
		return nil, ErrIMAPConnectionLost
	}

	uidSet := imap.UIDSet{}
	uidSet.AddNum(imap.UID(uid))

	bodySection := &imap.FetchItemBodySection{
		Specifier: imap.PartSpecifierNone,
	}
	fetchOptions := &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}

	messages, err := c.client.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("IMAP fetch body failed: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("message UID %d not found", uid)
	}

	msg := messages[0]
	bodyBytes := msg.FindBodySection(bodySection)
	if bodyBytes == nil {
		return nil, fmt.Errorf("no body section in response for UID %d", uid)
	}

	return &MessageBody{
		UID:     uint32(msg.UID),
		Literal: bodyBytes,
	}, nil
}

// SetFlags adds or removes flags on a message by UID.
func (c *IMAPClient) SetFlags(uid uint32, flags []imap.Flag, add bool) error {
	if c.client == nil {
		return ErrIMAPConnectionLost
	}

	uidSet := imap.UIDSet{}
	uidSet.AddNum(imap.UID(uid))

	storeFlags := &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: flags,
	}
	if !add {
		storeFlags.Op = imap.StoreFlagsDel
	}

	if err := c.client.Store(uidSet, storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("IMAP store flags failed: %w", err)
	}

	return nil
}

// Idle enters IMAP IDLE mode and blocks until context is canceled.
// The go-imap v2 IdleCommand handles periodic IDLE restart internally.
func (c *IMAPClient) Idle(ctx context.Context) error {
	if c.client == nil {
		return ErrIMAPConnectionLost
	}

	idleCmd, err := c.client.Idle()
	if err != nil {
		return fmt.Errorf("IMAP IDLE start failed: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()

	if err := idleCmd.Close(); err != nil {
		c.logger.Warn("IMAP IDLE close error", "error", err)
	}

	return ctx.Err()
}

// HasIDLE checks if the server supports the IDLE capability.
func (c *IMAPClient) HasIDLE() bool {
	if c.client == nil {
		return false
	}
	caps := c.client.Caps()
	return caps.Has(imap.CapIdle)
}

// Noop sends a NOOP command to the server (polling fallback).
func (c *IMAPClient) Noop() error {
	if c.client == nil {
		return ErrIMAPConnectionLost
	}
	if err := c.client.Noop().Wait(); err != nil {
		return fmt.Errorf("IMAP NOOP failed: %w", err)
	}
	return nil
}

// ListFolders enumerates all IMAP folders on the server.
func (c *IMAPClient) ListFolders() ([]*FolderInfo, error) {
	if c.client == nil {
		return nil, ErrIMAPConnectionLost
	}

	mailboxes, err := c.client.List("", "*", nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("IMAP LIST failed: %w", err)
	}

	var folders []*FolderInfo
	for _, mb := range mailboxes {
		folders = append(folders, &FolderInfo{
			Name:       mb.Mailbox,
			Delimiter:  mb.Delim,
			Attributes: mb.Attrs,
		})
	}

	c.logger.Info("IMAP folders listed", "count", len(folders))
	return folders, nil
}

// Close closes the IMAP connection.
func (c *IMAPClient) Close() error {
	if c.client == nil {
		return nil
	}
	err := c.client.Close()
	c.client = nil
	return err
}

// firstInReplyTo returns the first InReplyTo header value or empty string.
func firstInReplyTo(refs []string) string {
	if len(refs) > 0 {
		return strings.TrimSpace(refs[0])
	}
	return ""
}
