package settings_test

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/modules"
	"github.com/kmuhub/kmuhub/internal/settings"
)

// ============================================================================
// In-memory fakes
// ============================================================================

type fakeRepo struct {
	leads          []*settings.ModuleLead
	tenantSettings map[string][]*settings.SettingEntry     // key: tenantID+":"+moduleID
	userSettings   map[string][]*settings.SettingEntry     // key: tenantID+":"+userID+":"+moduleID
	activations    map[string]bool                         // key: tenantID+":"+moduleID
	grantCounts    map[string]int32                        // key: tenantID+":"+moduleID
	subscriptions  map[string]*settings.TenantSubscription // key: tenantID
	valueSets      map[string]*settings.ValueSet           // key: tenantID+":"+setKey
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		tenantSettings: make(map[string][]*settings.SettingEntry),
		userSettings:   make(map[string][]*settings.SettingEntry),
		valueSets:      make(map[string]*settings.ValueSet),
	}
}

func tsKey(tenantID uuid.UUID, moduleID string) string {
	return tenantID.String() + ":" + moduleID
}

func usKey(tenantID, userID uuid.UUID, moduleID string) string {
	return tenantID.String() + ":" + userID.String() + ":" + moduleID
}

func (r *fakeRepo) ListModuleLeads(_ context.Context, tenantID uuid.UUID, userID *uuid.UUID, moduleID *string) ([]*settings.ModuleLead, error) {
	var out []*settings.ModuleLead
	for _, l := range r.leads {
		if l.TenantID != tenantID {
			continue
		}
		if userID != nil && l.UserID != *userID {
			continue
		}
		if moduleID != nil && l.ModuleID != *moduleID {
			continue
		}
		out = append(out, l)
	}
	return out, nil
}

func (r *fakeRepo) GetModuleLead(_ context.Context, tenantID, userID uuid.UUID, moduleID string) (*settings.ModuleLead, error) {
	for _, l := range r.leads {
		if l.TenantID == tenantID && l.UserID == userID && l.ModuleID == moduleID {
			return l, nil
		}
	}
	return nil, settings.ErrNotFound
}

func (r *fakeRepo) GrantModuleLead(_ context.Context, tenantID, userID uuid.UUID, moduleID string, grantedBy *uuid.UUID) (*settings.ModuleLead, error) {
	ml := &settings.ModuleLead{TenantID: tenantID, UserID: userID, ModuleID: moduleID, GrantedBy: grantedBy}
	r.leads = append(r.leads, ml)
	return ml, nil
}

func (r *fakeRepo) RevokeModuleLead(_ context.Context, tenantID, userID uuid.UUID, moduleID string) error {
	newLeads := r.leads[:0]
	for _, l := range r.leads {
		if l.TenantID == tenantID && l.UserID == userID && l.ModuleID == moduleID {
			continue
		}
		newLeads = append(newLeads, l)
	}
	r.leads = newLeads
	return nil
}

