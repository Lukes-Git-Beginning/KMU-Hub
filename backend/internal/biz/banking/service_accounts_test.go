package banking

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func newAccountService(repo *fakeRepo) *Service {
	return NewService(repo, &fakeOpenItems{}, &fakePayments{}, nil)
}

func TestCreateAccount_NormalisesAndValidates(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()

	tests := []struct {
		name     string
		in       CreateAccountInput
		wantErr  error
		wantIBAN string
		wantCur  string
		wantBIC  string
	}{
		{
			name: "a grouped IBAN is stored compact and upper-case",
			in: CreateAccountInput{
				TenantID: tenant,
				BankName: "Commerzbank",
				IBAN:     "de89 3704 0044 0532 0130 00",
				BIC:      "cobadeffxxx",
			},
			wantIBAN: "DE89370400440532013000",
			wantBIC:  "COBADEFFXXX",
			// Nothing was sent, so it defaults rather than storing an empty
			// currency the frontend would render as a bare number.
			wantCur: "EUR",
		},
		{
			name: "a currency is upper-cased",
			in: CreateAccountInput{
				TenantID: tenant, BankName: "PostFinance",
				IBAN: "CH9300762011623852957", Currency: "chf",
			},
			wantIBAN: "CH9300762011623852957",
			wantCur:  "CHF",
		},
		{
			// One transposed digit. Stored uncorrected it would never match an
			// imported statement, and nothing would say why.
			name: "a failed check digit is refused",
			in: CreateAccountInput{
				TenantID: tenant, BankName: "Commerzbank",
				IBAN: "DE89 3704 0044 0532 0130 01",
			},
			wantErr: ErrInvalidIBAN,
		},
		{
			name: "a bank name is required",
			in: CreateAccountInput{
				TenantID: tenant, BankName: "   ",
				IBAN: "DE89370400440532013000",
			},
			wantErr: ErrInvalidAccount,
		},
		{
			name: "a malformed BIC is refused",
			in: CreateAccountInput{
				TenantID: tenant, BankName: "Commerzbank",
				IBAN: "DE89370400440532013000", BIC: "NOPE",
			},
			wantErr: ErrInvalidAccount,
		},
		{
			name: "a two-letter currency is refused",
			in: CreateAccountInput{
				TenantID: tenant, BankName: "Commerzbank",
				IBAN: "DE89370400440532013000", Currency: "EU",
			},
			wantErr: ErrInvalidAccount,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := newAccountService(newFakeRepo())

			acc, err := svc.CreateAccount(context.Background(), tc.in)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateAccount: %v", err)
			}
			if acc.IBAN != tc.wantIBAN {
				t.Errorf("iban = %q, want %q", acc.IBAN, tc.wantIBAN)
			}
			if acc.Currency != tc.wantCur {
				t.Errorf("currency = %q, want %q", acc.Currency, tc.wantCur)
			}
			if tc.wantBIC != "" && acc.BIC != tc.wantBIC {
				t.Errorf("bic = %q, want %q", acc.BIC, tc.wantBIC)
			}
			if acc.Connected {
				t.Error("a new account must start disconnected")
			}
			if !acc.Balance.IsZero() || acc.LastSync != nil {
				t.Errorf("a new account reported balance %s / lastSync %v, want 0 / nil",
					acc.Balance, acc.LastSync)
			}
		})
	}
}

// The grouped and the compact spelling of one IBAN must collide: if both could
// be stored the same account would exist twice and its balance would split
// across the copies.
func TestCreateAccount_SameIBANDifferentSpellingCollides(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	svc := newAccountService(newFakeRepo())
	ctx := context.Background()

	if _, err := svc.CreateAccount(ctx, CreateAccountInput{
		TenantID: tenant, BankName: "Commerzbank", IBAN: "DE89370400440532013000",
	}); err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err := svc.CreateAccount(ctx, CreateAccountInput{
		TenantID: tenant, BankName: "Commerzbank (2)", IBAN: "DE89 3704 0044 0532 0130 00",
	})
	if !errors.Is(err, ErrAccountExists) {
		t.Fatalf("err = %v, want ErrAccountExists", err)
	}
}

// The same IBAN under two tenants is two accounts, not a conflict — the unique
// constraint is per tenant.
func TestCreateAccount_SameIBANAcrossTenantsIsAllowed(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := newAccountService(repo)
	ctx := context.Background()

	for _, tenant := range []uuid.UUID{uuid.New(), uuid.New()} {
		if _, err := svc.CreateAccount(ctx, CreateAccountInput{
			TenantID: tenant, BankName: "Commerzbank", IBAN: "DE89370400440532013000",
		}); err != nil {
			t.Fatalf("create for %s: %v", tenant, err)
		}
	}
	if len(repo.accounts) != 2 {
		t.Fatalf("stored %d accounts, want 2", len(repo.accounts))
	}
}

