package send

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EmailAddress represents a name + email pair.
type EmailAddress struct {
	Name  string
	Email string
}

// String returns the formatted email address.
func (a EmailAddress) String() string {
	if a.Name != "" {
		return fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("utf-8", a.Name), a.Email)
	}
	return a.Email
}

// AttachmentData holds attachment data for MIME construction.
type AttachmentData struct {
	Filename    string
	ContentType string
	Data        io.Reader
	Size        int64
	ContentID   string // For inline images (CID)
	IsInline    bool
}

// MIMEInput holds the input for building a MIME message.
type MIMEInput struct {
	From       EmailAddress
	To         []EmailAddress
	CC         []EmailAddress
	BCC        []EmailAddress
	Subject    string
	BodyHTML   string
	BodyText   string
	Attachments []AttachmentData
	InReplyTo  string
	References []string
	MessageID  string
}

// MIMEBuilder constructs RFC-compliant MIME messages.
type MIMEBuilder struct{}

// NewMIMEBuilder creates a new MIME builder.
func NewMIMEBuilder() *MIMEBuilder {
	return &MIMEBuilder{}
}

// Build constructs a complete MIME message from the input.
func (b *MIMEBuilder) Build(input MIMEInput) ([]byte, error) {
	var buf bytes.Buffer

	// Generate Message-ID if not provided
	if input.MessageID == "" {
		input.MessageID = fmt.Sprintf("<%s@kmuhub>", uuid.New().String())
	}

	hasAttachments := len(input.Attachments) > 0
	hasInlineImages := false
	for _, att := range input.Attachments {
		if att.IsInline {
			hasInlineImages = true
			break
		}
	}

	// Write top-level headers
	writeHeader(&buf, "From", input.From.String())
	writeHeader(&buf, "To", formatAddressList(input.To))
	if len(input.CC) > 0 {
		writeHeader(&buf, "Cc", formatAddressList(input.CC))
	}
	writeHeader(&buf, "Date", time.Now().UTC().Format(time.RFC1123Z))
	writeHeader(&buf, "Subject", mime.QEncoding.Encode("utf-8", input.Subject))
	writeHeader(&buf, "Message-ID", input.MessageID)
	writeHeader(&buf, "MIME-Version", "1.0")

	// Threading headers
	if input.InReplyTo != "" {
		writeHeader(&buf, "In-Reply-To", input.InReplyTo)
	}
	if len(input.References) > 0 {
		writeHeader(&buf, "References", strings.Join(input.References, " "))
	}

	if hasAttachments {
		return b.buildWithAttachments(&buf, input, hasInlineImages)
	}
	return b.buildAlternative(&buf, input)
}

// buildAlternative creates a multipart/alternative message (text + HTML).
func (b *MIMEBuilder) buildAlternative(buf *bytes.Buffer, input MIMEInput) ([]byte, error) {
	writer := multipart.NewWriter(buf)
	writeHeader(buf, "Content-Type", fmt.Sprintf("multipart/alternative; boundary=%s", writer.Boundary()))
	buf.WriteString("\r\n")

	if err := writeTextPart(writer, input.BodyText); err != nil {
		return nil, err
	}
	if err := writeHTMLPart(writer, input.BodyHTML); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// buildWithAttachments creates a multipart/mixed message with text/html body and attachments.
func (b *MIMEBuilder) buildWithAttachments(buf *bytes.Buffer, input MIMEInput, hasInline bool) ([]byte, error) {
	mixedWriter := multipart.NewWriter(buf)
	writeHeader(buf, "Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", mixedWriter.Boundary()))
	buf.WriteString("\r\n")

	// First part: the body (alternative or related)
	bodyHeader := make(textproto.MIMEHeader)
	if hasInline {
		relatedWriter := multipart.NewWriter(nil) // just for boundary
		bodyHeader.Set("Content-Type", fmt.Sprintf("multipart/related; boundary=%s", relatedWriter.Boundary()))
		bodyPart, err := mixedWriter.CreatePart(bodyHeader)
		if err != nil {
			return nil, err
		}

		innerRelated := multipart.NewWriter(bodyPart)
		// Override boundary to match the one in the header
		if err := writeAlternativePart(innerRelated, input.BodyText, input.BodyHTML); err != nil {
			return nil, err
		}

		// Inline images
		for _, att := range input.Attachments {
			if att.IsInline {
				if err := writeAttachmentPart(innerRelated, att); err != nil {
					return nil, err
				}
			}
		}
		_ = innerRelated.Close()
	} else {
		altWriter := multipart.NewWriter(nil) // just for boundary
		bodyHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%s", altWriter.Boundary()))
		bodyPart, err := mixedWriter.CreatePart(bodyHeader)
		if err != nil {
			return nil, err
		}

		innerAlt := multipart.NewWriter(bodyPart)
		if err := writeTextPart(innerAlt, input.BodyText); err != nil {
			return nil, err
		}
		if err := writeHTMLPart(innerAlt, input.BodyHTML); err != nil {
			return nil, err
		}
		_ = innerAlt.Close()
	}

	// Attachment parts
	for _, att := range input.Attachments {
		if !att.IsInline {
			if err := writeAttachmentPart(mixedWriter, att); err != nil {
				return nil, err
			}
		}
	}

	if err := mixedWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// writeHeader writes a single header line.
func writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

// formatAddressList formats a list of email addresses.
func formatAddressList(addrs []EmailAddress) string {
	parts := make([]string, 0, len(addrs))
	for _, a := range addrs {
		parts = append(parts, a.String())
	}
	return strings.Join(parts, ", ")
}

// writeTextPart writes a text/plain part.
func writeTextPart(w *multipart.Writer, text string) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "text/plain; charset=utf-8")
	header.Set("Content-Transfer-Encoding", "quoted-printable")

	part, err := w.CreatePart(header)
	if err != nil {
		return err
	}

	qpw := quotedprintable.NewWriter(part)
	_, err = qpw.Write([]byte(text))
	if err != nil {
		return err
	}
	return qpw.Close()
}

// writeHTMLPart writes a text/html part.
func writeHTMLPart(w *multipart.Writer, html string) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Content-Transfer-Encoding", "quoted-printable")

	part, err := w.CreatePart(header)
	if err != nil {
		return err
	}

	qpw := quotedprintable.NewWriter(part)
	_, err = qpw.Write([]byte(html))
	if err != nil {
		return err
	}
	return qpw.Close()
}

// writeAlternativePart writes a multipart/alternative part (text + HTML).
func writeAlternativePart(w *multipart.Writer, text, html string) error {
	if err := writeTextPart(w, text); err != nil {
		return err
	}
	return writeHTMLPart(w, html)
}

// writeAttachmentPart writes an attachment or inline image part.
func writeAttachmentPart(w *multipart.Writer, att AttachmentData) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", att.ContentType)
	header.Set("Content-Transfer-Encoding", "base64")

	if att.IsInline && att.ContentID != "" {
		header.Set("Content-ID", fmt.Sprintf("<%s>", att.ContentID))
		header.Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", att.Filename))
	} else {
		header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.Filename))
	}

	part, err := w.CreatePart(header)
	if err != nil {
		return err
	}

	encoder := base64.NewEncoder(base64.StdEncoding, part)
	if _, err := io.Copy(encoder, att.Data); err != nil {
		return err
	}
	return encoder.Close()
}
