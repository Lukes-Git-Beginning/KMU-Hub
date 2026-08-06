package settings_test

// Value-sets (migration 000295) split their truth across two layers: the Go
// registry ships the baseline, the DB holds only what a tenant changed. Nearly
// every bug this design can have is a merge bug, so the service tests below all
// ask the same question in different shapes: what does a reader see after
// writing/deleting X?
//
// The DB test at the bottom is separate on purpose — it is the only place that
// proves the two tables are actually tenant-isolated under RLS. The in-memory
// fakeRepo cannot prove that; it would happily hand out a neighbour's rows.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/settings"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func findOption(t *testing.T, set *settings.ResolvedValueSet, key string) settings.ResolvedValueSetOption {
	t.Helper()
	for _, opt := range set.Options {
		if opt.Key == key {
			return opt
		}
	}
	t.Fatalf("option %q not found in value-set %q", key, set.Key)
	return settings.ResolvedValueSetOption{}
}

// A tenant that never opened the editor still reads complete lists — that is
// the whole point of keeping the baseline in code instead of backfilling rows
// per tenant.
func TestListValueSets_ShippedBaselineWithoutOverrides(t *testing.T) {
	svc := newService(newFakeRepo())

	sets, err := svc.ListValueSets(context.Background(), uuid.New(), false)
	require.NoError(t, err)
	require.Len(t, sets, len(settings.SystemValueSetKeys()))

	for _, set := range sets {
		assert.True(t, set.IsSystem, "%s should be a system list", set.Key)
		assert.Equal(t, settings.ProvenanceDefault, set.Provenance)
		require.NotEmpty(t, set.Options)
		for _, opt := range set.Options {
			assert.Equal(t, settings.ProvenanceDefault, opt.Provenance)
		}
	}
}

// The merge is per option key: an option the tenant did not mention keeps its
// shipped definition. Omitting an option must NOT delete it — live records may
// still point at it, which is why hiding one means Active=false.
func TestGetValueSet_OverrideMergesPerOptionKey(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)
	tenantID := uuid.New()
	ctx := context.Background()

	_, err := svc.UpsertValueSet(ctx, tenantID, nil, &settings.ValueSet{
		Key:  "ticket_priority",
		Name: "Dringlichkeit",
		Options: []settings.ValueSetOption{
			{Key: "low", Label: "Rückfrage", Color: "hsl(142 71% 45%)", Order: 0, Active: true},
			{Key: "critical", Label: "Kritisch", Order: 3, Active: false},
		},
	})
	require.NoError(t, err)

	set, err := svc.GetValueSet(ctx, tenantID, "ticket_priority", false)
	require.NoError(t, err)

	// Set-level name and provenance follow the override.
	assert.Equal(t, "Dringlichkeit", set.Name)
	assert.Equal(t, settings.ProvenanceTenant, set.Provenance)
	assert.True(t, set.IsSystem)

	// All four shipped options survive, two of them untouched.
	require.Len(t, set.Options, 4)

	low := findOption(t, set, "low")
	assert.Equal(t, "Rückfrage", low.Label)
	assert.Equal(t, settings.ProvenanceTenant, low.Provenance)

	medium := findOption(t, set, "medium")
	assert.Equal(t, "Mittel", medium.Label)
	assert.Equal(t, settings.ProvenanceDefault, medium.Provenance)

	critical := findOption(t, set, "critical")
	assert.False(t, critical.Active, "soft-deleted option stays in the list, hidden")
	assert.Equal(t, settings.ProvenanceTenant, critical.Provenance)

	// Options come back in display order regardless of how they were stored.
	for i := 1; i < len(set.Options); i++ {
		assert.LessOrEqual(t, set.Options[i-1].Order, set.Options[i].Order)
	}
}

// base=1 is the editor's "what did Cosmi ship" view and must ignore the
// tenant's own rows entirely.
func TestGetValueSet_BaseIgnoresOverride(t *testing.T) {
	svc := newService(newFakeRepo())
	tenantID := uuid.New()
	ctx := context.Background()

	_, err := svc.UpsertValueSet(ctx, tenantID, nil, &settings.ValueSet{
		Key:     "ticket_priority",
		Name:    "Dringlichkeit",
		Options: []settings.ValueSetOption{{Key: "low", Label: "Rückfrage", Order: 0, Active: true}},
	})
	require.NoError(t, err)

	base, err := svc.GetValueSet(ctx, tenantID, "ticket_priority", true)
	require.NoError(t, err)
	assert.Equal(t, "Ticket-Priorität", base.Name)
	assert.Equal(t, "Niedrig", findOption(t, base, "low").Label)
}

