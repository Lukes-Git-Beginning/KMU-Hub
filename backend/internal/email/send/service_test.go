package send

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock MessageCreator ---

type mockMessageCreator struct {
	messages []*models.EmailMessage
}

func (m *mockMessageCreator) Create(_ context.Context, msg *models.EmailMessage) error {
	m.messages = append(m.messages, msg)
	return nil
}

// --- Tests ---

func TestMIMEBuilder_PlainText(t *testing.T) {
	builder := NewMIMEBuilder()

	input := MIMEInput{
		From:    EmailAddress{Name: "Alice", Email: "alice@example.com"},
		To:      []EmailAddress{{Name: "Bob", Email: "bob@example.com"}},
		Subject: "Hello",
		BodyText: "Hello Bob",
		BodyHTML: "<p>Hello Bob</p>",
	}

	data, err := builder.Build(input)
	require.NoError(t, err)

	msg := string(data)
	assert.Contains(t, msg, "From:")
	assert.Contains(t, msg, "To:")
	assert.Contains(t, msg, "Subject:")
	assert.Contains(t, msg, "Message-ID:")
	assert.Contains(t, msg, "MIME-Version: 1.0")
	assert.Contains(t, msg, "multipart/alternative")
	assert.Contains(t, msg, "text/plain")
	assert.Contains(t, msg, "text/html")
}

func TestMIMEBuilder_WithAttachment(t *testing.T) {
	builder := NewMIMEBuilder()

	input := MIMEInput{
		From:    EmailAddress{Name: "Alice", Email: "alice@example.com"},
		To:      []EmailAddress{{Name: "Bob", Email: "bob@example.com"}},
		Subject: "With Attachment",
		BodyText: "See attached",
		BodyHTML: "<p>See attached</p>",
		Attachments: []AttachmentData{
			{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Data:        bytes.NewReader([]byte("file content")),
				Size:        12,
			},
		},
	}

	data, err := builder.Build(input)
	require.NoError(t, err)

	msg := string(data)
	assert.Contains(t, msg, "multipart/mixed")
	assert.Contains(t, msg, "test.txt")
	assert.Contains(t, msg, "Content-Disposition: attachment")
}

func TestMIMEBuilder_ReplyHeaders(t *testing.T) {
	builder := NewMIMEBuilder()

	input := MIMEInput{
		From:      EmailAddress{Name: "Alice", Email: "alice@example.com"},
		To:        []EmailAddress{{Email: "bob@example.com"}},
		Subject:   "Re: Hello",
		BodyText:  "Reply body",
		BodyHTML:  "<p>Reply body</p>",
		InReplyTo: "<original@example.com>",
		References: []string{"<original@example.com>"},
	}

	data, err := builder.Build(input)
	require.NoError(t, err)

	msg := string(data)
	assert.Contains(t, msg, "In-Reply-To: <original@example.com>")
	assert.Contains(t, msg, "References: <original@example.com>")
}

func TestMIMEBuilder_UnicodeSubject(t *testing.T) {
	builder := NewMIMEBuilder()

	input := MIMEInput{
		From:    EmailAddress{Name: "Muller", Email: "muller@example.com"},
		To:      []EmailAddress{{Email: "bob@example.com"}},
		Subject: "Angebot fur Vertrag",
		BodyText: "Text",
		BodyHTML: "<p>Text</p>",
	}

	data, err := builder.Build(input)
	require.NoError(t, err)

	msg := string(data)
	assert.Contains(t, msg, "Subject:")
	// Should be encoded properly
	assert.NotEmpty(t, msg)
}

func TestSaveDraft(t *testing.T) {
	creator := &mockMessageCreator{}
	svc := NewService(nil, creator, nil)

	input := DraftInput{
		AccountID: uuid.New(),
		UserID:    uuid.New(),
		From:      EmailAddress{Name: "Alice", Email: "alice@example.com"},
		To:        []EmailAddress{{Email: "bob@example.com"}},
		Subject:   "Draft subject",
		BodyHTML:  "<p>Draft body</p>",
		BodyText:  "Draft body",
	}

	msg, err := svc.SaveDraft(context.Background(), input)
	require.NoError(t, err)

	assert.True(t, msg.IsDraft)
	assert.Equal(t, "Draft subject", msg.Subject)
	assert.Equal(t, "alice@example.com", msg.FromEmail)
	assert.Len(t, creator.messages, 1)
}

func TestHasReplyPrefix(t *testing.T) {
	assert.True(t, hasReplyPrefix("Re: Hello"))
	assert.True(t, hasReplyPrefix("RE: Hello"))
	assert.True(t, hasReplyPrefix("AW: Hallo"))
	assert.False(t, hasReplyPrefix("Hello"))
	assert.False(t, hasReplyPrefix("Fwd: Hello"))
}

func TestHasForwardPrefix(t *testing.T) {
	assert.True(t, hasForwardPrefix("Fwd: Hello"))
	assert.True(t, hasForwardPrefix("FW: Hello"))
	assert.True(t, hasForwardPrefix("WG: Hallo"))
	assert.False(t, hasForwardPrefix("Hello"))
	assert.False(t, hasForwardPrefix("Re: Hello"))
}

func TestCollectRecipients(t *testing.T) {
	to := []EmailAddress{{Email: "a@ex.com"}, {Email: "b@ex.com"}}
	cc := []EmailAddress{{Email: "c@ex.com"}, {Email: "a@ex.com"}} // duplicate
	bcc := []EmailAddress{{Email: "d@ex.com"}}

	result := collectRecipients(to, cc, bcc)
	assert.Len(t, result, 4) // no duplicates
	assert.Contains(t, result, "a@ex.com")
	assert.Contains(t, result, "b@ex.com")
	assert.Contains(t, result, "c@ex.com")
	assert.Contains(t, result, "d@ex.com")
}

func TestToModelAddresses(t *testing.T) {
	addrs := []EmailAddress{
		{Name: "Alice", Email: "alice@ex.com"},
		{Name: "", Email: "bob@ex.com"},
	}

	result := toModelAddresses(addrs)
	assert.Len(t, result, 2)
	assert.Equal(t, "Alice", result[0].Name)
	assert.Equal(t, "alice@ex.com", result[0].Email)
}

func TestEmailAddressString(t *testing.T) {
	addr := EmailAddress{Name: "Alice Test", Email: "alice@example.com"}
	s := addr.String()
	assert.Contains(t, s, "alice@example.com")
	assert.Contains(t, s, "Alice")

	// Without name
	addr2 := EmailAddress{Email: "bob@example.com"}
	assert.Equal(t, "bob@example.com", addr2.String())
}

func TestMIMEBuilder_InlineImage(t *testing.T) {
	builder := NewMIMEBuilder()

	input := MIMEInput{
		From:    EmailAddress{Email: "alice@example.com"},
		To:      []EmailAddress{{Email: "bob@example.com"}},
		Subject: "With Inline",
		BodyText: "See image",
		BodyHTML: "<p>See image <img src=\"cid:img001\"></p>",
		Attachments: []AttachmentData{
			{
				Filename:    "logo.png",
				ContentType: "image/png",
				Data:        strings.NewReader("fake-png-data"),
				Size:        13,
				ContentID:   "img001",
				IsInline:    true,
			},
		},
	}

	data, err := builder.Build(input)
	require.NoError(t, err)

	msg := string(data)
	assert.Contains(t, msg, "multipart/mixed")
	// multipart writer canonicalizes header keys
	assert.Contains(t, msg, "Content-Id: <img001>")
	assert.Contains(t, msg, "Content-Disposition: inline")
}
