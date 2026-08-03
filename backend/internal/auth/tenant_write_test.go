package auth_test

// Users, invitations and sessions each already have an rls_*_test.go that
// proves the RLS policy in isolation, but all three seed their fixtures via
// testutil.SeedRow (a raw INSERT under system context) and never call
// PostgresRepository at all. That is exactly the shape of gap Nacht 1 found
// in this same package: invitations had no tenant_id column whatsoever
// (migration 000249) and nothing that exercised CreateInvitation against a
// real database would have noticed either way. This file closes that gap for
// the three write surfaces the backlog flagged as unproven: CreateUser/
// UpdateUser/UpdateProfile/UpdatePassword, CreateInvitation/
// DeleteInvitation/AcceptInvitation, and CreateSession/UpdateSessionActivity/
// DeleteSession/DeleteAllUserSessions.
//
// Provisioning is not repeated here: rls_provisioning_test.go already drives
// auth.NewService(...).ProvisionTenant end to end through the real
// repository, so ProvisionTenant is not an unproven write path.
//
// No dead write found: every INSERT/UPDATE below that has a tenant_id column
// to write already writes it. UpdateUser, UpdateProfile, UpdatePassword,
// UpdateSessionActivity, DeleteSession and DeleteAllUserSessions have no
// tenant_id predicate in their WHERE clause at all — RLS is the only thing
// stopping a foreign tenant from reaching the row, which each test below
// proves by calling the real method under a foreign ctx and checking the row
// was not touched.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestUsersWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Auth Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Auth Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := auth.NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	user := &models.User{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		Email:        "write-" + uuid.New().String() + "@test.local",
		PasswordHash: "x",
		FirstName:    "Write",
		LastName:     "Test",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.CreateUser(ctxOwn, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "users", user.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "users", user.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "users", user.ID, 0)

	// UpdateUser/UpdateProfile/UpdatePassword all key on `id = $1` alone —
	// a foreign tenant's write must still not reach the row.
	foreign := *user
	foreign.FirstName = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateUser(ctxOther, &foreign); err != nil {
		t.Fatalf("UpdateUser (foreign ctx): %v", err)
	}
	if err := repo.UpdateProfile(ctxOther, &models.User{ID: user.ID, FirstName: "Hacked2", LastName: "Hacked2", UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("UpdateProfile (foreign ctx): %v", err)
	}
	if err := repo.UpdatePassword(ctxOther, user.ID, "hacked-hash"); err != nil {
		t.Fatalf("UpdatePassword (foreign ctx): %v", err)
	}
	got, err := repo.GetUserByID(sysCtx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.FirstName != "Write" || got.PasswordHash != "x" {
		t.Fatalf("a foreign-tenant write reached the row: first_name=%q password_hash=%q", got.FirstName, got.PasswordHash)
	}

	// The owning tenant's own write does land.
	foreign.FirstName = "Renamed"
	if err := repo.UpdateUser(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateUser (own ctx): %v", err)
	}
	got, err = repo.GetUserByID(ctxOwn, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID (own ctx): %v", err)
	}
	if got.FirstName != "Renamed" {
		t.Fatalf("UpdateUser (own ctx): first_name not updated, got %q", got.FirstName)
	}
}

func TestInvitationsWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Auth Invitation Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Auth Invitation Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := auth.NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	inviter := &models.User{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		Email:        "inviter-" + uuid.New().String() + "@test.local",
		PasswordHash: "x",
		FirstName:    "Inviter",
		LastName:     "Write",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.CreateUser(ctxOwn, inviter); err != nil {
		t.Fatalf("CreateUser (inviter): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "users", inviter.ID)

	// RoleIDs, not just the legacy Role name: since migration 000280 that is
	// what AcceptInvitation resolves, and an invitation without it grants
	// nothing.
	var memberPresetID uuid.UUID
	if err := pool.QueryRow(sysCtx,
		`SELECT id FROM roles WHERE tenant_id IS NULL AND name = 'member'`).Scan(&memberPresetID); err != nil {
		t.Fatalf("member preset lookup: %v", err)
	}

	inv := &models.Invitation{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Email:     "invited-" + uuid.New().String() + "@test.local",
		Role:      "member",
		RoleIDs:   []uuid.UUID{memberPresetID},
		TokenHash: uuid.NewString(),
		CreatedBy: inviter.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: now,
	}
	if err := repo.CreateInvitation(ctxOwn, inv); err != nil {
		t.Fatalf("CreateInvitation: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "invitations", inv.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "invitations", inv.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "invitations", inv.ID, 0)

	// DeleteInvitation takes tenantID as an explicit parameter, but the
	// caller's ctx is still the real backstop: pass the correct owning
	// tenantID from a foreign ctx and the row must still not move.
	if err := repo.DeleteInvitation(ctxOther, tenantOwn, inv.ID); err != nil {
		t.Fatalf("DeleteInvitation (foreign ctx, correct tenantID param): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "invitations", inv.ID, 1)

	// AcceptInvitation, exercised end to end through the real repository —
	// rls_invitations_test.go only claims the row with a raw UPDATE, it
	// never calls this method.
	newUser := &models.User{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		Email:        inv.Email,
		PasswordHash: "y",
		FirstName:    "Invited",
		LastName:     "Write",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.AcceptInvitation(ctxOwn, inv, newUser); err != nil {
		t.Fatalf("AcceptInvitation: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "users", newUser.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "users", newUser.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "users", newUser.ID, 0)

	roles, err := repo.GetUserRoles(sysCtx, newUser.ID)
	if err != nil {
		t.Fatalf("GetUserRoles: %v", err)
	}
	if len(roles) != 1 || roles[0] != "member" {
		t.Fatalf("AcceptInvitation: expected role [member], got %v", roles)
	}

	// A replay under a foreign tenant must not create a second account —
	// the claiming UPDATE has to fail, whether RLS or accepted_at stops it.
	replay := &models.User{
		ID:           uuid.New(),
		TenantID:     tenantOther,
		Email:        "replay-" + uuid.New().String() + "@test.local",
		PasswordHash: "z",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	err = repo.AcceptInvitation(ctxOther, inv, replay)
	if !errors.Is(err, auth.ErrInvitationAlreadyUsed) {
		t.Fatalf("AcceptInvitation (foreign ctx, replay): expected ErrInvitationAlreadyUsed, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "users", replay.ID, 0)
}

func TestSessionsWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Auth Session Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Auth Session Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := auth.NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	owner := &models.User{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		Email:        "sess-owner-" + uuid.New().String() + "@test.local",
		PasswordHash: "x",
		FirstName:    "Session",
		LastName:     "Owner",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.CreateUser(ctxOwn, owner); err != nil {
		t.Fatalf("CreateUser (session owner): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "users", owner.ID)

	firstActive := time.Now().Add(-time.Hour).UTC()
	session := &models.UserSession{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		UserID:       owner.ID,
		DeviceName:   "Write Test Device",
		IPAddress:    "127.0.0.1",
		LastActiveAt: firstActive,
		CreatedAt:    now,
	}
	if err := repo.CreateSession(ctxOwn, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "user_sessions", session.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "user_sessions", session.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "user_sessions", session.ID, 0)

	// UpdateSessionActivity/DeleteSession/DeleteAllUserSessions key on id
	// (or user_id) alone with no tenant_id predicate in the query — RLS is
	// the only thing stopping a foreign tenant from reaching the row.
	if err := repo.UpdateSessionActivity(ctxOther, session.ID); err != nil {
		t.Fatalf("UpdateSessionActivity (foreign ctx): %v", err)
	}
	got, err := repo.GetSession(sysCtx, session.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	// Postgres truncates timestamptz to microsecond precision, so an exact
	// time.Time.Equal against the Go-side nanosecond value is not the right
	// check here — a sub-microsecond rounding diff is not a reached row.
	if diff := got.LastActiveAt.Sub(firstActive); diff < -time.Millisecond || diff > time.Millisecond {
		t.Fatalf("UpdateSessionActivity reached a foreign-tenant row: last_active_at changed to %v (diff %v)", got.LastActiveAt, diff)
	}

	if err := repo.DeleteSession(ctxOther, session.ID); err != nil {
		t.Fatalf("DeleteSession (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "user_sessions", session.ID, 1)

	if err := repo.DeleteAllUserSessions(ctxOther, owner.ID, nil); err != nil {
		t.Fatalf("DeleteAllUserSessions (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "user_sessions", session.ID, 1)

	// The owning tenant's own writes do land.
	if err := repo.UpdateSessionActivity(ctxOwn, session.ID); err != nil {
		t.Fatalf("UpdateSessionActivity (own ctx): %v", err)
	}
	got, err = repo.GetSession(ctxOwn, session.ID)
	if err != nil {
		t.Fatalf("GetSession (own ctx): %v", err)
	}
	if !got.LastActiveAt.After(firstActive) {
		t.Fatalf("UpdateSessionActivity (own ctx): last_active_at not advanced, got %v", got.LastActiveAt)
	}

	if err := repo.DeleteSession(ctxOwn, session.ID); err != nil {
		t.Fatalf("DeleteSession (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "user_sessions", session.ID, 0)
}
