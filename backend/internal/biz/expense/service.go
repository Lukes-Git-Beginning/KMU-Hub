// Package expense implements Ausgaben: an employee claims money already spent,
// somebody else approves or rejects it, and the bookkeeper assigns an SKR03/04
// account afterwards.
//
// lean: a receipt is a filename here, not a stored document. The frontend's
// receipt flow (finance-client.ts financeExpenseApi.attachReceipt) posts
// {receiptName} as JSON and renders a boolean indicator -- there is no upload
// widget and no download link to serve. Add the presign upload from
// internal/chat/file/minio_store.go, plus a document id column, when the
// receipt has to be retrievable (GoBD Belegarchiv already stores documents that
// way; internal/biz/gobdarchive is where such an expense receipt would land).
package expense

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// dateLayout is the wire format of an expense date. An expense is dated by the
// day money left the account, so it never carries a time or a zone.
const dateLayout = "2006-01-02"

// Service holds the expense business rules.
type Service struct {
	repo Repository
}

// NewService creates the service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput is a new expense as it arrives from the caller. Amount and Date
// are strings because that is what crosses the wire; parsing and rejecting them
// is this service's job, not the handler's.
type CreateInput struct {
	Description string
	Amount      string
	Date        string
	Category    string
	Supplier    string
	Project     string
	Account     string
	ReceiptName string
}

// UpdateInput carries the fields of an edit. A nil pointer means "leave as is",
// which is what lets the frontend's PATCH-shaped PUT (UpdateExpenseRequest is a
// Partial<CreateExpenseRequest>) work without clearing everything it omits.
type UpdateInput struct {
	Description *string
	Amount      *string
	Date        *string
	Category    *string
	Supplier    *string
	Project     *string
	Account     *string
	ReceiptName *string
}

// ListInput narrows a list query.
type ListInput struct {
	Status string
	// SubmittedBy comes from the caller's data scope, never from the request
	// body -- a client-supplied owner filter would be a suggestion, not a bound.
	SubmittedBy *uuid.UUID
	Page        int
	PageSize    int
}

// Create records a new expense. It always starts pending: an expense that
// arrives already approved would skip the very control this module exists for.
func (s *Service) Create(ctx context.Context, tenantID, actorID uuid.UUID, in CreateInput) (*models.Expense, error) {
	if actorID == uuid.Nil {
		return nil, ErrMissingActor
	}

	description := strings.TrimSpace(in.Description)
	if description == "" {
		return nil, ErrDescriptionRequired
	}
	amount, err := parseAmount(in.Amount)
	if err != nil {
		return nil, err
	}
	date, err := parseDate(in.Date)
	if err != nil {
		return nil, err
	}

	e := &models.Expense{
		TenantID:    tenantID,
		Description: description,
		Amount:      amount,
		ExpenseDate: date,
		Category:    strings.TrimSpace(in.Category),
		Supplier:    strings.TrimSpace(in.Supplier),
		Project:     optional(in.Project),
		Account:     optional(in.Account),
		ReceiptName: optional(in.ReceiptName),
		Status:      models.ExpenseStatusPending,
		SubmittedBy: &actorID,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "expense created",
		"expense_id", e.ID, "tenant_id", tenantID, "submitted_by", actorID)
	return e, nil
}

// Get returns one expense of the caller's tenant.
func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.Expense, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List returns a page of expenses plus the total matching the same filter.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, in ListInput) ([]*models.Expense, int, error) {
	page, pageSize := in.Page, in.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	return s.repo.List(ctx, tenantID, ListFilter{
		Status:      in.Status,
		SubmittedBy: in.SubmittedBy,
		Limit:       pageSize,
		Offset:      (page - 1) * pageSize,
	})
}

