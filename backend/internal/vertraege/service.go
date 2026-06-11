package vertraege

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service handles contract business logic.
type Service struct {
	repo Repository
}

// NewService creates a new vertraege Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ============================================================================
// Input types — Contracts
// ============================================================================

// CreateContractInput holds the data required to create a new contract.
type CreateContractInput struct {
	TenantID       uuid.UUID
	ContractNumber string
	Title          string
	ContractType   ContractType
	StartsOn       time.Time
	EndsOn         *time.Time
	DocumentURL    *string
	Notes          string
	CreatedBy      *uuid.UUID
}

// UpdateContractInput holds the fields that may be updated on a contract.
type UpdateContractInput struct {
	TenantID     uuid.UUID
	ContractID   uuid.UUID
	Title        *string
	ContractType *ContractType
	Status       *ContractStatus
	StartsOn     *time.Time
	EndsOn       *time.Time // set to zero value pointer to clear
	ClearEndsOn  bool       // explicitly set ends_on to NULL
	DocumentURL  *string
	Notes        *string
}

// ListContractsInput holds pagination and filter options for listing contracts.
type ListContractsInput struct {
	TenantID     uuid.UUID
	Status       *ContractStatus
	Type         *ContractType
	StartsAfter  *time.Time
	StartsBefore *time.Time
	EndsAfter    *time.Time
	EndsBefore   *time.Time
	Page         int
	PageSize     int
	// ContactID filters to contracts where at least one party has this contact_id.
	ContactID *uuid.UUID
}

// ============================================================================
// Input types — Parties
// ============================================================================

// AddPartyInput holds data for adding a party to a contract.
type AddPartyInput struct {
	TenantID       uuid.UUID
	ContractID     uuid.UUID
	PartyType      PartyType
	ContactID      *uuid.UUID
	CompanyID      *uuid.UUID
	ExternalName   *string
	RoleInContract string
	SignedOn       *time.Time
}

// ============================================================================
// Input types — Reminders
// ============================================================================

// CreateReminderInput holds data for creating a contract reminder.
type CreateReminderInput struct {
	TenantID     uuid.UUID
	ContractID   uuid.UUID
	RemindAt     time.Time
	ReminderType ReminderType
	Subject      string
	Message      string
}

// UpdateReminderInput holds data for updating a reminder.
type UpdateReminderInput struct {
	TenantID     uuid.UUID
	ReminderID   uuid.UUID
	RemindAt     *time.Time
	ReminderType *ReminderType
	Subject      *string
	Message      *string
	Status       *ReminderStatus
}

// ============================================================================
// Contract methods
// ============================================================================

// CreateContract creates a new contract.
func (s *Service) CreateContract(ctx context.Context, input CreateContractInput) (*Contract, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrInvalidInput
	}
	number := strings.TrimSpace(input.ContractNumber)
	if number == "" {
		return nil, ErrInvalidInput
	}
	if input.ContractType == "" {
		return nil, ErrInvalidInput
	}

	exists, err := s.repo.ContractNumberExists(ctx, input.TenantID, number, nil)
	if err != nil {
		return nil, fmt.Errorf("check contract number: %w", err)
	}
	if exists {
		return nil, ErrContractNumberTaken
	}

	now := time.Now()
	c := &Contract{
		ID:             uuid.New(),
		TenantID:       input.TenantID,
		ContractNumber: number,
		Title:          title,
		ContractType:   input.ContractType,
		Status:         ContractStatusDraft,
		StartsOn:       input.StartsOn,
		EndsOn:         input.EndsOn,
		DocumentURL:    input.DocumentURL,
		Notes:          input.Notes,
		CreatedBy:      input.CreatedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if createErr := s.repo.CreateContract(ctx, c); createErr != nil {
		return nil, fmt.Errorf("create contract: %w", createErr)
	}

	slog.Info("contract created",
		"contract_id", c.ID,
		"tenant_id", c.TenantID,
		"contract_number", c.ContractNumber,
	)
	return c, nil
}

// UpdateContract applies partial updates to a contract.
func (s *Service) UpdateContract(ctx context.Context, input UpdateContractInput) (*Contract, error) {
	c, err := s.repo.GetContract(ctx, input.TenantID, input.ContractID)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		t := strings.TrimSpace(*input.Title)
		if t == "" {
			return nil, ErrInvalidInput
		}
		c.Title = t
	}
	if input.ContractType != nil {
		c.ContractType = *input.ContractType
	}
	if input.Status != nil {
		c.Status = *input.Status
	}
	if input.StartsOn != nil {
		c.StartsOn = *input.StartsOn
	}
	if input.ClearEndsOn {
		c.EndsOn = nil
	} else if input.EndsOn != nil {
		c.EndsOn = input.EndsOn
	}
	if input.DocumentURL != nil {
		c.DocumentURL = input.DocumentURL
	}
	if input.Notes != nil {
		c.Notes = *input.Notes
	}

	c.UpdatedAt = time.Now()
	// Nullify embedded slices to avoid overwriting them in the repo
	c.Parties = nil
	c.Reminders = nil

	if updateErr := s.repo.UpdateContract(ctx, c); updateErr != nil {
		return nil, fmt.Errorf("update contract: %w", updateErr)
	}

	slog.Info("contract updated",
		"contract_id", c.ID,
		"tenant_id", c.TenantID,
	)
	return c, nil
}

