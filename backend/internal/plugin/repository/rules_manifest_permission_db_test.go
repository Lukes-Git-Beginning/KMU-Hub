package repository_test

// Read, update and list paths of the manifest, permission, validation-rule and
// workflow-rule repositories against the real database as kmuhub_app. The
// isolation tests next door cover Create/GetByID/Delete across tenants; this
// file pins what each remaining method actually returns.
//
// Every test mints its own tenants: the shared testutil.TenantA/B are seeded by
// many packages in parallel, and the validation/workflow lists filter on tenant.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin/repository"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func freshTenant(t *testing.T, pool *pgxpool.Pool, name string) (uuid.UUID, context.Context) {
	t.Helper()
	id := uuid.New()
	testutil.EnsureTenant(t, pool, id, name)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", id) })
	return id, testutil.WithTenantCtx(context.Background(), id)
}

func ids[T any](items []*T, id func(*T) uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		out = append(out, id(it))
	}
	return out
}

func equalIDs(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestManifestRepository_ListFiltersByTypeAndSortsByName(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant, ctx := freshTenant(t, pool, "manifest list")
	repo := repository.NewManifestRepository(pool)

	suffix := uuid.NewString()[:8]
	mk := func(name string, typ models.PluginType) *models.PluginManifest {
		m := manifestFixture(&tenant, "list-"+name+"-"+suffix)
		m.Name = name + "-" + suffix
		m.PluginType = typ
		if err := repo.Create(ctx, m); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "plugin_manifests", m.ID) })
		return m
	}
	bravo := mk("bravo", models.PluginTypeConfig)
	alpha := mk("alpha", models.PluginTypeWASM)
	charlie := mk("charlie", models.PluginTypeConfig)

	mine := map[uuid.UUID]bool{bravo.ID: true, alpha.ID: true, charlie.ID: true}
	onlyMine := func(list []*models.PluginManifest) []uuid.UUID {
		out := []uuid.UUID{}
		for _, m := range list {
			if mine[m.ID] {
				out = append(out, m.ID)
			}
		}
		return out
	}

	all, err := repo.List(ctx, "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if got, want := onlyMine(all), []uuid.UUID{alpha.ID, bravo.ID, charlie.ID}; !equalIDs(got, want) {
		t.Fatalf("unfiltered list order = %v, want name ascending %v", got, want)
	}

	cfg, err := repo.List(ctx, string(models.PluginTypeConfig))
	if err != nil {
		t.Fatalf("list config: %v", err)
	}
	if got, want := onlyMine(cfg), []uuid.UUID{bravo.ID, charlie.ID}; !equalIDs(got, want) {
		t.Fatalf("config list = %v, want %v", got, want)
	}
	for _, m := range cfg {
		if m.PluginType != models.PluginTypeConfig {
			t.Fatalf("type filter leaked %s manifest %s", m.PluginType, m.ID)
		}
	}

	wasm, err := repo.List(ctx, string(models.PluginTypeWASM))
	if err != nil {
		t.Fatalf("list wasm: %v", err)
	}
	if got, want := onlyMine(wasm), []uuid.UUID{alpha.ID}; !equalIDs(got, want) {
		t.Fatalf("wasm list = %v, want %v", got, want)
	}
}

func TestManifestRepository_ListIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	owner, ownerCtx := freshTenant(t, pool, "manifest list owner")
	_, otherCtx := freshTenant(t, pool, "manifest list other")
	repo := repository.NewManifestRepository(pool)

	m := manifestFixture(&owner, "list-scoped-"+uuid.NewString()[:8])
	if err := repo.Create(ownerCtx, m); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "plugin_manifests", m.ID) })

	has := func(ctx context.Context) bool {
		list, err := repo.List(ctx, "")
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		for _, it := range list {
			if it.ID == m.ID {
				return true
			}
		}
		return false
	}
	if !has(ownerCtx) {
		t.Fatal("owner does not see its own manifest in List")
	}
	if has(otherCtx) {
		t.Fatal("List leaked another tenant's manifest")
	}
}