func (r *fakeRepo) IsModuleLead(_ context.Context, tenantID, userID uuid.UUID, moduleID string) (bool, error) {
	for _, l := range r.leads {
		if l.TenantID == tenantID && l.UserID == userID && l.ModuleID == moduleID {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRepo) ListLeadModulesForUser(_ context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	var mods []string
	for _, l := range r.leads {
		if l.TenantID == tenantID && l.UserID == userID {
			mods = append(mods, l.ModuleID)
		}
	}
	return mods, nil
}

func (r *fakeRepo) ListModuleGrants(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ *string) ([]*settings.UserModuleGrant, error) {
	return nil, nil
}

func (r *fakeRepo) GrantModuleAccess(_ context.Context, tenantID, userID uuid.UUID, moduleID string, grantedBy *uuid.UUID) (*settings.UserModuleGrant, error) {
	return &settings.UserModuleGrant{TenantID: tenantID, UserID: userID, ModuleID: moduleID, GrantedBy: grantedBy}, nil
}

func (r *fakeRepo) RevokeModuleAccess(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}

func (r *fakeRepo) BulkRevokeModuleAccess(_ context.Context, _ uuid.UUID, refs []settings.ModuleGrantRef) (int, error) {
	return len(refs), nil
}

func (r *fakeRepo) GetTenantSettings(_ context.Context, tenantID uuid.UUID, moduleID string) ([]*settings.SettingEntry, error) {
	return r.tenantSettings[tsKey(tenantID, moduleID)], nil
}

func (r *fakeRepo) PutTenantSettings(_ context.Context, tenantID uuid.UUID, moduleID string, _ *uuid.UUID, entries []*settings.SettingEntry) ([]*settings.SettingEntry, error) {
	k := tsKey(tenantID, moduleID)
	existing := make(map[string]*settings.SettingEntry)
	for _, e := range r.tenantSettings[k] {
		existing[e.Key] = e
	}
	for _, e := range entries {
		existing[e.Key] = e
	}
	result := make([]*settings.SettingEntry, 0, len(existing))
	for _, e := range existing {
		result = append(result, e)
	}
	r.tenantSettings[k] = result
	return result, nil
}

func (r *fakeRepo) GetUserSettings(_ context.Context, tenantID, userID uuid.UUID, moduleID string) ([]*settings.SettingEntry, error) {
	return r.userSettings[usKey(tenantID, userID, moduleID)], nil
}

func (r *fakeRepo) PutUserSettings(_ context.Context, tenantID, userID uuid.UUID, moduleID string, entries []*settings.SettingEntry) ([]*settings.SettingEntry, error) {
	k := usKey(tenantID, userID, moduleID)
	existing := make(map[string]*settings.SettingEntry)
	for _, e := range r.userSettings[k] {
		existing[e.Key] = e
	}
	for _, e := range entries {
		existing[e.Key] = e
	}
	result := make([]*settings.SettingEntry, 0, len(existing))
	for _, e := range existing {
		result = append(result, e)
	}
	r.userSettings[k] = result
	return result, nil
}

// fakeRoleChecker always returns a configurable isAdmin value.
type fakeRoleChecker struct {
	adminIDs map[uuid.UUID]bool
}

func (f *fakeRoleChecker) IsAdmin(_ context.Context, _ uuid.UUID, userID uuid.UUID) (bool, error) {
	return f.adminIDs[userID], nil
}

// ============================================================================
// Test helpers
// ============================================================================

func rawJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func newService(repo *fakeRepo, adminIDs ...uuid.UUID) *settings.Service {
	adminSet := make(map[uuid.UUID]bool)
	for _, id := range adminIDs {
		adminSet[id] = true
	}
	return settings.NewService(repo, &fakeRoleChecker{adminIDs: adminSet})
}

// ============================================================================
// Resolve-order tests
// ============================================================================

func TestGetResolvedSettings_UserWinsOverTenant(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	moduleID := "calendar"

	repo := newFakeRepo()
	svc := newService(repo)

	// Seed tenant setting: defaultView = "month"
	_, err := repo.PutTenantSettings(context.Background(), tenantID, moduleID, nil, []*settings.SettingEntry{
		{Key: "defaultView", Value: rawJSON(t, "month")},
		{Key: "workStartHour", Value: rawJSON(t, 8)},
	})
	require.NoError(t, err)

	// Seed user override: defaultView = "week"
	_, err = repo.PutUserSettings(context.Background(), tenantID, userID, moduleID, []*settings.SettingEntry{
		{Key: "defaultView", Value: rawJSON(t, "week")},
	})
	require.NoError(t, err)

	resolved, err := svc.GetResolvedSettings(context.Background(), tenantID, userID, moduleID)
	require.NoError(t, err)

	// Build a map for easy assertion.
	m := make(map[string]json.RawMessage)
	for _, e := range resolved {
		m[e.Key] = e.Value
	}

	// User's "week" should override tenant's "month".
	var view string
	require.NoError(t, json.Unmarshal(m["defaultView"], &view))
	assert.Equal(t, "week", view, "user override must win over tenant default")

	// workStartHour only set at tenant level → falls through.
	var startHour float64
	require.NoError(t, json.Unmarshal(m["workStartHour"], &startHour))
	assert.Equal(t, float64(8), startHour)
}

func TestGetResolvedSettings_TenantOnlyWhenNoUserOverride(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	moduleID := "work"

	repo := newFakeRepo()
	svc := newService(repo)

	_, err := repo.PutTenantSettings(context.Background(), tenantID, moduleID, nil, []*settings.SettingEntry{
		{Key: "billableByDefault", Value: rawJSON(t, true)},
	})
	require.NoError(t, err)

	resolved, err := svc.GetResolvedSettings(context.Background(), tenantID, userID, moduleID)
	require.NoError(t, err)

	m := make(map[string]json.RawMessage)
	for _, e := range resolved {
		m[e.Key] = e.Value
	}

	var billable bool
	require.NoError(t, json.Unmarshal(m["billableByDefault"], &billable))
	assert.True(t, billable)
}

func TestGetResolvedSettings_EmptyWhenNothingSet(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)

	resolved, err := svc.GetResolvedSettings(context.Background(), uuid.New(), uuid.New(), "crm")
	require.NoError(t, err)
	assert.Empty(t, resolved)
}

func TestGetResolvedSettings_InvalidModuleID(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)
	_, err := svc.GetResolvedSettings(context.Background(), uuid.New(), uuid.New(), "")
	assert.ErrorIs(t, err, settings.ErrInvalidModuleID)
}

