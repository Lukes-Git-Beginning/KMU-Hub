package banking

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/payment"
	"github.com/kmuhub/kmuhub/internal/models"
)

// fakeRepo is an in-memory Repository. It keeps the unique-per-hash promise of
// the real schema so the idempotency test exercises the same rule.
type fakeRepo struct {
	statements map[string]*models.BankStatement // keyed by content hash
	txs        map[uuid.UUID][]*models.BankTransaction
	createErr  error
	updateErr  error
	creates    int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		statements: map[string]*models.BankStatement{},
		txs:        map[uuid.UUID][]*models.BankTransaction{},
	}
}

func (f *fakeRepo) GetStatementByHash(_ context.Context, _ uuid.UUID, hash string) (*models.BankStatement, error) {
	if s, ok := f.statements[hash]; ok {
		return s, nil
	}
	return nil, ErrStatementNotFound
}

func (f *fakeRepo) CreateStatement(_ context.Context, stmt *models.BankStatement, txs []*models.BankTransaction) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.creates++
	f.statements[stmt.ContentHash] = stmt
	f.txs[stmt.ID] = txs
	return nil
}

func (f *fakeRepo) GetStatement(_ context.Context, _, id uuid.UUID) (*models.BankStatement, error) {
	for _, s := range f.statements {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, ErrStatementNotFound
}

func (f *fakeRepo) ListStatements(context.Context, uuid.UUID, int, int) ([]*models.BankStatement, int, error) {
	return nil, 0, nil
}

func (f *fakeRepo) GetTransaction(_ context.Context, _, id uuid.UUID) (*models.BankTransaction, error) {
	for _, list := range f.txs {
		for _, t := range list {
			if t.ID == id {
				return t, nil
			}
		}
	}
	return nil, ErrTransactionNotFound
}

func (f *fakeRepo) ListTransactions(context.Context, uuid.UUID, models.BankTransactionFilter) ([]*models.BankTransaction, int, error) {
	return nil, 0, nil
}

func (f *fakeRepo) ListTransactionsByStatement(_ context.Context, _, statementID uuid.UUID) ([]*models.BankTransaction, error) {
	return f.txs[statementID], nil
}

func (f *fakeRepo) UpdateTransactionMatch(_ context.Context, _ *models.BankTransaction) error {
	return f.updateErr
}

type fakeOpenItems struct {
	items []*models.OpenItem
	err   error
}

func (f *fakeOpenItems) ListOpenItems(context.Context, uuid.UUID, models.OpenItemFilter) ([]*models.OpenItem, int, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.items, len(f.items), nil
}

type fakePayments struct {
	calls []payment.RecordInput
	err   error
}

func (f *fakePayments) Record(_ context.Context, in payment.RecordInput) (*models.Payment, error) {
	f.calls = append(f.calls, in)
	if f.err != nil {
		return nil, f.err
	}
	return &models.Payment{ID: uuid.New(), InvoiceID: in.InvoiceID, Amount: in.Amount}, nil
}

func TestImportAttachesSuggestions(t *testing.T) {
	repo := newFakeRepo()
	target := openItem("RE-2026-0001", "119.00")
	svc := NewService(repo, &fakeOpenItems{items: []*models.OpenItem{target}}, &fakePayments{}, nil)

	res, err := svc.Import(context.Background(), ImportInput{
		TenantID: uuid.New(), Filename: "statement.xml", Content: []byte(camtSample),
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if res.AlreadyImported {
		t.Error("first import reported as already imported")
	}
	if len(res.Transactions) != 2 {
		t.Fatalf("transactions = %d, want 2", len(res.Transactions))
	}
	credit := res.Transactions[0]
	if credit.MatchStatus != models.BankMatchSuggested {
		t.Errorf("credit match status = %q, want suggested", credit.MatchStatus)
	}
	if credit.MatchedInvoiceID == nil || *credit.MatchedInvoiceID != target.InvoiceID {
		t.Errorf("matched invoice = %v, want %s", credit.MatchedInvoiceID, target.InvoiceID)
	}
	// The debit stays untouched; nothing about it settles a receivable.
	if res.Transactions[1].MatchStatus != models.BankMatchUnmatched {
		t.Errorf("debit match status = %q, want unmatched", res.Transactions[1].MatchStatus)
	}
	if res.Statement.TransactionCount != 2 {
		t.Errorf("transaction count = %d", res.Statement.TransactionCount)
	}
}

func TestImportIsIdempotent(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeOpenItems{}, &fakePayments{}, nil)
	tenantID := uuid.New()
	input := ImportInput{TenantID: tenantID, Filename: "statement.sta", Content: []byte(mt940Sample)}

	first, err := svc.Import(context.Background(), input)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	second, err := svc.Import(context.Background(), input)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}

	if !second.AlreadyImported {
		t.Error("second import of the same file must report already imported")
	}
	if second.Statement.ID != first.Statement.ID {
		t.Errorf("statement id = %s, want the first import %s", second.Statement.ID, first.Statement.ID)
	}
	if repo.creates != 1 {
		t.Fatalf("creates = %d, want exactly one — a re-import must not duplicate transactions", repo.creates)
	}
}

func TestImportSurvivesUnavailableOpenItems(t *testing.T) {
	// Losing the receivables costs suggestions, not the import: the entries are
	// still worth storing and can be reconciled by hand.
	repo := newFakeRepo()
	svc := NewService(repo, &fakeOpenItems{err: errors.New("database down")}, &fakePayments{}, nil)

	res, err := svc.Import(context.Background(), ImportInput{TenantID: uuid.New(), Content: []byte(camtSample)})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(res.Transactions) != 2 {
		t.Fatalf("transactions = %d, want the entries stored anyway", len(res.Transactions))
	}
	if res.Transactions[0].MatchStatus != models.BankMatchUnmatched {
		t.Errorf("match status = %q, want unmatched without open items", res.Transactions[0].MatchStatus)
	}
}

func TestImportRejectsMalformedFile(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeOpenItems{}, &fakePayments{}, nil)

	_, err := svc.Import(context.Background(), ImportInput{TenantID: uuid.New(), Content: []byte("not a statement")})
	if !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("err = %v, want ErrUnknownFormat", err)
	}
	if repo.creates != 0 {
		t.Errorf("creates = %d, want nothing written for a rejected file", repo.creates)
	}
}

