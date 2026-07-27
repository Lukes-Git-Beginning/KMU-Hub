package datev

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

var (
	// ErrNotConnected is returned when no DATEV API credentials are wired, so
	// nothing can be transferred at all.
	ErrNotConnected = errors.New("datev: API not connected")

	// ErrNoAPIConfig is returned when this tenant has no active datev_api
	// integration config — the OAuth flow was never completed or was revoked.
	ErrNoAPIConfig = errors.New("datev: no active API configuration for this tenant")

	// ErrNoUploadConfig is returned when the tenant's upload configuration is
	// missing or carries no client number. DATEV files a batch under the client
	// number; without it the upload has no destination.
	ErrNoUploadConfig = errors.New("datev: upload configuration incomplete (client number missing)")

	// ErrAdvisorNumbersMissing is returned when Berater- or Mandantennummer are
	// unset on company settings. A batch without them cannot be assigned in
	// DATEV — the download path tolerates it because a human files that file,
	// the API upload does not.
	ErrAdvisorNumbersMissing = errors.New("datev: Beraternummer/Mandantennummer not configured in company settings")

	// ErrNothingToUpload is returned when the requested period contains no
	// bookable document. Uploading an empty batch and reporting success is the
	// exact failure this endpoint used to have.
	ErrNothingToUpload = errors.New("datev: no bookable documents in the requested period")
)

// BuchungsstapelUploader transfers a rendered CSV batch to DATEV.
// Implemented by *Uploader; an interface so the upload path is testable without
// a DATEV account.
type BuchungsstapelUploader interface {
	UploadBuchungsstapel(ctx context.Context, tenantID uuid.UUID, clientNumber string, csvData []byte) error
}

// BelegUploader transfers a single document image to DATEV.
// Implemented by *BelegbilderUploader.
type BelegUploader interface {
	UploadBeleg(ctx context.Context, tenantID uuid.UUID, clientNumber string, pdfData []byte, filename string) error
}

type UploadService struct {
	builder       *BuchungsstapelBuilder
	belegRenderer BelegSource
	uploader      BuchungsstapelUploader
	belegUploader BelegUploader
	uploadRepo    UploadRepository
	configRepo    IntegrationConfigRepo
	oauthManager  *OAuthManager
}

