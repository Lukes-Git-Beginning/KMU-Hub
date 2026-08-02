package berichte

// Double-delivery guard for the report scheduler.
//
// The scheduler runs inside every berichte replica, so two instances can see
// the same due schedule in the same tick. The only thing standing between that
// and a report mailed twice is ClaimSchedule's compare-and-set on last_run_at.
// scheduler_test.go covers the decision logic against a fake repository, which
// by construction cannot show whether the SQL is actually atomic — that needs a
// real Postgres and two concurrent claims on the same row.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestClaimSchedule_ConcurrentClaimsElectExactlyOneWinner(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte Claim Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC().Truncate(time.Microsecond)

	def := &Definition{
		ID:            uuid.New(),
		TenantID:      tenant,
		Name:          "Claim-Test-Definition",
		Module:        "cross",
		Kind:          "custom",
		QueryConfig:   []byte(`{}`),
		DefaultFormat: "csv",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateDefinition(ctx, def); err != nil {
		t.Fatalf("CreateDefinition: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "report_definitions", def.ID)

	sch := &Schedule{
		ID:             uuid.New(),
		TenantID:       tenant,
		DefinitionID:   def.ID,
		Name:           "Taeglicher Claim-Test",
		CronExpression: "0 6 * * *",
		Recipients:     []string{"empfaenger@example.test"},
		Format:         "csv",
		Params:         []byte(`{}`),
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.CreateSchedule(ctx, sch); err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "report_schedules", sch.ID)

	// Two schedulers racing on a never-run schedule: both see last_run_at NULL.
	const racers = 2
	results := make([]bool, racers)
	errs := make([]error, racers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range racers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx], errs[idx] = repo.ClaimSchedule(sysCtx, sch.ID, nil, now.Add(time.Duration(idx)*time.Millisecond))
		}(i)
	}
	close(start)
	wg.Wait()

	won := 0
	for i := range racers {
		if errs[i] != nil {
			t.Fatalf("ClaimSchedule (racer %d): %v", i, errs[i])
		}
		if results[i] {
			won++
		}
	}
	if won != 1 {
		t.Fatalf("concurrent claims: %d winners, want exactly 1 — the schedule would be delivered %d times", won, won)
	}

	// A replay of the already-consumed claim must lose as well: a scheduler
	// still holding the pre-claim last_run_at cannot re-run the schedule.
	claimedAgain, err := repo.ClaimSchedule(sysCtx, sch.ID, nil, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("ClaimSchedule (replay): %v", err)
	}
	if claimedAgain {
		t.Fatal("a stale claim (previousLastRunAt=nil) succeeded after the row was already claimed")
	}

	// The next tick, using the current last_run_at, must be able to claim again
	// — otherwise a schedule would fire exactly once and then be stuck forever.
	stored, err := repo.GetSchedule(sysCtx, tenant, sch.ID)
	if err != nil {
		t.Fatalf("GetSchedule: %v", err)
	}
	if stored.LastRunAt == nil {
		t.Fatal("last_run_at is still NULL after a successful claim")
	}
	nextClaim, err := repo.ClaimSchedule(sysCtx, sch.ID, stored.LastRunAt, now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("ClaimSchedule (next tick): %v", err)
	}
	if !nextClaim {
		t.Fatal("the following tick could not claim the schedule — it would never run again")
	}
}