// reconcileFixture imports the CAMT sample and returns the service, the payment
// recorder and the credit transaction that came out of it.
func reconcileFixture(t *testing.T) (*Service, *fakePayments, *models.BankTransaction, uuid.UUID) {
	t.Helper()
	repo := newFakeRepo()
	payments := &fakePayments{}
	target := openItem("RE-2026-0001", "119.00")
	svc := NewService(repo, &fakeOpenItems{items: []*models.OpenItem{target}}, payments, nil)
	tenantID := uuid.New()

	res, err := svc.Import(context.Background(), ImportInput{TenantID: tenantID, Content: []byte(camtSample)})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	return svc, payments, res.Transactions[0], tenantID
}

func TestReconcileBooksTheSuggestedInvoice(t *testing.T) {
	svc, payments, credit, tenantID := reconcileFixture(t)
	userID := uuid.New()

	got, err := svc.Reconcile(context.Background(), ReconcileInput{
		TenantID: tenantID, TransactionID: credit.ID, UserID: userID,
	})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	if len(payments.calls) != 1 {
		t.Fatalf("payment calls = %d, want 1", len(payments.calls))
	}
	call := payments.calls[0]
	if call.InvoiceID != *credit.MatchedInvoiceID {
		t.Errorf("booked against %s, want the suggested %s", call.InvoiceID, *credit.MatchedInvoiceID)
	}
	if !call.Amount.Equal(decimal.RequireFromString("119.00")) {
		t.Errorf("amount = %s, want 119.00", call.Amount)
	}
	if call.Date != credit.ValueDate {
		t.Errorf("payment date = %v, want the value date %v", call.Date, credit.ValueDate)
	}
	// The idempotency key is what stops a retried confirmation from booking a
	// second payment.
	if call.IdempotencyKey != "bank-tx:"+credit.ID.String() {
		t.Errorf("idempotency key = %q", call.IdempotencyKey)
	}
	if got.MatchStatus != models.BankMatchMatched {
		t.Errorf("match status = %q, want matched", got.MatchStatus)
	}
	if got.PaymentID == nil {
		t.Error("payment id not carried back onto the transaction")
	}
	if got.ReconciledBy == nil || *got.ReconciledBy != userID {
		t.Errorf("reconciled by = %v, want %s", got.ReconciledBy, userID)
	}
}

