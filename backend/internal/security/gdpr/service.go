package gdpr

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Service errors.
var (
	ErrExportAlreadyPending  = errors.New("gdpr: user already has a pending export request")
	ErrInvalidExportStatus   = errors.New("gdpr: export request is not in a valid state for this operation")
	ErrExportExpired         = errors.New("gdpr: export download has expired")
	ErrExportNotReady        = errors.New("gdpr: export is not ready for download")
	ErrErasureTargetNotFound = errors.New("gdpr: target user not found")
)

// Service orchestrates GDPR data export and right-to-erasure operations.
type Service struct {
	repo            Repository
	exportHandlers  []DataExportHandler
	erasureHandlers []ErasureHandler
}

// NewService creates a new GDPR service with the given repository.
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// RegisterExportHandler adds a per-module data export handler.
func (s *Service) RegisterExportHandler(handler DataExportHandler) {
	s.exportHandlers = append(s.exportHandlers, handler)
}

// RegisterErasureHandler adds a per-module data erasure handler.
func (s *Service) RegisterErasureHandler(handler ErasureHandler) {
	s.erasureHandlers = append(s.erasureHandlers, handler)
}

// ---------------------------------------------------------------------------
// Export workflow methods
// ---------------------------------------------------------------------------

// RequestExport creates a new data export request for the given user.
// Returns an error if the user already has a pending request.
func (s *Service) RequestExport(ctx context.Context, userID uuid.UUID) (*models.GDPRExportRequest, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("gdpr: tenant id missing from context: %w", err)
	}

	// Check for existing pending request
	existing, err := s.repo.ListExportRequests(ctx, tenantID, userID, models.ExportStatusPending)
	if err != nil {
		return nil, fmt.Errorf("gdpr: failed to check existing requests: %w", err)
	}
	if len(existing) > 0 {
		return nil, ErrExportAlreadyPending
	}

	req := &models.GDPRExportRequest{
		ID:          uuid.New(),
		UserID:      userID,
		Status:      models.ExportStatusPending,
		RequestedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateExportRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("gdpr: failed to create export request: %w", err)
	}

	slog.Info("gdpr: data export requested",
		"request_id", req.ID,
		"user_id", userID,
	)
	return req, nil
}

// ListExports returns export requests filtered by user and/or status.
// Pass uuid.Nil for userID to return all requests (admin view).
// Pass empty string for status to return all statuses.
func (s *Service) ListExports(ctx context.Context, userID uuid.UUID, status string) ([]*models.GDPRExportRequest, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("gdpr: tenant id missing from context: %w", err)
	}
	return s.repo.ListExportRequests(ctx, tenantID, userID, status)
}

// ApproveExport marks an export request as approved and triggers async export generation.
// The actual ZIP generation runs asynchronously via ExecuteExportAsync.
func (s *Service) ApproveExport(ctx context.Context, exportID uuid.UUID, reviewerID uuid.UUID, note string) (*models.GDPRExportRequest, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("gdpr: tenant id missing from context: %w", err)
	}

	req, err := s.repo.GetExportRequest(ctx, tenantID, exportID)
	if err != nil {
		return nil, err
	}

	if req.Status != models.ExportStatusPending {
		return nil, ErrInvalidExportStatus
	}

	now := time.Now().UTC()
	req.Status = models.ExportStatusApproved
	req.ReviewedBy = &reviewerID
	req.ReviewedAt = &now
	req.ReviewNote = note

	if err := s.repo.UpdateExportStatus(ctx, tenantID, req); err != nil {
		return nil, fmt.Errorf("gdpr: failed to approve export: %w", err)
	}

	slog.Info("gdpr: export approved",
		"request_id", exportID,
		"reviewer_id", reviewerID,
	)

	// Trigger async export generation. The tenant must be carried over to the
	// detached context: the RLS pool hook stamps app.tenant_id from the context,
	// so a bare Background() would hide every row from the export handlers.
	bgCtx := context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
	go func() {
		if execErr := s.ExecuteExportAsync(bgCtx, exportID); execErr != nil {
			slog.Error("gdpr: async export generation failed",
				"request_id", exportID,
				"error", execErr,
			)
		}
	}()

	return req, nil
}

