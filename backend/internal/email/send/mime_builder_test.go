package send

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type shallowPart struct {
	header textproto.MIMEHeader
	body   []byte
}

// parseOneLevel parses a single multipart layer into its immediate child parts,
// without recursing into children that are themselves multipart.
func parseOneLevel(t *testing.T, header textproto.MIMEHeader, body []byte) []shallowPart {
	t.Helper()

	_, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	require.NoError(t, err)

	mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	var parts []shallowPart
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		data, err := io.ReadAll(p)
		require.NoError(t, err)

		parts = append(parts, shallowPart{header: p.Header, body: data})
	}
	return parts
}

func decodeLeafBody(t *testing.T, l shallowPart) []byte {
	t.Helper()

	switch strings.ToLower(l.header.Get("Content-Transfer-Encoding")) {
	case "quoted-printable":
		data, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(l.body)))
		require.NoError(t, err)
		return data
	case "base64":
		data, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, bytes.NewReader(l.body)))
		require.NoError(t, err)
		return data
	default:
		return l.body
	}
}

// mail.Header and textproto.MIMEHeader share the same underlying map type, so
// the top-level message header can feed into the same parser used for
// multipart.Part headers.
func parseBuiltMessage(t *testing.T, raw []byte) (*mail.Message, []shallowPart) {
	t.Helper()

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	require.NoError(t, err)

	body, err := io.ReadAll(msg.Body)
	require.NoError(t, err)

	parts := parseOneLevel(t, textproto.MIMEHeader(msg.Header), body)
	return msg, parts
}

func TestMIMEBuilder_ParseBack_PlainAndHTML(t *testing.T) {
	builder := NewMIMEBuilder()

	input := MIMEInput{
		From:     EmailAddress{Name: "Alice", Email: "alice@example.com"},
		To:       []EmailAddress{{Name: "Bob", Email: "bob@example.com"}},
		Subject:  "Roundtrip test",
		BodyText: "Plain body content",
		BodyHTML: "<p>HTML body content</p>",
	}

	raw, err := builder.Build(input)
	require.NoError(t, err)

	msg, parts := parseBuiltMessage(t, raw)

	assert.Contains(t, msg.Header.Get("From"), "alice@example.com")
	assert.Contains(t, msg.Header.Get("To"), "bob@example.com")
	assert.NotEmpty(t, msg.Header.Get("Message-ID"))
	assert.Equal(t, "1.0", msg.Header.Get("MIME-Version"))

	// No attachments: buildAlternative writes text+html through the SAME
	// multipart.Writer whose Boundary() it declared in the header, so this
	// (unlike the attachment path below) parses cleanly in one level.
	require.Len(t, parts, 2)
	assert.Equal(t, "Plain body content", string(decodeLeafBody(t, parts[0])))
	assert.Equal(t, "<p>HTML body content</p>", string(decodeLeafBody(t, parts[1])))
}

func TestMIMEBuilder_SubjectRFC2047_Umlauts(t *testing.T) {
	builder := NewMIMEBuilder()
	subject := "Grüße äöü ß und Emoji 😀 Test"

	input := MIMEInput{
		From:     EmailAddress{Email: "alice@example.com"},
		To:       []EmailAddress{{Email: "bob@example.com"}},
		Subject:  subject,
		BodyText: "text",
		BodyHTML: "<p>text</p>",
	}

	raw, err := builder.Build(input)
	require.NoError(t, err)

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	require.NoError(t, err)

	rawSubject := msg.Header.Get("Subject")
	// A correctly RFC-2047-encoded subject never carries the raw UTF-8 bytes on
	// the header line itself - only decoding it back reveals them. Asserting on
	// the encoded-word marker (and absence of raw umlauts) is what actually
	// proves the encoding happened, not just that some string survived.
	require.True(t, strings.HasPrefix(rawSubject, "=?utf-8?"), "expected RFC 2047 encoded-word, got %q", rawSubject)
	assert.NotContains(t, rawSubject, "ü")

	decoded, err := (&mime.WordDecoder{}).DecodeHeader(rawSubject)
	require.NoError(t, err)
	assert.Equal(t, subject, decoded)
}