func TestManifestRepository_RoundtripAndGetBySlug(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant, ctx := freshTenant(t, pool, "manifest roundtrip")
	repo := repository.NewManifestRepository(pool)

	m := manifestFixture(&tenant, "roundtrip-"+uuid.NewString()[:8])
	m.Description = "does things"
	m.Author = "zentria"
	m.Permissions = []string{"contacts:read", "deals:write"}
	m.SettingsSchema = json.RawMessage(`{"type":"object"}`)
	m.HookRegistrations = []models.HookRegistration{{HookType: "before_create", Module: "crm", EntityType: "contact", Priority: 3}}
	m.WASMBinary = []byte{0x00, 0x61, 0x73, 0x6d}
	m.WASMBinaryHash = "abc123"
	m.PluginType = models.PluginTypeWASM
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "plugin_manifests", m.ID) })

	for name, load := range map[string]func() (*models.PluginManifest, error){
		"GetByID":   func() (*models.PluginManifest, error) { return repo.GetByID(ctx, m.ID) },
		"GetBySlug": func() (*models.PluginManifest, error) { return repo.GetBySlug(ctx, m.Slug) },
	} {
		got, err := load()
		if err != nil || got == nil {
			t.Fatalf("%s: %+v (err %v)", name, got, err)
		}
		if got.ID != m.ID || got.Description != "does things" || got.Author != "zentria" || got.Version != "1.0.0" {
			t.Fatalf("%s scalar fields wrong: %+v", name, got)
		}
		if len(got.Permissions) != 2 || got.Permissions[0] != "contacts:read" || got.Permissions[1] != "deals:write" {
			t.Fatalf("%s permissions = %v", name, got.Permissions)
		}
		if len(got.HookRegistrations) != 1 || got.HookRegistrations[0].HookType != "before_create" || got.HookRegistrations[0].Priority != 3 {
			t.Fatalf("%s hook registrations = %+v", name, got.HookRegistrations)
		}
		if got.WASMBinaryHash != "abc123" || got.PluginType != models.PluginTypeWASM {
			t.Fatalf("%s hash/type = %q/%s", name, got.WASMBinaryHash, got.PluginType)
		}
	}

	if got, err := repo.GetBySlug(ctx, "no-such-slug-"+uuid.NewString()); err != nil || got != nil {
		t.Fatalf("unknown slug = %+v (err %v), want nil, nil", got, err)
	}
	if got, err := repo.GetByID(ctx, uuid.New()); err != nil || got != nil {
		t.Fatalf("unknown id = %+v (err %v), want nil, nil", got, err)
	}
}

func TestManifestRepository_GetWASMBinary(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant, ctx := freshTenant(t, pool, "manifest wasm")
	_, otherCtx := freshTenant(t, pool, "manifest wasm other")
	repo := repository.NewManifestRepository(pool)

	m := manifestFixture(&tenant, "wasm-"+uuid.NewString()[:8])
	m.PluginType = models.PluginTypeWASM
	m.WASMBinary = []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "plugin_manifests", m.ID) })

	got, err := repo.GetWASMBinary(ctx, m.ID)
	if err != nil {
		t.Fatalf("get binary: %v", err)
	}
	if string(got) != string(m.WASMBinary) {
		t.Fatalf("binary = %v, want %v", got, m.WASMBinary)
	}

	if got, err := repo.GetWASMBinary(otherCtx, m.ID); err != nil || got != nil {
		t.Fatalf("foreign tenant read the binary: %v (err %v)", got, err)
	}
	if got, err := repo.GetWASMBinary(ctx, uuid.New()); err != nil || got != nil {
		t.Fatalf("unknown id = %v (err %v), want nil, nil", got, err)
	}
}

