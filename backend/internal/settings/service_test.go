package settings_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/settings"
)

// ============================================================================
// In-memory fakes
// ============================================================================

type fakeRepo struct {
	leads          []*settings.ModuleLead
	tenantSettings map[string][]*settings.SettingEntry // key: tenantID+":"+moduleID
	userSettings   map[string][]*settings.SettingEntry // key: tenantID+":"+userID+":"+moduleID
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		tenantSettings: make(map[string][]*settings.SettingEntry),
		userSettings:   make(map[string][]*settings.SettingEntry),
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