// parseNestedBodyPart re-parses a shallowPart that is itself a multipart body
// (multipart/alternative or multipart/related) using its OWN declared
// Content-Type boundary - the only way a real MIME client would ever read it.
func parseNestedBodyPart(t *testing.T, part shallowPart) []shallowPart {
	t.Helper()
	return parseOneLevel(t, part.header, part.body)
}

func TestMIMEBuilder_MultipleAttachments(t *testing.T) {
	builder := NewMIMEBuilder()

	att1 := []byte("first attachment payload")
	att2 := []byte("second, somewhat longer attachment payload")

	input := MIMEInput{
		From:     EmailAddress{Email: "alice@example.com"},
		To:       []EmailAddress{{Email: "bob@example.com"}},
		Subject:  "Two attachments",
		BodyText: "see attached",
		BodyHTML: "<p>see attached</p>",
		Attachments: []AttachmentData{
			{Filename: "one.txt", ContentType: "text/plain", Data: bytes.NewReader(att1), Size: int64(len(att1))},
			{Filename: "two.bin", ContentType: "application/octet-stream", Data: bytes.NewReader(att2), Size: int64(len(att2))},
		},
	}

	raw, err := builder.Build(input)
	require.NoError(t, err)

	_, parts := parseBuiltMessage(t, raw)
	// Top level (multipart/mixed): parts[0] is the nested text/html body,
	// parts[1] and parts[2] are the two attachments.
	require.Len(t, parts, 3)

	assert.Contains(t, parts[0].header.Get("Content-Type"), "multipart/alternative")

	bodyParts := parseNestedBodyPart(t, parts[0])
	require.Len(t, bodyParts, 2)
	assert.Equal(t, "see attached", string(decodeLeafBody(t, bodyParts[0])))
	assert.Equal(t, "<p>see attached</p>", string(decodeLeafBody(t, bodyParts[1])))

	assert.Contains(t, parts[1].header.Get("Content-Disposition"), "one.txt")
	assert.Equal(t, att1, decodeLeafBody(t, parts[1]))

	assert.Contains(t, parts[2].header.Get("Content-Disposition"), "two.bin")
	assert.Equal(t, att2, decodeLeafBody(t, parts[2]))
}

func TestMIMEBuilder_AttachmentWithoutFilename(t *testing.T) {
	builder := NewMIMEBuilder()

	input := MIMEInput{
		From:     EmailAddress{Email: "alice@example.com"},
		To:       []EmailAddress{{Email: "bob@example.com"}},
		Subject:  "No filename",
		BodyText: "text",
		BodyHTML: "<p>text</p>",
		Attachments: []AttachmentData{
			{Filename: "", ContentType: "application/octet-stream", Data: bytes.NewReader([]byte("data")), Size: 4},
		},
	}

	raw, err := builder.Build(input)
	require.NoError(t, err)

	_, parts := parseBuiltMessage(t, raw)
	require.Len(t, parts, 2) // nested text/html body + attachment

	_, params, err := mime.ParseMediaType(parts[1].header.Get("Content-Disposition"))
	require.NoError(t, err)
	assert.Equal(t, "", params["filename"])
	assert.Equal(t, []byte("data"), decodeLeafBody(t, parts[1]))
}

