package vermietung

// postgres_repository_db_test.go exercises PostgresRepository against the
// real schema, for cov-vermietung-repository-lowest-coverage-in-backend
// (BACKLOG.yml): at 3,9 % this file carried the lowest repository coverage
// in the whole backend, essentially only the constructor. Bug-focused, not a
// coverage exercise: tenant scoping on every read, soft-delete/list
// interaction, and the overlap-conflict mapping added to postgres_repository.go
// alongside these tests (see asRentalConflict).

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func newTestObject(tenantID uuid.UUID, name, category string) *RentalObject {
	now := time.Now().UTC()
	return &RentalObject{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		Category:  category,
		DailyRate: 45.0,
		Deposit:   200.0,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newTestRental(tenantID, objectID uuid.UUID, start, end time.Time) *Rental {
	now := time.Now().UTC()
	return &Rental{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ObjectID:   objectID,
		RenterName: "Mustermann Bau GmbH",
		StartDate:  start,
		EndDate:    end,
		Status:     RentalStatusReserved,
		TotalPrice: 90.0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// ============================================================================
// Objects
// ============================================================================

func TestPostgresObject_CreateGetUpdateSoftDelete(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vermietung Object CRUD")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	obj := newTestObject(tenantID, "Baugeruest 6m", "geruest")
	if err := repo.CreateObject(ctx, obj); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", obj.ID)

	got, err := repo.GetObject(ctx, tenantID, obj.ID)
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if got.Name != obj.Name || got.DailyRate != obj.DailyRate {
		t.Fatalf("GetObject mismatch: got %+v, want name=%s dailyRate=%v", got, obj.Name, obj.DailyRate)
	}

	newLocation := "Lager Nord"
	obj.Name = "Baugeruest 8m"
	obj.Location = &newLocation
	obj.Active = false
	if err := repo.UpdateObject(ctx, obj); err != nil {
		t.Fatalf("UpdateObject: %v", err)
	}
	got, err = repo.GetObject(ctx, tenantID, obj.ID)
	if err != nil {
		t.Fatalf("GetObject after update: %v", err)
	}
	if got.Name != "Baugeruest 8m" || got.Active || got.Location == nil || *got.Location != newLocation {
		t.Fatalf("UpdateObject did not persist changes, got %+v", got)
	}

	if err := repo.SoftDeleteObject(ctx, tenantID, obj.ID); err != nil {
		t.Fatalf("SoftDeleteObject: %v", err)
	}
	if _, err := repo.GetObject(ctx, tenantID, obj.ID); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound for soft-deleted object, got %v", err)
	}
	// Soft delete is not idempotent through the repo: the second call finds
	// zero rows matching `deleted_at IS NULL` and reports not-found, exactly
	// like deleting an object that never existed.
	if err := repo.SoftDeleteObject(ctx, tenantID, obj.ID); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected second SoftDeleteObject to report ErrObjectNotFound, got %v", err)
	}
}

func TestPostgresObject_GetAndUpdate_CrossTenant_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Vermietung Object Cross A")
	testutil.EnsureTenant(t, pool, tenantB, "Vermietung Object Cross B")
	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	obj := newTestObject(tenantA, "Betonmischer", "geraet")
	if err := repo.CreateObject(ctxA, obj); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", obj.ID)

	if _, err := repo.GetObject(ctxB, tenantB, obj.ID); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound reading tenant A's object as tenant B, got %v", err)
	}

	obj.TenantID = tenantB // caller-controlled tenantID in the WHERE clause, not RLS
	obj.Name = "Hijacked"
	if err := repo.UpdateObject(ctxB, obj); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound updating tenant A's object as tenant B, got %v", err)
	}
	if err := repo.SoftDeleteObject(ctxB, tenantB, obj.ID); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound soft-deleting tenant A's object as tenant B, got %v", err)
	}

	// Object must be untouched.
	got, err := repo.GetObject(ctxA, tenantA, obj.ID)
	if err != nil {
		t.Fatalf("GetObject as owning tenant: %v", err)
	}
	if got.Name != "Betonmischer" {
		t.Fatalf("cross-tenant UpdateObject leaked through: got name %q", got.Name)
	}
}

