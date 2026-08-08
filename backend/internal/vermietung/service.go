package vermietung

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service handles vermietung business logic.
type Service struct {
	repo Repository
}

// NewService creates a new vermietung service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ============================================================================
// Input types — Objects
// ============================================================================

type CreateObjectInput struct {
	TenantID    uuid.UUID
	Name        string
	Description *string
	Category    string
	DailyRate   float64
	Deposit     float64
	Location    *string
	Notes       *string
}

type UpdateObjectInput struct {
	TenantID    uuid.UUID
	ObjectID    uuid.UUID
	Name        *string
	Description *string
	Category    *string
	DailyRate   *float64
	Deposit     *float64
	Location    *string
	Notes       *string
	Active      *bool
}

type ListObjectsInput struct {
	TenantID   uuid.UUID
	Search     string
	Category   *string
	ActiveOnly bool
	Page       int
	PageSize   int
}

// ============================================================================
// Input types — Rentals
// ============================================================================

type CreateRentalInput struct {
	TenantID    uuid.UUID
	ObjectID    uuid.UUID
	ContactID   *uuid.UUID
	RenterName  string
	StartDate   time.Time
	EndDate     time.Time
	TotalPrice  float64
	DepositPaid bool
	Notes       *string
}

type UpdateRentalInput struct {
	TenantID    uuid.UUID
	RentalID    uuid.UUID
	RenterName  *string
	StartDate   *time.Time
	EndDate     *time.Time
	TotalPrice  *float64
	DepositPaid *bool
	Notes       *string
}

type ListRentalsInput struct {
	TenantID uuid.UUID
	ObjectID *uuid.UUID
	Status   *RentalStatus
	From     *time.Time
	To       *time.Time
	Page     int
	PageSize int
}

type CheckAvailabilityInput struct {
	TenantID        uuid.UUID
	ObjectID        uuid.UUID
	StartDate       time.Time
	EndDate         time.Time
	ExcludeRentalID *uuid.UUID
}

type CheckAvailabilityResult struct {
	Available          bool
	ConflictingRentals []*Rental
}

// ============================================================================
// Input types — Inspections
// ============================================================================

type CreateInspectionInput struct {
	TenantID    uuid.UUID
	RentalID    uuid.UUID
	Kind        InspectionKind
	Notes       string
	PhotoURLs   []string
	PerformedBy *uuid.UUID
	Checklist   []ChecklistItem
}

type UpdateInspectionInput struct {
	TenantID         uuid.UUID
	InspectionID     uuid.UUID
	Notes            *string
	PhotoURLs        []string // full replacement; nil = no change
	ReplacePhotos    bool
	Checklist        []ChecklistItem // full replacement; nil = no change
	ReplaceChecklist bool
	SignatureData    *string // nil = no change
}

type ListInspectionsInput struct {
	TenantID uuid.UUID
	RentalID uuid.UUID
	Page     int
	PageSize int
}

// ============================================================================
// Object methods
// ============================================================================

