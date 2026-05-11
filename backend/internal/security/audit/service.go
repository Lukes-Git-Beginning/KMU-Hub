package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service provides audit logging and chain verification functionality.
type Service struct {
	repo Repository
}

// NewService creates a new audit Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// LogEvent records an audit event. It never returns an error to the caller;
// failures are logged internally to avoid disrupting business operations.
// tenantID is the tenant that owns this audit entry (sentinel UUID for system events).
func (s *Service) LogEvent(
	ctx context.Context,
	tenantID uuid.UUID,
	userID *uuid.UUID,
	action, target, targetType string,
	details map[string]interface{},
	ipAddress, userAgent, result string,
) {
	entry := &models.AuditEntry{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Timestamp:  time.Now().UTC(),
		UserID:     userID,
		Action:     action,
		Target:     target,
		TargetType: targetType,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Result:     result,
	}

	if details != nil {
		detailsJSON, err := json.Marshal(details)
		if err != nil {
			slog.Error("failed to marshal audit details",
				"action", action,
				"error", err,
			)
		} else {
			entry.Details = string(detailsJSON)
		}
	}

	// audit_log.details is jsonb. The repository binds entry.Details as a
	// Go string, and Postgres' json input parser rejects the empty string
	// with SQLSTATE 22P02. Normalize the zero value to '{}' so callers
	// passing nil details (or whose json.Marshal failed above) still record
	// the audit entry instead of silently dropping it.
	if entry.Details == "" {
		entry.Details = "{}"
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		slog.Error("failed to create audit entry",
			"action", action,
			"target", target,
			"user_id", userID,
			"error", err,
		)
	}
}

// ListEntries returns audit entries matching the given filter.
func (s *Service) ListEntries(ctx context.Context, filter *models.AuditFilter) ([]*models.AuditEntry, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return s.repo.List(ctx, filter)
}

// VerifyChainIntegrity checks the hash chain integrity between two sequence numbers.
// Returns (valid, firstBrokenSequence, error).
func (s *Service) VerifyChainIntegrity(ctx context.Context, fromSequence, toSequence int64) (bool, int64, error) {
	if fromSequence < 1 {
		fromSequence = 1
	}
	if toSequence < fromSequence {
		return false, 0, fmt.Errorf("toSequence (%d) must be >= fromSequence (%d)", toSequence, fromSequence)
	}

	return s.repo.VerifyChain(ctx, fromSequence, toSequence)
}

// ExportEntries exports audit entries in the specified format ("csv" or "json").
func (s *Service) ExportEntries(ctx context.Context, filter *models.AuditFilter, format string) ([]byte, error) {
	// Remove pagination limits for export
	exportFilter := *filter
	exportFilter.Limit = 0
	exportFilter.Offset = 0

	entries, _, err := s.repo.List(ctx, &exportFilter)
	if err != nil {
		return nil, fmt.Errorf("list entries for export: %w", err)
	}

	switch format {
	case "csv":
		return ExportCSV(entries)
	case "json":
		return ExportJSON(entries)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}
