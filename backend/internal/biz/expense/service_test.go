package expense

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// fakeRepo is an in-memory Repository. It scopes on the tenant the same way the
// SQL does, so a service bug that drops the tenant shows up here rather than
// only against a real database.
type fakeRepo struct {
	rows      map[uuid.UUID]*models.Expense
	createErr error
	updates   int
	deletes   int
	// lastFilter records what List was asked for, so the paging bounds the
	// service applies can be asserted rather than assumed.
	lastFilter ListFilter
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{rows: map[uuid.UUID]*models.Expense{}}
}

func (f *fakeRepo) Create(_ context.Context, e *models.Expense) error {
	if f.createErr != nil {
		return f.createErr
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	stored := *e
	f.rows[e.ID] = &stored
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.Expense, error) {
	e, ok := f.rows[id]
	if !ok || e.TenantID != tenantID {
		return nil, ErrNotFound
	}
	copied := *e
	return &copied, nil
}

func (f *fakeRepo) Update(_ context.Context, e *models.Expense) error {
	stored, ok := f.rows[e.ID]
	if !ok || stored.TenantID != e.TenantID {
		return ErrNotFound
	}
	f.updates++
	copied := *e
	f.rows[e.ID] = &copied
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	e, ok := f.rows[id]
	if !ok || e.TenantID != tenantID {
		return ErrNotFound
	}
	f.deletes++
	delete(f.rows, id)
	return nil
}

func (f *fakeRepo) List(_ context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Expense, int, error) {
	f.lastFilter = filter
	var out []*models.Expense
	for _, e := range f.rows {
		if e.TenantID != tenantID {
			continue
		}
		if filter.Status != "" && e.Status != filter.Status {
			continue
		}
		if filter.SubmittedBy != nil && (e.SubmittedBy == nil || *e.SubmittedBy != *filter.SubmittedBy) {
			continue
		}
		copied := *e
		out = append(out, &copied)
	}
	return out, len(out), nil
}

func validInput() CreateInput {
	return CreateInput{
		Description: "Bürobedarf Q2",
		Amount:      "234.50",
		Date:        "2026-05-29",
		Category:    "Büromaterial",
		Supplier:    "Staples",
		Account:     "4930",
	}
}

func TestCreate_StartsPendingAndRecordsSubmitter(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeRepo())
	tenantID, actorID := uuid.New(), uuid.New()

	e, err := svc.Create(context.Background(), tenantID, actorID, validInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.Status != models.ExpenseStatusPending {
		t.Errorf("status = %q, want pending", e.Status)
	}
	if e.SubmittedBy == nil || *e.SubmittedBy != actorID {
		t.Errorf("submitted_by = %v, want %v", e.SubmittedBy, actorID)
	}
	if !e.Amount.Equal(decimal.RequireFromString("234.50")) {
		t.Errorf("amount = %s, want 234.50", e.Amount)
	}
	if e.Account == nil || *e.Account != "4930" {
		t.Errorf("account = %v, want 4930", e.Account)
	}
}

func TestCreate_RejectsBadInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mutate  func(*CreateInput)
		wantErr error
	}{
		{"empty description", func(in *CreateInput) { in.Description = "   " }, ErrDescriptionRequired},
		{"zero amount", func(in *CreateInput) { in.Amount = "0" }, ErrInvalidAmount},
		{"negative amount", func(in *CreateInput) { in.Amount = "-12.00" }, ErrInvalidAmount},
		{"unparsable amount", func(in *CreateInput) { in.Amount = "ein Euro" }, ErrInvalidAmount},
		{"empty date", func(in *CreateInput) { in.Date = "" }, ErrInvalidDate},
		{"date with time", func(in *CreateInput) { in.Date = "2026-05-29T10:00:00Z" }, ErrInvalidDate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewService(newFakeRepo())
			in := validInput()
			tt.mutate(&in)
			if _, err := svc.Create(context.Background(), uuid.New(), uuid.New(), in); !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreate_RequiresActor(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeRepo())
	if _, err := svc.Create(context.Background(), uuid.New(), uuid.Nil, validInput()); !errors.Is(err, ErrMissingActor) {
		t.Errorf("err = %v, want ErrMissingActor", err)
	}
}

// The four-eyes rule is the reason this module exists as more than a table.
func TestApprove_RefusesTheSubmitter(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeRepo())
	tenantID, submitter := uuid.New(), uuid.New()

	e, err := svc.Create(context.Background(), tenantID, submitter, validInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := svc.Approve(context.Background(), tenantID, e.ID, submitter); !errors.Is(err, ErrSelfApproval) {
		t.Fatalf("self approve err = %v, want ErrSelfApproval", err)
	}
	if _, err := svc.Reject(context.Background(), tenantID, e.ID, submitter); !errors.Is(err, ErrSelfApproval) {
		t.Fatalf("self reject err = %v, want ErrSelfApproval", err)
	}

	approver := uuid.New()
	decided, err := svc.Approve(context.Background(), tenantID, e.ID, approver)
	if err != nil {
		t.Fatalf("Approve by someone else: %v", err)
	}
	if decided.Status != models.ExpenseStatusApproved {
		t.Errorf("status = %q, want approved", decided.Status)
	}
	if decided.DecidedBy == nil || *decided.DecidedBy != approver {
		t.Errorf("decided_by = %v, want %v", decided.DecidedBy, approver)
	}
	if decided.DecidedAt == nil {
		t.Error("decided_at is nil after a decision")
	}
}

func TestDecide_OnlyFromPending(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeRepo())
	tenantID, submitter, approver := uuid.New(), uuid.New(), uuid.New()

	e, err := svc.Create(context.Background(), tenantID, submitter, validInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Reject(context.Background(), tenantID, e.ID, approver); err != nil {
		t.Fatalf("Reject: %v", err)
	}

	// A second decision would silently overwrite the first one.
	if _, err := svc.Approve(context.Background(), tenantID, e.ID, approver); !errors.Is(err, ErrNotPending) {
		t.Errorf("re-decide err = %v, want ErrNotPending", err)
	}
}

func TestDecide_ForeignTenantIsNotFound(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeRepo())
	tenantID, submitter := uuid.New(), uuid.New()

	e, err := svc.Create(context.Background(), tenantID, submitter, validInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Approve(context.Background(), uuid.New(), e.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestUpdate_DecidedExpenseKeepsBookkeepingFieldsOpen(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeRepo())
	tenantID, submitter, approver := uuid.New(), uuid.New(), uuid.New()

	in := validInput()
	in.Account = ""
	e, err := svc.Create(context.Background(), tenantID, submitter, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Approve(context.Background(), tenantID, e.ID, approver); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	// Kontierung after the fact is the normal case.
	account := "4930"
	updated, err := svc.Update(context.Background(), tenantID, e.ID, UpdateInput{Account: &account})
	if err != nil {
		t.Fatalf("Kontierung after approval: %v", err)
	}
	if updated.Account == nil || *updated.Account != account {
		t.Errorf("account = %v, want %q", updated.Account, account)
	}

	// The amount is what the approval was about.
	amount := "9999.00"
	if _, err := svc.Update(context.Background(), tenantID, e.ID, UpdateInput{Amount: &amount}); !errors.Is(err, ErrDecided) {
		t.Errorf("amount edit err = %v, want ErrDecided", err)
	}
}

func TestUpdate_OmittedFieldsSurvive(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeRepo())
	tenantID, submitter := uuid.New(), uuid.New()

	e, err := svc.Create(context.Background(), tenantID, submitter, validInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	category := "IT-Infrastruktur"
	updated, err := svc.Update(context.Background(), tenantID, e.ID, UpdateInput{Category: &category})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Description != e.Description {
		t.Errorf("description = %q, want %q — an omitted field was cleared", updated.Description, e.Description)
	}
	if !updated.Amount.Equal(e.Amount) {
		t.Errorf("amount = %s, want %s", updated.Amount, e.Amount)
	}
	if updated.Category != category {
		t.Errorf("category = %q, want %q", updated.Category, category)
	}
}

func TestAttachReceipt_WorksAfterADecision(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeRepo())
	tenantID, submitter, approver := uuid.New(), uuid.New(), uuid.New()

	e, err := svc.Create(context.Background(), tenantID, submitter, validInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Approve(context.Background(), tenantID, e.ID, approver); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	updated, err := svc.AttachReceipt(context.Background(), tenantID, e.ID, "staples-q2.pdf")
	if err != nil {
		t.Fatalf("AttachReceipt: %v", err)
	}
	if !updated.HasReceipt() {
		t.Error("HasReceipt() = false after attaching a filename")
	}
	if updated.ReceiptName == nil || *updated.ReceiptName != "staples-q2.pdf" {
		t.Errorf("receipt_name = %v, want staples-q2.pdf", updated.ReceiptName)
	}
}

func TestDelete_OnlyWhilePending(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := NewService(repo)
	tenantID, submitter, approver := uuid.New(), uuid.New(), uuid.New()

	pending, err := svc.Create(context.Background(), tenantID, submitter, validInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(context.Background(), tenantID, pending.ID); err != nil {
		t.Fatalf("Delete pending: %v", err)
	}

	decided, err := svc.Create(context.Background(), tenantID, submitter, validInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Approve(context.Background(), tenantID, decided.ID, approver); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if err := svc.Delete(context.Background(), tenantID, decided.ID); !errors.Is(err, ErrDecided) {
		t.Errorf("delete decided err = %v, want ErrDecided", err)
	}
	if repo.deletes != 1 {
		t.Errorf("repo deletes = %d, want 1", repo.deletes)
	}
}

func TestList_OwnScopeNarrowsToTheSubmitter(t *testing.T) {
	t.Parallel()
	svc := NewService(newFakeRepo())
	tenantID, mine, theirs := uuid.New(), uuid.New(), uuid.New()

	if _, err := svc.Create(context.Background(), tenantID, mine, validInput()); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Create(context.Background(), tenantID, theirs, validInput()); err != nil {
		t.Fatalf("Create: %v", err)
	}

	all, total, err := svc.List(context.Background(), tenantID, ListInput{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 || total != 2 {
		t.Fatalf("unfiltered list = %d rows / total %d, want 2 / 2", len(all), total)
	}

	own, ownTotal, err := svc.List(context.Background(), tenantID, ListInput{SubmittedBy: &mine})
	if err != nil {
		t.Fatalf("List own: %v", err)
	}
	// The total has to shrink with the rows, or paging describes a set the
	// caller cannot see.
	if len(own) != 1 || ownTotal != 1 {
		t.Fatalf("own list = %d rows / total %d, want 1 / 1", len(own), ownTotal)
	}
	if own[0].SubmittedBy == nil || *own[0].SubmittedBy != mine {
		t.Errorf("own list returned an expense of %v", own[0].SubmittedBy)
	}
}

func TestList_PagingIsBounded(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		in                    ListInput
		wantLimit, wantOffset int
	}{
		// A caller asking for the whole table at once must not get it.
		{"oversized page size falls back", ListInput{Page: 1, PageSize: 100000}, 50, 0},
		{"unset paging gets defaults", ListInput{}, 50, 0},
		{"page zero is page one", ListInput{Page: 0, PageSize: 20}, 20, 0},
		{"offset follows the page", ListInput{Page: 3, PageSize: 20}, 20, 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newFakeRepo()
			svc := NewService(repo)
			if _, _, err := svc.List(context.Background(), uuid.New(), tt.in); err != nil {
				t.Fatalf("List: %v", err)
			}
			if repo.lastFilter.Limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", repo.lastFilter.Limit, tt.wantLimit)
			}
			if repo.lastFilter.Offset != tt.wantOffset {
				t.Errorf("offset = %d, want %d", repo.lastFilter.Offset, tt.wantOffset)
			}
		})
	}
}
