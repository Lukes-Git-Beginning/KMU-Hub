package helpdesk

import (
	"context"

	"github.com/kmuhub/kmuhub/internal/email/send"
	"github.com/kmuhub/kmuhub/internal/email/systemmail"
)

// SystemMailCsatMailer delivers survey invitations through the shared system
// SMTP account -- the same transport scheduled reports use. There is no
// authenticated user behind a survey mail, so a user mailbox is not an option.
type SystemMailCsatMailer struct {
	sender *systemmail.Sender
}

// NewSystemMailCsatMailer wraps sender. Callers must only wire a configured
// sender: an unconfigured one fails every send, which would burn the dispatch
// attempts of every pending survey instead of simply not sending yet.
func NewSystemMailCsatMailer(sender *systemmail.Sender) *SystemMailCsatMailer {
	return &SystemMailCsatMailer{sender: sender}
}

// SendCsatSurvey implements CsatSurveyMailer.
func (m *SystemMailCsatMailer) SendCsatSurvey(ctx context.Context, mail CsatSurveyMail) error {
	return m.sender.Send(ctx, systemmail.Message{
		To:       []send.EmailAddress{{Name: mail.Name, Email: mail.Recipient}},
		Subject:  mail.Subject,
		BodyText: mail.BodyText,
		BodyHTML: mail.BodyHTML,
	})
}
