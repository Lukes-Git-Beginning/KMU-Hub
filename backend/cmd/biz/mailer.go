package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kmuhub/kmuhub/internal/email/send"
	"github.com/kmuhub/kmuhub/internal/email/systemmail"
)

// systemMailer implements dunning.NoticeSender on top of the shared system SMTP
// sender (internal/email/systemmail), which owns the dial/STARTTLS flow and its
// timeouts. When no SMTP host is configured the mailer logs instead of failing:
// the dunning notice is a side effect of a status change that is already
// persisted, so a dev/CI environment without SMTP must not turn the whole
// dunning run into an error.
type systemMailer struct {
	sender *systemmail.Sender
}

// SendDunningNotice implements dunning.NoticeSender.
func (m *systemMailer) SendDunningNotice(ctx context.Context, toEmail, toName string, pdfBytes []byte, filename, subject string, level int) error {
	body := dunningNoticeBody(toName, level)

	err := m.sender.Send(ctx, systemmail.Message{
		To:       []send.EmailAddress{{Name: toName, Email: toEmail}},
		Subject:  subject,
		BodyText: body,
		BodyHTML: "<p>" + body + "</p>",
		Attachments: []send.AttachmentData{{
			Filename:    filename,
			ContentType: "application/pdf",
			Data:        bytes.NewReader(pdfBytes),
			Size:        int64(len(pdfBytes)),
		}},
	})
	switch {
	case errors.Is(err, systemmail.ErrNotConfigured):
		slog.Info("system SMTP not configured — dunning notice email suppressed",
			"to", toEmail,
			"filename", filename,
			"level", level,
		)
		return nil
	case err != nil:
		return fmt.Errorf("system mailer: send dunning notice to %s: %w", toEmail, err)
	}

	slog.Info("dunning notice email sent", "to", toEmail, "level", level, "filename", filename)
	return nil
}

// dunningNoticeBody returns a level-appropriate German body text (friendly ->
// formal -> urgent), matching the PDF's own tone escalation
// (see pdf.buildDunningBody).
func dunningNoticeBody(toName string, level int) string {
	greeting := fmt.Sprintf("Hallo %s,", toName)
	switch level {
	case 1:
		return greeting + "\r\n\r\nanbei erhalten Sie eine freundliche Zahlungserinnerung zu einer offenen Rechnung. Bitte pruefen Sie den Vorgang.\r\n\r\n-- Ihr Cosmi-Team"
	case 2:
		return greeting + "\r\n\r\nanbei erhalten Sie die 1. Mahnung zu einer offenen Rechnung. Bitte begleichen Sie den Betrag zeitnah.\r\n\r\n-- Ihr Cosmi-Team"
	default:
		return greeting + "\r\n\r\nanbei erhalten Sie die 2./letzte Mahnung zu einer offenen Rechnung. Bitte begleichen Sie den Betrag umgehend, um weitere Schritte zu vermeiden.\r\n\r\n-- Ihr Cosmi-Team"
	}
}
