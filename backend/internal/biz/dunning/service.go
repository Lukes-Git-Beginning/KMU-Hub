// Package dunning provides the business logic for managing dunning records (Mahnungen).
// Supports 3-level semi-automatic dunning with configurable intervals and fees.
// Level 1: Zahlungserinnerung (friendly), Level 2: 1. Mahnung (formal),
// Level 3: 2. Mahnung / Letzte Mahnung (urgent, threatens Inkasso).
// Verzugszinsen are calculated per BGB section 288.
package dunning

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/pdf"
	"github.com/kmuhub/kmuhub/internal/models"
)

// MaxDunningLevel is the highest dunning level supported.
const MaxDunningLevel = 3

// Service handles dunning business logic.
type Service struct {
	repo       Repository
	configRepo ConfigRepository
	invReader  InvoiceReader

	// noticeMailer and companySettingsRepo are optional (wired via
	// SetNoticeMailer). When unset, Send/SendDunningNotice fall back to the
	// pre-email behavior: status flips to sent, no PDF/email attempted.
	noticeMailer        NoticeSender
	companySettingsRepo CompanySettingsRepo
}

// NewService creates a new dunning service.
func NewService(
	repo Repository,
	configRepo ConfigRepository,
	invReader InvoiceReader,
) *Service {
	return &Service{
		repo:       repo,
		configRepo: configRepo,
		invReader:  invReader,
	}
}

// CompanySettingsRepo provides access to per-tenant company settings needed to
// render the dunning PDF attachment. Signature matches invoice.CompanySettingsRepo
// so the same Postgres repo (quote.NewPostgresCompanySettingsRepo, already wired
// in cmd/biz/main.go) satisfies both without a shared package dependency.
type CompanySettingsRepo interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.CompanySettings, error)
}

// NoticeSender delivers the dunning notice email with the generated PDF
// attached. The concrete implementation (cmd/biz/mailer.go) wraps a dedicated
// system SMTP account.
type NoticeSender interface {
	SendDunningNotice(ctx context.Context, toEmail, toName string, pdfBytes []byte, filename, subject string, level int) error
}

// SetNoticeMailer wires the email + PDF dependencies for Send/SendDunningNotice
// (called from cmd/biz/main.go). Optional: if never called, sendAndNotify keeps
// the pre-email status-only behavior — safe for dev/CI and existing tests.
func (s *Service) SetNoticeMailer(mailer NoticeSender, settingsRepo CompanySettingsRepo) {
	s.noticeMailer = mailer
	s.companySettingsRepo = settingsRepo
}

// GetConfig returns the dunning configuration for a tenant.
// If no config exists, creates one with default values (14/14/14 days, 0/5/10 EUR fees).
func (s *Service) GetConfig(ctx context.Context, tenantID uuid.UUID) (*models.DunningConfig, error) {
	config, err := s.configRepo.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if config != nil {
		return config, nil
	}

	// Create default config
	config = &models.DunningConfig{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		Level1DaysAfterDue:    14,
		Level2DaysAfterLevel1: 14,
		Level3DaysAfterLevel2: 14,
		Level1Fee:             decimal.Zero,
		Level2Fee:             decimal.NewFromInt(5),
		Level3Fee:             decimal.NewFromInt(10),
		UpdatedAt:             time.Now(),
	}

	if upsertErr := s.configRepo.Upsert(ctx, config); upsertErr != nil {
		return nil, fmt.Errorf("create default dunning config: %w", upsertErr)
	}

	slog.Info("default dunning config created",
		"tenant_id", tenantID,
	)

	return config, nil
}

// UpdateConfigInput contains the data for updating dunning configuration.
type UpdateConfigInput struct {
	Level1DaysAfterDue    *int
	Level2DaysAfterLevel1 *int
	Level3DaysAfterLevel2 *int
	Level1Fee             *decimal.Decimal
	Level2Fee             *decimal.Decimal
	Level3Fee             *decimal.Decimal
}

