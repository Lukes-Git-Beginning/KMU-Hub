package auth

import "context"

// Mailer is the narrow interface the auth service uses to dispatch
// transactional emails (password-reset link). The concrete implementation
// in cmd/auth/main.go wraps email/send.Service.
type Mailer interface {
	SendPasswordResetEmail(ctx context.Context, toEmail, toName, resetLink string) error
}