// ============================================================================
// RBAC tests — tenant-scope write
// ============================================================================

func TestPutTenantSettings_AdminMayWrite(t *testing.T) {
	adminID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID) // adminID is admin

	entries, err := svc.PutTenantSettings(context.Background(), tenantID, adminID, "crm", []*settings.SettingEntry{
		{Key: "defaultCurrency", Value: rawJSON(t, "EUR")},
	})
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestPutTenantSettings_ModuleLeadMayWrite(t *testing.T) {
	leadID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo) // no admin

	// Grant lead rights in the fake repo directly.
	_, err := repo.GrantModuleLead(context.Background(), tenantID, leadID, "work", nil)
	require.NoError(t, err)

	entries, err := svc.PutTenantSettings(context.Background(), tenantID, leadID, "work", []*settings.SettingEntry{
		{Key: "billableByDefault", Value: rawJSON(t, false)},
	})
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestPutTenantSettings_NonLeadIsRejected(t *testing.T) {
	regularUser := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo) // no admin, regularUser is not a lead

	_, err := svc.PutTenantSettings(context.Background(), tenantID, regularUser, "work", []*settings.SettingEntry{
		{Key: "billableByDefault", Value: rawJSON(t, false)},
	})
	assert.ErrorIs(t, err, settings.ErrNotModuleLead)
}

// ============================================================================
// Cross-tenant isolation tests
// ============================================================================

func TestTenantSettings_CrossTenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	adminA := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminA)

	// Admin A writes to tenant A.
	_, err := svc.PutTenantSettings(context.Background(), tenantA, adminA, "crm", []*settings.SettingEntry{
		{Key: "secret", Value: rawJSON(t, "a-only")},
	})
	require.NoError(t, err)

	// Reading from tenant B should return empty — not tenant A's data.
	entries, err := svc.GetTenantSettings(context.Background(), tenantB, "crm")
	require.NoError(t, err)
	assert.Empty(t, entries, "tenant B must not see tenant A settings")
}

func TestModuleLeads_CrossTenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	adminA := uuid.New()
	userA := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminA)

	// Grant in tenant A.
	_, err := svc.GrantModuleLead(context.Background(), tenantA, adminA, userA, "crm")
	require.NoError(t, err)

	// List in tenant B — should return nothing.
	leads, err := svc.ListModuleLeads(context.Background(), tenantB, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, leads, "tenant B must not see tenant A module leads")
}

// ============================================================================
// Module-lead management tests
// ============================================================================

func TestGrantRevoke_ModuleLead(t *testing.T) {
	adminID := uuid.New()
	userID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID)

	// Grant
	ml, err := svc.GrantModuleLead(context.Background(), tenantID, adminID, userID, "finance")
	require.NoError(t, err)
	assert.Equal(t, "finance", ml.ModuleID)

	// My leads
	moduleIDs, isAdmin, err := svc.GetMyModuleLeads(context.Background(), tenantID, userID)
	require.NoError(t, err)
	assert.False(t, isAdmin)
	assert.Contains(t, moduleIDs, "finance")

	// Revoke
	err = svc.RevokeModuleLead(context.Background(), tenantID, adminID, userID, "finance")
	require.NoError(t, err)

	moduleIDs, _, err = svc.GetMyModuleLeads(context.Background(), tenantID, userID)
	require.NoError(t, err)
	assert.NotContains(t, moduleIDs, "finance")
}

