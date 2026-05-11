package datev

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

type UploadService struct {
	exporter      *Exporter
	uploader      *Uploader
	belegUploader *BelegbilderUploader
	uploadRepo    UploadRepository
	configRepo    IntegrationConfigRepo
	oauthManager  *OAuthManager
}

func NewUploadService(
	exporter *Exporter,
	uploader *Uploader,
	belegUploader *BelegbilderUploader,
	uploadRepo UploadRepository,
	configRepo IntegrationConfigRepo,
	oauthManager *OAuthManager,
) *UploadService {
	return &UploadService{
		exporter:      exporter,
		uploader:      uploader,
		belegUploader: belegUploader,
		uploadRepo:    uploadRepo,
		configRepo:    configRepo,
		oauthManager:  oauthManager,
	}
}

func (s *UploadService) IsConnected() bool {
	return s.uploader != nil && s.oauthManager != nil
}

func (s *UploadService) GetConnectionStatus(ctx context.Context) (bool, error) {
	if !s.IsConnected() {
		return false, nil
	}
	config, err := s.configRepo.GetByPlatform(ctx, "datev_api")
	if err != nil {
		return false, nil
	}
	return config.IsActive, nil
}

func (s *UploadService) GetAuthorizationURL(tenantID uuid.UUID, redirectURL, authBaseURL string) string {
	if s.oauthManager == nil {
		return ""
	}
	return s.oauthManager.GetAuthorizationURL(tenantID, redirectURL, authBaseURL)
}

// HandleOAuthCallback exchanges the DATEV authorization code for tokens.
// Pre-JWT path (OAuth redirect from DATEV with state token, no user JWT):
// wrap in sysctx so the underlying ExchangeCode INSERTs into integration_configs
// and vault_secrets (RLS-enabled in mig 122) pass WITH CHECK.
func (s *UploadService) HandleOAuthCallback(ctx context.Context, tenantID uuid.UUID, code, redirectURL string) error {
	if s.oauthManager == nil {
		return fmt.Errorf("datev: OAuth not configured")
	}
	ctx = sysctx.With(ctx)
	return s.oauthManager.ExchangeCode(ctx, tenantID, code, redirectURL)
}

func (s *UploadService) Disconnect(ctx context.Context, tenantID uuid.UUID) error {
	if s.oauthManager != nil {
		if err := s.oauthManager.RevokeTokens(ctx, tenantID); err != nil {
			slog.Error("datev: failed to revoke tokens", "error", err)
		}
	}

	config, err := s.configRepo.GetByPlatform(ctx, "datev_api")
	if err != nil {
		return nil
	}
	return s.configRepo.Deactivate(ctx, config.ID)
}

