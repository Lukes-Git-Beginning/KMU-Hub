// Package delivery holds the concrete scheduler collaborators for berichte:
// rendering a due report into a file and mailing it to the schedule's
// recipients. The scheduler itself only knows the narrow Exporter/Mailer
// interfaces; this package is where they meet the export renderers and the
// system SMTP account, so the whole chain (render -> attach -> SMTP) can be
// tested without standing up cmd/berichte.
package delivery

import (
	"bytes"
	"context"

	"github.com/kmuhub/kmuhub/internal/berichte"
	"github.com/kmuhub/kmuhub/internal/berichte/export"
	"github.com/kmuhub/kmuhub/internal/berichte/scheduler"
	"github.com/kmuhub/kmuhub/internal/email/send"
	"github.com/kmuhub/kmuhub/internal/email/systemmail"
)

// Exporter renders a report result in the schedule's format.
type Exporter struct{}

// NewExporter returns the scheduler's exporter.
func NewExporter() *Exporter { return &Exporter{} }

// Export implements scheduler.Exporter.
func (*Exporter) Export(_ context.Context, result *berichte.ReportResult, def *berichte.Definition, format string) (*scheduler.ExportedReport, error) {
	name := ""
	if def != nil {
		name = def.Name
	}
	rendered, err := export.Render(result, name, format)
	if err != nil {
		return nil, err
	}
	return &scheduler.ExportedReport{
		Payload:     rendered.Payload,
		ContentType: rendered.ContentType,
		Filename:    rendered.Filename,
	}, nil
}

// Mailer delivers a rendered report through the system SMTP account.
type Mailer struct {
	sender *systemmail.Sender
}

// NewMailer wraps sender. Callers must only wire a configured sender: an
// unconfigured one makes Send fail, which the scheduler records as a failed
// run -- correct, but noisier than never scheduling delivery at all.
func NewMailer(sender *systemmail.Sender) *Mailer { return &Mailer{sender: sender} }

// SendReport implements scheduler.Mailer.
func (m *Mailer) SendReport(ctx context.Context, in scheduler.MailInput) error {
	to := make([]send.EmailAddress, 0, len(in.Recipients))
	for _, addr := range in.Recipients {
		to = append(to, send.EmailAddress{Email: addr})
	}

	return m.sender.Send(ctx, systemmail.Message{
		To:       to,
		Subject:  in.Subject,
		BodyText: in.BodyText,
		BodyHTML: in.BodyHTML,
		Attachments: []send.AttachmentData{{
			Filename:    in.Attachment.Filename,
			ContentType: in.Attachment.ContentType,
			Data:        bytes.NewReader(in.Attachment.Payload),
			Size:        int64(len(in.Attachment.Payload)),
		}},
	})
}