func TestManifestRepository_HasActiveInstallations(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant, ctx := freshTenant(t, pool, "manifest installs")
	repo := repository.NewManifestRepository(pool)

	m := manifestFixture(&tenant, "installs-"+uuid.NewString()[:8])
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "plugin_manifests", m.ID) })

	has := func() bool {
		ok, err := repo.HasActiveInstallations(ctx, m.ID)
		if err != nil {
			t.Fatalf("has active: %v", err)
		}
		return ok
	}
	if has() {
		t.Fatal("manifest without installation reports active installations")
	}

	instID := testutil.SeedRow(t, pool, "plugin_installations", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    tenant,
		"manifest_id":  m.ID,
		"installed_by": uuid.New(),
		"status":       string(models.InstallationStatusUninstalled),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "plugin_installations", instID) })
	if has() {
		t.Fatal("an uninstalled installation counts as active")
	}

	for _, status := range []models.InstallationStatus{
		models.InstallationStatusPendingApproval,
		models.InstallationStatusActive,
		models.InstallationStatusDisabled,
		models.InstallationStatusError,
	} {
		if _, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
			`UPDATE plugin_installations SET status = $2 WHERE id = $1`, instID, string(status)); err != nil {
			t.Fatalf("set status %s: %v", status, err)
		}
		if !has() {
			t.Fatalf("installation in status %s is not counted", status)
		}
	}
}

func TestManifestRepository_DeleteRemovesOnlyTarget(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant, ctx := freshTenant(t, pool, "manifest delete")
	repo := repository.NewManifestRepository(pool)

	keep := manifestFixture(&tenant, "keep-"+uuid.NewString()[:8])
	drop := manifestFixture(&tenant, "drop-"+uuid.NewString()[:8])
	for _, m := range []*models.PluginManifest{keep, drop} {
		if err := repo.Create(ctx, m); err != nil {
			t.Fatalf("create: %v", err)
		}
		id := m.ID
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "plugin_manifests", id) })
	}

	if err := repo.Delete(ctx, drop.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	testutil.AssertRowCount(t, pool, ctx, "plugin_manifests", drop.ID, 0)
	testutil.AssertRowCount(t, pool, ctx, "plugin_manifests", keep.ID, 1)
}

func TestPermissionRepository_GrantListHasRevoke(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant, ctx := freshTenant(t, pool, "permission repo")
	repo := repository.NewPermissionRepository(pool)

	inst := seedInstallation(t, pool, tenant, "perm-repo-"+uuid.NewString()[:8])
	otherInst := seedInstallation(t, pool, tenant, "perm-repo-other-"+uuid.NewString()[:8])
	grantedBy := uuid.New()

	grant := func(installation uuid.UUID, perm string) {
		t.Helper()
		if err := repo.Grant(ctx, &models.PluginPermission{
			ID: uuid.New(), InstallationID: installation, Permission: perm,
			GrantedBy: grantedBy, GrantedAt: time.Now(),
		}); err != nil {
			t.Fatalf("grant %s: %v", perm, err)
		}
	}

	empty, err := repo.ListByInstallation(ctx, inst)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty list = %#v, want non-nil empty slice", empty)
	}

	grant(inst, "deals:write")
	grant(inst, "contacts:read")
	grant(inst, "contacts:read") // duplicate is a silent no-op, not an error
	grant(otherInst, "tasks:read")

	got, err := repo.ListByInstallation(ctx, inst)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 || got[0] != "contacts:read" || got[1] != "deals:write" {
		t.Fatalf("permissions = %v, want [contacts:read deals:write] sorted without duplicates", got)
	}

	for _, tc := range []struct {
		inst uuid.UUID
		perm string
		want bool
	}{
		{inst, "contacts:read", true},
		{inst, "deals:write", true},
		{inst, "tasks:read", false},
		{otherInst, "tasks:read", true},
		{otherInst, "contacts:read", false},
	} {
		ok, err := repo.HasPermission(ctx, tc.inst, tc.perm)
		if err != nil {
			t.Fatalf("has %s: %v", tc.perm, err)
		}
		if ok != tc.want {
			t.Fatalf("HasPermission(%s) = %v, want %v", tc.perm, ok, tc.want)
		}
	}

	if err := repo.RevokeAll(ctx, inst); err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	after, err := repo.ListByInstallation(ctx, inst)
	if err != nil || len(after) != 0 {
		t.Fatalf("after revoke: %v (err %v)", after, err)
	}
	if ok, _ := repo.HasPermission(ctx, inst, "contacts:read"); ok {
		t.Fatal("permission survived RevokeAll")
	}
	if rest, _ := repo.ListByInstallation(ctx, otherInst); len(rest) != 1 {
		t.Fatalf("RevokeAll touched another installation: %v", rest)
	}
}