// UpdateConfig updates the dunning configuration for a tenant.
func (s *Service) UpdateConfig(ctx context.Context, tenantID uuid.UUID, input UpdateConfigInput) (*models.DunningConfig, error) {
	config, err := s.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if input.Level1DaysAfterDue != nil {
		config.Level1DaysAfterDue = *input.Level1DaysAfterDue
	}
	if input.Level2DaysAfterLevel1 != nil {
		config.Level2DaysAfterLevel1 = *input.Level2DaysAfterLevel1
	}
	if input.Level3DaysAfterLevel2 != nil {
		config.Level3DaysAfterLevel2 = *input.Level3DaysAfterLevel2
	}
	if input.Level1Fee != nil {
		config.Level1Fee = *input.Level1Fee
	}
	if input.Level2Fee != nil {
		config.Level2Fee = *input.Level2Fee
	}
	if input.Level3Fee != nil {
		config.Level3Fee = *input.Level3Fee
	}
	config.UpdatedAt = time.Now()

	if upsertErr := s.configRepo.Upsert(ctx, config); upsertErr != nil {
		return nil, upsertErr
	}

	slog.Info("dunning config updated",
		"tenant_id", tenantID,
	)

	return config, nil
}

// DetectAndCreateDunnings detects overdue invoices and creates appropriate dunning drafts.
// This is semi-automatic: it creates draft dunnings that the user must review and send manually.
//
// Logic:
//   - Invoice overdue + no dunning yet + days_overdue >= level1_days -> create Level 1 draft
//   - Invoice has Level 1 sent + days_since_level1_sent >= level2_days -> create Level 2 draft
//   - Invoice has Level 2 sent + days_since_level2_sent >= level3_days -> create Level 3 draft
//   - Never creates if max level (3) already sent
func (s *Service) DetectAndCreateDunnings(ctx context.Context, tenantID uuid.UUID) ([]*models.DunningRecord, error) {
	config, err := s.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get dunning config: %w", err)
	}

	overdueInvoices, err := s.invReader.GetOverdue(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get overdue invoices: %w", err)
	}

	// Pre-fetch existing dunning records for all overdue invoices in one query to
	// avoid an N+1 (previously one GetByInvoiceID per overdue invoice).
	invoiceIDs := make([]uuid.UUID, len(overdueInvoices))
	for i, inv := range overdueInvoices {
		invoiceIDs[i] = inv.ID
	}
	dunningByInvoice, err := s.repo.GetByInvoiceIDs(ctx, tenantID, invoiceIDs)
	if err != nil {
		return nil, fmt.Errorf("get dunning records for overdue invoices: %w", err)
	}

	var created []*models.DunningRecord
	now := time.Now()

	for _, inv := range overdueInvoices {
		// Existing dunning records for this invoice (from the batch pre-fetch).
		existing := dunningByInvoice[inv.ID]

		// Determine the highest sent dunning level
		highestSentLevel := 0
		var lastSentAt *time.Time
		hasDraft := false

		for _, d := range existing {
			if d.Status == models.DunningStatusSent && d.Level > highestSentLevel {
				highestSentLevel = d.Level
				lastSentAt = d.SentAt
			}
			if d.Status == models.DunningStatusDraft {
				hasDraft = true
			}
		}

		// Skip if already at max level or there's already a draft pending
		if highestSentLevel >= MaxDunningLevel || hasDraft {
			continue
		}

		// Determine if we should create the next level
		nextLevel := highestSentLevel + 1
		shouldCreate := false
		var fee decimal.Decimal

		switch nextLevel {
		case 1:
			daysOverdue := daysBetween(inv.DueDate, now)
			if daysOverdue >= config.Level1DaysAfterDue {
				shouldCreate = true
				fee = config.Level1Fee
			}
		case 2:
			if lastSentAt != nil {
				daysSinceLast := daysBetween(*lastSentAt, now)
				if daysSinceLast >= config.Level2DaysAfterLevel1 {
					shouldCreate = true
					fee = config.Level2Fee
				}
			}
		case 3:
			if lastSentAt != nil {
				daysSinceLast := daysBetween(*lastSentAt, now)
				if daysSinceLast >= config.Level3DaysAfterLevel2 {
					shouldCreate = true
					fee = config.Level3Fee
				}
			}
		}

		if !shouldCreate {
			continue
		}

		record := &models.DunningRecord{
			ID:        uuid.New(),
			TenantID:  tenantID,
			InvoiceID: inv.ID,
			Level:     nextLevel,
			Status:    models.DunningStatusDraft,
			Fee:       fee,
			// lean: Verzugszinsen are never computed — CalculateInterest below has zero
			// callers (verified 2026-08-23). Not "calculated elsewhere"; simply not wired.
			// Upgrade trigger: wire once (a) a B2B/B2C flag exists on invoice/customer and
			// (b) CompanySettings.Basiszinssatz is read here instead of left at zero.
			Interest:  decimal.Zero,
			CreatedBy: uuid.Nil, // System-generated
			CreatedAt: now,
		}

		if createErr := s.repo.Create(ctx, record); createErr != nil {
			slog.Warn("failed to create dunning record",
				"invoice_id", inv.ID,
				"level", nextLevel,
				"error", createErr,
			)
			continue
		}

		created = append(created, record)

		slog.Info("dunning record created",
			"dunning_id", record.ID,
			"invoice_id", inv.ID,
			"level", nextLevel,
			"fee", fee,
		)
	}

	return created, nil
}

