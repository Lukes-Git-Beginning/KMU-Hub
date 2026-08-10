package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/modules"
	"github.com/kmuhub/kmuhub/internal/settings"
	settingsv1 "github.com/kmuhub/kmuhub/proto/settings/v1"
)

// errStubRepoFailure is a generic non-sentinel error used to reach the fallback
// "codes.Internal" branch of every mapping in settings_grpc.go.
var errStubRepoFailure = errors.New("stub repository failure")

// ---------------------------------------------------------------------------
// stub settings.Repository
// ---------------------------------------------------------------------------

func tKey(tenantID uuid.UUID, moduleID string) string {
	return tenantID.String() + ":" + moduleID
}

func uKey(tenantID, userID uuid.UUID, moduleID string) string {
	return tenantID.String() + ":" + userID.String() + ":" + moduleID
}

func vKey(tenantID uuid.UUID, key string) string {
	return tenantID.String() + ":" + key
}

// stubSettingsRepo is an in-memory settings.Repository double. forceErr, when
// set, is returned by every method — used to reach the generic
// "slog.Error + codes.Internal" branch every RPC in settings_grpc.go shares,
// without needing 20 near-identical injection points.
type stubSettingsRepo struct {
	forceErr error

	leads          []*settings.ModuleLead
	grants         []*settings.UserModuleGrant
	tenantSettings map[string][]*settings.SettingEntry
	userSettings   map[string][]*settings.SettingEntry
	activations    map[string]bool
	grantCounts    map[string]int32
	subscriptions  map[string]*settings.TenantSubscription
	valueSets      map[string]*settings.ValueSet
}

func newStubSettingsRepo() *stubSettingsRepo {
	return &stubSettingsRepo{
		tenantSettings: make(map[string][]*settings.SettingEntry),
		userSettings:   make(map[string][]*settings.SettingEntry),
		activations:    make(map[string]bool),
		grantCounts:    make(map[string]int32),
		subscriptions:  make(map[string]*settings.TenantSubscription),
		valueSets:      make(map[string]*settings.ValueSet),
	}
}

func (r *stubSettingsRepo) ListModuleLeads(_ context.Context, tenantID uuid.UUID, userID *uuid.UUID, moduleID *string) ([]*settings.ModuleLead, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
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

func (r *stubSettingsRepo) GetModuleLead(_ context.Context, tenantID, userID uuid.UUID, moduleID string) (*settings.ModuleLead, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	for _, l := range r.leads {
		if l.TenantID == tenantID && l.UserID == userID && l.ModuleID == moduleID {
			return l, nil
		}
	}
	return nil, settings.ErrNotFound
}

func (r *stubSettingsRepo) GrantModuleLead(_ context.Context, tenantID, userID uuid.UUID, moduleID string, grantedBy *uuid.UUID) (*settings.ModuleLead, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	ml := &settings.ModuleLead{TenantID: tenantID, UserID: userID, ModuleID: moduleID, GrantedBy: grantedBy, GrantedAt: time.Now()}
	r.leads = append(r.leads, ml)
	return ml, nil
}

func (r *stubSettingsRepo) RevokeModuleLead(_ context.Context, tenantID, userID uuid.UUID, moduleID string) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	kept := r.leads[:0]
	for _, l := range r.leads {
		if l.TenantID == tenantID && l.UserID == userID && l.ModuleID == moduleID {
			continue
		}
		kept = append(kept, l)
	}
	r.leads = kept
	return nil
}

func (r *stubSettingsRepo) IsModuleLead(_ context.Context, tenantID, userID uuid.UUID, moduleID string) (bool, error) {
	if r.forceErr != nil {
		return false, r.forceErr
	}
	for _, l := range r.leads {
		if l.TenantID == tenantID && l.UserID == userID && l.ModuleID == moduleID {
			return true, nil
		}
	}
	return false, nil
}

func (r *stubSettingsRepo) ListLeadModulesForUser(_ context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	var out []string
	for _, l := range r.leads {
		if l.TenantID == tenantID && l.UserID == userID {
			out = append(out, l.ModuleID)
		}
	}
	return out, nil
}

func (r *stubSettingsRepo) ListModuleGrants(_ context.Context, tenantID uuid.UUID, userID *uuid.UUID, moduleID *string) ([]*settings.UserModuleGrant, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	var out []*settings.UserModuleGrant
	for _, g := range r.grants {
		if g.TenantID != tenantID {
			continue
		}
		if userID != nil && g.UserID != *userID {
			continue
		}
		if moduleID != nil && g.ModuleID != *moduleID {
			continue
		}
		out = append(out, g)
	}
	return out, nil
}

func (r *stubSettingsRepo) GrantModuleAccess(_ context.Context, tenantID, userID uuid.UUID, moduleID string, grantedBy *uuid.UUID) (*settings.UserModuleGrant, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	g := &settings.UserModuleGrant{TenantID: tenantID, UserID: userID, ModuleID: moduleID, GrantedBy: grantedBy, GrantedAt: time.Now()}
	r.grants = append(r.grants, g)
	return g, nil
}

func (r *stubSettingsRepo) RevokeModuleAccess(_ context.Context, tenantID, userID uuid.UUID, moduleID string) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	kept := r.grants[:0]
	for _, g := range r.grants {
		if g.TenantID == tenantID && g.UserID == userID && g.ModuleID == moduleID {
			continue
		}
		kept = append(kept, g)
	}
	r.grants = kept
	return nil
}

func (r *stubSettingsRepo) BulkRevokeModuleAccess(_ context.Context, tenantID uuid.UUID, refs []settings.ModuleGrantRef) (int, error) {
	if r.forceErr != nil {
		return 0, r.forceErr
	}
	n := 0
	for _, ref := range refs {
		kept := r.grants[:0]
		for _, g := range r.grants {
			if g.TenantID == tenantID && g.UserID == ref.UserID && g.ModuleID == ref.ModuleID {
				n++
				continue
			}
			kept = append(kept, g)
		}
		r.grants = kept
	}
	return n, nil
}

func (r *stubSettingsRepo) GetTenantSettings(_ context.Context, tenantID uuid.UUID, moduleID string) ([]*settings.SettingEntry, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	return r.tenantSettings[tKey(tenantID, moduleID)], nil
}

