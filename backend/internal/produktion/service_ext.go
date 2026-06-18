package produktion

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Input types — BOMs
// ============================================================================

type CreateBomItemInput struct {
	MaterialName string
	Quantity     float64
	Unit         string
	SortOrder    int
}

type CreateBOMInput struct {
	TenantID    uuid.UUID
	ProductName string
	SKU         string
	Version     string
	Active      bool
	Notes       string
	Items       []CreateBomItemInput
	CreatedBy   *uuid.UUID
}

type UpdateBOMInput struct {
	TenantID    uuid.UUID
	BOMID       uuid.UUID
	ProductName *string
	SKU         *string
	Version     *string
	Active      *bool
	Notes       *string
}

type ListBOMsInput struct {
	TenantID   uuid.UUID
	ActiveOnly *bool
	Page       int
	PageSize   int
}

// ============================================================================
// Input types — Work Steps
// ============================================================================

type CreateWorkStepInput struct {
	TenantID        uuid.UUID
	OrderID         uuid.UUID
	StepNr          int
	Name            string
	Description     string
	DurationMinutes int
	Assignee        string
}

type UpdateWorkStepInput struct {
	TenantID        uuid.UUID
	StepID          uuid.UUID
	Name            *string
	Description     *string
	DurationMinutes *int
	Status          *WorkStepStatus
	Assignee        *string
}

// ============================================================================
// Input types — Machines
// ============================================================================

type CreateMachineInput struct {
	TenantID  uuid.UUID
	Name      string
	Type      string
	Notes     string
	CreatedBy *uuid.UUID
}

type UpdateMachineInput struct {
	TenantID  uuid.UUID
	MachineID uuid.UUID
	Name      *string
	Type      *string
	Status    *MachineStatus
	Notes     *string
}

type ListMachinesInput struct {
	TenantID uuid.UUID
	Status   *MachineStatus
	Page     int
	PageSize int
}

// ============================================================================
// Input types — Quality Checks
// ============================================================================

type CreateQualityCheckInput struct {
	TenantID     uuid.UUID
	OrderID      uuid.UUID
	Inspector    string
	CheckedAt    time.Time
	Passed       bool
	DefectsFound int
	Notes        string
	CreatedBy    *uuid.UUID
}

type ListQualityChecksInput struct {
	TenantID uuid.UUID
	OrderID  *uuid.UUID
	Page     int
	PageSize int
}

// ============================================================================
// BOM methods
// ============================================================================