// Send transitions a draft dunning record to sent, emailing the PDF notice if
// a NoticeSender is configured (see SetNoticeMailer), and sets the sent_at
// timestamp. This is the entrypoint the "Mahnung senden" UI button calls.
func (s *Service) Send(ctx context.Context, tenantID, id, userID uuid.UUID) error {
	_, err := s.sendAndNotify(ctx, tenantID, id, userID)
	return err
}

// sendAndNotify is the shared implementation behind Send and SendDunningNotice
// (Fix #2 / Sprint 3 email integration) — both entrypoints must go through the
// same status-flip + email path, otherwise only one of them would actually
// deliver the dunning PDF.
//
// Fail-open vs fail-closed:
//   - noticeMailer == nil (SetNoticeMailer never called — dev/CI/tests without
//     mail wiring): behaves exactly like the pre-email code — status flips to
//     sent, no email attempted, only logged.
//   - noticeMailer configured but the send itself fails: fail-closed — the
//     error is returned and the record stays in draft. GoBD requires "sent" to
//     mean the notice was actually delivered, so flipping status on a failed
//     send would misrepresent the audit trail.
//   - noticeMailer configured but the tenant has no company settings
//     (ErrCompanySettingsMissing): fail-open like the nil-mailer branch. That is
//     a missing-master-data gap, not a delivery attempt that failed, so the GoBD
//     argument does not apply — blocking the whole dunning run on it would only
//     leave the record stuck in draft behind a 500.
func (s *Service) sendAndNotify(ctx context.Context, tenantID, id, userID uuid.UUID) (*models.DunningRecord, error) {
	record, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if record.Status != models.DunningStatusDraft {
		return nil, ErrDunningNotDraft
	}

	if s.noticeMailer != nil {
		if sendErr := s.emailNotice(ctx, tenantID, record); sendErr != nil {
			if !errors.Is(sendErr, ErrCompanySettingsMissing) {
				return nil, fmt.Errorf("send dunning notice: %w", sendErr)
			}
			slog.Error("dunning notice email skipped, company settings missing; flipping status anyway",
				"dunning_id", id,
				"tenant_id", tenantID,
				"error", sendErr,
			)
		}
	} else {
		slog.Warn("notice mailer not configured, dunning notice email not sent",
			"dunning_id", id,
			"tenant_id", tenantID,
		)
	}

	now := time.Now()
	if updateErr := s.repo.UpdateStatus(ctx, tenantID, id, models.DunningStatusSent, &now); updateErr != nil {
		return nil, fmt.Errorf("mark dunning sent: %w", updateErr)
	}

	record.Status = models.DunningStatusSent
	record.SentAt = &now

	slog.Info("dunning notice sent",
		"dunning_id", id,
		"invoice_id", record.InvoiceID,
		"tenant_id", tenantID,
		"level", record.Level,
		"sent_by", userID,
	)

	return record, nil
}

// dunningDocPrefix maps a dunning level to its German document/filename prefix,
// matching the convention already used for manual PDF downloads
// (BizGRPCServer.GenerateDunningPDF).
func dunningDocPrefix(level int) string {
	switch level {
	case 1:
		return "Zahlungserinnerung"
	case 2:
		return "1_Mahnung"
	default:
		return "2_Mahnung"
	}
}

