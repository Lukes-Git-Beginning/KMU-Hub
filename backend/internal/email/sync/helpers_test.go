package sync

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func TestDetectFolderType(t *testing.T) {
	tests := []struct {
		name     string
		imapName string
		want     string
	}{
		{"inbox", "INBOX", models.FolderTypeInbox},
		{"inbox lowercase", "inbox", models.FolderTypeInbox},
		{"sent", "Sent", models.FolderTypeSent},
		{"sent items", "Sent Items", models.FolderTypeSent},
		{"sent messages", "Sent Messages", models.FolderTypeSent},
		{"gesendet", "Gesendet", models.FolderTypeSent},
		{"gesendete objekte", "Gesendete Objekte", models.FolderTypeSent},
		{"drafts", "Drafts", models.FolderTypeDrafts},
		{"entwuerfe umlaut", "Entwürfe", models.FolderTypeDrafts},
		{"entwuerfe ascii", "Entwuerfe", models.FolderTypeDrafts},
		{"trash", "Trash", models.FolderTypeTrash},
		{"deleted", "Deleted", models.FolderTypeTrash},
		{"deleted items", "Deleted Items", models.FolderTypeTrash},
		{"papierkorb", "Papierkorb", models.FolderTypeTrash},
		{"spam", "Spam", models.FolderTypeSpam},
		{"junk", "Junk", models.FolderTypeSpam},
		{"junk-e-mail", "Junk-E-Mail", models.FolderTypeSpam},
		{"archive", "Archive", models.FolderTypeArchive},
		{"archiv", "Archiv", models.FolderTypeArchive},
		{"unknown custom folder", "Projects/2026", models.FolderTypeCustom},
		{"empty name", "", models.FolderTypeCustom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DetectFolderType(tt.imapName))
		})
	}
}

func TestParseEnvelopeDate(t *testing.T) {
	t.Run("valid RFC3339 date parses exactly", func(t *testing.T) {
		got := parseEnvelopeDate("2026-03-05T14:30:00Z")
		want := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
		assert.True(t, want.Equal(got), "expected %v, got %v", want, got)
	})

	t.Run("valid date with offset parses exactly", func(t *testing.T) {
		got := parseEnvelopeDate("2026-03-05T14:30:00+02:00")
		want := time.Date(2026, 3, 5, 14, 30, 0, 0, time.FixedZone("", 2*60*60))
		assert.True(t, want.Equal(got), "expected %v, got %v", want, got)
	})

	t.Run("unparseable date falls back to now, not zero value", func(t *testing.T) {
		got := parseEnvelopeDate("not a date at all")
		assert.WithinDuration(t, time.Now().UTC(), got, 5*time.Second)
		assert.NotEqual(t, time.Time{}, got)
	})

	t.Run("empty string falls back to now", func(t *testing.T) {
		got := parseEnvelopeDate("")
		assert.WithinDuration(t, time.Now().UTC(), got, 5*time.Second)
	})
}

func TestFirstInReplyTo(t *testing.T) {
	t.Run("empty list yields empty string", func(t *testing.T) {
		assert.Equal(t, "", firstInReplyTo(nil))
		assert.Equal(t, "", firstInReplyTo([]string{}))
	})

	t.Run("returns first entry trimmed", func(t *testing.T) {
		assert.Equal(t, "<msg1@example.com>", firstInReplyTo([]string{"  <msg1@example.com>  ", "<msg2@example.com>"}))
	})

	t.Run("single entry without whitespace", func(t *testing.T) {
		assert.Equal(t, "<only@example.com>", firstInReplyTo([]string{"<only@example.com>"}))
	})
}

