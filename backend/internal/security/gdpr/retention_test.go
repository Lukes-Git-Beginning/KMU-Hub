package gdpr

// Coverage for the retention engine (Lauf 10, A10). The engine ships with an
// empty registry on purpose — the per-resource handlers are separate units —
// so the integration test brings its own handler over a real table
// (`notifications`). That keeps the proof honest: RLS, the run log and the
// idempotency guarantee are all exercised against Postgres, not a fake.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestParseRetentionMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in       string
		wantMode RetentionMode
		wantOK   bool
	}{
		{"enforce", RetentionModeEnforce, true},
		{"ENFORCE", RetentionModeEnforce, true},
		{"  enforce  ", RetentionModeEnforce, true},
		{"dry_run", RetentionModeDryRun, true},
		{"", RetentionModeDryRun, true},
		// A typo must never arm the engine, and it must be reported.
		{"enfroce", RetentionModeDryRun, false},
		{"true", RetentionModeDryRun, false},
		{"on", RetentionModeDryRun, false},
	}
	for _, tc := range cases {
		mode, ok := ParseRetentionMode(tc.in)
		assert.Equal(t, tc.wantMode, mode, "mode for %q", tc.in)
		assert.Equal(t, tc.wantOK, ok, "ok for %q", tc.in)
	}
}

func TestRetentionRegistry_LookupIsExact(t *testing.T) {
	t.Parallel()

	h := &fakeRetentionHandler{resourceType: "contacts"}
	reg := NewRetentionRegistry(h)

	got, ok := reg.Lookup("contacts")
	require.True(t, ok)
	assert.Same(t, h, got)

	// Near misses must not resolve — a policy for "Contacts" governs nothing.
	for _, miss := range []string{"Contacts", "contacts ", "contact", "kontakte"} {
		_, ok := reg.Lookup(miss)
		assert.False(t, ok, "lookup %q must not resolve", miss)
	}

	assert.Equal(t, []string{"contacts"}, reg.ResourceTypes())

	// nil handlers are ignored instead of panicking on ResourceType().
	reg.Register(nil)
	assert.Equal(t, []string{"contacts"}, reg.ResourceTypes())
}

func TestRetentionEngine_EmptyRegistryReportsUnmapped(t *testing.T) {
	t.Parallel()

	// The default registry is empty by design: until the per-resource
	// handlers land, every policy must be visible as "nicht zugeordnet".
	assert.Empty(t, NewRetentionEngine(nil, nil, nil).Registry().ResourceTypes())
}

// ---------------------------------------------------------------------------
// Integration
// ---------------------------------------------------------------------------