func TestGetMyModuleLeads_AdminGetsAll(t *testing.T) {
	adminID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID)

	moduleIDs, isAdmin, err := svc.GetMyModuleLeads(context.Background(), tenantID, adminID)
	require.NoError(t, err)
	assert.True(t, isAdmin, "admin flag must be set")
	assert.Empty(t, moduleIDs, "admin returns empty list (FE short-circuits via is_admin flag)")
}

// ============================================================================
// Licensing fakes
// ============================================================================

func (r *fakeRepo) ListModuleActivations(_ context.Context, tenantID uuid.UUID) (map[string]bool, error) {
	out := make(map[string]bool)
	for k, v := range r.activations {
		t, moduleID, _ := strings.Cut(k, ":")
		if t == tenantID.String() {
			out[moduleID] = v
		}
	}
	return out, nil
}

func (r *fakeRepo) SetModuleActivation(_ context.Context, tenantID uuid.UUID, moduleID string, active bool, _ *uuid.UUID) error {
	if r.activations == nil {
		r.activations = make(map[string]bool)
	}
	r.activations[tenantID.String()+":"+moduleID] = active
	return nil
}

func (r *fakeRepo) CountGrantsByModule(_ context.Context, tenantID uuid.UUID) (map[string]int32, error) {
	out := make(map[string]int32)
	for k, v := range r.grantCounts {
		t, moduleID, _ := strings.Cut(k, ":")
		if t == tenantID.String() {
			out[moduleID] = v
		}
	}
	return out, nil
}

func (r *fakeRepo) GetTenantSubscription(_ context.Context, tenantID uuid.UUID) (*settings.TenantSubscription, error) {
	sub, ok := r.subscriptions[tenantID.String()]
	if !ok {
		return nil, settings.ErrNotFound
	}
	return sub, nil
}

func valueSetKey(tenantID uuid.UUID, setKey string) string {
	return tenantID.String() + ":" + setKey
}