// TestMIMEBuilder_WithAttachments_NestedBodyRoundTrips proves the fix for a real
// production bug: buildWithAttachments (mime_builder.go) used to declare the
// nested text/html body part's boundary from one throwaway multipart.Writer
// ("just for the boundary") but write the actual text/html content through a
// SECOND, independently-created multipart.Writer with its own random boundary.
// The declared and actual boundaries never matched, so no RFC-2046-compliant
// client could recover the text/html body of an email carrying at least one
// attachment. This asserts a full recursive parse - using the boundary the
// nested part itself declares, exactly as a real client would - now succeeds
// for both the "no inline images" and "has inline images" branches.
func TestMIMEBuilder_WithAttachments_NestedBodyRoundTrips(t *testing.T) {
	t.Run("no inline images", func(t *testing.T) {
		builder := NewMIMEBuilder()

		input := MIMEInput{
			From:     EmailAddress{Email: "alice@example.com"},
			To:       []EmailAddress{{Email: "bob@example.com"}},
			Subject:  "Boundary roundtrip",
			BodyText: "text",
			BodyHTML: "<p>text</p>",
			Attachments: []AttachmentData{
				{Filename: "a.txt", ContentType: "text/plain", Data: bytes.NewReader([]byte("x")), Size: 1},
			},
		}

		raw, err := builder.Build(input)
		require.NoError(t, err)

		_, parts := parseBuiltMessage(t, raw)
		require.Len(t, parts, 2)

		bodyPart := parts[0]
		mediaType, _, err := mime.ParseMediaType(bodyPart.header.Get("Content-Type"))
		require.NoError(t, err)
		require.Equal(t, "multipart/alternative", mediaType)

		bodyParts := parseNestedBodyPart(t, bodyPart)
		require.Len(t, bodyParts, 2)
		assert.Equal(t, "text", string(decodeLeafBody(t, bodyParts[0])))
		assert.Equal(t, "<p>text</p>", string(decodeLeafBody(t, bodyParts[1])))
	})

	t.Run("has inline images", func(t *testing.T) {
		builder := NewMIMEBuilder()

		input := MIMEInput{
			From:     EmailAddress{Email: "alice@example.com"},
			To:       []EmailAddress{{Email: "bob@example.com"}},
			Subject:  "Boundary roundtrip with inline image",
			BodyText: "see image",
			BodyHTML: `<p>see image <img src="cid:img001"></p>`,
			Attachments: []AttachmentData{
				{Filename: "logo.png", ContentType: "image/png", Data: bytes.NewReader([]byte("fake-png")), Size: 8, ContentID: "img001", IsInline: true},
				{Filename: "report.pdf", ContentType: "application/pdf", Data: bytes.NewReader([]byte("fake-pdf")), Size: 8},
			},
		}

		raw, err := builder.Build(input)
		require.NoError(t, err)

		_, parts := parseBuiltMessage(t, raw)
		require.Len(t, parts, 2) // nested related body + the non-inline attachment

		bodyPart := parts[0]
		mediaType, _, err := mime.ParseMediaType(bodyPart.header.Get("Content-Type"))
		require.NoError(t, err)
		require.Equal(t, "multipart/related", mediaType)

		// writeAlternativePart writes text and html as direct sibling parts of
		// the related writer (not nested in their own multipart/alternative),
		// so the related body has three flat children: text, html, inline image.
		relatedParts := parseNestedBodyPart(t, bodyPart)
		require.Len(t, relatedParts, 3)

		assert.Equal(t, "see image", string(decodeLeafBody(t, relatedParts[0])))
		assert.Equal(t, `<p>see image <img src="cid:img001"></p>`, string(decodeLeafBody(t, relatedParts[1])))

		assert.Contains(t, relatedParts[2].header.Get("Content-Id"), "img001")
		assert.Equal(t, []byte("fake-png"), decodeLeafBody(t, relatedParts[2]))

		assert.Contains(t, parts[1].header.Get("Content-Disposition"), "report.pdf")
		assert.Equal(t, []byte("fake-pdf"), decodeLeafBody(t, parts[1]))
	})
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("boom: attachment read failed")
}

func TestMIMEBuilder_AttachmentReadError(t *testing.T) {
	builder := NewMIMEBuilder()

	input := MIMEInput{
		From:     EmailAddress{Email: "alice@example.com"},
		To:       []EmailAddress{{Email: "bob@example.com"}},
		Subject:  "Broken attachment",
		BodyText: "text",
		BodyHTML: "<p>text</p>",
		Attachments: []AttachmentData{
			{Filename: "broken.bin", ContentType: "application/octet-stream", Data: errReader{}},
		},
	}

	_, err := builder.Build(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}
