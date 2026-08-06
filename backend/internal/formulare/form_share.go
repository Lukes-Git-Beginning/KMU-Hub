package formulare

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// FormShareLink is one public fill-out link for a form schema (table
// form_share_tokens, migration 000293). The token is the whole credential the
// public route accepts, so this struct carries exactly what deciding "may this
// link still be used, and for whom" needs.
type FormShareLink struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	FormSchemaID uuid.UUID  `json:"form_schema_id"`
	Token        string     `json:"token"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	// MaxSubmissions nil = unlimited.
	MaxSubmissions  *int       `json:"max_submissions,omitempty"`
	SubmissionCount int        `json:"submission_count"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// Share link status values, mirroring ShareLinkStatus in
// desktop/src/renderer/src/api/formulare-types.ts.
const (
	ShareLinkStatusActive   = "active"
	ShareLinkStatusExpired  = "expired"
	ShareLinkStatusDisabled = "disabled"
)

// Usable reports whether the link may still be redeemed. Every negative
// verdict here becomes the same ErrShareLinkNotFound an invented token gets:
// revoked, expired, used up and never-existed must not be distinguishable from
// outside. It is an advisory check only -- the authoritative one lives in the
// redeem statement, because two visitors submitting at once is exactly the
// race a read cannot cover.
func (l *FormShareLink) Usable(now time.Time) bool {
	if l.RevokedAt != nil {
		return false
	}
	if l.ExpiresAt != nil && !now.Before(*l.ExpiresAt) {
		return false
	}
	if l.MaxSubmissions != nil && l.SubmissionCount >= *l.MaxSubmissions {
		return false
	}
	return true
}

// Status is the derived state the frontend renders. Order matters: a link that
// was cut by hand reads "disabled" even if it would also have expired by now,
// because that is the fact its author acted on.
func (l *FormShareLink) Status(now time.Time) string {
	switch {
	case l.RevokedAt != nil:
		return ShareLinkStatusDisabled
	case l.ExpiresAt != nil && !now.Before(*l.ExpiresAt):
		return ShareLinkStatusExpired
	case l.MaxSubmissions != nil && l.SubmissionCount >= *l.MaxSubmissions:
		return ShareLinkStatusExpired
	default:
		return ShareLinkStatusActive
	}
}

const (
	// shareTokenBytes matches the report share links (000252) and the CSAT
	// survey links: 32 bytes of crypto/rand, base64url-encoded to 43 URL-safe
	// characters. The token IS the credential, so it is drawn the same way.
	shareTokenBytes = 32

	// shareTokenMaxLength is the longest token string worth a database round
	// trip. Real tokens are 43 characters; anything past this is a probe, and
	// answering it should not cost a query.
	shareTokenMaxLength = 128

	// maxPublicAnswersBytes bounds the decoded answers of one public
	// submission. The gateway caps the raw body too; this is the service-side
	// floor, so the limit holds no matter who calls the RPC.
	maxPublicAnswersBytes = 256 * 1024

	// fileFieldType is the form builder's file-upload field
	// (FormFieldType in desktop/src/renderer/src/api/formulare-types.ts).
	// Public submission does not carry uploads -- see SubmitByShareToken.
	fileFieldType = "file"
)

// CreateShareLinkInput is the authenticated request to mint a public link.
type CreateShareLinkInput struct {
	TenantID     uuid.UUID
	FormSchemaID uuid.UUID
	ExpiresAt    *time.Time
	// MaxSubmissions nil = unlimited.
	MaxSubmissions *int
	CreatedBy      *uuid.UUID
}

// ShareTokenSubmission is what a redeemed public submission reports back. It
// carries no schema content and no answers: the visitor already has both. The
// ids are here because the caller (the gateway) needs them to run the intake
// dispatch, which is where a bound form turns into a module record.
type ShareTokenSubmission struct {
	SubmissionID uuid.UUID
	TenantID     uuid.UUID
	FormSchemaID uuid.UUID
}

