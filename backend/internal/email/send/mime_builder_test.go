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
	// Top level (multipart/mixed) is unaffected by the nested-boundary bug
	// documented in TestMIMEBuilder_WithAttachments_NestedBoundaryMismatch_
	// DocumentsCurrentGap below: parts[0] is the (currently undecodable)
	// text/html body, parts[1] and parts[2] are the two attachments.
	require.Len(t, parts, 3)

	assert.Contains(t, parts[0].header.Get("Content-Type"), "multipart/alternative")

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
	require.Len(t, parts, 2) // body part (undecodable, see gap test) + attachment

	_, params, err := mime.ParseMediaType(parts[1].header.Get("Content-Disposition"))
	require.NoError(t, err)
	assert.Equal(t, "", params["filename"])
	assert.Equal(t, []byte("data"), decodeLeafBody(t, parts[1]))
}

// TestMIMEBuilder_WithAttachments_NestedBoundaryMismatch_DocumentsCurrentGap proves
// a real production bug rather than a test artifact: buildWithAttachments
// (mime_builder.go) declares the nested text/html body part's boundary from one
// throwaway multipart.Writer ("just for the boundary") but then writes the actual
// text/html content through a SECOND, independently-created multipart.Writer with
// its own random boundary. The declared and actual boundaries never match, so no
// RFC-2046-compliant client can ever recover the text/html body of an email that
// carries at least one attachment (inline or not - both branches share this
// pattern). This is not fixed here: it is a genuine behavior change belonging to
// its own backlog unit (see BACKLOG.yml), not this coverage unit.
func TestMIMEBuilder_WithAttachments_NestedBoundaryMismatch_DocumentsCurrentGap(t *testing.T) {
	builder := NewMIMEBuilder()

	input := MIMEInput{
		From:     EmailAddress{Email: "alice@example.com"},
		To:       []EmailAddress{{Email: "bob@example.com"}},
		Subject:  "Boundary mismatch",
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
	mediaType, params, err := mime.ParseMediaType(bodyPart.header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/alternative", mediaType)
	declaredBoundary := params["boundary"]

	mr := multipart.NewReader(bytes.NewReader(bodyPart.body), declaredBoundary)
	_, err = mr.NextPart()
	assert.ErrorIs(t, err, io.EOF, "expected the declared boundary to find zero parts (current gap) - if this now finds a part, the bug is fixed and this test should be rewritten to assert the correct roundtrip")

	assert.NotContains(t, string(bodyPart.body), "--"+declaredBoundary)
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