// A list the tenant invented has no baseline. It must still be listed — the FE
// mock lists only its code defaults, which would leave such a list invisible in
// the editor that created it (documented divergence).
func TestListValueSets_IncludesTenantOwnedList(t *testing.T) {
	svc := newService(newFakeRepo())
	tenantID := uuid.New()
	ctx := context.Background()

	_, err := svc.UpsertValueSet(ctx, tenantID, nil, &settings.ValueSet{
		Key:     "machine_condition",
		Name:    "Maschinenzustand",
		Options: []settings.ValueSetOption{{Key: "ok", Label: "In Ordnung", Order: 0, Active: true}},
	})
	require.NoError(t, err)

	sets, err := svc.ListValueSets(ctx, tenantID, false)
	require.NoError(t, err)
	require.Len(t, sets, len(settings.SystemValueSetKeys())+1)

	var own *settings.ResolvedValueSet
	for _, set := range sets {
		if set.Key == "machine_condition" {
			own = set
		}
	}
	require.NotNil(t, own)
	assert.False(t, own.IsSystem)
	assert.Equal(t, settings.ProvenanceTenant, own.Provenance)

	// base=1 has nothing to say about a list without a baseline.
	bases, err := svc.ListValueSets(ctx, tenantID, true)
	require.NoError(t, err)
	assert.Len(t, bases, len(settings.SystemValueSetKeys()))
}

// Deleting a system list is a RESET, not a deletion: the list survives on its
// shipped baseline. This is the "system lists cannot be deleted" guarantee —
// it holds structurally, not via a guard that someone could forget.
func TestDeleteValueSet_SystemListFallsBackToBaseline(t *testing.T) {
	svc := newService(newFakeRepo())
	tenantID := uuid.New()
	ctx := context.Background()

	_, err := svc.UpsertValueSet(ctx, tenantID, nil, &settings.ValueSet{
		Key:     "deal_stages",
		Name:    "Behandlungs-Pipeline",
		Options: []settings.ValueSetOption{{Key: "lead", Label: "Interessent", Order: 0, Active: true}},
	})
	require.NoError(t, err)

	after, err := svc.DeleteValueSet(ctx, tenantID, "deal_stages")
	require.NoError(t, err)
	require.NotNil(t, after, "a system list must survive its override being deleted")
	assert.Equal(t, "Deal-Phasen", after.Name)
	assert.True(t, after.IsSystem)
	assert.Equal(t, "Lead", findOption(t, after, "lead").Label)

	// And a fresh read agrees — the reset is persisted, not just returned.
	reread, err := svc.GetValueSet(ctx, tenantID, "deal_stages", false)
	require.NoError(t, err)
	assert.Equal(t, "Deal-Phasen", reread.Name)
	assert.Equal(t, settings.ProvenanceDefault, reread.Provenance)
}

// Resetting a system list nobody overrode asks for a state that already holds.
func TestDeleteValueSet_SystemListWithoutOverrideIsNoOp(t *testing.T) {
	svc := newService(newFakeRepo())

	after, err := svc.DeleteValueSet(context.Background(), uuid.New(), "project_status")
	require.NoError(t, err)
	require.NotNil(t, after)
	assert.True(t, after.IsSystem)
}

func TestDeleteValueSet_TenantOwnedListIsGone(t *testing.T) {
	svc := newService(newFakeRepo())
	tenantID := uuid.New()
	ctx := context.Background()

	_, err := svc.UpsertValueSet(ctx, tenantID, nil, &settings.ValueSet{
		Key:     "machine_condition",
		Name:    "Maschinenzustand",
		Options: []settings.ValueSetOption{{Key: "ok", Label: "In Ordnung", Order: 0, Active: true}},
	})
	require.NoError(t, err)

	after, err := svc.DeleteValueSet(ctx, tenantID, "machine_condition")
	require.NoError(t, err)
	assert.Nil(t, after, "a tenant-owned list has no baseline to fall back to")

	_, err = svc.GetValueSet(ctx, tenantID, "machine_condition", false)
	assert.ErrorIs(t, err, settings.ErrNotFound)
}

func TestDeleteValueSet_UnknownKey(t *testing.T) {
	svc := newService(newFakeRepo())

	_, err := svc.DeleteValueSet(context.Background(), uuid.New(), "never_existed")
	assert.ErrorIs(t, err, settings.ErrNotFound)
}

func TestGetValueSet_UnknownKey(t *testing.T) {
	svc := newService(newFakeRepo())

	_, err := svc.GetValueSet(context.Background(), uuid.New(), "never_existed", false)
	assert.ErrorIs(t, err, settings.ErrNotFound)
}

