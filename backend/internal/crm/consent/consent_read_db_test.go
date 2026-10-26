package consent

// Read-path coverage against the app-role database: the history/latest/last
// lookups, the IP address round trip through scanConsentRecord(Row) and the
// consent-gate repository (ActiveConsent, ContactHasEmail). The existing DB
// tests only exercise the write side and RLS visibility.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

type consentFixture struct {
	pool                 *pgxpool.Pool
	tenantOwn, tenantOth uuid.UUID
	ctxOwn, ctxOther     context.Context
	contactID, userID    uuid.UUID
	repo                 *PostgresRepository
	assertRepo           *PostgresAssertRepo
	base                 time.Time
}

func newConsentFixture(t *testing.T) *consentFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	f := &consentFixture{
		pool: pool, tenantOwn: uuid.New(), tenantOth: uuid.New(),
		base: time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second),
	}
	testutil.EnsureTenant(t, pool, f.tenantOwn, "Consent Read Own")
	testutil.EnsureTenant(t, pool, f.tenantOth, "Consent Read Other")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", f.tenantOth) })
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", f.tenantOwn) })

	f.userID = testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": f.tenantOwn, "email": fmt.Sprintf("consent-read-%s@example.test", uuid.New().String()[:8]), "password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", f.userID) })
	f.contactID = f.contact(t, "Own", fmt.Sprintf("read-own-%s@example.test", uuid.New().String()[:8]))

	f.ctxOwn = testutil.WithTenantCtx(context.Background(), f.tenantOwn)
	f.ctxOther = testutil.WithTenantCtx(context.Background(), f.tenantOth)
	f.repo = NewPostgresRepository(pool)
	f.assertRepo = NewPostgresAssertRepo(pool)
	return f
}

// contact seeds a contact in the fixture's own tenant; email nil leaves the
// column NULL.
func (f *consentFixture) contact(t *testing.T, last string, email any) uuid.UUID {
	t.Helper()
	cols := map[string]any{"tenant_id": f.tenantOwn, "first_name": "Read", "last_name": last, "created_by": f.userID}
	if email != nil {
		cols["email"] = email
	}
	id := testutil.SeedRow(t, f.pool, "contacts", cols)
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "contacts", id) })
	return id
}

// add inserts a consent record for the fixture's own tenant, created `age`
// after the fixture base time so the ordering is deterministic.
func (f *consentFixture) add(t *testing.T, contact uuid.UUID, ctype string, granted bool, age time.Duration, mutate func(*ConsentRecord)) *ConsentRecord {
	t.Helper()
	at := f.base.Add(age)
	rec := &ConsentRecord{
		ID: uuid.New(), TenantID: f.tenantOwn, ContactID: contact, ConsentType: ctype,
		Granted: granted, LegalBasis: "consent", CreatedAt: at,
	}
	if granted {
		rec.GrantedAt = &at
	}
	if mutate != nil {
		mutate(rec)
	}
	if err := f.repo.CreateConsentRecord(f.ctxOwn, rec); err != nil {
		t.Fatalf("CreateConsentRecord(%s): %v", ctype, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "consent_records", rec.ID) })
	return rec
}

func recordIDs(recs []*ConsentRecord) []uuid.UUID {
	ids := make([]uuid.UUID, len(recs))
	for i, r := range recs {
		ids[i] = r.ID
	}
	return ids
}