func TestConnectAccount(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	svc := newAccountService(newFakeRepo())
	ctx := context.Background()

	acc, err := svc.CreateAccount(ctx, CreateAccountInput{
		TenantID: tenant, BankName: "Sparkasse München", IBAN: "DE89370400440532013000",
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	connected, err := svc.ConnectAccount(ctx, tenant, acc.ID)
	if err != nil {
		t.Fatalf("ConnectAccount: %v", err)
	}
	if !connected.Connected || connected.ConnectedAt == nil {
		t.Fatalf("connected = %v, connectedAt = %v, want true / set",
			connected.Connected, connected.ConnectedAt)
	}

	// A retried request must not move the timestamp: the desktop client fires
	// connect from a dialog that can be reopened.
	again, err := svc.ConnectAccount(ctx, tenant, acc.ID)
	if err != nil {
		t.Fatalf("second ConnectAccount: %v", err)
	}
	if !again.ConnectedAt.Equal(*connected.ConnectedAt) {
		t.Errorf("connectedAt moved from %v to %v on a repeated connect",
			connected.ConnectedAt, again.ConnectedAt)
	}

	if _, err := svc.ConnectAccount(ctx, uuid.New(), acc.ID); !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("connect across tenants: err = %v, want ErrAccountNotFound", err)
	}
}

// Disconnecting clears the timestamp: a stale connected_at next to
// connected=false would read as "was linked all along".
func TestUpdateAccount_DisconnectClearsTimestamp(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	svc := newAccountService(newFakeRepo())
	ctx := context.Background()

	acc, err := svc.CreateAccount(ctx, CreateAccountInput{
		TenantID: tenant, BankName: "ING", IBAN: "DE89370400440532013000",
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if _, err := svc.ConnectAccount(ctx, tenant, acc.ID); err != nil {
		t.Fatalf("ConnectAccount: %v", err)
	}

	off := false
	updated, err := svc.UpdateAccount(ctx, UpdateAccountInput{
		TenantID: tenant, ID: acc.ID, Connected: &off,
	})
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	if updated.Connected || updated.ConnectedAt != nil {
		t.Errorf("connected = %v, connectedAt = %v, want false / nil",
			updated.Connected, updated.ConnectedAt)
	}
}

// A partial edit leaves the untouched fields alone. Sending only the BIC must
// not blank the bank name — the frontend's edit form posts what changed.
func TestUpdateAccount_PartialLeavesTheRestAlone(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	svc := newAccountService(newFakeRepo())
	ctx := context.Background()

	acc, err := svc.CreateAccount(ctx, CreateAccountInput{
		TenantID: tenant, BankName: "DKB", IBAN: "DE89370400440532013000", Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	bic := "BYLADEM1001"
	updated, err := svc.UpdateAccount(ctx, UpdateAccountInput{
		TenantID: tenant, ID: acc.ID, BIC: &bic,
	})
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	if updated.BankName != "DKB" || updated.IBAN != "DE89370400440532013000" || updated.Currency != "EUR" {
		t.Errorf("a BIC-only edit changed the rest: %+v", updated)
	}
	if updated.BIC != bic {
		t.Errorf("bic = %q, want %q", updated.BIC, bic)
	}
}

func TestUpdateAccount_RejectsAnInvalidIBAN(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	svc := newAccountService(newFakeRepo())
	ctx := context.Background()

	acc, err := svc.CreateAccount(ctx, CreateAccountInput{
		TenantID: tenant, BankName: "DKB", IBAN: "DE89370400440532013000",
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	bad := "DE89370400440532013001"
	if _, err := svc.UpdateAccount(ctx, UpdateAccountInput{
		TenantID: tenant, ID: acc.ID, IBAN: &bad,
	}); !errors.Is(err, ErrInvalidIBAN) {
		t.Fatalf("err = %v, want ErrInvalidIBAN", err)
	}
}

func TestDeleteAccount_StaysInsideTheTenant(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	repo := newFakeRepo()
	svc := newAccountService(repo)
	ctx := context.Background()

	acc, err := svc.CreateAccount(ctx, CreateAccountInput{
		TenantID: tenant, BankName: "N26", IBAN: "DE89370400440532013000",
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	if err := svc.DeleteAccount(ctx, uuid.New(), acc.ID); !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("delete across tenants: err = %v, want ErrAccountNotFound", err)
	}
	if _, err := svc.GetAccount(ctx, tenant, acc.ID); err != nil {
		t.Errorf("the row went missing after a cross-tenant delete: %v", err)
	}
	if err := svc.DeleteAccount(ctx, tenant, acc.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if _, err := svc.GetAccount(ctx, tenant, acc.ID); !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("after delete: err = %v, want ErrAccountNotFound", err)
	}
}