// DeleteContract removes a contract — only draft contracts may be deleted.
func (s *Service) DeleteContract(ctx context.Context, tenantID, contractID uuid.UUID) error {
	c, err := s.repo.GetContract(ctx, tenantID, contractID)
	if err != nil {
		return err
	}
	if c.Status != ContractStatusDraft {
		return ErrDeleteNonDraft
	}

	if delErr := s.repo.DeleteContract(ctx, tenantID, contractID); delErr != nil {
		return fmt.Errorf("delete contract: %w", delErr)
	}

	slog.Info("contract deleted",
		"contract_id", contractID,
		"tenant_id", tenantID,
	)
	return nil
}

// GetContract retrieves a contract including its parties and reminders.
func (s *Service) GetContract(ctx context.Context, tenantID, contractID uuid.UUID) (*Contract, error) {
	return s.repo.GetContract(ctx, tenantID, contractID)
}

// ListContracts returns paginated contracts for a tenant with optional filters.
func (s *Service) ListContracts(ctx context.Context, input ListContractsInput) ([]*Contract, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}
	offset := (input.Page - 1) * input.PageSize
	filter := ListContractsFilter{
		Status:       input.Status,
		Type:         input.Type,
		StartsAfter:  input.StartsAfter,
		StartsBefore: input.StartsBefore,
		EndsAfter:    input.EndsAfter,
		EndsBefore:   input.EndsBefore,
		ContactID:    input.ContactID,
	}
	return s.repo.ListContracts(ctx, input.TenantID, filter, offset, input.PageSize)
}

// UploadDocument returns a stub signed URL for document upload.
// TODO: Sprint 3 — integrate with document/MinIO service for actual signed URL generation.
func (s *Service) UploadDocument(_ context.Context, tenantID, contractID uuid.UUID) (string, error) {
	// Stub: returns a placeholder path. Phase D wires this to MinIO presign.
	return fmt.Sprintf("/api/v1/vertraege/contracts/%s/document", contractID), nil
}

// ExportContract produces a plain-text dump of the contract.
// TODO: Sprint 3 — replace with PDF renderer (e.g. wkhtmltopdf or gotenberg).
func (s *Service) ExportContract(ctx context.Context, tenantID, contractID uuid.UUID) ([]byte, error) {
	c, err := s.repo.GetContract(ctx, tenantID, contractID)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "CONTRACT: %s\n", c.Title)
	fmt.Fprintf(&sb, "Number:   %s\n", c.ContractNumber)
	fmt.Fprintf(&sb, "Type:     %s\n", c.ContractType)
	fmt.Fprintf(&sb, "Status:   %s\n", c.Status)
	fmt.Fprintf(&sb, "Starts:   %s\n", c.StartsOn.Format("2006-01-02"))
	if c.EndsOn != nil {
		fmt.Fprintf(&sb, "Ends:     %s\n", c.EndsOn.Format("2006-01-02"))
	} else {
		sb.WriteString("Ends:     open-ended\n")
	}
	if c.Notes != "" {
		fmt.Fprintf(&sb, "Notes:    %s\n", c.Notes)
	}

	fmt.Fprintf(&sb, "\nParties (%d):\n", len(c.Parties))
	for _, p := range c.Parties {
		name := string(p.PartyType)
		if p.ExternalName != nil {
			name = *p.ExternalName
		}
		fmt.Fprintf(&sb, "  - %s (%s)\n", name, p.RoleInContract)
	}

	return []byte(sb.String()), nil
}

// ============================================================================
// Party methods
// ============================================================================

// AddParty attaches a party to a contract.
func (s *Service) AddParty(ctx context.Context, input AddPartyInput) (*ContractParty, error) {
	if input.RoleInContract == "" {
		return nil, ErrInvalidInput
	}
	if input.PartyType == PartyTypeExternal && (input.ExternalName == nil || strings.TrimSpace(*input.ExternalName) == "") {
		return nil, ErrInvalidInput
	}

	p := &ContractParty{
		ID:             uuid.New(),
		TenantID:       input.TenantID,
		ContractID:     input.ContractID,
		PartyType:      input.PartyType,
		ContactID:      input.ContactID,
		CompanyID:      input.CompanyID,
		ExternalName:   input.ExternalName,
		RoleInContract: input.RoleInContract,
		SignedOn:       input.SignedOn,
		CreatedAt:      time.Now(),
	}

	if addErr := s.repo.AddParty(ctx, p); addErr != nil {
		return nil, fmt.Errorf("add party: %w", addErr)
	}

	slog.Info("contract party added",
		"party_id", p.ID,
		"contract_id", p.ContractID,
	)
	return p, nil
}