func TestImapAddressesToModel(t *testing.T) {
	t.Run("nil input yields nil", func(t *testing.T) {
		assert.Nil(t, imapAddressesToModel(nil))
	})

	t.Run("empty input yields nil", func(t *testing.T) {
		assert.Nil(t, imapAddressesToModel([]imap.Address{}))
	})

	t.Run("populated input maps name and Addr()", func(t *testing.T) {
		addrs := []imap.Address{
			{Name: "Alice", Mailbox: "alice", Host: "example.com"},
			{Name: "", Mailbox: "bob", Host: "example.com"},
		}
		got := imapAddressesToModel(addrs)
		require.Len(t, got, 2)
		assert.Equal(t, models.EmailAddress{Name: "Alice", Email: "alice@example.com"}, got[0])
		assert.Equal(t, models.EmailAddress{Name: "", Email: "bob@example.com"}, got[1])
	})
}

func newTestWorker() *Worker {
	return &Worker{
		account: &models.EmailAccount{
			ID:       uuid.New(),
			TenantID: uuid.New(),
		},
	}
}

func TestEnvelopeToMessage(t *testing.T) {
	w := newTestWorker()
	folder := &models.EmailFolder{ID: uuid.New(), AccountID: w.account.ID}
	date := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)

	t.Run("maps addresses, flags and in-reply-to", func(t *testing.T) {
		env := &MessageEnvelope{
			UID:       42,
			MessageID: "<abc@example.com>",
			InReplyTo: []string{"<parent@example.com>"},
			Subject:   "Hello",
			From:      []imap.Address{{Name: "Sender", Mailbox: "sender", Host: "example.com"}},
			To:        []imap.Address{{Mailbox: "to", Host: "example.com"}},
			CC:        []imap.Address{{Mailbox: "cc", Host: "example.com"}},
			BCC:       []imap.Address{{Mailbox: "bcc", Host: "example.com"}},
			Flags:     []imap.Flag{imap.FlagSeen, imap.FlagFlagged},
			Size:      1024,
		}

		msg := w.envelopeToMessage(env, folder, date)

		assert.Equal(t, w.account.TenantID, msg.TenantID)
		assert.Equal(t, w.account.ID, msg.AccountID)
		assert.Equal(t, folder.ID, msg.FolderID)
		assert.Equal(t, int64(42), msg.UID)
		assert.Equal(t, "<abc@example.com>", msg.MessageID)
		assert.Equal(t, "<parent@example.com>", msg.InReplyTo)
		assert.Equal(t, []string{"<parent@example.com>"}, msg.References)
		assert.Equal(t, "Sender", msg.FromName)
		assert.Equal(t, "sender@example.com", msg.FromEmail)
		assert.Equal(t, []models.EmailAddress{{Email: "to@example.com"}}, msg.ToAddresses)
		assert.Equal(t, []models.EmailAddress{{Email: "cc@example.com"}}, msg.CcAddresses)
		assert.Equal(t, []models.EmailAddress{{Email: "bcc@example.com"}}, msg.BccAddresses)
		assert.True(t, msg.IsRead)
		assert.True(t, msg.IsStarred)
		assert.False(t, msg.IsDraft)
		assert.Equal(t, date, msg.Date)
		assert.Equal(t, int64(1024), msg.SizeBytes)
		assert.Equal(t, "Hello", msg.Preview)
	})

	t.Run("draft flag and no from address", func(t *testing.T) {
		env := &MessageEnvelope{
			UID:     7,
			Subject: "Draft",
			Flags:   []imap.Flag{imap.FlagDraft},
		}

		msg := w.envelopeToMessage(env, folder, date)

		assert.True(t, msg.IsDraft)
		assert.False(t, msg.IsRead)
		assert.Empty(t, msg.FromName)
		assert.Empty(t, msg.FromEmail)
		assert.Nil(t, msg.ToAddresses)
		assert.Empty(t, msg.InReplyTo)
		assert.Nil(t, msg.References)
	})

	t.Run("subject longer than 200 chars is truncated for preview", func(t *testing.T) {
		longSubject := strings.Repeat("a", 250)
		env := &MessageEnvelope{UID: 1, Subject: longSubject}

		msg := w.envelopeToMessage(env, folder, date)

		assert.Len(t, msg.Preview, 200)
		assert.Equal(t, longSubject[:200], msg.Preview)
	})
}