// newShareToken draws a link secret from crypto/rand. base64url keeps it
// usable as a URL path segment without escaping.
func newShareToken() (string, error) {
	buf := make([]byte, shareTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate form share token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// withTenant attaches a tenant resolved from a share link to the context, so
// the repository and the RLS session GUCs see it. This is the single place a
// public path in this service may set a tenant, and it is only ever reached
// with the tenant the link itself resolved to.
func withTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
}

// CreateShareLink mints a public fill-out link for a schema.
//
// `is_public` is the precondition, not the gate: an author has to open the
// form up deliberately before a link can exist for it, but what a visitor
// presents is the token. Both are checked -- here at mint time, and again on
// every redemption, so revoking `is_public` kills the links that were already
// handed out instead of leaving them live.
func (s *Service) CreateShareLink(ctx context.Context, in CreateShareLinkInput) (*FormShareLink, error) {
	if in.MaxSubmissions != nil && *in.MaxSubmissions <= 0 {
		return nil, fmt.Errorf("max_submissions must be positive: %w", ErrInvalidShareLink)
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(time.Now().UTC()) {
		return nil, fmt.Errorf("expires_at must be in the future: %w", ErrInvalidShareLink)
	}

	schema, err := s.repo.GetSchema(ctx, in.FormSchemaID, in.TenantID)
	if err != nil {
		return nil, err
	}
	if schema.DeletedAt != nil {
		return nil, ErrSchemaDeleted
	}
	if !schema.IsPublic {
		return nil, ErrSchemaNotPublic
	}

	token, err := newShareToken()
	if err != nil {
		return nil, err
	}

	link := &FormShareLink{
		ID:             uuid.New(),
		TenantID:       in.TenantID,
		FormSchemaID:   in.FormSchemaID,
		Token:          token,
		ExpiresAt:      in.ExpiresAt,
		MaxSubmissions: in.MaxSubmissions,
		CreatedBy:      in.CreatedBy,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.repo.CreateShareLink(ctx, link); err != nil {
		return nil, fmt.Errorf("create form share link: %w", err)
	}

	s.logger.Info("formulare share link created",
		"share_link_id", link.ID,
		"form_schema_id", link.FormSchemaID,
		"tenant_id", link.TenantID,
		"expires_at", link.ExpiresAt,
	)
	return link, nil
}

// ListShareLinks returns every link of one schema, newest first, including
// revoked and expired ones: an author looking at an external write path needs
// to see what was cut, not just what is live.
func (s *Service) ListShareLinks(ctx context.Context, formSchemaID, tenantID uuid.UUID) ([]*FormShareLink, error) {
	return s.repo.ListShareLinks(ctx, formSchemaID, tenantID)
}

// RevokeShareLink cuts a link. Soft revoke: the row and its submission count
// survive, which is the only record of how often the link was used.
// Re-revoking is a no-op, not an error.
func (s *Service) RevokeShareLink(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.RevokeShareLink(ctx, id, tenantID, time.Now().UTC()); err != nil {
		return err
	}
	s.logger.Info("formulare share link revoked", "share_link_id", id, "tenant_id", tenantID)
	return nil
}

// SubmitByShareToken stores a submission handed in through a public link.
//
// This is the one path in the service that begins without a tenant, so the
// order of steps is the security property, not a matter of taste (same shape
// as helpdesk.Service.SubmitCsatByToken and berichte GetSharedDocument):
//
//  1. resolve the token in the system context -- the single read that must
//     escape RLS, because which tenant the caller may write to is precisely
//     what it answers;
//  2. refuse revoked, expired and used-up links as the same "not found" an
//     invented token gets;
//  3. re-enter tenant scope with the tenant the token itself named, and only
//     then read the schema and write. Everything after step 1 is ordinary
//     tenant-scoped work; the system context is never carried past the lookup.
//
// The schema is re-checked on every redemption, not just at mint time: a form
// switched back to private, archived or deleted must stop accepting answers
// through links already handed out. Every one of those verdicts is the same
// 404, for the same reason -- the route must not report which tokens exist.
func (s *Service) SubmitByShareToken(
	ctx context.Context,
	token string,
	answers []byte,
	ipAddress string,
) (*ShareTokenSubmission, error) {
	// Shape checks before the lookup: they are the caller's own error and say
	// nothing about the token, and an oversized body must not buy a query.
	if len(answers) == 0 {
		return nil, fmt.Errorf("answers are required: %w", ErrInvalidFields)
	}
	if len(answers) > maxPublicAnswersBytes {
		return nil, ErrAnswersTooLarge
	}
	if token == "" || len(token) > shareTokenMaxLength {
		return nil, ErrShareLinkNotFound
	}

	link, err := s.repo.GetShareLinkByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if !link.Usable(now) {
		s.logger.Info("formulare share link refused",
			"share_link_id", link.ID,
			"revoked", link.RevokedAt != nil,
			"expired", link.ExpiresAt != nil && !now.Before(*link.ExpiresAt),
			"quota_reached", link.MaxSubmissions != nil && link.SubmissionCount >= *link.MaxSubmissions,
		)
		return nil, ErrShareLinkNotFound
	}

	ctx = withTenant(ctx, link.TenantID)

	schema, err := s.repo.GetSchema(ctx, link.FormSchemaID, link.TenantID)
	if err != nil {
		// A link whose schema is gone is a dead link, not a server fault.
		if errors.Is(err, ErrSchemaNotFound) {
			return nil, ErrShareLinkNotFound
		}
		return nil, err
	}
	if schema.DeletedAt != nil || !schema.IsPublic || schema.Status == FormSchemaStatusArchived {
		s.logger.Info("formulare share link refused by schema state",
			"share_link_id", link.ID,
			"form_schema_id", schema.ID,
			"is_public", schema.IsPublic,
			"status", schema.Status,
			"deleted", schema.DeletedAt != nil,
		)
		return nil, ErrShareLinkNotFound
	}
	if hasFileField(schema.Fields) {
		// Rejected outright rather than half-supported: a public submission
		// carries JSON answers, and there is no upload path on this route to
		// put a file behind a "file" answer. Accepting the form anyway would
		// store an attachment reference pointing at nothing.
		return nil, ErrShareLinkFileFieldUnsupported
	}

	webhooks, err := s.repo.ListActiveWebhooksForSchema(ctx, link.TenantID, link.FormSchemaID)
	if err != nil {
		return nil, fmt.Errorf("list active webhooks: %w", err)
	}
	webhookIDs := make([]uuid.UUID, 0, len(webhooks))
	for _, w := range webhooks {
		webhookIDs = append(webhookIDs, w.ID)
	}

	submission := &FormSubmission{
		ID:           uuid.New(),
		FormSchemaID: &link.FormSchemaID,
		TenantID:     link.TenantID,
		Answers:      answers,
		Status:       FormSubmissionStatusNew,
		IPAddress:    parseIPAddress(ipAddress),
		SubmittedAt:  now,
	}

	// The repository repeats "still live, still under quota" inside the write.
	// Two visitors redeeming the last slot at once is exactly the race the
	// check above cannot cover, and the quota must hold.
	redeemed, err := s.repo.RedeemShareLinkTx(ctx, link.ID, link.TenantID, submission, webhookIDs, now)
	if err != nil {
		return nil, fmt.Errorf("redeem form share link: %w", err)
	}
	if !redeemed {
		return nil, ErrShareLinkNotFound
	}

	s.logger.Info("formulare public submission stored",
		"submission_id", submission.ID,
		"share_link_id", link.ID,
		"form_schema_id", link.FormSchemaID,
		"tenant_id", link.TenantID,
		"webhook_count", len(webhookIDs),
	)
	return &ShareTokenSubmission{
		SubmissionID: submission.ID,
		TenantID:     link.TenantID,
		FormSchemaID: link.FormSchemaID,
	}, nil
}

// hasFileField reports whether a schema's raw `fields` JSON declares an upload
// field. A malformed array answers false: rejecting a public submission on the
// strength of JSON this code could not read would be the wrong call, and the
// mapper downstream reports the same problem with a better message.
func hasFileField(raw []byte) bool {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return false
	}
	var fields []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return false
	}
	for _, f := range fields {
		if strings.EqualFold(strings.TrimSpace(f.Type), fileFieldType) {
			return true
		}
	}
	return false
}