func TestReconcileRefusesRepeat(t *testing.T) {
	svc, _, credit, tenantID := reconcileFixture(t)
	if _, err := svc.Reconcile(context.Background(), ReconcileInput{TenantID: tenantID, TransactionID: credit.ID}); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}

	_, err := svc.Reconcile(context.Background(), ReconcileInput{TenantID: tenantID, TransactionID: credit.ID})
	if !errors.Is(err, ErrAlreadyReconciled) {
		t.Fatalf("err = %v, want ErrAlreadyReconciled", err)
	}
}

func TestReconcileRefusesDebit(t *testing.T) {
	repo := newFakeRepo()
	payments := &fakePayments{}
	svc := NewService(repo, &fakeOpenItems{}, payments, nil)
	tenantID := uuid.New()
	res, err := svc.Import(context.Background(), ImportInput{TenantID: tenantID, Content: []byte(camtSample)})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	debit := res.Transactions[1]

	_, err = svc.Reconcile(context.Background(), ReconcileInput{
		TenantID: tenantID, TransactionID: debit.ID, InvoiceID: uuid.New(),
	})
	if !errors.Is(err, ErrNotACredit) {
		t.Fatalf("err = %v, want ErrNotACredit", err)
	}
	if len(payments.calls) != 0 {
		t.Errorf("payment calls = %d, want none for a debit", len(payments.calls))
	}
}

func TestReconcileDoesNotMarkMatchedWhenPaymentFails(t *testing.T) {
	svc, payments, credit, tenantID := reconcileFixture(t)
	payments.err = errors.New("invoice already paid")

	if _, err := svc.Reconcile(context.Background(), ReconcileInput{TenantID: tenantID, TransactionID: credit.ID}); err == nil {
		t.Fatal("expected the failed payment to fail the reconcile")
	}
	if credit.MatchStatus == models.BankMatchMatched {
		t.Error("transaction marked matched although no payment was recorded")
	}
}

func TestIgnoreTakesEntryOutOfTheQueue(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, &fakeOpenItems{}, &fakePayments{}, nil)
	tenantID := uuid.New()
	res, err := svc.Import(context.Background(), ImportInput{TenantID: tenantID, Content: []byte(camtSample)})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	debit := res.Transactions[1]

	got, err := svc.Ignore(context.Background(), tenantID, debit.ID, uuid.New())
	if err != nil {
		t.Fatalf("Ignore: %v", err)
	}
	if got.MatchStatus != models.BankMatchIgnored {
		t.Errorf("match status = %q, want ignored", got.MatchStatus)
	}
	if got.MatchedInvoiceID != nil {
		t.Error("an ignored entry must not keep an invoice suggestion")
	}
}

func TestIgnoreRefusesBookedEntry(t *testing.T) {
	svc, _, credit, tenantID := reconcileFixture(t)
	if _, err := svc.Reconcile(context.Background(), ReconcileInput{TenantID: tenantID, TransactionID: credit.ID}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	// Undoing a booked entry means deleting its payment — the payment service's
	// decision, not this one's.
	if _, err := svc.Ignore(context.Background(), tenantID, credit.ID, uuid.New()); !errors.Is(err, ErrAlreadyReconciled) {
		t.Fatalf("err = %v, want ErrAlreadyReconciled", err)
	}
}