// DenyExport marks an export request as denied with a review note.
func (s *Service) DenyExport(ctx context.Context, exportID uuid.UUID, reviewerID uuid.UUID, note string) (*models.GDPRExportRequest, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("gdpr: tenant id missing from context: %w", err)
	}

	req, err := s.repo.GetExportRequest(ctx, tenantID, exportID)
	if err != nil {
		return nil, err
	}

	if req.Status != models.ExportStatusPending {
		return nil, ErrInvalidExportStatus
	}

	now := time.Now().UTC()
	req.Status = models.ExportStatusDenied
	req.ReviewedBy = &reviewerID
	req.ReviewedAt = &now
	req.ReviewNote = note

	if err := s.repo.UpdateExportStatus(ctx, tenantID, req); err != nil {
		return nil, fmt.Errorf("gdpr: failed to deny export: %w", err)
	}

	slog.Info("gdpr: export denied",
		"request_id", exportID,
		"reviewer_id", reviewerID,
	)
	return req, nil
}

// ExecuteExportAsync generates the ZIP export file and stores the result.
// This is called asynchronously after admin approval.
func (s *Service) ExecuteExportAsync(ctx context.Context, exportID uuid.UUID) error {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return fmt.Errorf("gdpr: tenant id missing from context: %w", err)
	}

	req, err := s.repo.GetExportRequest(ctx, tenantID, exportID)
	if err != nil {
		return err
	}

	if req.Status != models.ExportStatusApproved {
		return ErrInvalidExportStatus
	}

	// Update status to processing
	req.Status = models.ExportStatusProcessing
	if err := s.repo.UpdateExportStatus(ctx, tenantID, req); err != nil {
		return fmt.Errorf("gdpr: failed to set processing status: %w", err)
	}

	// Execute the export across all registered handlers
	zipData, err := ExecuteExport(ctx, req.UserID, s.exportHandlers)
	if err != nil {
		// Revert to approved status on failure so retry is possible
		req.Status = models.ExportStatusApproved
		_ = s.repo.UpdateExportStatus(ctx, tenantID, req)
		return fmt.Errorf("gdpr: export generation failed: %w", err)
	}

	// Generate secure download token
	token, err := generateDownloadToken()
	if err != nil {
		return fmt.Errorf("gdpr: failed to generate download token: %w", err)
	}

	// Download link expires in 7 days
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)

	if err := s.repo.StoreExportResult(ctx, tenantID, exportID, zipData, token, expiresAt); err != nil {
		return fmt.Errorf("gdpr: failed to store export result: %w", err)
	}

	slog.Info("gdpr: export ready for download",
		"request_id", exportID,
		"user_id", req.UserID,
		"size_bytes", len(zipData),
	)
	return nil
}

// GetExportDownload retrieves the export data by download token.
// Marks the export as downloaded after successful retrieval.
func (s *Service) GetExportDownload(ctx context.Context, token string) ([]byte, string, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("gdpr: tenant id missing from context: %w", err)
	}

	req, err := s.repo.GetExportByToken(ctx, tenantID, token)
	if err != nil {
		return nil, "", err
	}

	if req.Status != models.ExportStatusReady && req.Status != models.ExportStatusDownloaded {
		return nil, "", ErrExportNotReady
	}

	// Check expiration
	if req.DownloadExpiresAt != nil && time.Now().UTC().After(*req.DownloadExpiresAt) {
		return nil, "", ErrExportExpired
	}

	// Mark as downloaded
	if req.Status == models.ExportStatusReady {
		if markErr := s.repo.MarkDownloaded(ctx, tenantID, req.ID); markErr != nil {
			slog.Error("gdpr: failed to mark export as downloaded",
				"request_id", req.ID,
				"error", markErr,
			)
			// Non-fatal: user still gets the data
		}
	}

	filename := fmt.Sprintf("dsgvo-export-%s.zip", req.UserID.String()[:8])
	return req.ExportData, filename, nil
}

// ---------------------------------------------------------------------------
// Erasure workflow methods
// ---------------------------------------------------------------------------

// PreviewErasure returns a preview of what data will be affected across all modules.
func (s *Service) PreviewErasure(ctx context.Context, userID uuid.UUID) ([]*models.ModuleErasurePreview, int, error) {
	var previews []*models.ModuleErasurePreview
	totalRecords := 0

	for _, handler := range s.erasureHandlers {
		preview, err := handler.PreviewErasure(ctx, userID)
		if err != nil {
			slog.Error("gdpr: erasure preview failed for module",
				"module", handler.ModuleName(),
				"user_id", userID,
				"error", err,
			)
			// Include error entry but continue with other modules
			previews = append(previews, &models.ModuleErasurePreview{
				ModuleName:  handler.ModuleName(),
				RecordCount: -1, // Indicates error
				Action:      "error",
			})
			continue
		}

		previews = append(previews, preview)
		if preview.RecordCount > 0 {
			totalRecords += preview.RecordCount
		}
	}

	return previews, totalRecords, nil
}