// RemoveParty detaches a party from a contract.
func (s *Service) RemoveParty(ctx context.Context, tenantID, partyID uuid.UUID) error {
	if delErr := s.repo.RemoveParty(ctx, tenantID, partyID); delErr != nil {
		return fmt.Errorf("remove party: %w", delErr)
	}
	slog.Info("contract party removed", "party_id", partyID, "tenant_id", tenantID)
	return nil
}

// ListParties returns all parties for a contract.
func (s *Service) ListParties(ctx context.Context, tenantID, contractID uuid.UUID) ([]*ContractParty, error) {
	return s.repo.ListParties(ctx, tenantID, contractID)
}

// ============================================================================
// Reminder methods
// ============================================================================

// CreateReminder schedules a new reminder for a contract.
func (s *Service) CreateReminder(ctx context.Context, input CreateReminderInput) (*ContractReminder, error) {
	if strings.TrimSpace(input.Subject) == "" {
		return nil, ErrInvalidInput
	}
	if input.RemindAt.IsZero() {
		return nil, ErrInvalidInput
	}

	rem := &ContractReminder{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		ContractID:   input.ContractID,
		RemindAt:     input.RemindAt,
		ReminderType: input.ReminderType,
		Subject:      input.Subject,
		Message:      input.Message,
		Status:       ReminderStatusPending,
		CreatedAt:    time.Now(),
	}

	if createErr := s.repo.CreateReminder(ctx, rem); createErr != nil {
		return nil, fmt.Errorf("create reminder: %w", createErr)
	}

	slog.Info("contract reminder created",
		"reminder_id", rem.ID,
		"contract_id", rem.ContractID,
		"remind_at", rem.RemindAt,
	)
	return rem, nil
}

// UpdateReminder applies partial updates to a reminder.
func (s *Service) UpdateReminder(ctx context.Context, input UpdateReminderInput) (*ContractReminder, error) {
	rem, err := s.repo.GetReminder(ctx, input.TenantID, input.ReminderID)
	if err != nil {
		return nil, err
	}

	if input.RemindAt != nil {
		rem.RemindAt = *input.RemindAt
	}
	if input.ReminderType != nil {
		rem.ReminderType = *input.ReminderType
	}
	if input.Subject != nil {
		rem.Subject = *input.Subject
	}
	if input.Message != nil {
		rem.Message = *input.Message
	}
	if input.Status != nil {
		rem.Status = *input.Status
	}

	if updateErr := s.repo.UpdateReminder(ctx, rem); updateErr != nil {
		return nil, fmt.Errorf("update reminder: %w", updateErr)
	}
	return rem, nil
}

// DeleteReminder removes a reminder.
func (s *Service) DeleteReminder(ctx context.Context, tenantID, reminderID uuid.UUID) error {
	if delErr := s.repo.DeleteReminder(ctx, tenantID, reminderID); delErr != nil {
		return fmt.Errorf("delete reminder: %w", delErr)
	}
	slog.Info("contract reminder deleted", "reminder_id", reminderID)
	return nil
}

// ListReminders returns all reminders for a contract, optionally filtered to pending only.
func (s *Service) ListReminders(ctx context.Context, tenantID, contractID uuid.UUID, onlyPending bool) ([]*ContractReminder, error) {
	return s.repo.ListReminders(ctx, tenantID, contractID, onlyPending)
}

// ============================================================================
// Signature
// ============================================================================

// contractSignatureValidPrefixes lists the accepted MIME-type prefixes for EES inline signatures.
var contractSignatureValidPrefixes = []string{
	"data:image/png;base64,",
	"data:image/svg+xml;base64,",
}

const contractSignatureMaxBytes = 1_048_576 // 1 MiB

// SaveSignature stores an EES inline signature for a contract.
// Validates the data-URI prefix, enforces the 1 MiB size limit, and delegates persistence.
func (s *Service) SaveSignature(ctx context.Context, tenantID, contractID, signatureData, signedBy string) (*Contract, error) {
	if signatureData == "" {
		return nil, ErrInvalidInput
	}

	hasValidPrefix := false
	for _, prefix := range contractSignatureValidPrefixes {
		if len(signatureData) >= len(prefix) && signatureData[:len(prefix)] == prefix {
			hasValidPrefix = true
			break
		}
	}
	if !hasValidPrefix {
		return nil, ErrInvalidInput
	}

	if len(signatureData) > contractSignatureMaxBytes {
		return nil, ErrInvalidInput
	}

	if strings.TrimSpace(signedBy) == "" {
		return nil, ErrInvalidInput
	}

	contract, err := s.repo.SaveSignature(ctx, tenantID, contractID, signatureData, signedBy)
	if err != nil {
		return nil, fmt.Errorf("save contract signature: %w", err)
	}

	slog.Info("contract signature saved", "contract_id", contractID, "tenant_id", tenantID, "signed_by", signedBy)
	return contract, nil
}