func (r *fakeRepo) ListValueSetOverrides(_ context.Context, tenantID uuid.UUID) ([]*settings.ValueSet, error) {
	out := make([]*settings.ValueSet, 0)
	prefix := tenantID.String() + ":"
	for k, set := range r.valueSets {
		if strings.HasPrefix(k, prefix) {
			out = append(out, set)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (r *fakeRepo) GetValueSetOverride(_ context.Context, tenantID uuid.UUID, setKey string) (*settings.ValueSet, error) {
	set, ok := r.valueSets[valueSetKey(tenantID, setKey)]
	if !ok {
		return nil, settings.ErrNotFound
	}
	return set, nil
}

func (r *fakeRepo) UpsertValueSetOverride(_ context.Context, tenantID uuid.UUID, _ *uuid.UUID, set *settings.ValueSet) (*settings.ValueSet, error) {
	stored := &settings.ValueSet{Key: set.Key, Name: set.Name}
	stored.Options = append(stored.Options, set.Options...)
	r.valueSets[valueSetKey(tenantID, set.Key)] = stored
	return stored, nil
}

func (r *fakeRepo) DeleteValueSetOverride(_ context.Context, tenantID uuid.UUID, setKey string) error {
	k := valueSetKey(tenantID, setKey)
	if _, ok := r.valueSets[k]; !ok {
		return settings.ErrNotFound
	}
	delete(r.valueSets, k)
	return nil
}

// ============================================================================
// Licensing tests
// ============================================================================

// The catalogue drives the response, not the activation table: a tenant that
// has never touched the licence tab still sees every module, all inactive.
func TestGetTenantLicense_ReturnsWholeCatalogue(t *testing.T) {
	tenantID := uuid.New()
	svc := newService(newFakeRepo())

	mods, err := svc.GetTenantLicense(context.Background(), tenantID)
	require.NoError(t, err)
	require.Len(t, mods, len(modules.Catalog))
	for _, m := range mods {
		assert.False(t, m.Active, "module %s should be inactive without a row", m.ModuleID)
		assert.Zero(t, m.AssignedSeats)
	}
}

func TestGetTenantLicense_ReportsSeatsOnlyWhileActive(t *testing.T) {
	tenantID := uuid.New()
	other := uuid.New()
	repo := newFakeRepo()
	repo.activations = map[string]bool{
		tenantID.String() + ":crm":  true,
		tenantID.String() + ":wiki": false,
		// A different tenant with the same module must not leak in.
		other.String() + ":inventar": true,
	}
	repo.grantCounts = map[string]int32{
		tenantID.String() + ":crm":  7,
		tenantID.String() + ":wiki": 3,
	}
	svc := newService(repo)

	mods, err := svc.GetTenantLicense(context.Background(), tenantID)
	require.NoError(t, err)

	byID := make(map[string]*settings.TenantModule, len(mods))
	for _, m := range mods {
		byID[m.ModuleID] = m
	}
	assert.True(t, byID["crm"].Active)
	assert.Equal(t, int32(7), byID["crm"].AssignedSeats)
	// Deactivated: the grants stay in the database, the cost view reads 0.
	assert.False(t, byID["wiki"].Active)
	assert.Zero(t, byID["wiki"].AssignedSeats)
	assert.False(t, byID["inventar"].Active, "another tenant's activation must not leak")
}

func TestGetTenantLicense_CarriesFlagKeyForGateway(t *testing.T) {
	svc := newService(newFakeRepo())

	mods, err := svc.GetTenantLicense(context.Background(), uuid.New())
	require.NoError(t, err)

	byID := make(map[string]*settings.TenantModule, len(mods))
	for _, m := range mods {
		byID[m.ModuleID] = m
	}
	assert.Equal(t, "modules.buchhaltung", byID["finance"].FlagKey)
	assert.Empty(t, byID["crm"].FlagKey, "crm ships in every deployment")
}

func TestSetTenantModuleActive_RejectsUnknownModule(t *testing.T) {
	svc := newService(newFakeRepo())

	_, err := svc.SetTenantModuleActive(context.Background(), uuid.New(), "not-a-module", true, nil)
	assert.ErrorIs(t, err, settings.ErrUnknownModule)

	_, err = svc.SetTenantModuleActive(context.Background(), uuid.New(), "", true, nil)
	assert.ErrorIs(t, err, settings.ErrInvalidModuleID)
}

func TestSetTenantModuleActive_PersistsAndReadsBack(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeRepo()
	repo.grantCounts = map[string]int32{tenantID.String() + ":helpdesk": 5}
	svc := newService(repo)

	m, err := svc.SetTenantModuleActive(context.Background(), tenantID, "helpdesk", true, nil)
	require.NoError(t, err)
	assert.True(t, m.Active)
	assert.Equal(t, int32(5), m.AssignedSeats)
	assert.Equal(t, "industry", m.Group)

	m, err = svc.SetTenantModuleActive(context.Background(), tenantID, "helpdesk", false, nil)
	require.NoError(t, err)
	assert.False(t, m.Active)
	assert.Zero(t, m.AssignedSeats)

	mods, err := svc.GetTenantLicense(context.Background(), tenantID)
	require.NoError(t, err)
	for _, mod := range mods {
		if mod.ModuleID == "helpdesk" {
			assert.False(t, mod.Active)
		}
	}
}

func TestGetTenantSubscription_MissingTenant(t *testing.T) {
	svc := newService(newFakeRepo())

	_, err := svc.GetTenantSubscription(context.Background(), uuid.New())
	assert.ErrorIs(t, err, settings.ErrNotFound)
}

func TestGetTenantSubscription_ReturnsBookedPlan(t *testing.T) {
	tenantID := uuid.New()
	seats := int32(14)
	repo := newFakeRepo()
	repo.subscriptions = map[string]*settings.TenantSubscription{
		tenantID.String(): {PlanType: "cosmi", SupportTier: "priority", Status: "active", TotalSeats: &seats},
	}
	svc := newService(repo)

	sub, err := svc.GetTenantSubscription(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, "cosmi", sub.PlanType)
	assert.Equal(t, "priority", sub.SupportTier)
	require.NotNil(t, sub.TotalSeats)
	assert.Equal(t, int32(14), *sub.TotalSeats)
	assert.Nil(t, sub.BillingPeriodEnd)
}