func TestPermissionRepository_ForeignTenantSeesNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant, ctx := freshTenant(t, pool, "permission owner")
	_, otherCtx := freshTenant(t, pool, "permission other")
	repo := repository.NewPermissionRepository(pool)

	inst := seedInstallation(t, pool, tenant, "perm-foreign-"+uuid.NewString()[:8])
	if err := repo.Grant(ctx, &models.PluginPermission{
		ID: uuid.New(), InstallationID: inst, Permission: "contacts:read",
		GrantedBy: uuid.New(), GrantedAt: time.Now(),
	}); err != nil {
		t.Fatalf("grant: %v", err)
	}

	if list, err := repo.ListByInstallation(otherCtx, inst); err != nil || len(list) != 0 {
		t.Fatalf("foreign list = %v (err %v), want empty", list, err)
	}
	if ok, err := repo.HasPermission(otherCtx, inst, "contacts:read"); err != nil || ok {
		t.Fatalf("foreign HasPermission = %v (err %v), want false", ok, err)
	}
	if err := repo.RevokeAll(otherCtx, inst); err != nil {
		t.Fatalf("foreign revoke errored: %v", err)
	}
	if list, _ := repo.ListByInstallation(ctx, inst); len(list) != 1 {
		t.Fatalf("foreign RevokeAll deleted the owner's grant: %v", list)
	}
}