// ExportAndUpload exports DATEV CSV and uploads via API. Falls back to export-only when no API credentials.
func (s *UploadService) ExportAndUpload(ctx context.Context, tenantID uuid.UUID, invoices []*models.Invoice, creditNotes []*models.CreditNote, fiscalYearStart time.Time) ([]byte, error) {
	csvData, err := s.exporter.Export(invoices, creditNotes, fiscalYearStart, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("datev export: %w", err)
	}

	if !s.IsConnected() {
		slog.Info("datev: API not connected, returning CSV only")
		return csvData, nil
	}

	config, err := s.configRepo.GetByPlatform(ctx, "datev_api")
	if err != nil {
		slog.Warn("datev: no API config found, returning CSV only", "error", err)
		return csvData, nil
	}

	if !config.IsActive {
		return csvData, nil
	}

	uploadConfig, err := s.uploadRepo.GetUploadConfig(ctx, config.ID)
	if err != nil {
		slog.Warn("datev: no upload config found, returning CSV only", "error", err)
		return csvData, nil
	}

	now := time.Now().UTC()
	uploadLog := &models.DatevUploadLog{
		ConfigID:      config.ID,
		UploadType:    "buchungsstapel",
		Status:        "uploading",
		FileSize:      len(csvData),
		DocumentCount: len(invoices) + len(creditNotes),
		StartedAt:     now,
		Metadata:      map[string]any{"fiscal_year_start": fiscalYearStart.Format("2006-01-02")},
	}
	if err := s.uploadRepo.CreateUploadLog(ctx, uploadLog); err != nil {
		slog.Error("datev: failed to create upload log", "error", err)
	}

	if err := s.uploader.UploadBuchungsstapel(ctx, tenantID, uploadConfig.ClientNumber, csvData); err != nil {
		uploadLog.Status = "failed"
		errMsg := err.Error()
		uploadLog.ErrorMessage = &errMsg
		uploadLog.CompletedAt = &now
		_ = s.uploadRepo.UpdateUploadLog(ctx, uploadLog)
		return csvData, fmt.Errorf("datev upload failed (CSV still available): %w", err)
	}

	completed := time.Now().UTC()
	uploadLog.Status = "completed"
	uploadLog.CompletedAt = &completed
	_ = s.uploadRepo.UpdateUploadLog(ctx, uploadLog)

	slog.Info("datev export and upload completed",
		"tenant_id", tenantID,
		"documents", uploadLog.DocumentCount,
		"size_bytes", uploadLog.FileSize,
	)

	return csvData, nil
}

// UploadBeleg uploads a PDF invoice as a DATEV Belegbild.
func (s *UploadService) UploadBeleg(ctx context.Context, tenantID uuid.UUID, pdfData []byte, filename string) error {
	if s.belegUploader == nil {
		return fmt.Errorf("datev: Belegbilder upload not available (API not connected)")
	}

	config, err := s.configRepo.GetByPlatform(ctx, "datev_api")
	if err != nil {
		return fmt.Errorf("datev: no API config: %w", err)
	}

	uploadConfig, err := s.uploadRepo.GetUploadConfig(ctx, config.ID)
	if err != nil {
		return fmt.Errorf("datev: no upload config: %w", err)
	}

	now := time.Now().UTC()
	uploadLog := &models.DatevUploadLog{
		ConfigID:      config.ID,
		UploadType:    "belegbild",
		Status:        "uploading",
		FileSize:      len(pdfData),
		DocumentCount: 1,
		StartedAt:     now,
		Metadata:      map[string]any{"filename": filename},
	}
	if err := s.uploadRepo.CreateUploadLog(ctx, uploadLog); err != nil {
		slog.Error("datev: failed to create upload log", "error", err)
	}

	if err := s.belegUploader.UploadBeleg(ctx, tenantID, uploadConfig.ClientNumber, pdfData, filename); err != nil {
		uploadLog.Status = "failed"
		errMsg := err.Error()
		uploadLog.ErrorMessage = &errMsg
		uploadLog.CompletedAt = &now
		_ = s.uploadRepo.UpdateUploadLog(ctx, uploadLog)
		return err
	}

	completed := time.Now().UTC()
	uploadLog.Status = "completed"
	uploadLog.CompletedAt = &completed
	_ = s.uploadRepo.UpdateUploadLog(ctx, uploadLog)

	return nil
}

func (s *UploadService) GetUploadConfig(ctx context.Context) (*models.DatevUploadConfig, error) {
	config, err := s.configRepo.GetByPlatform(ctx, "datev_api")
	if err != nil {
		return nil, err
	}
	return s.uploadRepo.GetUploadConfig(ctx, config.ID)
}

func (s *UploadService) UpdateUploadConfig(ctx context.Context, cfg *models.DatevUploadConfig) error {
	return s.uploadRepo.UpsertUploadConfig(ctx, cfg)
}

func (s *UploadService) ListUploadLogs(ctx context.Context, limit int) ([]models.DatevUploadLog, error) {
	config, err := s.configRepo.GetByPlatform(ctx, "datev_api")
	if err != nil {
		return nil, err
	}
	return s.uploadRepo.ListUploadLogs(ctx, config.ID, limit)
}
