package action

import (
	"context"
	"encoding/json"
	"fmt"

	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// SendEmailAction sends an email via the Email service gRPC.
type SendEmailAction struct {
	client emailv1.EmailServiceClient
}

// NewSendEmailAction creates a new SendEmailAction.
func NewSendEmailAction(client emailv1.EmailServiceClient) *SendEmailAction {
	return &SendEmailAction{client: client}
}

func (a *SendEmailAction) Type() string { return "email.send" }

type sendEmailConfig struct {
	AccountID string `json:"account_id"` // email account to send from
	To        string `json:"to"`         // template: recipient email address
	Subject   string `json:"subject"`    // template
	Body      string `json:"body"`       // template (HTML or plain text)
}

func (a *SendEmailAction) Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error) {
	if a.client == nil {
		return ActionResult{Success: false, Error: "Email service unavailable"}, fmt.Errorf("Email gRPC client is nil")
	}

	var cfg sendEmailConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ActionResult{Success: false, Error: "invalid config: " + err.Error()}, err
	}

	to := resolveTemplate(cfg.To, env)
	subject := resolveTemplate(cfg.Subject, env)
	body := resolveTemplate(cfg.Body, env)
	accountID := resolveTemplate(cfg.AccountID, env)

	resp, err := a.client.SendEmail(ctx, &emailv1.SendEmailRequest{
		AccountId: accountID,
		To: []*emailv1.EmailAddress{
			{Email: to},
		},
		Subject:  subject,
		BodyHtml: body,
		BodyText: body,
	})
	if err != nil {
		return ActionResult{Success: false, Error: err.Error()}, err
	}

	return ActionResult{
		Success: true,
		Output: map[string]any{
			"message_id": resp.GetMessage().GetId(),
		},
	}, nil
}

// SendEmailDefinition returns the action definition for the catalog.
func SendEmailDefinition() *ActionDefinition {
	return &ActionDefinition{
		Type:        "email.send",
		Module:      "email",
		Name:        "E-Mail senden",
		Description: "Sendet eine E-Mail ueber den E-Mail-Service",
		Params: []ConfigParam{
			{Key: "account_id", Label: "E-Mail-Konto", Type: "string", Required: true},
			{Key: "to", Label: "Empfaenger", Type: "template", Required: true},
			{Key: "subject", Label: "Betreff", Type: "template", Required: true},
			{Key: "body", Label: "Inhalt", Type: "template", Required: true},
		},
		OutputFields: []OutputField{
			{Key: "message_id", Label: "Nachrichten-ID", Type: "string"},
		},
	}
}