func TestValidationRuleRepository_ListUpdateScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant, ctx := freshTenant(t, pool, "validation lists")
	other, otherCtx := freshTenant(t, pool, "validation lists other")
	repo := repository.NewValidationRuleRepository(pool)

	now := time.Now().UTC()
	mk := func(ctx context.Context, owner uuid.UUID, name, entity string, prio int, enabled bool) *models.ValidationRule {
		r := &models.ValidationRule{
			ID: uuid.New(), TenantID: owner, Name: name, Description: "d-" + name,
			EntityType: entity, FieldName: "email", RuleType: models.ValidationRuleType("format"),
			RuleConfig: json.RawMessage(`{"pattern":"x"}`), ErrorMessage: "bad " + name,
			Priority: prio, Enabled: enabled, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "validation_rules", r.ID) })
		return r
	}
	// c is inserted before b so the name tie-break, not insertion order, decides.
	c := mk(ctx, tenant, "c-same-prio", "contact", 20, true)
	b := mk(ctx, tenant, "b-second", "contact", 20, true)
	a := mk(ctx, tenant, "a-first", "contact", 10, true)
	off := mk(ctx, tenant, "off", "contact", 5, false)
	deal := mk(ctx, tenant, "deal-rule", "deal", 1, true)
	foreign := mk(otherCtx, other, "foreign", "contact", 1, true)

	list, err := repo.List(ctx, tenant, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// priority ascending, name ascending as tie-break
	want := []uuid.UUID{deal.ID, off.ID, a.ID, b.ID, c.ID}
	if got := ids(list, func(r *models.ValidationRule) uuid.UUID { return r.ID }); !equalIDs(got, want) {
		t.Fatalf("list order = %v, want %v", got, want)
	}

	contacts, err := repo.List(ctx, tenant, "contact")
	if err != nil {
		t.Fatalf("list contact: %v", err)
	}
	want = []uuid.UUID{off.ID, a.ID, b.ID, c.ID}
	if got := ids(contacts, func(r *models.ValidationRule) uuid.UUID { return r.ID }); !equalIDs(got, want) {
		t.Fatalf("contact list = %v, want %v", got, want)
	}

	enabled, err := repo.ListEnabled(ctx, tenant, "contact")
	if err != nil {
		t.Fatalf("list enabled: %v", err)
	}
	if len(enabled) != 3 || enabled[0].ID != a.ID {
		t.Fatalf("enabled contact rules = %v, want a-first then the two prio-20 rules", ids(enabled, func(r *models.ValidationRule) uuid.UUID { return r.ID }))
	}
	for _, r := range enabled {
		if !r.Enabled || r.EntityType != "contact" || r.TenantID != tenant {
			t.Fatalf("ListEnabled returned %+v", r)
		}
	}

	// The tenant argument alone must not reach another tenant's rule.
	crossList, err := repo.List(ctx, other, "")
	if err != nil {
		t.Fatalf("cross list: %v", err)
	}
	if len(crossList) != 0 {
		t.Fatalf("tenant context saw %d rules of another tenant", len(crossList))
	}
	crossEnabled, err := repo.ListEnabled(ctx, other, "contact")
	if err != nil || len(crossEnabled) != 0 {
		t.Fatalf("cross enabled = %v (err %v)", crossEnabled, err)
	}

	// Update touches the editable columns only.
	a.Name, a.Description, a.ErrorMessage, a.Priority, a.Enabled = "a-renamed", "new d", "new msg", 99, false
	a.RuleConfig = json.RawMessage(`{"pattern":"y"}`)
	a.EntityType, a.FieldName = "deal", "other-field"
	if err := repo.Update(ctx, a); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := repo.GetByID(ctx, a.ID)
	if err != nil || got == nil {
		t.Fatalf("reload: %+v (err %v)", got, err)
	}
	if got.Name != "a-renamed" || got.Description != "new d" || got.ErrorMessage != "new msg" || got.Priority != 99 || got.Enabled {
		t.Fatalf("update not applied: %+v", got)
	}
	var cfg map[string]string
	if err := json.Unmarshal(got.RuleConfig, &cfg); err != nil || cfg["pattern"] != "y" {
		t.Fatalf("rule_config = %s (err %v)", got.RuleConfig, err)
	}
	if got.EntityType != "contact" || got.FieldName != "email" {
		t.Fatalf("update rewrote entity_type/field_name: %s/%s", got.EntityType, got.FieldName)
	}
	if !got.UpdatedAt.After(now) {
		t.Fatalf("updated_at not advanced: %s <= %s", got.UpdatedAt, now)
	}
	if neighbour, _ := repo.GetByID(ctx, b.ID); neighbour == nil || neighbour.Name != "b-second" {
		t.Fatalf("update changed a neighbour: %+v", neighbour)
	}

	// Update has no tenant predicate; RLS must stop it on a foreign row.
	foreign.Name = "hijacked"
	if err := repo.Update(ctx, foreign); err != nil {
		t.Fatalf("cross-tenant update errored: %v", err)
	}
	if kept, _ := repo.GetByID(otherCtx, foreign.ID); kept == nil || kept.Name != "foreign" {
		t.Fatalf("cross-tenant update changed the row: %+v", kept)
	}
}