func TestPostgresListObjects_FiltersAndTenantScoping(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Vermietung List A")
	testutil.EnsureTenant(t, pool, tenantB, "Vermietung List B")
	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	active := newTestObject(tenantA, "Kettensaege Stihl", "geraet")
	inactive := newTestObject(tenantA, "Kettensaege Husqvarna", "geraet")
	inactive.Active = false
	otherCategory := newTestObject(tenantA, "Anhaenger", "fahrzeug")
	foreign := newTestObject(tenantB, "Kettensaege Fremd", "geraet")

	for _, o := range []*RentalObject{active, inactive, otherCategory, foreign} {
		ctx := ctxA
		if o.TenantID == tenantB {
			ctx = ctxB
		}
		if err := repo.CreateObject(ctx, o); err != nil {
			t.Fatalf("CreateObject %s: %v", o.Name, err)
		}
		defer testutil.CleanupRow(t, pool, "rental_objects", o.ID)
	}
	if err := repo.UpdateObject(ctxA, inactive); err != nil {
		t.Fatalf("UpdateObject inactive: %v", err)
	}

	objs, total, err := repo.ListObjects(ctxA, tenantA, ListObjectsFilter{Search: "kettensaege"}, 0, 50)
	if err != nil {
		t.Fatalf("ListObjects search: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 matches for search 'kettensaege' within tenant A, got %d", total)
	}
	for _, o := range objs {
		if o.TenantID != tenantA {
			t.Fatalf("ListObjects leaked a row from another tenant: %+v", o)
		}
	}

	geraetCategory := "geraet"
	_, total, err = repo.ListObjects(ctxA, tenantA, ListObjectsFilter{Category: &geraetCategory}, 0, 50)
	if err != nil {
		t.Fatalf("ListObjects category: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 'geraet' objects for tenant A, got %d", total)
	}

	objs, total, err = repo.ListObjects(ctxA, tenantA, ListObjectsFilter{ActiveOnly: true}, 0, 50)
	if err != nil {
		t.Fatalf("ListObjects activeOnly: %v", err)
	}
	for _, o := range objs {
		if !o.Active {
			t.Fatalf("ActiveOnly filter returned an inactive object: %+v", o)
		}
	}
	if total != 2 { // active + otherCategory
		t.Fatalf("expected 2 active objects for tenant A, got %d", total)
	}

	// tenant B never sees tenant A's rows regardless of filter
	_, totalB, err := repo.ListObjects(ctxB, tenantB, ListObjectsFilter{Search: "kettensaege"}, 0, 50)
	if err != nil {
		t.Fatalf("ListObjects as tenant B: %v", err)
	}
	if totalB != 1 {
		t.Fatalf("expected tenant B to see only its own 'Kettensaege Fremd', got total %d", totalB)
	}
}

// ============================================================================
// Rentals
// ============================================================================

func TestPostgresRental_CreateGetUpdateDelete(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vermietung Rental CRUD")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	obj := newTestObject(tenantID, "Anhaenger 2t", "fahrzeug")
	if err := repo.CreateObject(ctx, obj); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", obj.ID)

	now := time.Now().UTC().Truncate(time.Second)
	rental := newTestRental(tenantID, obj.ID, now, now.Add(48*time.Hour))
	if err := repo.CreateRental(ctx, rental); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", rental.ID)

	got, err := repo.GetRental(ctx, tenantID, rental.ID)
	if err != nil {
		t.Fatalf("GetRental: %v", err)
	}
	if got.RenterName != rental.RenterName || !got.StartDate.Equal(rental.StartDate) {
		t.Fatalf("GetRental mismatch: got %+v", got)
	}

	rental.RenterName = "Neuer Mieter AG"
	rental.TotalPrice = 120.0
	if err := repo.UpdateRental(ctx, rental); err != nil {
		t.Fatalf("UpdateRental: %v", err)
	}
	got, err = repo.GetRental(ctx, tenantID, rental.ID)
	if err != nil {
		t.Fatalf("GetRental after update: %v", err)
	}
	if got.RenterName != "Neuer Mieter AG" || got.TotalPrice != 120.0 {
		t.Fatalf("UpdateRental did not persist changes, got %+v", got)
	}

	if err := repo.DeleteRental(ctx, tenantID, rental.ID); err != nil {
		t.Fatalf("DeleteRental: %v", err)
	}
	if _, err := repo.GetRental(ctx, tenantID, rental.ID); !errors.Is(err, ErrRentalNotFound) {
		t.Fatalf("expected ErrRentalNotFound after DeleteRental, got %v", err)
	}
	if err := repo.DeleteRental(ctx, tenantID, rental.ID); !errors.Is(err, ErrRentalNotFound) {
		t.Fatalf("expected ErrRentalNotFound deleting an already-deleted rental, got %v", err)
	}
}