// Validation lives at the service boundary so the editor gets a precise error
// instead of a 23514 from the CHECK constraints that say the same thing.
func TestUpsertValueSet_Validation(t *testing.T) {
	longLabel := ""
	for i := 0; i < 130; i++ {
		longLabel += "x"
	}

	cases := []struct {
		name string
		set  *settings.ValueSet
		want error
	}{
		{
			name: "empty key",
			set:  &settings.ValueSet{Key: "", Name: "X", Options: []settings.ValueSetOption{{Key: "a", Label: "A", Active: true}}},
			want: settings.ErrInvalidValueSetKey,
		},
		{
			name: "key with uppercase",
			set:  &settings.ValueSet{Key: "Deal_Stages", Name: "X", Options: []settings.ValueSetOption{{Key: "a", Label: "A", Active: true}}},
			want: settings.ErrInvalidValueSetKey,
		},
		{
			name: "blank name",
			set:  &settings.ValueSet{Key: "deal_stages", Name: "   ", Options: []settings.ValueSetOption{{Key: "a", Label: "A", Active: true}}},
			want: settings.ErrInvalidValueSet,
		},
		{
			name: "no options",
			set:  &settings.ValueSet{Key: "deal_stages", Name: "X"},
			want: settings.ErrInvalidValueSet,
		},
		{
			name: "duplicate option key",
			set: &settings.ValueSet{Key: "deal_stages", Name: "X", Options: []settings.ValueSetOption{
				{Key: "a", Label: "A", Active: true},
				{Key: "a", Label: "B", Active: true},
			}},
			want: settings.ErrInvalidValueSetOption,
		},
		{
			name: "blank option label",
			set: &settings.ValueSet{Key: "deal_stages", Name: "X", Options: []settings.ValueSetOption{
				{Key: "a", Label: " ", Active: true},
			}},
			want: settings.ErrInvalidValueSetOption,
		},
		{
			name: "oversized option label",
			set: &settings.ValueSet{Key: "deal_stages", Name: "X", Options: []settings.ValueSetOption{
				{Key: "a", Label: longLabel, Active: true},
			}},
			want: settings.ErrInvalidValueSetOption,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newService(newFakeRepo())
			_, err := svc.UpsertValueSet(context.Background(), uuid.New(), nil, tc.set)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

// ============================================================================
// DB-backed: the only proof that both tables are tenant-isolated under RLS.
// ============================================================================

func TestValueSetOverrides_TenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	// Dedicated tenants, not testutil.TenantA/B: those are shared across the
	// suite and a value-set key collision would make this test flap.
	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Value-Set Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Value-Set Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantWrite)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	author := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantWrite,
		"email":         fmt.Sprintf("valueset-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", author)

	repo := settings.NewPostgresRepository(pool)
	ctxWrite := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	stored, err := repo.UpsertValueSetOverride(ctxWrite, tenantWrite, &author, &settings.ValueSet{
		Key:  "ticket_priority",
		Name: "Dringlichkeit",
		Options: []settings.ValueSetOption{
			{Key: "low", Label: "Rückfrage", Color: "hsl(142 71% 45%)", Order: 0, Active: true},
			{Key: "critical", Label: "Kritisch", Order: 3, Active: false},
		},
	})
	require.NoError(t, err)
	require.Len(t, stored.Options, 2)
	assert.Equal(t, "hsl(142 71% 45%)", stored.Options[0].Color)
	assert.Empty(t, stored.Options[1].Color, "a NULL color reads back as the empty string, not as a scan error")
	assert.False(t, stored.Options[1].Active)

	// Own tenant sees both rows; the neighbour sees neither — asserted with the
	// writer's tenant_id still named explicitly, so this is the RLS policy
	// talking, not the WHERE clause.
	assertRowCountWhere(t, pool, ctxWrite, "customization_value_sets",
		"tenant_id = $1 AND set_key = $2", 1, tenantWrite, "ticket_priority")
	assertRowCountWhere(t, pool, ctxOther, "customization_value_sets",
		"tenant_id = $1 AND set_key = $2", 0, tenantWrite, "ticket_priority")
	assertRowCountWhere(t, pool, ctxWrite, "customization_value_set_options",
		"tenant_id = $1", 2, tenantWrite)
	assertRowCountWhere(t, pool, ctxOther, "customization_value_set_options",
		"tenant_id = $1", 0, tenantWrite)

	// The repository read agrees with RLS: the neighbour gets nothing back.
	_, err = repo.GetValueSetOverride(ctxOther, tenantWrite, "ticket_priority")
	assert.ErrorIs(t, err, settings.ErrNotFound)

	foreignList, err := repo.ListValueSetOverrides(ctxOther, tenantWrite)
	require.NoError(t, err)
	assert.Empty(t, foreignList)

	// A second upsert REPLACES the option list wholesale (PUT semantics) rather
	// than accumulating rows — the failure mode would be silent duplicates.
	stored, err = repo.UpsertValueSetOverride(ctxWrite, tenantWrite, &author, &settings.ValueSet{
		Key:     "ticket_priority",
		Name:    "Priorität",
		Options: []settings.ValueSetOption{{Key: "low", Label: "Niedrig", Order: 0, Active: true}},
	})
	require.NoError(t, err)
	require.Len(t, stored.Options, 1)
	assert.Equal(t, "Priorität", stored.Name)
	assertRowCountWhere(t, pool, ctxWrite, "customization_value_set_options",
		"tenant_id = $1", 1, tenantWrite)

	// Delete cascades to the options via the composite FK.
	require.NoError(t, repo.DeleteValueSetOverride(ctxWrite, tenantWrite, "ticket_priority"))
	assertRowCountWhere(t, pool, ctxWrite, "customization_value_sets",
		"tenant_id = $1", 0, tenantWrite)
	assertRowCountWhere(t, pool, ctxWrite, "customization_value_set_options",
		"tenant_id = $1", 0, tenantWrite)

	// Deleting someone else's override must not report success.
	assert.ErrorIs(t, repo.DeleteValueSetOverride(ctxOther, tenantWrite, "ticket_priority"), settings.ErrNotFound)
}
