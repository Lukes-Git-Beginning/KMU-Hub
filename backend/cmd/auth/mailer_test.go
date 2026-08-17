package main

import (
	"bytes"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
	"time"
)

const (
	testResetLink = "https://app.zentria.tech/reset-password?token=abc123&x=1"
	testToEmail   = "empfaenger@example.org"
)

// buildTestMessage returns a fixed message so assertions do not depend on
// randomness or the clock.
func buildTestMessage(t *testing.T) []byte {
	t.Helper()
	m := &systemMailer{host: "smtp.example.com", port: 587, from: "noreply@zentria.tech"}
	return m.buildPasswordResetMessage(testToEmail, "Luke", testResetLink,
		"boundary0123456789", "<fixed@zentria.tech>",
		time.Date(2026, 8, 17, 15, 4, 5, 0, time.UTC))
}

// parseParts walks the multipart/alternative body and returns each part's media
// type mapped to its decoded text. Parsing rather than substring matching is the
// point: quoted-printable rewrites the '=' in the reset link's query string, so
// a raw strings.Contains would pass on a message no client can read.
func parseParts(t *testing.T, raw []byte) map[string]string {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("message is not well-formed: %v", err)
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse Content-Type: %v", err)
	}
	if mediaType != "multipart/alternative" {
		t.Fatalf("Content-Type = %q, want multipart/alternative", mediaType)
	}

	parts := map[string]string{}
	mr := multipart.NewReader(msg.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		// multipart.Part hides Content-Transfer-Encoding and decodes
		// quoted-printable transparently, so read the part directly — the raw
		// header is asserted separately below.
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read part: %v", err)
		}
		pt, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("parse part Content-Type: %v", err)
		}
		parts[pt] = string(body)
	}
	return parts
}

func TestPasswordResetMessageHeaders(t *testing.T) {
	msg, err := mail.ReadMessage(bytes.NewReader(buildTestMessage(t)))
	if err != nil {
		t.Fatalf("message is not well-formed: %v", err)
	}

	// Date and Message-ID were both missing before; their absence costs spam
	// score on its own, independent of SPF and DKIM.
	if got := msg.Header.Get("Date"); got != "Mon, 17 Aug 2026 15:04:05 +0000" {
		t.Errorf("Date = %q", got)
	}
	if got := msg.Header.Get("Message-ID"); got != "<fixed@zentria.tech>" {
		t.Errorf("Message-ID = %q", got)
	}
	if got := msg.Header.Get("Auto-Submitted"); got != "auto-generated" {
		t.Errorf("Auto-Submitted = %q", got)
	}
	if got := msg.Header.Get("To"); got != testToEmail {
		t.Errorf("To = %q", got)
	}

	// The subject carries an umlaut, so it must be RFC 2047 encoded on the wire
	// and decode back to the German original.
	rawSubject := msg.Header.Get("Subject")
	if !strings.HasPrefix(rawSubject, "=?") {
		t.Errorf("Subject is not RFC 2047 encoded: %q", rawSubject)
	}
	decoded, err := new(mime.WordDecoder).DecodeHeader(rawSubject)
	if err != nil {
		t.Fatalf("decode subject: %v", err)
	}
	if want := "Passwort für dein Cosmi-Konto zurücksetzen"; decoded != want {
		t.Errorf("Subject = %q, want %q", decoded, want)
	}
}

func TestPasswordResetMessageBothPartsCarryLinkAndAccount(t *testing.T) {
	raw := buildTestMessage(t)
	parts := parseParts(t, raw)

	// Asserted on the wire format, because multipart.Part hides this header.
	if got := strings.Count(string(raw), "Content-Transfer-Encoding: quoted-printable"); got != 2 {
		t.Errorf("quoted-printable declared %d times, want 2", got)
	}

	if len(parts) != 2 {
		t.Fatalf("got %d parts, want text/plain and text/html", len(parts))
	}
	// The html part must escape the query string's '&' to '&amp;' inside the
	// href, so each part expects its own rendering of the same link.
	wantLink := map[string]string{
		"text/plain": testResetLink,
		"text/html":  html.EscapeString(testResetLink),
	}
	for _, pt := range []string{"text/plain", "text/html"} {
		body, ok := parts[pt]
		if !ok {
			t.Fatalf("missing %s part", pt)
		}
		if !strings.Contains(body, wantLink[pt]) {
			t.Errorf("%s part does not carry the reset link", pt)
		}
		// Naming the account is what keeps someone with several accounts from
		// resetting one and signing in to another.
		if !strings.Contains(body, testToEmail) {
			t.Errorf("%s part does not name the account address", pt)
		}
		if !strings.Contains(body, "Cosmi") {
			t.Errorf("%s part is unbranded", pt)
		}
	}

	// German is the product locale; an English mail was the reported defect.
	if !strings.Contains(parts["text/plain"], "Passwort") {
		t.Error("plain part is not German")
	}
	// No remote asset may creep in: that would be a tracking pixel under a
	// data-sovereignty promise, and it would break in offline clients.
	for _, needle := range []string{"<img", "@font-face", "http://"} {
		if strings.Contains(parts["text/html"], needle) {
			t.Errorf("html part contains %q", needle)
		}
	}
}

func TestMessageIDFallsBackWhenSenderHasNoDomain(t *testing.T) {
	if got := messageID("noreply@zentria.tech"); !strings.HasSuffix(got, "@zentria.tech>") {
		t.Errorf("messageID = %q, want the sender domain", got)
	}
	if got := messageID("garbage"); !strings.HasSuffix(got, "@localhost>") {
		t.Errorf("messageID = %q, want the localhost fallback", got)
	}
}