// NewUploadService wires the DATEV upload path. uploader and belegUploader are
// interfaces: pass a real value or an untyped nil, never a typed nil pointer —
// IsConnected reads them as the presence of API credentials.
func NewUploadService(
	builder *BuchungsstapelBuilder,
	belegRenderer BelegSource,
	uploader BuchungsstapelUploader,
	belegUploader BelegUploader,
	uploadRepo UploadRepository,
	configRepo IntegrationConfigRepo,
	oauthManager *OAuthManager,
) *UploadService {
	return &UploadService{
		builder:       builder,
		belegRenderer: belegRenderer,
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

// UploadBuchungsstapel renders the tenant's booking batch for a period and
// transfers it to DATEV.
//
// Every precondition is checked before anything is rendered, and each failure is
// an error rather than a batch that goes out half-configured: the predecessor of
// this method exported whatever it was handed — including nothing at all — and
// wrote a "completed" upload log for it, so the client was told the accounting
// data had reached the tax advisor when an empty CSV had. There is no
// export-only fallback here either; a caller who wants the file without sending
// it uses the GoBD export.
func (s *UploadService) UploadBuchungsstapel(
	ctx context.Context,
	tenantID uuid.UUID,
	startDate, endDate, fiscalYearStart time.Time,
) (*Buchungsstapel, error) {
	if s.builder == nil {
		return nil, ErrBuilderNotConfigured
	}
	if !s.IsConnected() {
		return nil, ErrNotConnected
	}

	config, err := s.configRepo.GetByPlatform(ctx, "datev_api")
	if err != nil || config == nil || !config.IsActive {
		return nil, ErrNoAPIConfig
	}

	uploadConfig, err := s.uploadRepo.GetUploadConfig(ctx, config.ID)
	if err != nil || uploadConfig == nil || uploadConfig.ClientNumber == "" {
		return nil, ErrNoUploadConfig
	}

	batch, err := s.builder.Build(ctx, tenantID, startDate, endDate, fiscalYearStart)
	if err != nil {
		return nil, err
	}
	if batch.DocumentCount == 0 {
		return nil, ErrNothingToUpload
	}
	// The header numbers decide where DATEV files the batch. Empty ones produce
	// an import the tax advisor cannot assign, which looks like a successful
	// transfer on both ends and is only noticed at the bookkeeping.
	if batch.BeraterNr == "" || batch.MandantNr == "" {
		return nil, ErrAdvisorNumbersMissing
	}

	now := time.Now().UTC()
	uploadLog := &models.DatevUploadLog{
		ConfigID:      config.ID,
		UploadType:    "buchungsstapel",
		Status:        "uploading",
		FileSize:      len(batch.CSV),
		DocumentCount: batch.DocumentCount,
		StartedAt:     now,
		Metadata: map[string]any{
			"fiscal_year_start": fiscalYearStart.Format("2006-01-02"),
			"start_date":        startDate.Format("2006-01-02"),
			"end_date":          endDate.Format("2006-01-02"),
			"line_count":        batch.LineCount,
		},
	}
	if err := s.uploadRepo.CreateUploadLog(ctx, uploadLog); err != nil {
		slog.Error("datev: failed to create upload log", "error", err)
	}

	if err := s.uploader.UploadBuchungsstapel(ctx, tenantID, uploadConfig.ClientNumber, batch.CSV); err != nil {
		s.failUploadLog(ctx, uploadLog, err)
		return nil, fmt.Errorf("datev upload failed: %w", err)
	}

	s.completeUploadLog(ctx, uploadLog)

	slog.Info("datev buchungsstapel uploaded",
		"tenant_id", tenantID,
		"documents", batch.DocumentCount,
		"lines", batch.LineCount,
		"size_bytes", len(batch.CSV),
	)

	return batch, nil
}

// UploadInvoiceBeleg renders an invoice PDF and transfers it as a Belegbild.
// The rendering happens here rather than in the caller so no path can report a
// transferred document image without one having been produced.
func (s *UploadService) UploadInvoiceBeleg(ctx context.Context, tenantID, invoiceID uuid.UUID) error {
	if s.belegRenderer == nil {
		return ErrBuilderNotConfigured
	}
	if s.belegUploader == nil {
		return ErrNotConnected
	}

	pdfData, filename, err := s.belegRenderer.RenderInvoice(ctx, tenantID, invoiceID)
	if err != nil {
		return err
	}
	if len(pdfData) == 0 {
		return fmt.Errorf("datev: rendered Belegbild is empty")
	}

	return s.UploadBeleg(ctx, tenantID, pdfData, filename)
}

// failUploadLog marks an upload log entry as failed with the transfer error.
func (s *UploadService) failUploadLog(ctx context.Context, uploadLog *models.DatevUploadLog, cause error) {
	failed := time.Now().UTC()
	msg := cause.Error()
	uploadLog.Status = "failed"
	uploadLog.ErrorMessage = &msg
	uploadLog.CompletedAt = &failed
	if err := s.uploadRepo.UpdateUploadLog(ctx, uploadLog); err != nil {
		slog.Error("datev: failed to update upload log", "error", err)
	}
}

// completeUploadLog marks an upload log entry as completed.
func (s *UploadService) completeUploadLog(ctx context.Context, uploadLog *models.DatevUploadLog) {
	completed := time.Now().UTC()
	uploadLog.Status = "completed"
	uploadLog.CompletedAt = &completed
	if err := s.uploadRepo.UpdateUploadLog(ctx, uploadLog); err != nil {
		slog.Error("datev: failed to update upload log", "error", err)
	}
}

// UploadBeleg uploads a PDF invoice as a DATEV Belegbild.
func (s *UploadService) UploadBeleg(ctx context.Context, tenantID uuid.UUID, pdfData []byte, filename string) error {
	if s.belegUploader == nil {
		return ErrNotConnected
	}

	config, err := s.configRepo.GetByPlatform(ctx, "datev_api")
	if err != nil || config == nil || !config.IsActive {
		return ErrNoAPIConfig
	}

	uploadConfig, err := s.uploadRepo.GetUploadConfig(ctx, config.ID)
	if err != nil || uploadConfig == nil || uploadConfig.ClientNumber == "" {
		return ErrNoUploadConfig
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
		s.failUploadLog(ctx, uploadLog, err)
		return err
	}

	s.completeUploadLog(ctx, uploadLog)

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