func TestConsentRead_HistoryAndLatest(t *testing.T) {
	f := newConsentFixture(t)

	ip := "203.0.113.7"
	userID := f.userID
	first := f.add(t, f.contactID, "marketing_email", true, 0, func(r *ConsentRecord) {
		r.Source, r.Notes, r.IPAddress, r.CreatedBy = "web_form", "double opt-in", &ip, &userID
	})
	revoked := f.add(t, f.contactID, "marketing_email", false, time.Hour, func(r *ConsentRecord) {
		rv := f.base.Add(time.Hour)
		r.RevokedAt = &rv
	})
	third := f.add(t, f.contactID, "marketing_email", true, 2*time.Hour, nil)
	phone := f.add(t, f.contactID, "marketing_phone", false, 30*time.Minute, nil)
	f.add(t, f.contactID, "newsletter", true, 90*time.Minute, nil)
	profiling := f.add(t, f.contactID, "profiling", true, 100*time.Minute, func(r *ConsentRecord) {
		r.IPAddress = &ip
	})

	t.Run("history is newest first and scoped to the type", func(t *testing.T) {
		got, err := f.repo.GetConsentHistory(f.ctxOwn, f.tenantOwn, f.contactID, "marketing_email")
		if err != nil {
			t.Fatalf("GetConsentHistory: %v", err)
		}
		want := []uuid.UUID{third.ID, revoked.ID, first.ID}
		ids := recordIDs(got)
		if len(ids) != len(want) {
			t.Fatalf("history = %v, want %v", ids, want)
		}
		for i := range want {
			if ids[i] != want[i] {
				t.Fatalf("history[%d] = %s, want %s (order must be created_at DESC)", i, ids[i], want[i])
			}
		}
	})

	t.Run("all columns survive the scan", func(t *testing.T) {
		got, err := f.repo.GetConsentHistory(f.ctxOwn, f.tenantOwn, f.contactID, "marketing_email")
		if err != nil {
			t.Fatalf("GetConsentHistory: %v", err)
		}
		rec := got[2]
		if rec.ID != first.ID || rec.ContactID != f.contactID || rec.ConsentType != "marketing_email" || !rec.Granted ||
			rec.LegalBasis != "consent" || rec.Source != "web_form" || rec.Notes != "double opt-in" {
			t.Fatalf("scanned record = %+v", rec)
		}
		// inet::text carries the prefix length.
		if rec.IPAddress == nil || *rec.IPAddress != ip+"/32" {
			t.Fatalf("ip_address = %v, want %s/32", rec.IPAddress, ip)
		}
		if rec.CreatedBy == nil || *rec.CreatedBy != userID {
			t.Fatalf("created_by = %v, want %s", rec.CreatedBy, userID)
		}
		if rec.GrantedAt == nil || !rec.GrantedAt.Equal(first.CreatedAt) || rec.RevokedAt != nil {
			t.Fatalf("granted_at/revoked_at = %v / %v", rec.GrantedAt, rec.RevokedAt)
		}
		if got[1].RevokedAt == nil || got[1].Granted {
			t.Fatalf("revocation not scanned: %+v", got[1])
		}
		if got[0].IPAddress != nil {
			t.Fatalf("NULL ip_address must stay nil, got %v", *got[0].IPAddress)
		}
	})

	t.Run("latest keeps one record per type", func(t *testing.T) {
		got, err := f.repo.GetLatestConsents(f.ctxOwn, f.tenantOwn, f.contactID)
		if err != nil {
			t.Fatalf("GetLatestConsents: %v", err)
		}
		if len(got) != 4 {
			t.Fatalf("latest = %d records, want 4 (email, phone, newsletter, profiling)", len(got))
		}
		byType := map[string]uuid.UUID{}
		for _, r := range got {
			byType[r.ConsentType] = r.ID
		}
		if byType["marketing_email"] != third.ID || byType["marketing_phone"] != phone.ID {
			t.Fatalf("latest per type wrong: %v", byType)
		}
	})

	t.Run("last by type returns the newest or not found", func(t *testing.T) {
		got, err := f.repo.GetLastConsentByType(f.ctxOwn, f.tenantOwn, f.contactID, "marketing_email")
		if err != nil || got.ID != third.ID {
			t.Fatalf("GetLastConsentByType = %v, %v; want %s", got, err, third.ID)
		}
		withIP, err := f.repo.GetLastConsentByType(f.ctxOwn, f.tenantOwn, f.contactID, "profiling")
		if err != nil || withIP.ID != profiling.ID || withIP.IPAddress == nil || *withIP.IPAddress != ip+"/32" {
			t.Fatalf("GetLastConsentByType(profiling) = %+v, %v; want %s with ip %s/32", withIP, err, profiling.ID, ip)
		}
		if _, err := f.repo.GetLastConsentByType(f.ctxOwn, f.tenantOwn, f.contactID, "data_sharing"); err != ErrConsentNotFound {
			t.Fatalf("no record: err = %v, want ErrConsentNotFound", err)
		}
	})

	t.Run("another tenant sees nothing", func(t *testing.T) {
		hist, err := f.repo.GetConsentHistory(f.ctxOther, f.tenantOth, f.contactID, "marketing_email")
		if err != nil || len(hist) != 0 {
			t.Fatalf("history for foreign tenant = %d, %v", len(hist), err)
		}
		latest, err := f.repo.GetLatestConsents(f.ctxOther, f.tenantOth, f.contactID)
		if err != nil || len(latest) != 0 {
			t.Fatalf("latest for foreign tenant = %d, %v", len(latest), err)
		}
		if _, err := f.repo.GetLastConsentByType(f.ctxOther, f.tenantOth, f.contactID, "marketing_email"); err != ErrConsentNotFound {
			t.Fatalf("last for foreign tenant: err = %v, want ErrConsentNotFound", err)
		}
	})

	t.Run("tenant predicate holds without RLS", func(t *testing.T) {
		sys := testutil.WithSystemCtx(context.Background())
		hist, err := f.repo.GetConsentHistory(sys, f.tenantOth, f.contactID, "marketing_email")
		if err != nil || len(hist) != 0 {
			t.Fatalf("history with foreign tenant id under system ctx = %d, %v", len(hist), err)
		}
		latest, err := f.repo.GetLatestConsents(sys, f.tenantOth, f.contactID)
		if err != nil || len(latest) != 0 {
			t.Fatalf("latest with foreign tenant id under system ctx = %d, %v", len(latest), err)
		}
		if _, err := f.repo.GetLastConsentByType(sys, f.tenantOth, f.contactID, "marketing_email"); err != ErrConsentNotFound {
			t.Fatalf("last with foreign tenant id under system ctx: err = %v", err)
		}
	})

	t.Run("a cancelled context surfaces as an error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(f.ctxOwn)
		cancel()
		if _, err := f.repo.GetConsentHistory(ctx, f.tenantOwn, f.contactID, "marketing_email"); err == nil {
			t.Fatal("GetConsentHistory: want error")
		}
		if _, err := f.repo.GetLatestConsents(ctx, f.tenantOwn, f.contactID); err == nil {
			t.Fatal("GetLatestConsents: want error")
		}
		if _, err := f.repo.GetLastConsentByType(ctx, f.tenantOwn, f.contactID, "marketing_email"); err == nil || err == ErrConsentNotFound {
			t.Fatalf("GetLastConsentByType: want a real error, got %v", err)
		}
	})
}