func TestPostgresRental_GetUpdateDelete_CrossTenant_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Vermietung Rental Cross A")
	testutil.EnsureTenant(t, pool, tenantB, "Vermietung Rental Cross B")
	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	obj := newTestObject(tenantA, "Buehne", "event")
	if err := repo.CreateObject(ctxA, obj); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", obj.ID)

	now := time.Now().UTC().Truncate(time.Second)
	rental := newTestRental(tenantA, obj.ID, now, now.Add(24*time.Hour))
	if err := repo.CreateRental(ctxA, rental); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", rental.ID)

	if _, err := repo.GetRental(ctxB, tenantB, rental.ID); !errors.Is(err, ErrRentalNotFound) {
		t.Fatalf("expected ErrRentalNotFound reading tenant A's rental as tenant B, got %v", err)
	}

	hijack := *rental
	hijack.TenantID = tenantB
	hijack.RenterName = "Hijacked"
	if err := repo.UpdateRental(ctxB, &hijack); !errors.Is(err, ErrRentalNotFound) {
		t.Fatalf("expected ErrRentalNotFound updating tenant A's rental as tenant B, got %v", err)
	}
	if err := repo.DeleteRental(ctxB, tenantB, rental.ID); !errors.Is(err, ErrRentalNotFound) {
		t.Fatalf("expected ErrRentalNotFound deleting tenant A's rental as tenant B, got %v", err)
	}

	got, err := repo.GetRental(ctxA, tenantA, rental.ID)
	if err != nil {
		t.Fatalf("GetRental as owning tenant: %v", err)
	}
	if got.RenterName != rental.RenterName {
		t.Fatalf("cross-tenant UpdateRental leaked through: got renter_name %q", got.RenterName)
	}
}