func (r *stubSettingsRepo) PutTenantSettings(_ context.Context, tenantID uuid.UUID, moduleID string, _ *uuid.UUID, entries []*settings.SettingEntry) ([]*settings.SettingEntry, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	k := tKey(tenantID, moduleID)
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

func (r *stubSettingsRepo) GetUserSettings(_ context.Context, tenantID, userID uuid.UUID, moduleID string) ([]*settings.SettingEntry, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	return r.userSettings[uKey(tenantID, userID, moduleID)], nil
}

func (r *stubSettingsRepo) PutUserSettings(_ context.Context, tenantID, userID uuid.UUID, moduleID string, entries []*settings.SettingEntry) ([]*settings.SettingEntry, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	k := uKey(tenantID, userID, moduleID)
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

func (r *stubSettingsRepo) ReplaceUserSettings(_ context.Context, tenantID, userID uuid.UUID, moduleID string, entries []*settings.SettingEntry) ([]*settings.SettingEntry, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	k := uKey(tenantID, userID, moduleID)
	result := make([]*settings.SettingEntry, len(entries))
	copy(result, entries)
	r.userSettings[k] = result
	return result, nil
}

func (r *stubSettingsRepo) ListValueSetOverrides(_ context.Context, tenantID uuid.UUID) ([]*settings.ValueSet, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	var out []*settings.ValueSet
	prefix := tenantID.String() + ":"
	for k, v := range r.valueSets {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, v)
		}
	}
	return out, nil
}

func (r *stubSettingsRepo) GetValueSetOverride(_ context.Context, tenantID uuid.UUID, setKey string) (*settings.ValueSet, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	v, ok := r.valueSets[vKey(tenantID, setKey)]
	if !ok {
		return nil, settings.ErrNotFound
	}
	return v, nil
}

func (r *stubSettingsRepo) UpsertValueSetOverride(_ context.Context, tenantID uuid.UUID, _ *uuid.UUID, set *settings.ValueSet) (*settings.ValueSet, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	stored := *set
	r.valueSets[vKey(tenantID, set.Key)] = &stored
	return &stored, nil
}

func (r *stubSettingsRepo) DeleteValueSetOverride(_ context.Context, tenantID uuid.UUID, setKey string) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	k := vKey(tenantID, setKey)
	if _, ok := r.valueSets[k]; !ok {
		return settings.ErrNotFound
	}
	delete(r.valueSets, k)
	return nil
}

func (r *stubSettingsRepo) ListModuleActivations(_ context.Context, tenantID uuid.UUID) (map[string]bool, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	out := make(map[string]bool)
	prefix := tenantID.String() + ":"
	for k, v := range r.activations {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out[k[len(prefix):]] = v
		}
	}
	return out, nil
}

func (r *stubSettingsRepo) SetModuleActivation(_ context.Context, tenantID uuid.UUID, moduleID string, active bool, _ *uuid.UUID) error {
	if r.forceErr != nil {
		return r.forceErr
	}
	r.activations[tKey(tenantID, moduleID)] = active
	return nil
}

func (r *stubSettingsRepo) CountGrantsByModule(_ context.Context, tenantID uuid.UUID) (map[string]int32, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	out := make(map[string]int32)
	prefix := tenantID.String() + ":"
	for k, v := range r.grantCounts {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out[k[len(prefix):]] = v
		}
	}
	return out, nil
}

func (r *stubSettingsRepo) GetTenantSubscription(_ context.Context, tenantID uuid.UUID) (*settings.TenantSubscription, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	sub, ok := r.subscriptions[tenantID.String()]
	if !ok {
		return nil, settings.ErrNotFound
	}
	return sub, nil
}

// ---------------------------------------------------------------------------
// stub settings.RoleChecker
// ---------------------------------------------------------------------------

type stubRoleChecker struct {
	admins map[string]bool
	err    error
}

func newStubRoleChecker() *stubRoleChecker {
	return &stubRoleChecker{admins: make(map[string]bool)}
}

func (r *stubRoleChecker) IsAdmin(_ context.Context, tenantID, userID uuid.UUID) (bool, error) {
	if r.err != nil {
		return false, r.err
	}
	return r.admins[tenantID.String()+":"+userID.String()], nil
}

func (r *stubRoleChecker) setAdmin(tenantID, userID uuid.UUID) {
	r.admins[tenantID.String()+":"+userID.String()] = true
}

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

func newTestSettingsServer() (*SettingsGRPCServer, *stubSettingsRepo, *stubRoleChecker) {
	repo := newStubSettingsRepo()
	roles := newStubRoleChecker()
	svc := settings.NewService(repo, roles)
	return NewSettingsGRPCServer(svc), repo, roles
}

func callerCtx(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDKey, userID.String())
}

// ---------------------------------------------------------------------------
// Module-lead RPCs
// ---------------------------------------------------------------------------