func TestWorkflowRuleRepository_ListUpdateScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant, ctx := freshTenant(t, pool, "workflow lists")
	other, otherCtx := freshTenant(t, pool, "workflow lists other")
	repo := repository.NewWorkflowRuleRepository(pool)

	now := time.Now().UTC()
	mk := func(ctx context.Context, owner uuid.UUID, name, trigger string, prio int, enabled bool) *models.WorkflowRule {
		r := &models.WorkflowRule{
			ID: uuid.New(), TenantID: owner, Name: name, Description: "d-" + name,
			TriggerEvent: trigger, Conditions: json.RawMessage(`{"all":[]}`), Actions: json.RawMessage(`[{"type":"noop"}]`),
			Priority: prio, Enabled: enabled, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "workflow_rules", r.ID) })
		return r
	}
	// c is inserted before b so the name tie-break, not insertion order, decides.
	c := mk(ctx, tenant, "c-same-prio", "deal.won", 20, true)
	b := mk(ctx, tenant, "b-second", "deal.won", 20, true)
	a := mk(ctx, tenant, "a-first", "deal.won", 10, true)
	off := mk(ctx, tenant, "off", "deal.won", 5, false)
	lost := mk(ctx, tenant, "lost-rule", "deal.lost", 1, true)
	foreign := mk(otherCtx, other, "foreign", "deal.won", 1, true)

	list, err := repo.List(ctx, tenant, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := []uuid.UUID{lost.ID, off.ID, a.ID, b.ID, c.ID}
	if got := ids(list, func(r *models.WorkflowRule) uuid.UUID { return r.ID }); !equalIDs(got, want) {
		t.Fatalf("list order = %v, want %v", got, want)
	}

	won, err := repo.List(ctx, tenant, "deal.won")
	if err != nil {
		t.Fatalf("list won: %v", err)
	}
	want = []uuid.UUID{off.ID, a.ID, b.ID, c.ID}
	if got := ids(won, func(r *models.WorkflowRule) uuid.UUID { return r.ID }); !equalIDs(got, want) {
		t.Fatalf("trigger list = %v, want %v", got, want)
	}

	enabled, err := repo.ListEnabled(ctx, tenant, "deal.won")
	if err != nil {
		t.Fatalf("list enabled: %v", err)
	}
	if len(enabled) != 3 || enabled[0].ID != a.ID {
		t.Fatalf("enabled rules = %v, want a-first then the two prio-20 rules", ids(enabled, func(r *models.WorkflowRule) uuid.UUID { return r.ID }))
	}
	for _, r := range enabled {
		if !r.Enabled || r.TriggerEvent != "deal.won" || r.TenantID != tenant {
			t.Fatalf("ListEnabled returned %+v", r)
		}
	}

	if cross, err := repo.List(ctx, other, ""); err != nil || len(cross) != 0 {
		t.Fatalf("tenant context saw another tenant's rules: %d (err %v)", len(cross), err)
	}
	if cross, err := repo.ListEnabled(ctx, other, "deal.won"); err != nil || len(cross) != 0 {
		t.Fatalf("cross enabled = %v (err %v)", cross, err)
	}

	a.Name, a.Description, a.Priority, a.Enabled = "a-renamed", "new d", 99, false
	a.Conditions = json.RawMessage(`{"all":[1]}`)
	a.Actions = json.RawMessage(`[{"type":"mail"}]`)
	a.TriggerEvent = "deal.lost"
	if err := repo.Update(ctx, a); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := repo.GetByID(ctx, a.ID)
	if err != nil || got == nil {
		t.Fatalf("reload: %+v (err %v)", got, err)
	}
	if got.Name != "a-renamed" || got.Description != "new d" || got.Priority != 99 || got.Enabled {
		t.Fatalf("update not applied: %+v", got)
	}
	if string(got.Conditions) != `{"all": [1]}` || string(got.Actions) != `[{"type": "mail"}]` {
		t.Fatalf("conditions/actions = %s / %s", got.Conditions, got.Actions)
	}
	if got.TriggerEvent != "deal.won" {
		t.Fatalf("update rewrote trigger_event: %s", got.TriggerEvent)
	}
	if !got.UpdatedAt.After(now) {
		t.Fatalf("updated_at not advanced: %s <= %s", got.UpdatedAt, now)
	}
	if neighbour, _ := repo.GetByID(ctx, b.ID); neighbour == nil || neighbour.Name != "b-second" {
		t.Fatalf("update changed a neighbour: %+v", neighbour)
	}

	foreign.Name = "hijacked"
	if err := repo.Update(ctx, foreign); err != nil {
		t.Fatalf("cross-tenant update errored: %v", err)
	}
	if kept, _ := repo.GetByID(otherCtx, foreign.ID); kept == nil || kept.Name != "foreign" {
		t.Fatalf("cross-tenant update changed the row: %+v", kept)
	}
}