func (s *Service) CreateObject(ctx context.Context, input CreateObjectInput) (*RentalObject, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	if input.DailyRate < 0 {
		return nil, ErrInvalidInput
	}
	if input.Deposit < 0 {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	obj := &RentalObject{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		Name:        name,
		Description: input.Description,
		Category:    input.Category,
		DailyRate:   input.DailyRate,
		Deposit:     input.Deposit,
		Location:    input.Location,
		Notes:       input.Notes,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateObject(ctx, obj); err != nil {
		return nil, fmt.Errorf("create rental object: %w", err)
	}

	slog.Info("rental object created", "object_id", obj.ID, "tenant_id", obj.TenantID)
	return obj, nil
}

func (s *Service) UpdateObject(ctx context.Context, input UpdateObjectInput) (*RentalObject, error) {
	obj, err := s.repo.GetObject(ctx, input.TenantID, input.ObjectID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		n := strings.TrimSpace(*input.Name)
		if n == "" {
			return nil, ErrInvalidInput
		}
		obj.Name = n
	}
	if input.Description != nil {
		obj.Description = input.Description
	}
	if input.Category != nil {
		obj.Category = *input.Category
	}
	if input.DailyRate != nil {
		if *input.DailyRate < 0 {
			return nil, ErrInvalidInput
		}
		obj.DailyRate = *input.DailyRate
	}
	if input.Deposit != nil {
		if *input.Deposit < 0 {
			return nil, ErrInvalidInput
		}
		obj.Deposit = *input.Deposit
	}
	if input.Location != nil {
		obj.Location = input.Location
	}
	if input.Notes != nil {
		obj.Notes = input.Notes
	}
	if input.Active != nil {
		obj.Active = *input.Active
	}

	obj.UpdatedAt = time.Now()
	if err := s.repo.UpdateObject(ctx, obj); err != nil {
		return nil, fmt.Errorf("update rental object: %w", err)
	}

	slog.Info("rental object updated", "object_id", obj.ID, "tenant_id", obj.TenantID)
	return obj, nil
}

func (s *Service) DeleteObject(ctx context.Context, tenantID, objectID uuid.UUID) error {
	if _, err := s.repo.GetObject(ctx, tenantID, objectID); err != nil {
		return err
	}
	if err := s.repo.SoftDeleteObject(ctx, tenantID, objectID); err != nil {
		return fmt.Errorf("soft delete rental object: %w", err)
	}
	slog.Info("rental object deleted", "object_id", objectID, "tenant_id", tenantID)
	return nil
}

func (s *Service) GetObject(ctx context.Context, tenantID, objectID uuid.UUID) (*RentalObject, error) {
	return s.repo.GetObject(ctx, tenantID, objectID)
}

func (s *Service) ListObjects(ctx context.Context, input ListObjectsInput) ([]*RentalObject, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	return s.repo.ListObjects(ctx, input.TenantID, ListObjectsFilter{
		Search:     input.Search,
		Category:   input.Category,
		ActiveOnly: input.ActiveOnly,
	}, offset, input.PageSize)
}

// ============================================================================
// Availability check
// ============================================================================

func (s *Service) CheckAvailability(ctx context.Context, input CheckAvailabilityInput) (*CheckAvailabilityResult, error) {
	if !input.EndDate.After(input.StartDate) {
		return nil, ErrInvalidInput
	}

	overlap, err := s.repo.HasOverlap(ctx, input.TenantID, input.ObjectID, input.StartDate, input.EndDate, input.ExcludeRentalID)
	if err != nil {
		return nil, fmt.Errorf("check overlap: %w", err)
	}

	result := &CheckAvailabilityResult{Available: !overlap}
	if overlap {
		// Fetch conflicting rentals for the caller to display
		statusActive := RentalStatusActive
		_ = statusActive // included in ListRentals via From/To filter
		conflicts, _, listErr := s.repo.ListRentals(ctx, input.TenantID, ListRentalsFilter{
			ObjectID: &input.ObjectID,
			From:     &input.StartDate,
			To:       &input.EndDate,
		}, 0, 50)
		if listErr == nil {
			// Filter out cancelled ones
			for _, r := range conflicts {
				if r.Status != RentalStatusCancelled {
					result.ConflictingRentals = append(result.ConflictingRentals, r)
				}
			}
		}
	}

	return result, nil
}

// ============================================================================
// Rental methods
// ============================================================================

// CreateRental creates a new rental after verifying date-range availability.
// This is the critical guard: overlap query BEFORE insert.
func (s *Service) CreateRental(ctx context.Context, input CreateRentalInput) (*Rental, error) {
	if strings.TrimSpace(input.RenterName) == "" {
		return nil, ErrInvalidInput
	}
	if !input.EndDate.After(input.StartDate) {
		return nil, ErrInvalidInput
	}

	// Ensure the object exists and belongs to tenant
	if _, err := s.repo.GetObject(ctx, input.TenantID, input.ObjectID); err != nil {
		return nil, err
	}

	// Pre-check: overlap detection BEFORE insert
	overlap, err := s.repo.HasOverlap(ctx, input.TenantID, input.ObjectID, input.StartDate, input.EndDate, nil)
	if err != nil {
		return nil, fmt.Errorf("check rental overlap: %w", err)
	}
	if overlap {
		return nil, ErrRentalConflict
	}

	now := time.Now()
	rental := &Rental{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		ObjectID:    input.ObjectID,
		ContactID:   input.ContactID,
		RenterName:  strings.TrimSpace(input.RenterName),
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
		Status:      RentalStatusReserved,
		TotalPrice:  input.TotalPrice,
		DepositPaid: input.DepositPaid,
		Notes:       input.Notes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateRental(ctx, rental); err != nil {
		return nil, fmt.Errorf("create rental: %w", err)
	}

	slog.Info("rental created",
		"rental_id", rental.ID,
		"object_id", rental.ObjectID,
		"tenant_id", rental.TenantID,
		"start", rental.StartDate,
		"end", rental.EndDate,
	)
	return rental, nil
}

func (s *Service) UpdateRental(ctx context.Context, input UpdateRentalInput) (*Rental, error) {
	rental, err := s.repo.GetRental(ctx, input.TenantID, input.RentalID)
	if err != nil {
		return nil, err
	}

	if input.RenterName != nil {
		n := strings.TrimSpace(*input.RenterName)
		if n == "" {
			return nil, ErrInvalidInput
		}
		rental.RenterName = n
	}

	newStart := rental.StartDate
	newEnd := rental.EndDate

	if input.StartDate != nil {
		newStart = *input.StartDate
	}
	if input.EndDate != nil {
		newEnd = *input.EndDate
	}

	if (input.StartDate != nil || input.EndDate != nil) && (newStart != rental.StartDate || newEnd != rental.EndDate) {
		if !newEnd.After(newStart) {
			return nil, ErrInvalidInput
		}
		// Re-check overlap excluding this rental
		overlap, overlapErr := s.repo.HasOverlap(ctx, input.TenantID, rental.ObjectID, newStart, newEnd, &rental.ID)
		if overlapErr != nil {
			return nil, fmt.Errorf("check rental overlap on update: %w", overlapErr)
		}
		if overlap {
			return nil, ErrRentalConflict
		}
		rental.StartDate = newStart
		rental.EndDate = newEnd
	}

	if input.TotalPrice != nil {
		rental.TotalPrice = *input.TotalPrice
	}
	if input.DepositPaid != nil {
		rental.DepositPaid = *input.DepositPaid
	}
	if input.Notes != nil {
		rental.Notes = input.Notes
	}

	rental.UpdatedAt = time.Now()
	if err := s.repo.UpdateRental(ctx, rental); err != nil {
		return nil, fmt.Errorf("update rental: %w", err)
	}

	slog.Info("rental updated", "rental_id", rental.ID, "tenant_id", rental.TenantID)
	return rental, nil
}

func (s *Service) DeleteRental(ctx context.Context, tenantID, rentalID uuid.UUID) error {
	if _, err := s.repo.GetRental(ctx, tenantID, rentalID); err != nil {
		return err
	}
	if err := s.repo.DeleteRental(ctx, tenantID, rentalID); err != nil {
		return fmt.Errorf("delete rental: %w", err)
	}
	slog.Info("rental deleted", "rental_id", rentalID, "tenant_id", tenantID)
	return nil
}

func (s *Service) GetRental(ctx context.Context, tenantID, rentalID uuid.UUID) (*Rental, error) {
	return s.repo.GetRental(ctx, tenantID, rentalID)
}

func (s *Service) ListRentals(ctx context.Context, input ListRentalsInput) ([]*Rental, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	return s.repo.ListRentals(ctx, input.TenantID, ListRentalsFilter{
		ObjectID: input.ObjectID,
		Status:   input.Status,
		From:     input.From,
		To:       input.To,
	}, offset, input.PageSize)
}

// StartRental transitions a rental from reserved → active.
// Guard: rental.Status MUST be "reserved".
func (s *Service) StartRental(ctx context.Context, tenantID, rentalID uuid.UUID) (*Rental, error) {
	rental, err := s.repo.GetRental(ctx, tenantID, rentalID)
	if err != nil {
		return nil, err
	}

	if rental.Status != RentalStatusReserved {
		return nil, ErrInvalidStateTransition
	}

	rental.Status = RentalStatusActive
	rental.UpdatedAt = time.Now()

	if err := s.repo.UpdateRental(ctx, rental); err != nil {
		return nil, fmt.Errorf("start rental: %w", err)
	}

	slog.Info("rental started", "rental_id", rental.ID, "tenant_id", rental.TenantID)
	return rental, nil
}

// EndRental transitions a rental from active → completed.
// Guard: rental.Status MUST be "active".
func (s *Service) EndRental(ctx context.Context, tenantID, rentalID uuid.UUID) (*Rental, error) {
	rental, err := s.repo.GetRental(ctx, tenantID, rentalID)
	if err != nil {
		return nil, err
	}

	if rental.Status != RentalStatusActive {
		return nil, ErrInvalidStateTransition
	}

	rental.Status = RentalStatusCompleted
	rental.UpdatedAt = time.Now()

	if err := s.repo.UpdateRental(ctx, rental); err != nil {
		return nil, fmt.Errorf("end rental: %w", err)
	}

	slog.Info("rental ended", "rental_id", rental.ID, "tenant_id", rental.TenantID)
	return rental, nil
}

// ============================================================================
// Inspection methods
// ============================================================================

func (s *Service) CreateInspection(ctx context.Context, input CreateInspectionInput) (*RentalInspection, error) {
	if input.Kind != InspectionKindHandover && input.Kind != InspectionKindReturn {
		return nil, ErrInvalidInput
	}

	// Verify rental exists and belongs to tenant
	if _, err := s.repo.GetRental(ctx, input.TenantID, input.RentalID); err != nil {
		return nil, err
	}

	// Guard: only one inspection of each kind per rental (e.g. no two handover inspections).
	// The DB UNIQUE constraint (migration 000101) is the hard stop; this pre-check provides
	// a friendly error message rather than a raw constraint violation.
	existing, err := s.repo.GetInspectionByKind(ctx, input.TenantID, input.RentalID, input.Kind)
	if err == nil && existing != nil {
		return nil, ErrInspectionKindExists
	}

	photos := input.PhotoURLs
	if photos == nil {
		photos = []string{}
	}

	checklist, err := normalizeChecklist(input.Checklist)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	ins := &RentalInspection{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		RentalID:    input.RentalID,
		Kind:        input.Kind,
		Notes:       input.Notes,
		PhotoURLs:   photos,
		PerformedBy: input.PerformedBy,
		Checklist:   checklist,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateInspection(ctx, ins); err != nil {
		return nil, fmt.Errorf("create inspection: %w", err)
	}

	slog.Info("inspection created",
		"inspection_id", ins.ID,
		"rental_id", ins.RentalID,
		"kind", ins.Kind,
	)
	return ins, nil
}

func (s *Service) UpdateInspection(ctx context.Context, input UpdateInspectionInput) (*RentalInspection, error) {
	ins, err := s.repo.GetInspection(ctx, input.TenantID, input.InspectionID)
	if err != nil {
		return nil, err
	}

	if input.Notes != nil {
		ins.Notes = *input.Notes
	}
	if input.ReplacePhotos {
		photos := input.PhotoURLs
		if photos == nil {
			photos = []string{}
		}
		ins.PhotoURLs = photos
	}
	if input.ReplaceChecklist {
		checklist, err := normalizeChecklist(input.Checklist)
		if err != nil {
			return nil, err
		}
		ins.Checklist = checklist
	}
	if input.SignatureData != nil {
		if err := validateInlineSignature(*input.SignatureData); err != nil {
			return nil, err
		}
		ins.SignatureData = input.SignatureData
	}

	ins.UpdatedAt = time.Now()
	if err := s.repo.UpdateInspection(ctx, ins); err != nil {
		return nil, fmt.Errorf("update inspection: %w", err)
	}

	slog.Info("inspection updated", "inspection_id", ins.ID, "tenant_id", ins.TenantID)
	return ins, nil
}

func (s *Service) GetInspection(ctx context.Context, tenantID, inspectionID uuid.UUID) (*RentalInspection, error) {
	return s.repo.GetInspection(ctx, tenantID, inspectionID)
}

func (s *Service) ListInspections(ctx context.Context, input ListInspectionsInput) ([]*RentalInspection, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	return s.repo.ListInspections(ctx, input.TenantID, input.RentalID, offset, input.PageSize)
}

// ============================================================================
// Signature
// ============================================================================

// signatureValidPrefixes lists the accepted MIME-type prefixes for EES inline signatures.
var signatureValidPrefixes = []string{
	"data:image/png;base64,",
	"data:image/svg+xml;base64,",
}

const signatureMaxBytes = 1_048_576 // 1 MiB

// validateInlineSignature checks an EES inline signature data-URI against the
// accepted MIME-type prefixes and the size limit. Shared by rental signatures
// (SaveSignature) and inspection signatures (UpdateInspection) so both use the
// same storage form and the same validation.
func validateInlineSignature(signatureData string) error {
	if signatureData == "" {
		return ErrInvalidInput
	}

	hasValidPrefix := false
	for _, prefix := range signatureValidPrefixes {
		if len(signatureData) >= len(prefix) && signatureData[:len(prefix)] == prefix {
			hasValidPrefix = true
			break
		}
	}
	if !hasValidPrefix {
		return ErrInvalidInput
	}

	if len(signatureData) > signatureMaxBytes {
		return ErrInvalidInput
	}

	return nil
}

// normalizeChecklist validates and defaults an inspection checklist: every
// position needs a non-blank label, and a nil input becomes an empty (not
// nil) slice so callers never have to distinguish "no checklist" from "empty
// checklist".
func normalizeChecklist(items []ChecklistItem) ([]ChecklistItem, error) {
	if items == nil {
		return []ChecklistItem{}, nil
	}
	for _, item := range items {
		if strings.TrimSpace(item.Label) == "" {
			return nil, ErrInvalidInput
		}
	}
	return items, nil
}

// SaveSignature stores an EES inline signature for a rental.
// Validates the data-URI prefix, enforces the 1 MiB size limit, and delegates persistence.
func (s *Service) SaveSignature(ctx context.Context, tenantID, rentalID, signatureData, signedBy string) (*Rental, error) {
	if err := validateInlineSignature(signatureData); err != nil {
		return nil, err
	}

	if strings.TrimSpace(signedBy) == "" {
		return nil, ErrInvalidInput
	}

	rental, err := s.repo.SaveSignature(ctx, tenantID, rentalID, signatureData, signedBy)
	if err != nil {
		return nil, fmt.Errorf("save rental signature: %w", err)
	}

	slog.Info("rental signature saved", "rental_id", rentalID, "tenant_id", tenantID, "signed_by", signedBy)
	return rental, nil
}

// ============================================================================
// Calendar
// ============================================================================

// CalendarEntry groups rentals by object for a calendar view.
type CalendarEntry struct {
	Object  *RentalObject
	Rentals []*Rental
}

// GetRentalCalendar returns all rentals for a given month grouped by object.
func (s *Service) GetRentalCalendar(ctx context.Context, tenantID uuid.UUID, year, month int) ([]*CalendarEntry, error) {
	loc := time.UTC
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
	monthEnd := monthStart.AddDate(0, 1, 0)

	rentals, _, err := s.repo.ListRentals(ctx, tenantID, ListRentalsFilter{
		From: &monthStart,
		To:   &monthEnd,
	}, 0, 10000)
	if err != nil {
		return nil, fmt.Errorf("get calendar rentals: %w", err)
	}

	// Group by objectID — exclude cancelled rentals from calendar view
	byObject := make(map[uuid.UUID][]*Rental)
	for _, r := range rentals {
		if r.Status == RentalStatusCancelled {
			continue
		}
		byObject[r.ObjectID] = append(byObject[r.ObjectID], r)
	}

	var entries []*CalendarEntry
	for objectID, objectRentals := range byObject {
		obj, objErr := s.repo.GetObject(ctx, tenantID, objectID)
		if objErr != nil {
			// Object may have been soft-deleted; skip
			slog.Warn("calendar: object not found", "object_id", objectID, "error", objErr)
			continue
		}
		entries = append(entries, &CalendarEntry{
			Object:  obj,
			Rentals: objectRentals,
		})
	}

	return entries, nil
}