// emailNotice loads the linked invoice + company settings, renders the dunning
// PDF, and dispatches it via the configured NoticeSender. Failures are
// fail-closed by the caller (sendAndNotify) — draft status is preserved — with
// the single exception of ErrCompanySettingsMissing, which the caller treats as
// non-fatal.
func (s *Service) emailNotice(ctx context.Context, tenantID uuid.UUID, record *models.DunningRecord) error {
	invoice, err := s.invReader.GetByID(ctx, tenantID, record.InvoiceID)
	if err != nil {
		return fmt.Errorf("load linked invoice: %w", err)
	}

	settings, err := s.companySettingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("load company settings: %w", err)
	}
	if settings == nil {
		return fmt.Errorf("%w %s", ErrCompanySettingsMissing, tenantID)
	}

	pdfBytes, err := pdf.NewGenerator(*settings).GenerateDunningPDF(*record, *invoice, record.Level)
	if err != nil {
		return fmt.Errorf("generate dunning pdf: %w", err)
	}

	docPrefix := dunningDocPrefix(record.Level)
	filename := fmt.Sprintf("%s_%s.pdf", docPrefix, invoice.InvoiceNumber)
	subject := fmt.Sprintf("%s zu Rechnung %s", strings.ReplaceAll(docPrefix, "_", ". "), invoice.InvoiceNumber)

	if err := s.noticeMailer.SendDunningNotice(ctx, invoice.CustomerEmail, invoice.CustomerName, pdfBytes, filename, subject, record.Level); err != nil {
		return fmt.Errorf("deliver email: %w", err)
	}
	return nil
}

// GetByID retrieves a single dunning record by its ID.
func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.DunningRecord, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List retrieves dunning records with optional filtering.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.DunningRecord, int, error) {
	return s.repo.List(ctx, tenantID, filter)
}

// CalculateInterest calculates Verzugszinsen per BGB section 288.
// B2C: Basiszinssatz + 5 percentage points
// B2B: Basiszinssatz + 9 percentage points
// Pro-rata daily calculation from due_date to today.
//
// lean: has zero production callers (verified 2026-08-23, grepped internal/, cmd/ and
// proto/biz/v1/biz.proto). The basiszinssatz parameter itself CAN be populated today —
// CompanySettings.Basiszinssatz is a persisted, tenant-configurable field wired end to
// end (route_biz.go HandleUpdateCompanySettings -> biz_grpc.go:229 -> models.CompanySettings)
// — but nothing reads it into a dunning record. Upgrade trigger: wire this once a B2B/B2C
// distinction exists on invoice/customer (models.Invoice has none today).
// Two fachliche Mängel to fix at the same time as the wiring, not before:
//   - the divisor below is a fixed 365; a full leap year overstates the daily rate slightly.
//   - Basiszinssatz changes twice a year (section 247 BGB); a period spanning a change
//     needs to be split and each half rated with the rate that applied during it, not one
//     rate applied to the whole span.
func (s *Service) CalculateInterest(
	_ context.Context,
	invoiceGrossTotal decimal.Decimal,
	dueDate time.Time,
	basiszinssatz decimal.Decimal,
	isB2B bool,
) decimal.Decimal {
	now := time.Now()
	if now.Before(dueDate) {
		return decimal.Zero
	}

	// Calculate days overdue
	days := daysBetween(dueDate, now)
	if days <= 0 {
		return decimal.Zero
	}

	// Interest rate per BGB section 288
	var addPoints decimal.Decimal
	if isB2B {
		addPoints = decimal.NewFromInt(9)
	} else {
		addPoints = decimal.NewFromInt(5)
	}
	annualRate := basiszinssatz.Add(addPoints)

	// Pro-rata daily: (principal * rate/100 * days) / 365
	hundred := decimal.NewFromInt(100)
	daysInYear := decimal.NewFromInt(365)
	daysDecimal := decimal.NewFromInt(int64(days))

	interest := invoiceGrossTotal.
		Mul(annualRate).
		Div(hundred).
		Mul(daysDecimal).
		Div(daysInYear).
		Round(2)

	return interest
}

// daysBetween returns the number of full days between two times.
func daysBetween(from, to time.Time) int {
	duration := to.Sub(from)
	return int(math.Floor(duration.Hours() / 24))
}
