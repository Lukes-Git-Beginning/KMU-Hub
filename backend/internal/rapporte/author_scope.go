package rapporte

import (
	"context"

	"github.com/google/uuid"
)

type authorOnlyKey struct{}

// WithAuthorOnly marks ctx as belonging to a caller whose grant is scoped to "own":
// they may only reach reports they wrote, and only while the report is still theirs
// to edit (draft or rejected). The gRPC layer sets it from the gateway's
// restrict_to_author flag plus the x-user-id the gateway propagates — never from a
// request field, so the body cannot name somebody else.
func WithAuthorOnly(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, authorOnlyKey{}, userID)
}

func authorOnly(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(authorOnlyKey{}).(uuid.UUID)
	return id, ok
}

// loadReport fetches a report; for an author-only caller somebody else's report is
// reported as not found, so its existence does not leak.
func (s *Service) loadReport(ctx context.Context, tenantID, reportID uuid.UUID) (*WorkReport, error) {
	report, err := s.repo.GetReport(ctx, tenantID, reportID)
	if err != nil {
		return nil, err
	}
	if author, restricted := authorOnly(ctx); restricted && report.AuthorID != author {
		return nil, ErrReportNotFound
	}
	return report, nil
}

// requireEditable is the write guard for a report's content (fields, lines,
// attachments, signature). Approved reports are immutable for everyone; an
// author-only caller additionally loses the report once it is submitted.
// lean: lock gilt nur für own-Scope; Manager/Admin korrigieren eingereichte Rapporte
// weiter, bis es eine eigene Review-Bearbeitung gibt.
func requireEditable(ctx context.Context, report *WorkReport) error {
	if report.Status == StatusApproved {
		return ErrAlreadyApproved
	}
	if _, restricted := authorOnly(ctx); restricted && report.Status != StatusDraft && report.Status != StatusRejected {
		return ErrInvalidStateTransition
	}
	return nil
}

// AuthorizeReport lets callers outside the service (measurements go straight to the
// repository) apply the same author check and editability guard.
func (s *Service) AuthorizeReport(ctx context.Context, tenantID, reportID uuid.UUID, write bool) error {
	report, err := s.loadReport(ctx, tenantID, reportID)
	if err != nil {
		return err
	}
	if write {
		return requireEditable(ctx, report)
	}
	return nil
}
