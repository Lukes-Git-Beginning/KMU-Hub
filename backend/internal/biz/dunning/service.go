// Package dunning provides the business logic for managing dunning records (Mahnungen).
// Supports 3-level semi-automatic dunning with configurable intervals and fees.
// Level 1: Zahlungserinnerung (friendly), Level 2: 1. Mahnung (formal),
// Level 3: 2. Mahnung / Letzte Mahnung (urgent, threatens Inkasso).
// Verzugszinsen are calculated per BGB section 288.
package dunning

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MaxDunningLevel is the highest dunning level supported.
const MaxDunningLevel = 3

// Service handles dunning business logic.
type Service struct {
	repo       Repository
	configRepo ConfigRepository
	invReader  InvoiceReader
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

	var created []*models.DunningRecord
	now := time.Now()

	for _, inv := range overdueInvoices {
		// Get existing dunning records for this invoice
		existing, existErr := s.repo.GetByInvoiceID(ctx, tenantID, inv.ID)
		if existErr != nil {
			slog.Warn("failed to get dunning records for invoice",
				"invoice_id", inv.ID,
				"error", existErr,
			)
			continue
		}

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
			Interest:  decimal.Zero, // Interest is calculated separately at send time or display time
			CreatedBy: uuid.Nil,     // System-generated
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

// Send transitions a draft dunning record to sent and sets the sent_at timestamp.
func (s *Service) Send(ctx context.Context, tenantID, id, userID uuid.UUID) error {
	record, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if record.Status != models.DunningStatusDraft {
		return ErrDunningNotDraft
	}

	now := time.Now()
	if updateErr := s.repo.UpdateStatus(ctx, tenantID, id, models.DunningStatusSent, &now); updateErr != nil {
		return updateErr
	}

	slog.Info("dunning record sent",
		"dunning_id", id,
		"invoice_id", record.InvoiceID,
		"tenant_id", tenantID,
		"level", record.Level,
	)
	return nil
}

// List retrieves dunning records with optional filtering.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.DunningRecord, int, error) {
	return s.repo.List(ctx, tenantID, filter)
}

// CalculateInterest calculates Verzugszinsen per BGB section 288.
// B2C: Basiszinssatz + 5 percentage points
// B2B: Basiszinssatz + 9 percentage points
// Pro-rata daily calculation from due_date to today.
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