// ExecuteErasure performs cascading data erasure across all registered modules.
// Generates an anonymized label, executes erasure on each module (continuing on individual failures),
// computes a SHA-256 confirmation hash, and logs the operation.
func (s *Service) ExecuteErasure(ctx context.Context, userID uuid.UUID, executedBy uuid.UUID) (*models.GDPRErasureLog, error) {
	// Get the next anonymized label
	anonymizedLabel, err := s.repo.GetNextAnonymizedLabel(ctx)
	if err != nil {
		return nil, fmt.Errorf("gdpr: failed to get anonymized label: %w", err)
	}

	slog.Info("gdpr: starting erasure execution",
		"user_id", userID,
		"executed_by", executedBy,
		"anonymized_label", anonymizedLabel,
	)

	// Execute erasure across all modules, collecting results
	modulesAffected := make(map[string]string)
	var moduleErrors []string

	for _, handler := range s.erasureHandlers {
		moduleName := handler.ModuleName()

		// Determine action based on handler's preview
		preview, previewErr := handler.PreviewErasure(ctx, userID)
		action := ErasureAnonymize
		if previewErr == nil && preview != nil {
			action = ErasureAction(preview.Action)
		}

		recordsAffected, execErr := handler.ExecuteErasure(ctx, userID, anonymizedLabel, action)
		if execErr != nil {
			slog.Error("gdpr: erasure failed for module",
				"module", moduleName,
				"user_id", userID,
				"error", execErr,
			)
			modulesAffected[moduleName] = fmt.Sprintf("error: %s", execErr.Error())
			moduleErrors = append(moduleErrors, fmt.Sprintf("%s: %s", moduleName, execErr.Error()))
			// Continue with next module -- partial erasure is better than no erasure
			continue
		}

		modulesAffected[moduleName] = fmt.Sprintf("%s: %d records", action, recordsAffected)

		slog.Info("gdpr: module erasure complete",
			"module", moduleName,
			"user_id", userID,
			"action", action,
			"records_affected", recordsAffected,
		)
	}

	// Generate SHA-256 confirmation hash from erasure details
	confirmationHash := computeConfirmationHash(userID, executedBy, anonymizedLabel, modulesAffected)

	// Create erasure log entry
	entry := &models.GDPRErasureLog{
		ID:               uuid.New(),
		OriginalUserID:   userID,
		AnonymizedLabel:  anonymizedLabel,
		ExecutedBy:       executedBy,
		ExecutedAt:       time.Now().UTC(),
		ModulesAffected:  modulesAffected,
		ConfirmationHash: confirmationHash,
	}

	if err := s.repo.CreateErasureLog(ctx, entry); err != nil {
		return nil, fmt.Errorf("gdpr: failed to create erasure log: %w", err)
	}

	if len(moduleErrors) > 0 {
		slog.Warn("gdpr: erasure completed with errors",
			"user_id", userID,
			"errors", strings.Join(moduleErrors, "; "),
			"confirmation_hash", confirmationHash,
		)
	} else {
		slog.Info("gdpr: erasure completed successfully",
			"user_id", userID,
			"anonymized_label", anonymizedLabel,
			"confirmation_hash", confirmationHash,
		)
	}

	return entry, nil
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// generateDownloadToken creates a cryptographically random 32-byte hex token.
func generateDownloadToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("gdpr: failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// computeConfirmationHash generates a SHA-256 hash of the erasure operation details.
// This provides a tamper-evident confirmation receipt.
func computeConfirmationHash(userID, executedBy uuid.UUID, label string, modules map[string]string) string {
	h := sha256.New()
	h.Write([]byte(userID.String()))
	h.Write([]byte(executedBy.String()))
	h.Write([]byte(label))
	h.Write([]byte(time.Now().UTC().Format(time.RFC3339)))

	for k, v := range modules {
		h.Write([]byte(k))
		h.Write([]byte(v))
	}

	return hex.EncodeToString(h.Sum(nil))
}