func TestListModuleLeads_InvalidIDs(t *testing.T) {
	srv, _, _ := newTestSettingsServer()

	_, err := srv.ListModuleLeads(context.Background(), &settingsv1.ListModuleLeadsRequest{TenantId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	tenantID := uuid.New().String()
	badUser := "not-a-uuid"
	_, err = srv.ListModuleLeads(context.Background(), &settingsv1.ListModuleLeadsRequest{TenantId: tenantID, UserId: &badUser})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListModuleLeads_FiltersAndSuccess(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()
	tenantID := uuid.New()
	userA := uuid.New()
	userB := uuid.New()
	repo.leads = []*settings.ModuleLead{
		{TenantID: tenantID, UserID: userA, ModuleID: "crm"},
		{TenantID: tenantID, UserID: userB, ModuleID: "finance"},
		{TenantID: uuid.New(), UserID: userA, ModuleID: "crm"}, // different tenant
	}

	resp, err := srv.ListModuleLeads(context.Background(), &settingsv1.ListModuleLeadsRequest{TenantId: tenantID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.GetLeads(), 2)

	moduleID := "crm"
	userIDStr := userA.String()
	resp, err = srv.ListModuleLeads(context.Background(), &settingsv1.ListModuleLeadsRequest{
		TenantId: tenantID.String(), UserId: &userIDStr, ModuleId: &moduleID,
	})
	requireGRPCOK(t, err)
	require.Len(t, resp.GetLeads(), 1)
	assert.Equal(t, "crm", resp.GetLeads()[0].GetModuleId())
	assert.NotNil(t, resp.GetLeads()[0].GetGrantedAt())
}

func TestListModuleLeads_RepoError(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()
	repo.forceErr = errStubRepoFailure
	_, err := srv.ListModuleLeads(context.Background(), &settingsv1.ListModuleLeadsRequest{TenantId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Internal)
}

func TestGrantModuleLead_InvalidIDs(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	valid := uuid.New().String()

	cases := []*settingsv1.GrantModuleLeadRequest{
		{TenantId: "bad", UserId: valid, ModuleId: "crm", GrantedBy: valid},
		{TenantId: valid, UserId: "bad", ModuleId: "crm", GrantedBy: valid},
		{TenantId: valid, UserId: valid, ModuleId: "crm", GrantedBy: "bad"},
	}
	for _, req := range cases {
		_, err := srv.GrantModuleLead(context.Background(), req)
		requireGRPCCode(t, err, codes.InvalidArgument)
	}
}

func TestGrantModuleLead_EmptyModuleIDBeforeAdminCheck(t *testing.T) {
	// A non-admin caller with an empty module_id must get InvalidArgument, not
	// PermissionDenied — the service validates module_id before checking role.
	srv, _, _ := newTestSettingsServer()
	tenantID, userID, caller := uuid.New(), uuid.New(), uuid.New()

	_, err := srv.GrantModuleLead(context.Background(), &settingsv1.GrantModuleLeadRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "", GrantedBy: caller.String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGrantModuleLead_NotAdmin(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	tenantID, userID, caller := uuid.New(), uuid.New(), uuid.New()

	_, err := srv.GrantModuleLead(context.Background(), &settingsv1.GrantModuleLeadRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm", GrantedBy: caller.String(),
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestGrantModuleLead_Success(t *testing.T) {
	srv, _, roles := newTestSettingsServer()
	tenantID, userID, caller := uuid.New(), uuid.New(), uuid.New()
	roles.setAdmin(tenantID, caller)

	resp, err := srv.GrantModuleLead(context.Background(), &settingsv1.GrantModuleLeadRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm", GrantedBy: caller.String(),
	})
	requireGRPCOK(t, err)
	assert.Equal(t, "crm", resp.GetModuleId())
	assert.Equal(t, userID.String(), resp.GetUserId())
	assert.Equal(t, caller.String(), resp.GetGrantedBy())
}

func TestRevokeModuleLead_CallerNotAuthenticated(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	tenantID, userID := uuid.New(), uuid.New()

	// No UserIDKey in context: middleware.GetUserID returns "", uuid.Parse fails.
	_, err := srv.RevokeModuleLead(context.Background(), &settingsv1.RevokeModuleLeadRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestRevokeModuleLead_InvalidIDs(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	ctx := callerCtx(uuid.New())
	valid := uuid.New().String()

	_, err := srv.RevokeModuleLead(ctx, &settingsv1.RevokeModuleLeadRequest{TenantId: "bad", UserId: valid, ModuleId: "crm"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.RevokeModuleLead(ctx, &settingsv1.RevokeModuleLeadRequest{TenantId: valid, UserId: "bad", ModuleId: "crm"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRevokeModuleLead_NotAdmin(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	tenantID, userID, caller := uuid.New(), uuid.New(), uuid.New()

	_, err := srv.RevokeModuleLead(callerCtx(caller), &settingsv1.RevokeModuleLeadRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestRevokeModuleLead_EmptyModuleID(t *testing.T) {
	srv, _, roles := newTestSettingsServer()
	tenantID, userID, caller := uuid.New(), uuid.New(), uuid.New()
	roles.setAdmin(tenantID, caller)

	_, err := srv.RevokeModuleLead(callerCtx(caller), &settingsv1.RevokeModuleLeadRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRevokeModuleLead_Success(t *testing.T) {
	srv, repo, roles := newTestSettingsServer()
	tenantID, userID, caller := uuid.New(), uuid.New(), uuid.New()
	roles.setAdmin(tenantID, caller)
	repo.leads = []*settings.ModuleLead{{TenantID: tenantID, UserID: userID, ModuleID: "crm"}}

	_, err := srv.RevokeModuleLead(callerCtx(caller), &settingsv1.RevokeModuleLeadRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
	})
	requireGRPCOK(t, err)
	assert.Empty(t, repo.leads)
}

func TestGetMyModuleLeads_InvalidIDs(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	_, err := srv.GetMyModuleLeads(context.Background(), &settingsv1.GetMyModuleLeadsRequest{TenantId: "bad", UserId: uuid.New().String()})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.GetMyModuleLeads(context.Background(), &settingsv1.GetMyModuleLeadsRequest{TenantId: uuid.New().String(), UserId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetMyModuleLeads_Admin(t *testing.T) {
	srv, _, roles := newTestSettingsServer()
	tenantID, userID := uuid.New(), uuid.New()
	roles.setAdmin(tenantID, userID)

	resp, err := srv.GetMyModuleLeads(context.Background(), &settingsv1.GetMyModuleLeadsRequest{TenantId: tenantID.String(), UserId: userID.String()})
	requireGRPCOK(t, err)
	assert.True(t, resp.GetIsAdmin())
	assert.Empty(t, resp.GetModuleIds())
}

func TestGetMyModuleLeads_NonAdminWithLeads(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()
	tenantID, userID := uuid.New(), uuid.New()
	repo.leads = []*settings.ModuleLead{
		{TenantID: tenantID, UserID: userID, ModuleID: "crm"},
		{TenantID: tenantID, UserID: userID, ModuleID: "finance"},
	}

	resp, err := srv.GetMyModuleLeads(context.Background(), &settingsv1.GetMyModuleLeadsRequest{TenantId: tenantID.String(), UserId: userID.String()})
	requireGRPCOK(t, err)
	assert.False(t, resp.GetIsAdmin())
	assert.ElementsMatch(t, []string{"crm", "finance"}, resp.GetModuleIds())
}

func TestGetMyModuleLeads_RepoError(t *testing.T) {
	srv, _, roles := newTestSettingsServer()
	roles.err = errStubRepoFailure
	_, err := srv.GetMyModuleLeads(context.Background(), &settingsv1.GetMyModuleLeadsRequest{TenantId: uuid.New().String(), UserId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Internal)
}

// ---------------------------------------------------------------------------
// Module-access grant RPCs
// ---------------------------------------------------------------------------

func TestListModuleGrants_InvalidIDs(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	_, err := srv.ListModuleGrants(context.Background(), &settingsv1.ListModuleGrantsRequest{TenantId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	tenantID := uuid.New().String()
	badUser := "bad"
	_, err = srv.ListModuleGrants(context.Background(), &settingsv1.ListModuleGrantsRequest{TenantId: tenantID, UserId: &badUser})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListModuleGrants_Success(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()
	tenantID, userID := uuid.New(), uuid.New()
	repo.grants = []*settings.UserModuleGrant{{TenantID: tenantID, UserID: userID, ModuleID: "crm", UserName: "Ada"}}

	resp, err := srv.ListModuleGrants(context.Background(), &settingsv1.ListModuleGrantsRequest{TenantId: tenantID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.GetGrants(), 1)
	assert.Equal(t, "Ada", resp.GetGrants()[0].GetUserName())
}

func TestGrantModuleAccess_ErrorMapping(t *testing.T) {
	tenantID, userID, caller := uuid.New(), uuid.New(), uuid.New()

	t.Run("not_admin", func(t *testing.T) {
		srv, _, _ := newTestSettingsServer()
		_, err := srv.GrantModuleAccess(context.Background(), &settingsv1.GrantModuleAccessRequest{
			TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm", GrantedBy: caller.String(),
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})

	t.Run("empty_module_id", func(t *testing.T) {
		srv, _, roles := newTestSettingsServer()
		roles.setAdmin(tenantID, caller)
		_, err := srv.GrantModuleAccess(context.Background(), &settingsv1.GrantModuleAccessRequest{
			TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "", GrantedBy: caller.String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("repo_error_is_internal", func(t *testing.T) {
		srv, repo, roles := newTestSettingsServer()
		roles.setAdmin(tenantID, caller)
		repo.forceErr = errStubRepoFailure
		_, err := srv.GrantModuleAccess(context.Background(), &settingsv1.GrantModuleAccessRequest{
			TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm", GrantedBy: caller.String(),
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestGrantModuleAccess_InvalidIDs(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	valid := uuid.New().String()

	_, err := srv.GrantModuleAccess(context.Background(), &settingsv1.GrantModuleAccessRequest{TenantId: "bad", UserId: valid, ModuleId: "crm", GrantedBy: valid})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.GrantModuleAccess(context.Background(), &settingsv1.GrantModuleAccessRequest{TenantId: valid, UserId: "bad", ModuleId: "crm", GrantedBy: valid})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.GrantModuleAccess(context.Background(), &settingsv1.GrantModuleAccessRequest{TenantId: valid, UserId: valid, ModuleId: "crm", GrantedBy: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGrantModuleAccess_Success(t *testing.T) {
	srv, _, roles := newTestSettingsServer()
	tenantID, userID, caller := uuid.New(), uuid.New(), uuid.New()
	roles.setAdmin(tenantID, caller)

	resp, err := srv.GrantModuleAccess(context.Background(), &settingsv1.GrantModuleAccessRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm", GrantedBy: caller.String(),
	})
	requireGRPCOK(t, err)
	assert.Equal(t, "crm", resp.GetModuleId())
}

func TestRevokeModuleAccess_CallerNotAuthenticated(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	_, err := srv.RevokeModuleAccess(context.Background(), &settingsv1.RevokeModuleAccessRequest{
		TenantId: uuid.New().String(), UserId: uuid.New().String(), ModuleId: "crm",
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestRevokeModuleAccess_NotAdmin(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	caller := uuid.New()
	_, err := srv.RevokeModuleAccess(callerCtx(caller), &settingsv1.RevokeModuleAccessRequest{
		TenantId: uuid.New().String(), UserId: uuid.New().String(), ModuleId: "crm",
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestRevokeModuleAccess_Success(t *testing.T) {
	srv, repo, roles := newTestSettingsServer()
	tenantID, userID, caller := uuid.New(), uuid.New(), uuid.New()
	roles.setAdmin(tenantID, caller)
	repo.grants = []*settings.UserModuleGrant{{TenantID: tenantID, UserID: userID, ModuleID: "crm"}}

	_, err := srv.RevokeModuleAccess(callerCtx(caller), &settingsv1.RevokeModuleAccessRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
	})
	requireGRPCOK(t, err)
	assert.Empty(t, repo.grants)
}

func TestBulkRevokeModuleAccess_CallerNotAuthenticated(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	_, err := srv.BulkRevokeModuleAccess(context.Background(), &settingsv1.BulkRevokeModuleAccessRequest{TenantId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestBulkRevokeModuleAccess_InvalidRefUUID(t *testing.T) {
	srv, _, roles := newTestSettingsServer()
	tenantID, caller := uuid.New(), uuid.New()
	roles.setAdmin(tenantID, caller)

	_, err := srv.BulkRevokeModuleAccess(callerCtx(caller), &settingsv1.BulkRevokeModuleAccessRequest{
		TenantId: tenantID.String(),
		Refs:     []*settingsv1.ModuleGrantRef{{UserId: "not-a-uuid", ModuleId: "crm"}},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestBulkRevokeModuleAccess_NotAdmin(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	caller := uuid.New()
	_, err := srv.BulkRevokeModuleAccess(callerCtx(caller), &settingsv1.BulkRevokeModuleAccessRequest{
		TenantId: uuid.New().String(),
		Refs:     []*settingsv1.ModuleGrantRef{{UserId: uuid.New().String(), ModuleId: "crm"}},
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestBulkRevokeModuleAccess_EmptyModuleIDInRefs(t *testing.T) {
	srv, _, roles := newTestSettingsServer()
	tenantID, caller := uuid.New(), uuid.New()
	roles.setAdmin(tenantID, caller)

	_, err := srv.BulkRevokeModuleAccess(callerCtx(caller), &settingsv1.BulkRevokeModuleAccessRequest{
		TenantId: tenantID.String(),
		Refs:     []*settingsv1.ModuleGrantRef{{UserId: uuid.New().String(), ModuleId: ""}},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestBulkRevokeModuleAccess_Success(t *testing.T) {
	srv, repo, roles := newTestSettingsServer()
	tenantID, caller := uuid.New(), uuid.New()
	roles.setAdmin(tenantID, caller)
	userA, userB := uuid.New(), uuid.New()
	repo.grants = []*settings.UserModuleGrant{
		{TenantID: tenantID, UserID: userA, ModuleID: "crm"},
		{TenantID: tenantID, UserID: userB, ModuleID: "finance"},
	}

	resp, err := srv.BulkRevokeModuleAccess(callerCtx(caller), &settingsv1.BulkRevokeModuleAccessRequest{
		TenantId: tenantID.String(),
		Refs: []*settingsv1.ModuleGrantRef{
			{UserId: userA.String(), ModuleId: "crm"},
			{UserId: userB.String(), ModuleId: "finance"},
		},
	})
	requireGRPCOK(t, err)
	assert.Equal(t, int32(2), resp.GetRevokedCount())
	assert.Empty(t, repo.grants)
}

// ---------------------------------------------------------------------------
// Settings RPCs — three-level resolution
// ---------------------------------------------------------------------------

func TestGetResolvedSettings_ThreeLevelResolution(t *testing.T) {
	srv, _, roles := newTestSettingsServer()
	tenantID, userID, admin, lead := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	roles.setAdmin(tenantID, admin)

	// Level 1: tenant default written by an admin.
	_, err := srv.PutTenantSettings(context.Background(), &settingsv1.PutTenantSettingsRequest{
		TenantId: tenantID.String(), ModuleId: "crm", UpdatedBy: admin.String(),
		Entries: []*settingsv1.SettingEntry{{Key: "theme", Value: structpb.NewStringValue("light")}},
	})
	requireGRPCOK(t, err)

	// Level 2: module-lead override at tenant scope (same write path, different
	// caller — proves the RBAC gate, not a separate storage level).
	_, err = srv.GrantModuleLead(context.Background(), &settingsv1.GrantModuleLeadRequest{
		TenantId: tenantID.String(), UserId: lead.String(), ModuleId: "crm", GrantedBy: admin.String(),
	})
	requireGRPCOK(t, err)
	_, err = srv.PutTenantSettings(callerCtx(lead), &settingsv1.PutTenantSettingsRequest{
		TenantId: tenantID.String(), ModuleId: "crm", UpdatedBy: lead.String(),
		Entries: []*settingsv1.SettingEntry{{Key: "theme", Value: structpb.NewStringValue("dark")}},
	})
	requireGRPCOK(t, err)

	resolved, err := srv.GetResolvedSettings(context.Background(), &settingsv1.GetResolvedSettingsRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
	})
	requireGRPCOK(t, err)
	require.Len(t, resolved.GetEntries(), 1)
	assert.Equal(t, "dark", resolved.GetEntries()[0].GetValue().GetStringValue())

	// Level 3: personal override wins over the tenant-scope value.
	_, err = srv.PutUserSettings(context.Background(), &settingsv1.PutUserSettingsRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
		Entries: []*settingsv1.SettingEntry{{Key: "theme", Value: structpb.NewStringValue("solarized")}},
	})
	requireGRPCOK(t, err)

	resolved, err = srv.GetResolvedSettings(context.Background(), &settingsv1.GetResolvedSettingsRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
	})
	requireGRPCOK(t, err)
	require.Len(t, resolved.GetEntries(), 1)
	assert.Equal(t, "solarized", resolved.GetEntries()[0].GetValue().GetStringValue())

	// A different user in the same tenant never wrote a personal override —
	// they still see the tenant/lead-scope value, not the first user's.
	other := uuid.New()
	resolved, err = srv.GetResolvedSettings(context.Background(), &settingsv1.GetResolvedSettingsRequest{
		TenantId: tenantID.String(), UserId: other.String(), ModuleId: "crm",
	})
	requireGRPCOK(t, err)
	require.Len(t, resolved.GetEntries(), 1)
	assert.Equal(t, "dark", resolved.GetEntries()[0].GetValue().GetStringValue())
}

func TestGetResolvedSettings_InvalidIDsAndEmptyModule(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	valid := uuid.New().String()

	_, err := srv.GetResolvedSettings(context.Background(), &settingsv1.GetResolvedSettingsRequest{TenantId: "bad", UserId: valid, ModuleId: "crm"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.GetResolvedSettings(context.Background(), &settingsv1.GetResolvedSettingsRequest{TenantId: valid, UserId: "bad", ModuleId: "crm"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.GetResolvedSettings(context.Background(), &settingsv1.GetResolvedSettingsRequest{TenantId: valid, UserId: valid, ModuleId: ""})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetResolvedSettings_EmptyIsEmptyListNotNull(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	resp, err := srv.GetResolvedSettings(context.Background(), &settingsv1.GetResolvedSettingsRequest{
		TenantId: uuid.New().String(), UserId: uuid.New().String(), ModuleId: "crm",
	})
	requireGRPCOK(t, err)
	assert.NotNil(t, resp.GetEntries())
	assert.Empty(t, resp.GetEntries())
}

func TestGetTenantSettings_InvalidModuleIDAndSuccess(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	tenantID := uuid.New()

	_, err := srv.GetTenantSettings(context.Background(), &settingsv1.GetTenantSettingsRequest{TenantId: tenantID.String(), ModuleId: ""})
	requireGRPCCode(t, err, codes.InvalidArgument)

	resp, err := srv.GetTenantSettings(context.Background(), &settingsv1.GetTenantSettingsRequest{TenantId: tenantID.String(), ModuleId: "crm"})
	requireGRPCOK(t, err)
	assert.Empty(t, resp.GetEntries())
}

func TestGetTenantSettings_RepoError(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()
	repo.forceErr = errStubRepoFailure
	_, err := srv.GetTenantSettings(context.Background(), &settingsv1.GetTenantSettingsRequest{TenantId: uuid.New().String(), ModuleId: "crm"})
	requireGRPCCode(t, err, codes.Internal)
}

func TestPutTenantSettings_PermissionDenied(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	tenantID, caller := uuid.New(), uuid.New()

	_, err := srv.PutTenantSettings(context.Background(), &settingsv1.PutTenantSettingsRequest{
		TenantId: tenantID.String(), ModuleId: "crm", UpdatedBy: caller.String(),
		Entries: []*settingsv1.SettingEntry{{Key: "k", Value: structpb.NewBoolValue(true)}},
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestPutTenantSettings_ModuleLeadAllowed(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()
	tenantID, lead := uuid.New(), uuid.New()
	repo.leads = []*settings.ModuleLead{{TenantID: tenantID, UserID: lead, ModuleID: "crm"}}

	_, err := srv.PutTenantSettings(context.Background(), &settingsv1.PutTenantSettingsRequest{
		TenantId: tenantID.String(), ModuleId: "crm", UpdatedBy: lead.String(),
		Entries: []*settingsv1.SettingEntry{{Key: "k", Value: structpb.NewBoolValue(true)}},
	})
	requireGRPCOK(t, err)
}

func TestPutTenantSettings_InvalidUpdatedByAndEmptyKey(t *testing.T) {
	srv, _, roles := newTestSettingsServer()
	tenantID, admin := uuid.New(), uuid.New()
	roles.setAdmin(tenantID, admin)

	_, err := srv.PutTenantSettings(context.Background(), &settingsv1.PutTenantSettingsRequest{
		TenantId: tenantID.String(), ModuleId: "crm", UpdatedBy: "bad",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.PutTenantSettings(context.Background(), &settingsv1.PutTenantSettingsRequest{
		TenantId: tenantID.String(), ModuleId: "crm", UpdatedBy: admin.String(),
		Entries: []*settingsv1.SettingEntry{{Key: "", Value: structpb.NewBoolValue(true)}},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.PutTenantSettings(context.Background(), &settingsv1.PutTenantSettingsRequest{
		TenantId: tenantID.String(), ModuleId: "", UpdatedBy: admin.String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetUserSettings_InvalidIDsAndSuccess(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	valid := uuid.New().String()

	_, err := srv.GetUserSettings(context.Background(), &settingsv1.GetUserSettingsRequest{TenantId: "bad", UserId: valid, ModuleId: "crm"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.GetUserSettings(context.Background(), &settingsv1.GetUserSettingsRequest{TenantId: valid, UserId: "bad", ModuleId: "crm"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.GetUserSettings(context.Background(), &settingsv1.GetUserSettingsRequest{TenantId: valid, UserId: valid, ModuleId: ""})
	requireGRPCCode(t, err, codes.InvalidArgument)

	resp, err := srv.GetUserSettings(context.Background(), &settingsv1.GetUserSettingsRequest{TenantId: valid, UserId: valid, ModuleId: "crm"})
	requireGRPCOK(t, err)
	assert.Empty(t, resp.GetEntries())
}

func TestPutUserSettings_InvalidIDsEmptyKeyAndSuccess(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	tenantID, userID := uuid.New(), uuid.New()

	_, err := srv.PutUserSettings(context.Background(), &settingsv1.PutUserSettingsRequest{TenantId: "bad", UserId: userID.String(), ModuleId: "crm"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.PutUserSettings(context.Background(), &settingsv1.PutUserSettingsRequest{TenantId: tenantID.String(), UserId: "bad", ModuleId: "crm"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.PutUserSettings(context.Background(), &settingsv1.PutUserSettingsRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
		Entries: []*settingsv1.SettingEntry{{Key: "", Value: structpb.NewNullValue()}},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)

	resp, err := srv.PutUserSettings(context.Background(), &settingsv1.PutUserSettingsRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
		Entries: []*settingsv1.SettingEntry{{Key: "density", Value: structpb.NewNumberValue(42)}},
	})
	requireGRPCOK(t, err)
	require.Len(t, resp.GetEntries(), 1)
	assert.InDelta(t, 42, resp.GetEntries()[0].GetValue().GetNumberValue(), 0.0001)
}

func TestReplaceUserSettings_FullReplaceSemantics(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	tenantID, userID := uuid.New(), uuid.New()

	_, err := srv.PutUserSettings(context.Background(), &settingsv1.PutUserSettingsRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
		Entries: []*settingsv1.SettingEntry{
			{Key: "a", Value: structpb.NewStringValue("1")},
			{Key: "b", Value: structpb.NewStringValue("2")},
		},
	})
	requireGRPCOK(t, err)

	resp, err := srv.ReplaceUserSettings(context.Background(), &settingsv1.ReplaceUserSettingsRequest{
		TenantId: tenantID.String(), UserId: userID.String(), ModuleId: "crm",
		Entries: []*settingsv1.SettingEntry{{Key: "a", Value: structpb.NewStringValue("only-this-survives")}},
	})
	requireGRPCOK(t, err)
	require.Len(t, resp.GetEntries(), 1)
	assert.Equal(t, "a", resp.GetEntries()[0].GetKey())

	_, err = srv.ReplaceUserSettings(context.Background(), &settingsv1.ReplaceUserSettingsRequest{
		TenantId: "bad", UserId: userID.String(), ModuleId: "crm",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ---------------------------------------------------------------------------
// Branding RPCs
// ---------------------------------------------------------------------------

func TestGetBranding_InvalidTenantIDAndDefaults(t *testing.T) {
	srv, _, _ := newTestSettingsServer()

	_, err := srv.GetBranding(context.Background(), &settingsv1.GetBrandingRequest{TenantId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	resp, err := srv.GetBranding(context.Background(), &settingsv1.GetBrandingRequest{TenantId: uuid.New().String()})
	requireGRPCOK(t, err)
	assert.Equal(t, "#10B981", resp.GetBranding().GetAccentColor())
	assert.Empty(t, resp.GetBranding().GetName())
}

func TestGetBranding_RepoError(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()
	repo.forceErr = errStubRepoFailure
	_, err := srv.GetBranding(context.Background(), &settingsv1.GetBrandingRequest{TenantId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Internal)
}

func TestPutBranding_ValidationErrors(t *testing.T) {
	tenantID, admin := uuid.New(), uuid.New()

	cases := []struct {
		name     string
		branding *settingsv1.Branding
	}{
		{"name_too_long", &settingsv1.Branding{Name: string(make([]byte, 201)), AccentColor: "#10B981"}},
		{"accent_color_not_allowed", &settingsv1.Branding{Name: "Cosmi", AccentColor: "#000000"}},
		{"object_key_wrong_tenant", &settingsv1.Branding{Name: "Cosmi", AccentColor: "#10B981", LogoObjectKey: uuid.New().String() + "/branding/logo.png"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _, roles := newTestSettingsServer()
			roles.setAdmin(tenantID, admin)
			_, err := srv.PutBranding(context.Background(), &settingsv1.PutBrandingRequest{
				TenantId: tenantID.String(), UpdatedBy: admin.String(), Branding: tc.branding,
			})
			requireGRPCCode(t, err, codes.InvalidArgument)
		})
	}
}

func TestPutBranding_InvalidUpdatedBy(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	_, err := srv.PutBranding(context.Background(), &settingsv1.PutBrandingRequest{
		TenantId: uuid.New().String(), UpdatedBy: "bad", Branding: &settingsv1.Branding{AccentColor: "#10B981"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestPutBranding_PermissionDenied(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	tenantID, caller := uuid.New(), uuid.New()
	_, err := srv.PutBranding(context.Background(), &settingsv1.PutBrandingRequest{
		TenantId: tenantID.String(), UpdatedBy: caller.String(), Branding: &settingsv1.Branding{Name: "Cosmi", AccentColor: "#10B981"},
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestPutBranding_Success(t *testing.T) {
	srv, _, roles := newTestSettingsServer()
	tenantID, admin := uuid.New(), uuid.New()
	roles.setAdmin(tenantID, admin)

	resp, err := srv.PutBranding(context.Background(), &settingsv1.PutBrandingRequest{
		TenantId: tenantID.String(), UpdatedBy: admin.String(),
		Branding: &settingsv1.Branding{Name: "Cosmi", AccentColor: "#3B82F6", LogoObjectKey: tenantID.String() + "/branding/logo.png"},
	})
	requireGRPCOK(t, err)
	assert.Equal(t, "Cosmi", resp.GetBranding().GetName())
	assert.Equal(t, "#3B82F6", resp.GetBranding().GetAccentColor())

	get, err := srv.GetBranding(context.Background(), &settingsv1.GetBrandingRequest{TenantId: tenantID.String()})
	requireGRPCOK(t, err)
	assert.Equal(t, "Cosmi", get.GetBranding().GetName())
}

// ---------------------------------------------------------------------------
// Licensing RPCs
// ---------------------------------------------------------------------------

func TestGetTenantLicense_InvalidTenantIDAndSuccess(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()

	_, err := srv.GetTenantLicense(context.Background(), &settingsv1.GetTenantLicenseRequest{TenantId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	tenantID := uuid.New()
	repo.activations[tKey(tenantID, "crm")] = true
	repo.grantCounts[tKey(tenantID, "crm")] = 3

	resp, err := srv.GetTenantLicense(context.Background(), &settingsv1.GetTenantLicenseRequest{TenantId: tenantID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.GetModules(), len(modules.Catalog))
	for _, m := range resp.GetModules() {
		if m.GetModuleId() == "crm" {
			assert.True(t, m.GetActive())
			assert.Equal(t, int32(3), m.GetAssignedSeats())
		} else {
			assert.False(t, m.GetActive())
			assert.Zero(t, m.GetAssignedSeats())
		}
	}
}

func TestSetTenantModuleActive_Validation(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	tenantID := uuid.New()

	_, err := srv.SetTenantModuleActive(context.Background(), &settingsv1.SetTenantModuleActiveRequest{TenantId: tenantID.String(), ModuleId: ""})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.SetTenantModuleActive(context.Background(), &settingsv1.SetTenantModuleActiveRequest{TenantId: tenantID.String(), ModuleId: "does-not-exist"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.SetTenantModuleActive(context.Background(), &settingsv1.SetTenantModuleActiveRequest{TenantId: tenantID.String(), ModuleId: "crm", UpdatedBy: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetTenantModuleActive_Success(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()
	tenantID, admin := uuid.New(), uuid.New()
	repo.grantCounts[tKey(tenantID, "crm")] = 5

	resp, err := srv.SetTenantModuleActive(context.Background(), &settingsv1.SetTenantModuleActiveRequest{
		TenantId: tenantID.String(), ModuleId: "crm", Active: true, UpdatedBy: admin.String(),
	})
	requireGRPCOK(t, err)
	assert.True(t, resp.GetActive())
	assert.Equal(t, int32(5), resp.GetAssignedSeats())

	// Deactivating reports 0 assigned seats even though the grants stay stored —
	// a booked-off module does not occupy seats in the cost view.
	resp, err = srv.SetTenantModuleActive(context.Background(), &settingsv1.SetTenantModuleActiveRequest{
		TenantId: tenantID.String(), ModuleId: "crm", Active: false,
	})
	requireGRPCOK(t, err)
	assert.False(t, resp.GetActive())
	assert.Zero(t, resp.GetAssignedSeats())
}

func TestGetTenantSubscription_NotFoundAndSuccess(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()

	_, err := srv.GetTenantSubscription(context.Background(), &settingsv1.GetTenantSubscriptionRequest{TenantId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	tenantID := uuid.New()
	_, err = srv.GetTenantSubscription(context.Background(), &settingsv1.GetTenantSubscriptionRequest{TenantId: tenantID.String()})
	requireGRPCCode(t, err, codes.NotFound)

	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	seats := int32(25)
	repo.subscriptions[tenantID.String()] = &settings.TenantSubscription{
		PlanType: "cosmi", SupportTier: "priority", Status: "active", BillingPeriodEnd: &end, TotalSeats: &seats,
	}

	resp, err := srv.GetTenantSubscription(context.Background(), &settingsv1.GetTenantSubscriptionRequest{TenantId: tenantID.String()})
	requireGRPCOK(t, err)
	assert.Equal(t, "cosmi", resp.GetPlanType())
	assert.Equal(t, "2026-12-31", resp.GetBillingPeriodEnd())
	assert.Equal(t, int32(25), resp.GetTotalSeats())
}

// ---------------------------------------------------------------------------
// Value-Set RPCs
// ---------------------------------------------------------------------------

func TestListValueSets_BaseAndOverride(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()

	_, err := srv.ListValueSets(context.Background(), &settingsv1.ListValueSetsRequest{TenantId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	tenantID := uuid.New()
	base, err := srv.ListValueSets(context.Background(), &settingsv1.ListValueSetsRequest{TenantId: tenantID.String(), Base: true})
	requireGRPCOK(t, err)
	assert.Len(t, base.GetValueSets(), 4) // deal_stages, ticket_priority, ticket_status, project_status

	// A tenant-only custom list has no baseline and must still show up in the
	// non-base view.
	repo.valueSets[vKey(tenantID, "custom_tags")] = &settings.ValueSet{
		Key: "custom_tags", Name: "Custom Tags",
		Options: []settings.ValueSetOption{{Key: "vip", Label: "VIP", Order: 0, Active: true}},
	}
	resolved, err := srv.ListValueSets(context.Background(), &settingsv1.ListValueSetsRequest{TenantId: tenantID.String()})
	requireGRPCOK(t, err)
	assert.Len(t, resolved.GetValueSets(), 5)
}

func TestGetValueSet_InvalidKeyNotFoundAndSuccess(t *testing.T) {
	srv, repo, _ := newTestSettingsServer()
	tenantID := uuid.New()

	_, err := srv.GetValueSet(context.Background(), &settingsv1.GetValueSetRequest{TenantId: "bad", Key: "deal_stages"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.GetValueSet(context.Background(), &settingsv1.GetValueSetRequest{TenantId: tenantID.String(), Key: "Not Valid!"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.GetValueSet(context.Background(), &settingsv1.GetValueSetRequest{TenantId: tenantID.String(), Key: "unknown_list", Base: true})
	requireGRPCCode(t, err, codes.NotFound)

	_, err = srv.GetValueSet(context.Background(), &settingsv1.GetValueSetRequest{TenantId: tenantID.String(), Key: "unknown_list"})
	requireGRPCCode(t, err, codes.NotFound)

	resp, err := srv.GetValueSet(context.Background(), &settingsv1.GetValueSetRequest{TenantId: tenantID.String(), Key: "deal_stages"})
	requireGRPCOK(t, err)
	assert.True(t, resp.GetIsSystem())
	assert.Equal(t, "default", resp.GetProvenance())

	repo.valueSets[vKey(tenantID, "deal_stages")] = &settings.ValueSet{
		Key: "deal_stages", Name: "Verkaufsphasen",
		Options: []settings.ValueSetOption{{Key: "lead", Label: "Interessent", Order: 0, Active: true}},
	}
	resp, err = srv.GetValueSet(context.Background(), &settingsv1.GetValueSetRequest{TenantId: tenantID.String(), Key: "deal_stages"})
	requireGRPCOK(t, err)
	assert.Equal(t, "Verkaufsphasen", resp.GetName())
	assert.Equal(t, "tenant", resp.GetProvenance())
}

func TestUpsertValueSet_ValidationErrors(t *testing.T) {
	tenantID := uuid.New()
	cases := []struct {
		name string
		set  *settingsv1.ValueSet
	}{
		{"bad_key", &settingsv1.ValueSet{Key: "BadKey!", Name: "X", Options: []*settingsv1.ValueSetOption{{Key: "a", Label: "A"}}}},
		{"empty_name", &settingsv1.ValueSet{Key: "custom_list", Name: "", Options: []*settingsv1.ValueSetOption{{Key: "a", Label: "A"}}}},
		{"no_options", &settingsv1.ValueSet{Key: "custom_list", Name: "X", Options: nil}},
		{"duplicate_option_key", &settingsv1.ValueSet{Key: "custom_list", Name: "X", Options: []*settingsv1.ValueSetOption{
			{Key: "a", Label: "A"}, {Key: "a", Label: "A2"},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _, _ := newTestSettingsServer()
			_, err := srv.UpsertValueSet(context.Background(), &settingsv1.UpsertValueSetRequest{
				TenantId: tenantID.String(), ValueSet: tc.set,
			})
			requireGRPCCode(t, err, codes.InvalidArgument)
		})
	}
}

func TestUpsertValueSet_InvalidUpdatedByAndSuccess(t *testing.T) {
	srv, _, _ := newTestSettingsServer()
	tenantID := uuid.New()
	validSet := &settingsv1.ValueSet{Key: "custom_list", Name: "Custom", Options: []*settingsv1.ValueSetOption{{Key: "a", Label: "A", Active: true}}}

	_, err := srv.UpsertValueSet(context.Background(), &settingsv1.UpsertValueSetRequest{
		TenantId: tenantID.String(), UpdatedBy: "bad", ValueSet: validSet,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)

	resp, err := srv.UpsertValueSet(context.Background(), &settingsv1.UpsertValueSetRequest{
		TenantId: tenantID.String(), ValueSet: validSet,
	})
	requireGRPCOK(t, err)
	assert.Equal(t, "custom_list", resp.GetKey())
	assert.Equal(t, "tenant", resp.GetProvenance())
	assert.False(t, resp.GetIsSystem())
	require.Len(t, resp.GetOptions(), 1)
	assert.Equal(t, "tenant", resp.GetOptions()[0].GetProvenance())
}

// ---------------------------------------------------------------------------
// mapModuleGrantError — table test
// ---------------------------------------------------------------------------

func TestMapModuleGrantError_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"not_admin", settings.ErrNotAdmin, codes.PermissionDenied},
		{"invalid_module_id", settings.ErrInvalidModuleID, codes.InvalidArgument},
		{"generic", errStubRepoFailure, codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapModuleGrantError(tc.err, "grant")
			requireGRPCCode(t, err, tc.code)
		})
	}
}