func TestConsentRead_AssertRepoActiveConsent(t *testing.T) {
	f := newConsentFixture(t)

	// Email: granted, then revoked as the newest record -> inactive.
	f.add(t, f.contactID, "marketing_email", true, 0, nil)
	f.add(t, f.contactID, "marketing_email", false, time.Hour, func(r *ConsentRecord) {
		rv := f.base.Add(time.Hour)
		r.RevokedAt = &rv
	})
	// Phone: granted, then a non-granted record -> inactive.
	f.add(t, f.contactID, "marketing_phone", true, 0, nil)
	f.add(t, f.contactID, "marketing_phone", false, time.Hour, nil)

	check := func(t *testing.T, ctx context.Context, contact uuid.UUID, ch Channel, want bool) {
		t.Helper()
		got, err := f.assertRepo.ActiveConsent(ctx, contact, ch)
		if err != nil {
			t.Fatalf("ActiveConsent(%s): %v", ch, err)
		}
		if got != want {
			t.Fatalf("ActiveConsent(%s) = %v, want %v", ch, got, want)
		}
	}

	t.Run("newest record is a revocation", func(t *testing.T) {
		check(t, f.ctxOwn, f.contactID, ChannelEmail, false)
		check(t, f.ctxOwn, f.contactID, ChannelPhone, false)
	})

	t.Run("a grant that carries revoked_at is not active", func(t *testing.T) {
		contact := f.contact(t, "Grant", nil)
		f.add(t, contact, "marketing_phone", true, 0, func(r *ConsentRecord) {
			rv := f.base.Add(time.Minute)
			r.RevokedAt = &rv
		})
		check(t, f.ctxOwn, contact, ChannelPhone, false)
	})

	t.Run("a newer grant re-activates", func(t *testing.T) {
		f.add(t, f.contactID, "marketing_email", true, 2*time.Hour, nil)
		check(t, f.ctxOwn, f.contactID, ChannelEmail, true)
		check(t, f.ctxOwn, f.contactID, ChannelPhone, false)
	})

	t.Run("no record means no consent", func(t *testing.T) {
		check(t, f.ctxOwn, uuid.New(), ChannelEmail, false)
	})

	t.Run("RLS hides the consent from another tenant", func(t *testing.T) {
		check(t, f.ctxOther, f.contactID, ChannelEmail, false)
	})

	t.Run("lookup errors fail closed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(f.ctxOwn)
		cancel()
		check(t, ctx, f.contactID, ChannelEmail, false)
	})
}