func TestRetentionEngine_Run_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Retention Engine Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOther, "Retention Engine Foreign Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := seedExportUser(t, pool, tenantOwn, "retention-subject")
	defer testutil.CleanupRow(t, pool, "users", userID)
	foreignUserID := seedExportUser(t, pool, tenantOther, "retention-foreign")
	defer testutil.CleanupRow(t, pool, "users", foreignUserID)

	old := time.Now().UTC().AddDate(0, 0, -90)
	fresh := time.Now().UTC().AddDate(0, 0, -2)

	expiredA := seedRetentionNotification(t, pool, tenantOwn, userID, "Alt A", old)
	defer testutil.CleanupRow(t, pool, "notifications", expiredA)
	expiredB := seedRetentionNotification(t, pool, tenantOwn, userID, "Alt B", old)
	defer testutil.CleanupRow(t, pool, "notifications", expiredB)
	// Inside the retention period — must survive every run.
	recent := seedRetentionNotification(t, pool, tenantOwn, userID, "Neu", fresh)
	defer testutil.CleanupRow(t, pool, "notifications", recent)
	// Old enough, but belongs to another tenant — must never be touched.
	foreign := seedRetentionNotification(t, pool, tenantOther, foreignUserID, "Fremd", old)
	defer testutil.CleanupRow(t, pool, "notifications", foreign)

	mappedPolicy := testutil.SeedRow(t, pool, "retention_policies", map[string]any{
		"tenant_id":      tenantOwn,
		"resource_type":  "notifications",
		"retention_days": 30,
		"action":         models.RetentionActionDelete,
		"enabled":        true,
	})
	defer testutil.CleanupRow(t, pool, "retention_policies", mappedPolicy)

	// Free-text resource_type that no handler knows — the dangerous case:
	// it looks like a fulfilled deletion duty and deletes nothing.
	unmappedPolicy := testutil.SeedRow(t, pool, "retention_policies", map[string]any{
		"tenant_id":      tenantOwn,
		"resource_type":  "kontakte",
		"retention_days": 30,
		"action":         models.RetentionActionDelete,
		"enabled":        true,
	})
	defer testutil.CleanupRow(t, pool, "retention_policies", unmappedPolicy)

	// Disabled policies must not run at all.
	disabledPolicy := testutil.SeedRow(t, pool, "retention_policies", map[string]any{
		"tenant_id":      tenantOwn,
		"resource_type":  "tickets",
		"retention_days": 30,
		"action":         models.RetentionActionDelete,
		"enabled":        false,
	})
	defer testutil.CleanupRow(t, pool, "retention_policies", disabledPolicy)

	engine := NewRetentionEngine(pool, NewPostgresRepository(pool),
		NewRetentionRegistry(newNotificationRetentionHandler(pool)))

	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	// --- dry run: counts, changes nothing ---------------------------------
	dry, err := engine.Run(ctx, RetentionModeDryRun, "test")
	require.NoError(t, err)
	defer testutil.CleanupRow(t, pool, "retention_runs", dry.RunID)

	assert.Equal(t, RetentionModeDryRun, dry.Mode)
	assert.Equal(t, 2, dry.PoliciesTotal, "disabled policy must be ignored")
	assert.Equal(t, 2, dry.RecordsMatched)
	assert.Equal(t, 0, dry.RecordsAffected, "a dry run must not touch data")

	mappedItem := itemFor(t, dry, "notifications")
	assert.Equal(t, RetentionItemDryRun, mappedItem.Status)
	assert.Equal(t, 2, mappedItem.Matched)
	assert.Equal(t, 0, mappedItem.Affected)

	unmappedItem := itemFor(t, dry, "kontakte")
	assert.Equal(t, RetentionItemUnmapped, unmappedItem.Status)
	assert.Contains(t, unmappedItem.Message, "nicht zugeordnet")
	assert.Equal(t, 0, unmappedItem.Matched)

	testutil.AssertRowCount(t, pool, ctx, "notifications", expiredA, 1)
	testutil.AssertRowCount(t, pool, ctx, "notifications", expiredB, 1)

	// --- enforce: deletes exactly the expired rows of this tenant ---------
	armed, err := engine.Run(ctx, RetentionModeEnforce, "test")
	require.NoError(t, err)
	defer testutil.CleanupRow(t, pool, "retention_runs", armed.RunID)

	assert.Equal(t, RetentionModeEnforce, armed.Mode)
	assert.Equal(t, 2, armed.RecordsMatched)
	assert.Equal(t, 2, armed.RecordsAffected)
	assert.Equal(t, RetentionItemApplied, itemFor(t, armed, "notifications").Status)

	testutil.AssertRowCount(t, pool, ctx, "notifications", expiredA, 0)
	testutil.AssertRowCount(t, pool, ctx, "notifications", expiredB, 0)
	testutil.AssertRowCount(t, pool, ctx, "notifications", recent, 1)

	foreignCtx := testutil.WithTenantCtx(context.Background(), tenantOther)
	testutil.AssertRowCount(t, pool, foreignCtx, "notifications", foreign, 1)

	// --- second enforce run right after the first changes nothing ---------
	again, err := engine.Run(ctx, RetentionModeEnforce, "test")
	require.NoError(t, err)
	defer testutil.CleanupRow(t, pool, "retention_runs", again.RunID)

	assert.Equal(t, 0, again.RecordsMatched, "second run must find nothing left")
	assert.Equal(t, 0, again.RecordsAffected)

	// --- run log ----------------------------------------------------------
	var status, mode string
	var matched, affected int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT status, mode, records_matched, records_affected
		   FROM retention_runs WHERE id = $1 AND tenant_id = $2`,
		armed.RunID, tenantOwn,
	).Scan(&status, &mode, &matched, &affected))
	assert.Equal(t, "completed", status)
	assert.Equal(t, string(RetentionModeEnforce), mode)
	assert.Equal(t, 2, matched)
	assert.Equal(t, 2, affected)

	var itemCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM retention_run_items WHERE run_id = $1`, armed.RunID,
	).Scan(&itemCount))
	assert.Equal(t, 2, itemCount, "one item per enabled policy")

	// The unmapped policy must be readable in the log, not just in memory.
	var loggedStatus, loggedMessage string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT status, message FROM retention_run_items
		  WHERE run_id = $1 AND resource_type = 'kontakte'`, armed.RunID,
	).Scan(&loggedStatus, &loggedMessage))
	assert.Equal(t, RetentionItemUnmapped, loggedStatus)
	assert.Contains(t, loggedMessage, "nicht zugeordnet")

	// A foreign tenant must not see this run at all (RLS).
	var foreignVisible int
	require.NoError(t, pool.QueryRow(foreignCtx,
		`SELECT COUNT(*) FROM retention_runs WHERE id = $1`, armed.RunID,
	).Scan(&foreignVisible))
	assert.Equal(t, 0, foreignVisible, "retention runs must be tenant-isolated")
}

func TestRetentionEngine_Run_HandlerFailuresAreRecorded(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Retention Failure Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	brokenPolicy := testutil.SeedRow(t, pool, "retention_policies", map[string]any{
		"tenant_id":      tenantID,
		"resource_type":  "broken",
		"retention_days": 10,
		"action":         models.RetentionActionDelete,
		"enabled":        true,
	})
	defer testutil.CleanupRow(t, pool, "retention_policies", brokenPolicy)

	anonPolicy := testutil.SeedRow(t, pool, "retention_policies", map[string]any{
		"tenant_id":      tenantID,
		"resource_type":  "delete-only",
		"retention_days": 10,
		"action":         models.RetentionActionAnonymize,
		"enabled":        true,
	})
	defer testutil.CleanupRow(t, pool, "retention_policies", anonPolicy)

	engine := NewRetentionEngine(pool, NewPostgresRepository(pool), NewRetentionRegistry(
		&fakeRetentionHandler{resourceType: "broken", planErr: fmt.Errorf("Tabelle fehlt")},
		&fakeRetentionHandler{resourceType: "delete-only", deleteOnly: true},
	))

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	result, err := engine.Run(ctx, RetentionModeEnforce, "test")
	require.NoError(t, err, "one broken handler must not abort the whole run")
	defer testutil.CleanupRow(t, pool, "retention_runs", result.RunID)

	broken := itemFor(t, result, "broken")
	assert.Equal(t, RetentionItemFailed, broken.Status)
	assert.Contains(t, broken.Message, "Tabelle fehlt")

	unsupported := itemFor(t, result, "delete-only")
	assert.Equal(t, RetentionItemUnsupported, unsupported.Status)
	assert.Contains(t, unsupported.Message, "anonymize")

	var logged int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM retention_run_items WHERE run_id = $1 AND status IN ('failed','unsupported')`,
		result.RunID,
	).Scan(&logged))
	assert.Equal(t, 2, logged)
}