// Update edits an expense.
//
// Once a decision was made, only the bookkeeping fields -- the SKR03/04 account
// and the receipt filename -- stay open. Kontierung and a late receipt are
// normal after-the-fact work; changing the amount or the date of an approved
// expense would make the approval describe something that no longer exists.
func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, in UpdateInput) (*models.Expense, error) {
	e, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if e.Status != models.ExpenseStatusPending && touchesDecidedFields(in) {
		return nil, ErrDecided
	}

	if in.Description != nil {
		description := strings.TrimSpace(*in.Description)
		if description == "" {
			return nil, ErrDescriptionRequired
		}
		e.Description = description
	}
	if in.Amount != nil {
		amount, parseErr := parseAmount(*in.Amount)
		if parseErr != nil {
			return nil, parseErr
		}
		e.Amount = amount
	}
	if in.Date != nil {
		date, parseErr := parseDate(*in.Date)
		if parseErr != nil {
			return nil, parseErr
		}
		e.ExpenseDate = date
	}
	if in.Category != nil {
		e.Category = strings.TrimSpace(*in.Category)
	}
	if in.Supplier != nil {
		e.Supplier = strings.TrimSpace(*in.Supplier)
	}
	if in.Project != nil {
		e.Project = optional(*in.Project)
	}
	if in.Account != nil {
		e.Account = optional(*in.Account)
	}
	if in.ReceiptName != nil {
		e.ReceiptName = optional(*in.ReceiptName)
	}

	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// AttachReceipt records the filename of a document handed in for this expense.
// It stays open after a decision: a receipt arriving late is the normal case,
// and refusing it would leave the expense permanently undocumented.
func (s *Service) AttachReceipt(ctx context.Context, tenantID, id uuid.UUID, receiptName string) (*models.Expense, error) {
	e, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	e.ReceiptName = optional(receiptName)
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// Approve accepts a pending expense.
func (s *Service) Approve(ctx context.Context, tenantID, id, actorID uuid.UUID) (*models.Expense, error) {
	return s.decide(ctx, tenantID, id, actorID, models.ExpenseStatusApproved)
}

// Reject refuses a pending expense.
func (s *Service) Reject(ctx context.Context, tenantID, id, actorID uuid.UUID) (*models.Expense, error) {
	return s.decide(ctx, tenantID, id, actorID, models.ExpenseStatusRejected)
}

// decide is the shared transition behind Approve and Reject. Both guards live
// here rather than in the two callers so neither can be forgotten in one of
// them: only a pending expense can be decided, and never by its submitter.
func (s *Service) decide(ctx context.Context, tenantID, id, actorID uuid.UUID, status string) (*models.Expense, error) {
	if actorID == uuid.Nil {
		return nil, ErrMissingActor
	}
	e, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if e.Status != models.ExpenseStatusPending {
		return nil, ErrNotPending
	}
	if e.SubmittedBy != nil && *e.SubmittedBy == actorID {
		return nil, ErrSelfApproval
	}

	now := time.Now().UTC()
	e.Status = status
	e.DecidedBy = &actorID
	e.DecidedAt = &now

	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "expense decided",
		"expense_id", e.ID, "tenant_id", tenantID, "status", status, "decided_by", actorID)
	return e, nil
}

// Delete removes a pending expense. A decided one stays: it is the record of a
// decision somebody made, and withdrawing it is a new decision, not an erasure.
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	e, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if e.Status != models.ExpenseStatusPending {
		return ErrDecided
	}
	return s.repo.Delete(ctx, tenantID, id)
}

// touchesDecidedFields reports whether an edit reaches past the bookkeeping
// fields that stay open after a decision.
func touchesDecidedFields(in UpdateInput) bool {
	return in.Description != nil || in.Amount != nil || in.Date != nil ||
		in.Category != nil || in.Supplier != nil || in.Project != nil
}

func parseAmount(raw string) (decimal.Decimal, error) {
	amount, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || !amount.IsPositive() {
		return decimal.Zero, ErrInvalidAmount
	}
	// The column is NUMERIC(15,2); rounding here rather than letting Postgres do
	// it keeps the value the caller gets back equal to the value that was stored.
	return amount.Round(2), nil
}

func parseDate(raw string) (time.Time, error) {
	date, err := time.Parse(dateLayout, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}
	return date, nil
}

// optional maps an empty string to nil, so an omitted optional field is stored
// as NULL instead of as an empty string that reads as "set to nothing".
func optional(v string) *string {
	if trimmed := strings.TrimSpace(v); trimmed != "" {
		return &trimmed
	}
	return nil
}