func TestConsentRead_AssertRepoContactHasEmail(t *testing.T) {
	f := newConsentFixture(t)

	withEmail, emptyEmail, noEmail := f.contactID, f.contact(t, "Empty", ""), f.contact(t, "Null", nil)

	for name, tc := range map[string]struct {
		ctx     context.Context
		contact uuid.UUID
		want    bool
	}{
		"address present":           {f.ctxOwn, withEmail, true},
		"empty string":              {f.ctxOwn, emptyEmail, false},
		"NULL":                      {f.ctxOwn, noEmail, false},
		"unknown contact":           {f.ctxOwn, uuid.New(), false},
		"contact of another tenant": {f.ctxOther, withEmail, false},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := f.assertRepo.ContactHasEmail(tc.ctx, tc.contact)
			if err != nil || got != tc.want {
				t.Fatalf("ContactHasEmail = %v, %v; want %v", got, err, tc.want)
			}
		})
	}

	t.Run("a cancelled context is an error, not a silent false", func(t *testing.T) {
		ctx, cancel := context.WithCancel(f.ctxOwn)
		cancel()
		if got, err := f.assertRepo.ContactHasEmail(ctx, withEmail); err == nil || got {
			t.Fatalf("ContactHasEmail = %v, %v; want false and an error", got, err)
		}
	})
}

func TestConsentRead_AsserterAgainstRealRepo(t *testing.T) {
	f := newConsentFixture(t)
	a := NewAsserter(f.assertRepo, nil)

	if err := a.Assert(f.ctxOwn, f.contactID, ChannelEmail); err != ErrNoConsent {
		t.Fatalf("no consent on record: err = %v, want ErrNoConsent", err)
	}
	f.add(t, f.contactID, "marketing_email", true, 0, nil)
	if err := a.Assert(f.ctxOwn, f.contactID, ChannelEmail); err != nil {
		t.Fatalf("granted consent: %v", err)
	}
	if err := a.Assert(f.ctxOwn, f.contactID, ChannelPhone); err != ErrNoConsent {
		t.Fatalf("phone has no consent: err = %v, want ErrNoConsent", err)
	}
	// An email-less contact needs no email consent.
	noMail := f.contact(t, "NoMail", nil)
	if err := a.Assert(f.ctxOwn, noMail, ChannelEmail); err != nil {
		t.Fatalf("contact without email: %v", err)
	}
}