// TestRetentionEngine_Run_PartialBatchFailure_UndercountsButHandlerStaysIdempotent
// answers the scope question "what happens when a handler fails mid-batch —
// does half stay deleted, and is the next run idempotent?" using a stateful
// fake handler that really does persist the records it processes before
// erroring out (a bookkeeping map, standing in for a real handler's table
// writes — the engine itself never touches the resource table, only the
// handler does).
//
// It surfaces a real discrepancy: runPolicy (retention.go:400-407) discards
// Apply's returned affected count entirely when Apply also returns an error,
// so a handler that genuinely processed some records before failing is
// logged with Affected = 0. The next run is still idempotent from a DATA
// standpoint — Plan only re-offers what the handler itself never marked
// done — but the run log for the failed attempt understates what happened,
// which matters for an audit trail. Filed as a follow-up unit rather than
// fixed here: a coverage unit proves behaviour, it does not change it.
func TestRetentionEngine_Run_PartialBatchFailure_UndercountsButHandlerStaysIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Retention Partial Failure Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	policy := testutil.SeedRow(t, pool, "retention_policies", map[string]any{
		"tenant_id":      tenantID,
		"resource_type":  "partial",
		"retention_days": 10,
		"action":         models.RetentionActionDelete,
		"enabled":        true,
	})
	defer testutil.CleanupRow(t, pool, "retention_policies", policy)

	handler := &fakePartialFailureHandler{
		resourceType: "partial",
		due:          []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()},
		processed:    make(map[uuid.UUID]bool),
		failAfter:    2,
	}
	engine := NewRetentionEngine(pool, NewPostgresRepository(pool), NewRetentionRegistry(handler))
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	first, err := engine.Run(ctx, RetentionModeEnforce, "test")
	require.NoError(t, err, "one handler's mid-batch error must not abort Run itself")
	defer testutil.CleanupRow(t, pool, "retention_runs", first.RunID)

	item := itemFor(t, first, "partial")
	assert.Equal(t, RetentionItemFailed, item.Status)
	assert.Equal(t, 4, item.Matched, "Plan found all four records before anything ran")
	assert.Equal(t, 0, item.Affected,
		"known gap: runPolicy drops Apply's partial affected count on error, "+
			"even though the handler really processed some records (see handler.processed below)")
	assert.Len(t, handler.processed, 2, "the handler itself DID persist two of the four records before failing")

	var loggedAffected int
	var loggedStatus string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT status, affected FROM retention_run_items WHERE run_id = $1 AND resource_type = 'partial'`,
		first.RunID,
	).Scan(&loggedStatus, &loggedAffected))
	assert.Equal(t, RetentionItemFailed, loggedStatus)
	assert.Equal(t, 0, loggedAffected, "the run log itself, not just the in-memory item, understates the real work done")

	// --- second run: idempotent from the data's point of view -------------
	second, err := engine.Run(ctx, RetentionModeEnforce, "test")
	require.NoError(t, err)
	defer testutil.CleanupRow(t, pool, "retention_runs", second.RunID)

	secondItem := itemFor(t, second, "partial")
	assert.Equal(t, 2, secondItem.Matched, "only the two records the handler did NOT mark done in run 1 are still due")
}

// fakePartialFailureHandler simulates a handler whose Apply persists some
// records (via its own bookkeeping, standing in for real table writes) before
// erroring out partway through a batch.
type fakePartialFailureHandler struct {
	resourceType string
	due          []uuid.UUID
	processed    map[uuid.UUID]bool
	failAfter    int
}

func (h *fakePartialFailureHandler) ResourceType() string         { return h.resourceType }
func (h *fakePartialFailureHandler) Table() string                { return h.resourceType }
func (h *fakePartialFailureHandler) DateColumn() string           { return "created_at" }
func (h *fakePartialFailureHandler) SupportsAction(_ string) bool { return true }

func (h *fakePartialFailureHandler) Plan(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (*RetentionPlan, error) {
	plan := &RetentionPlan{}
	for _, id := range h.due {
		if !h.processed[id] {
			plan.Due = append(plan.Due, id)
		}
	}
	return plan, nil
}

func (h *fakePartialFailureHandler) Apply(_ context.Context, _ uuid.UUID, ids []uuid.UUID, _, _ string) (int, error) {
	affected := 0
	for i, id := range ids {
		if i >= h.failAfter {
			return affected, fmt.Errorf("simulated failure after %d of %d records", affected, len(ids))
		}
		h.processed[id] = true
		affected++
	}
	return affected, nil
}

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// fakeRetentionHandler covers the branches that need no table behind them.
type fakeRetentionHandler struct {
	resourceType string
	planErr      error
	deleteOnly   bool
}

func (h *fakeRetentionHandler) ResourceType() string { return h.resourceType }
func (h *fakeRetentionHandler) Table() string        { return h.resourceType }
func (h *fakeRetentionHandler) DateColumn() string   { return "created_at" }

func (h *fakeRetentionHandler) SupportsAction(action string) bool {
	if h.deleteOnly {
		return action == models.RetentionActionDelete
	}
	return true
}

func (h *fakeRetentionHandler) Plan(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (*RetentionPlan, error) {
	if h.planErr != nil {
		return nil, h.planErr
	}
	return &RetentionPlan{}, nil
}

func (h *fakeRetentionHandler) Apply(_ context.Context, _ uuid.UUID, ids []uuid.UUID, _, _ string) (int, error) {
	return len(ids), nil
}

// notificationRetentionHandler is a real handler over a real table, used to
// prove the engine end to end. It is deliberately test-local: which resource
// types production maps is decided by the follow-up handler units, not here.
type notificationRetentionHandler struct {
	pool *pgxpool.Pool
}

func newNotificationRetentionHandler(pool *pgxpool.Pool) *notificationRetentionHandler {
	return &notificationRetentionHandler{pool: pool}
}

func (h *notificationRetentionHandler) ResourceType() string { return "notifications" }
func (h *notificationRetentionHandler) Table() string        { return "notifications" }
func (h *notificationRetentionHandler) DateColumn() string   { return "created_at" }

func (h *notificationRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

func (h *notificationRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM notifications WHERE tenant_id = $1 AND created_at < $2 ORDER BY created_at ASC`,
		tenantID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	plan := &RetentionPlan{}
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, scanErr
		}
		plan.Due = append(plan.Due, id)
	}
	return plan, rows.Err()
}

func (h *notificationRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, _, _ string) (int, error) {
	tag, err := h.pool.Exec(ctx,
		`DELETE FROM notifications WHERE tenant_id = $1 AND id = ANY($2)`, tenantID, ids)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func seedRetentionNotification(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, title string, createdAt time.Time) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "notifications", map[string]any{
		"tenant_id":      tenantID,
		"user_id":        userID,
		"event_type_key": "retention.test",
		"module_id":      "security",
		"title":          title,
		"created_at":     createdAt,
	})
}

func itemFor(t *testing.T, result *RetentionRunResult, resourceType string) RetentionRunItem {
	t.Helper()
	for _, item := range result.Items {
		if item.ResourceType == resourceType {
			return item
		}
	}
	t.Fatalf("no run item for resource_type %q", resourceType)
	return RetentionRunItem{}
}