func TestPostgresListRentals_FiltersAndTenantScoping(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Vermietung List Rentals A")
	testutil.EnsureTenant(t, pool, tenantB, "Vermietung List Rentals B")
	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	objA1 := newTestObject(tenantA, "Objekt A1", "geraet")
	objA2 := newTestObject(tenantA, "Objekt A2", "geraet")
	objB := newTestObject(tenantB, "Objekt B", "geraet")
	for _, o := range []*RentalObject{objA1, objA2, objB} {
		ctx := ctxA
		if o.TenantID == tenantB {
			ctx = ctxB
		}
		if err := repo.CreateObject(ctx, o); err != nil {
			t.Fatalf("CreateObject %s: %v", o.Name, err)
		}
		defer testutil.CleanupRow(t, pool, "rental_objects", o.ID)
	}

	base := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	early := newTestRental(tenantA, objA1.ID, base, base.Add(48*time.Hour))
	late := newTestRental(tenantA, objA2.ID, base.Add(10*24*time.Hour), base.Add(12*24*time.Hour))
	late.Status = RentalStatusActive
	foreign := newTestRental(tenantB, objB.ID, base, base.Add(48*time.Hour))

	if err := repo.CreateRental(ctxA, early); err != nil {
		t.Fatalf("CreateRental early: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", early.ID)
	if err := repo.CreateRental(ctxA, late); err != nil {
		t.Fatalf("CreateRental late: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", late.ID)
	if err := repo.CreateRental(ctxB, foreign); err != nil {
		t.Fatalf("CreateRental foreign: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", foreign.ID)

	// ObjectID filter
	rentals, total, err := repo.ListRentals(ctxA, tenantA, ListRentalsFilter{ObjectID: &objA1.ID}, 0, 50)
	if err != nil {
		t.Fatalf("ListRentals by object: %v", err)
	}
	if total != 1 || len(rentals) != 1 || rentals[0].ID != early.ID {
		t.Fatalf("expected only 'early' for objA1, got total=%d rentals=%+v", total, rentals)
	}

	// Status filter
	statusActive := RentalStatusActive
	rentals, total, err = repo.ListRentals(ctxA, tenantA, ListRentalsFilter{Status: &statusActive}, 0, 50)
	if err != nil {
		t.Fatalf("ListRentals by status: %v", err)
	}
	if total != 1 || rentals[0].ID != late.ID {
		t.Fatalf("expected only 'late' (active), got total=%d rentals=%+v", total, rentals)
	}

	// From/To range filter — early only overlaps the first window
	from := base
	to := base.Add(5 * 24 * time.Hour)
	rentals, total, err = repo.ListRentals(ctxA, tenantA, ListRentalsFilter{From: &from, To: &to}, 0, 50)
	if err != nil {
		t.Fatalf("ListRentals by date range: %v", err)
	}
	if total != 1 || rentals[0].ID != early.ID {
		t.Fatalf("expected only 'early' within [%v,%v], got total=%d rentals=%+v", from, to, total, rentals)
	}

	// Tenant B never sees tenant A's rentals
	_, totalB, err := repo.ListRentals(ctxB, tenantB, ListRentalsFilter{}, 0, 50)
	if err != nil {
		t.Fatalf("ListRentals as tenant B: %v", err)
	}
	if totalB != 1 {
		t.Fatalf("expected tenant B to see only its own rental, got total %d", totalB)
	}
}

// ============================================================================
// HasOverlap
// ============================================================================

func TestPostgresHasOverlap_DetectsAndExcludesCancelled(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vermietung HasOverlap")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	obj := newTestObject(tenantID, "Buehnentruck", "event")
	if err := repo.CreateObject(ctx, obj); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", obj.ID)

	base := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	booked := newTestRental(tenantID, obj.ID, base, base.Add(72*time.Hour))
	if err := repo.CreateRental(ctx, booked); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", booked.ID)

	// Overlapping window
	overlap, err := repo.HasOverlap(ctx, tenantID, obj.ID, base.Add(24*time.Hour), base.Add(96*time.Hour), nil)
	if err != nil {
		t.Fatalf("HasOverlap overlapping: %v", err)
	}
	if !overlap {
		t.Fatal("expected overlap for a window intersecting the existing booking")
	}

	// end_date is exclusive: a window starting exactly at the existing end must not overlap
	adjacent, err := repo.HasOverlap(ctx, tenantID, obj.ID, booked.EndDate, booked.EndDate.Add(24*time.Hour), nil)
	if err != nil {
		t.Fatalf("HasOverlap adjacent: %v", err)
	}
	if adjacent {
		t.Fatal("expected no overlap for a window starting exactly at the existing rental's end_date (half-open range)")
	}

	// Excluding the booking itself (as UpdateRental does) must not self-conflict
	self, err := repo.HasOverlap(ctx, tenantID, obj.ID, booked.StartDate, booked.EndDate, &booked.ID)
	if err != nil {
		t.Fatalf("HasOverlap excludeRentalID: %v", err)
	}
	if self {
		t.Fatal("expected no self-overlap when excluding the rental's own ID")
	}

	// Cancel the booking, then the same overlapping window must be free
	booked.Status = RentalStatusCancelled
	if err := repo.UpdateRental(ctx, booked); err != nil {
		t.Fatalf("UpdateRental to cancelled: %v", err)
	}
	overlap, err = repo.HasOverlap(ctx, tenantID, obj.ID, base.Add(24*time.Hour), base.Add(96*time.Hour), nil)
	if err != nil {
		t.Fatalf("HasOverlap after cancel: %v", err)
	}
	if overlap {
		t.Fatal("expected a cancelled rental to be exempt from overlap detection")
	}
}

// TestPostgresCreateRental_ConcurrentOverlap_OneWinnerOneConflict pins the
// contract that motivated asRentalConflict in postgres_repository.go:
// service.CreateRental's HasOverlap pre-check and the INSERT are not in the
// same transaction, so two concurrent bookings for the same object/date-range
// can both pass the pre-check and race the DB. The uq_rentals_no_overlap
// GIST exclusion constraint (migration 000101) is the real backstop — this
// test proves the loser sees the domain ErrRentalConflict, not a raw
// exclusion_violation Postgres error.
func TestPostgresCreateRental_ConcurrentOverlap_OneWinnerOneConflict(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vermietung Concurrent Overlap")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	obj := newTestObject(tenantID, "Konzertzelt", "event")
	if err := repo.CreateObject(ctx, obj); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", obj.ID)

	base := time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC)
	rentals := [2]*Rental{
		newTestRental(tenantID, obj.ID, base, base.Add(48*time.Hour)),
		newTestRental(tenantID, obj.ID, base.Add(12*time.Hour), base.Add(60*time.Hour)),
	}
	defer func() {
		for _, r := range rentals {
			testutil.CleanupRow(t, pool, "rentals", r.ID)
		}
	}()

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 2)
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = repo.CreateRental(ctx, rentals[i])
		}(i)
	}
	close(start)
	wg.Wait()

	winners, conflicts := 0, 0
	for i, err := range errs {
		switch {
		case err == nil:
			winners++
		case errors.Is(err, ErrRentalConflict):
			conflicts++
		default:
			t.Fatalf("goroutine %d: expected nil or ErrRentalConflict, got %v (not the raw pg error)", i, err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("expected exactly one winner and one ErrRentalConflict, got %d winners / %d conflicts", winners, conflicts)
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM rentals WHERE tenant_id=$1 AND object_id=$2", tenantID, obj.ID).Scan(&count); err != nil {
		t.Fatalf("count rentals: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one rental to have been inserted despite the race, got %d", count)
	}
}

// ============================================================================
// SaveSignature
// ============================================================================

func TestPostgresSaveSignature_UpdatesAndReturnsRental(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vermietung SaveSignature")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	obj := newTestObject(tenantID, "Hebebuehne", "geraet")
	if err := repo.CreateObject(ctx, obj); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", obj.ID)

	now := time.Now().UTC().Truncate(time.Second)
	rental := newTestRental(tenantID, obj.ID, now, now.Add(24*time.Hour))
	if err := repo.CreateRental(ctx, rental); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", rental.ID)

	sig := "data:image/png;base64,iVBORw0KGgo="
	got, err := repo.SaveSignature(ctx, tenantID.String(), rental.ID.String(), sig, "Max Mustermann")
	if err != nil {
		t.Fatalf("SaveSignature: %v", err)
	}
	if got.SignatureData == nil || *got.SignatureData != sig {
		t.Fatalf("expected signature_data %q, got %+v", sig, got.SignatureData)
	}
	if got.SignedBy == nil || *got.SignedBy != "Max Mustermann" {
		t.Fatalf("expected signed_by set, got %+v", got.SignedBy)
	}
	if got.SignedAt == nil {
		t.Fatal("expected signed_at set")
	}

	if _, err := repo.SaveSignature(ctx, tenantID.String(), uuid.New().String(), sig, "Nobody"); !errors.Is(err, ErrRentalNotFound) {
		t.Fatalf("expected ErrRentalNotFound for unknown rental, got %v", err)
	}
}

// ============================================================================
// Inspections
// ============================================================================

func TestPostgresInspection_CreateGetUpdateAndKindLookup(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Vermietung Inspection CRUD")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	obj := newTestObject(tenantID, "Rammler", "geraet")
	if err := repo.CreateObject(ctx, obj); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", obj.ID)

	now := time.Now().UTC().Truncate(time.Second)
	rental := newTestRental(tenantID, obj.ID, now, now.Add(24*time.Hour))
	if err := repo.CreateRental(ctx, rental); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", rental.ID)

	ins := &RentalInspection{
		ID:        uuid.New(),
		TenantID:  tenantID,
		RentalID:  rental.ID,
		Kind:      InspectionKindHandover,
		Notes:     "Zustand gut",
		PhotoURLs: []string{"https://example.invalid/a.jpg"},
		Checklist: []ChecklistItem{{Label: "Motor", Condition: "intakt"}},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateInspection(ctx, ins); err != nil {
		t.Fatalf("CreateInspection: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_inspections", ins.ID)

	got, err := repo.GetInspection(ctx, tenantID, ins.ID)
	if err != nil {
		t.Fatalf("GetInspection: %v", err)
	}
	if got.Notes != ins.Notes || len(got.Checklist) != 1 || got.Checklist[0].Label != "Motor" {
		t.Fatalf("GetInspection mismatch: got %+v", got)
	}

	byKind, err := repo.GetInspectionByKind(ctx, tenantID, rental.ID, InspectionKindHandover)
	if err != nil {
		t.Fatalf("GetInspectionByKind: %v", err)
	}
	if byKind.ID != ins.ID {
		t.Fatalf("GetInspectionByKind returned wrong inspection: %+v", byKind)
	}
	if _, err := repo.GetInspectionByKind(ctx, tenantID, rental.ID, InspectionKindReturn); !errors.Is(err, ErrInspectionNotFound) {
		t.Fatalf("expected ErrInspectionNotFound for a kind that was never created, got %v", err)
	}

	ins.Notes = "Zustand: kleiner Kratzer"
	ins.PhotoURLs = []string{"https://example.invalid/a.jpg", "https://example.invalid/b.jpg"}
	if err := repo.UpdateInspection(ctx, ins); err != nil {
		t.Fatalf("UpdateInspection: %v", err)
	}
	got, err = repo.GetInspection(ctx, tenantID, ins.ID)
	if err != nil {
		t.Fatalf("GetInspection after update: %v", err)
	}
	if got.Notes != "Zustand: kleiner Kratzer" || len(got.PhotoURLs) != 2 {
		t.Fatalf("UpdateInspection did not persist changes, got %+v", got)
	}

	if _, err := repo.GetInspection(ctx, tenantID, uuid.New()); !errors.Is(err, ErrInspectionNotFound) {
		t.Fatalf("expected ErrInspectionNotFound for unknown inspection, got %v", err)
	}

	// DB-level backstop for the app pre-check in service.CreateInspection:
	// a second handover inspection for the same rental must fail the unique
	// constraint (migration 000101), not silently duplicate.
	dup := &RentalInspection{
		ID:        uuid.New(),
		TenantID:  tenantID,
		RentalID:  rental.ID,
		Kind:      InspectionKindHandover,
		PhotoURLs: []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateInspection(ctx, dup); err == nil {
		testutil.CleanupRow(t, pool, "rental_inspections", dup.ID)
		t.Fatal("expected a second handover inspection for the same rental to violate uq_rental_inspections_kind")
	}
}

func TestPostgresListInspections_TenantScopedAndOrdered(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Vermietung List Inspections A")
	testutil.EnsureTenant(t, pool, tenantB, "Vermietung List Inspections B")
	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	objA := newTestObject(tenantA, "Kran", "geraet")
	if err := repo.CreateObject(ctxA, objA); err != nil {
		t.Fatalf("CreateObject A: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", objA.ID)
	objB := newTestObject(tenantB, "Kran Fremd", "geraet")
	if err := repo.CreateObject(ctxB, objB); err != nil {
		t.Fatalf("CreateObject B: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", objB.ID)

	now := time.Now().UTC().Truncate(time.Second)
	rentalA := newTestRental(tenantA, objA.ID, now, now.Add(24*time.Hour))
	if err := repo.CreateRental(ctxA, rentalA); err != nil {
		t.Fatalf("CreateRental A: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", rentalA.ID)
	rentalB := newTestRental(tenantB, objB.ID, now, now.Add(24*time.Hour))
	if err := repo.CreateRental(ctxB, rentalB); err != nil {
		t.Fatalf("CreateRental B: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", rentalB.ID)

	handover := &RentalInspection{ID: uuid.New(), TenantID: tenantA, RentalID: rentalA.ID, Kind: InspectionKindHandover, PhotoURLs: []string{}, CreatedAt: now, UpdatedAt: now}
	ret := &RentalInspection{ID: uuid.New(), TenantID: tenantA, RentalID: rentalA.ID, Kind: InspectionKindReturn, PhotoURLs: []string{}, CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute)}
	foreign := &RentalInspection{ID: uuid.New(), TenantID: tenantB, RentalID: rentalB.ID, Kind: InspectionKindHandover, PhotoURLs: []string{}, CreatedAt: now, UpdatedAt: now}
	for ctx, ins := range map[context.Context]*RentalInspection{ctxA: handover, ctxB: foreign} {
		if err := repo.CreateInspection(ctx, ins); err != nil {
			t.Fatalf("CreateInspection %s: %v", ins.Kind, err)
		}
		defer testutil.CleanupRow(t, pool, "rental_inspections", ins.ID)
	}
	if err := repo.CreateInspection(ctxA, ret); err != nil {
		t.Fatalf("CreateInspection ret: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_inspections", ret.ID)

	list, total, err := repo.ListInspections(ctxA, tenantA, rentalA.ID, 0, 50)
	if err != nil {
		t.Fatalf("ListInspections: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("expected 2 inspections for rentalA, got total=%d len=%d", total, len(list))
	}
	if list[0].Kind != InspectionKindHandover || list[1].Kind != InspectionKindReturn {
		t.Fatalf("expected ORDER BY created_at ASC (handover, return), got %s, %s", list[0].Kind, list[1].Kind)
	}

	// Tenant B never sees tenant A's inspections even when querying rentalA's ID directly
	_, totalCrossTenant, err := repo.ListInspections(ctxB, tenantB, rentalA.ID, 0, 50)
	if err != nil {
		t.Fatalf("ListInspections cross-tenant: %v", err)
	}
	if totalCrossTenant != 0 {
		t.Fatalf("expected 0 inspections when tenant B queries tenant A's rental, got %d", totalCrossTenant)
	}
}