func (s *Service) CreateBOM(ctx context.Context, input CreateBOMInput) (*BOM, error) {
	name := strings.TrimSpace(input.ProductName)
	if name == "" {
		return nil, ErrInvalidInput
	}
	sku := strings.TrimSpace(input.SKU)
	if sku == "" {
		return nil, ErrInvalidInput
	}
	version := strings.TrimSpace(input.Version)
	if version == "" {
		version = "1.0"
	}

	now := time.Now()
	bom := &BOM{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		ProductName: name,
		SKU:         sku,
		Version:     version,
		Active:      input.Active,
		Notes:       input.Notes,
		CreatedBy:   input.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	for i, itemInput := range input.Items {
		bom.Items = append(bom.Items, &BomItem{
			ID:           uuid.New(),
			TenantID:     input.TenantID,
			BomID:        bom.ID,
			MaterialName: strings.TrimSpace(itemInput.MaterialName),
			Quantity:     itemInput.Quantity,
			Unit:         itemInput.Unit,
			SortOrder:    i,
		})
	}

	if err := s.repo.CreateBOM(ctx, bom); err != nil {
		return nil, fmt.Errorf("create bom: %w", err)
	}

	slog.Info("bom created", "bom_id", bom.ID, "tenant_id", bom.TenantID, "sku", bom.SKU)
	return bom, nil
}

func (s *Service) UpdateBOM(ctx context.Context, input UpdateBOMInput) (*BOM, error) {
	bom, err := s.repo.GetBOM(ctx, input.TenantID, input.BOMID)
	if err != nil {
		return nil, err
	}

	if input.ProductName != nil {
		name := strings.TrimSpace(*input.ProductName)
		if name == "" {
			return nil, ErrInvalidInput
		}
		bom.ProductName = name
	}
	if input.SKU != nil {
		sku := strings.TrimSpace(*input.SKU)
		if sku == "" {
			return nil, ErrInvalidInput
		}
		bom.SKU = sku
	}
	if input.Version != nil {
		bom.Version = *input.Version
	}
	if input.Active != nil {
		bom.Active = *input.Active
	}
	if input.Notes != nil {
		bom.Notes = *input.Notes
	}
	bom.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateBOM(ctx, bom); updateErr != nil {
		return nil, fmt.Errorf("update bom: %w", updateErr)
	}

	slog.Info("bom updated", "bom_id", bom.ID, "tenant_id", bom.TenantID)
	return bom, nil
}

func (s *Service) GetBOM(ctx context.Context, tenantID, bomID uuid.UUID) (*BOM, error) {
	return s.repo.GetBOM(ctx, tenantID, bomID)
}

func (s *Service) ListBOMs(ctx context.Context, input ListBOMsInput) ([]*BOM, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	return s.repo.ListBOMs(ctx, input.TenantID, input.ActiveOnly, offset, input.PageSize)
}

func (s *Service) DeleteBOM(ctx context.Context, tenantID, bomID uuid.UUID) error {
	if _, err := s.repo.GetBOM(ctx, tenantID, bomID); err != nil {
		return err
	}
	if delErr := s.repo.DeleteBOM(ctx, tenantID, bomID); delErr != nil {
		return fmt.Errorf("delete bom: %w", delErr)
	}
	slog.Info("bom deleted", "bom_id", bomID, "tenant_id", tenantID)
	return nil
}

// ============================================================================
// Work Step methods
// ============================================================================

func (s *Service) CreateWorkStep(ctx context.Context, input CreateWorkStepInput) (*WorkStep, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	step := &WorkStep{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		OrderID:         input.OrderID,
		StepNr:          input.StepNr,
		Name:            name,
		Description:     input.Description,
		DurationMinutes: input.DurationMinutes,
		Status:          WorkStepStatusPending,
		Assignee:        input.Assignee,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateWorkStep(ctx, step); err != nil {
		return nil, fmt.Errorf("create work step: %w", err)
	}

	slog.Info("work step created", "step_id", step.ID, "order_id", step.OrderID)
	return step, nil
}

func (s *Service) UpdateWorkStep(ctx context.Context, input UpdateWorkStepInput) (*WorkStep, error) {
	step, err := s.repo.GetWorkStep(ctx, input.TenantID, input.StepID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrInvalidInput
		}
		step.Name = name
	}
	if input.Description != nil {
		step.Description = *input.Description
	}
	if input.DurationMinutes != nil {
		step.DurationMinutes = *input.DurationMinutes
	}
	if input.Status != nil {
		step.Status = *input.Status
	}
	if input.Assignee != nil {
		step.Assignee = *input.Assignee
	}
	step.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateWorkStep(ctx, step); updateErr != nil {
		return nil, fmt.Errorf("update work step: %w", updateErr)
	}

	slog.Info("work step updated", "step_id", step.ID)
	return step, nil
}

func (s *Service) ListWorkSteps(ctx context.Context, tenantID, orderID uuid.UUID) ([]*WorkStep, error) {
	return s.repo.ListWorkSteps(ctx, tenantID, orderID)
}

func (s *Service) DeleteWorkStep(ctx context.Context, tenantID, stepID uuid.UUID) error {
	if _, err := s.repo.GetWorkStep(ctx, tenantID, stepID); err != nil {
		return err
	}
	if delErr := s.repo.DeleteWorkStep(ctx, tenantID, stepID); delErr != nil {
		return fmt.Errorf("delete work step: %w", delErr)
	}
	slog.Info("work step deleted", "step_id", stepID)
	return nil
}

// ============================================================================
// Machine methods
// ============================================================================

func (s *Service) CreateMachine(ctx context.Context, input CreateMachineInput) (*Machine, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	machine := &Machine{
		ID:        uuid.New(),
		TenantID:  input.TenantID,
		Name:      name,
		Type:      input.Type,
		Status:    MachineStatusAvailable,
		Notes:     input.Notes,
		CreatedBy: input.CreatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateMachine(ctx, machine); err != nil {
		return nil, fmt.Errorf("create machine: %w", err)
	}

	slog.Info("machine created", "machine_id", machine.ID, "tenant_id", machine.TenantID)
	return machine, nil
}

func (s *Service) UpdateMachine(ctx context.Context, input UpdateMachineInput) (*Machine, error) {
	machine, err := s.repo.GetMachine(ctx, input.TenantID, input.MachineID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrInvalidInput
		}
		machine.Name = name
	}
	if input.Type != nil {
		machine.Type = *input.Type
	}
	if input.Status != nil {
		machine.Status = *input.Status
	}
	if input.Notes != nil {
		machine.Notes = *input.Notes
	}
	machine.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateMachine(ctx, machine); updateErr != nil {
		return nil, fmt.Errorf("update machine: %w", updateErr)
	}

	slog.Info("machine updated", "machine_id", machine.ID)
	return machine, nil
}

func (s *Service) GetMachine(ctx context.Context, tenantID, machineID uuid.UUID) (*Machine, error) {
	return s.repo.GetMachine(ctx, tenantID, machineID)
}

func (s *Service) ListMachines(ctx context.Context, input ListMachinesInput) ([]*Machine, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	return s.repo.ListMachines(ctx, input.TenantID, input.Status, offset, input.PageSize)
}

func (s *Service) DeleteMachine(ctx context.Context, tenantID, machineID uuid.UUID) error {
	if _, err := s.repo.GetMachine(ctx, tenantID, machineID); err != nil {
		return err
	}
	if delErr := s.repo.DeleteMachine(ctx, tenantID, machineID); delErr != nil {
		return fmt.Errorf("delete machine: %w", delErr)
	}
	slog.Info("machine deleted", "machine_id", machineID)
	return nil
}

// ============================================================================
// Quality Check methods
// ============================================================================

func (s *Service) CreateQualityCheck(ctx context.Context, input CreateQualityCheckInput) (*QualityCheck, error) {
	inspector := strings.TrimSpace(input.Inspector)
	if inspector == "" {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	check := &QualityCheck{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		OrderID:      input.OrderID,
		Inspector:    inspector,
		CheckedAt:    input.CheckedAt,
		Passed:       input.Passed,
		DefectsFound: input.DefectsFound,
		Notes:        input.Notes,
		CreatedBy:    input.CreatedBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateQualityCheck(ctx, check); err != nil {
		return nil, fmt.Errorf("create quality check: %w", err)
	}

	slog.Info("quality check created", "check_id", check.ID, "order_id", check.OrderID, "passed", check.Passed)
	return check, nil
}

func (s *Service) GetQualityCheck(ctx context.Context, tenantID, checkID uuid.UUID) (*QualityCheck, error) {
	return s.repo.GetQualityCheck(ctx, tenantID, checkID)
}

func (s *Service) ListQualityChecks(ctx context.Context, input ListQualityChecksInput) ([]*QualityCheck, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	return s.repo.ListQualityChecks(ctx, input.TenantID, input.OrderID, offset, input.PageSize)
}
