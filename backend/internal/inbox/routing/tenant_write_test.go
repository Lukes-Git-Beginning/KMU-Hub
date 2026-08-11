package routing

// routing_rules already has a raw-row cross-tenant proof
// (team/tenant_isolation_phase2_test.go), but PostgresRepository itself --
// the thing routing.Service actually calls -- had no DB test at all
// (e-cov-inbox-repo-infra). Every write below carries an explicit tenant_id
// predicate already; this proves RLS backs it up rather than the WHERE
// clause alone by passing the row's *real* owning tenantID as the explicit
// parameter from a foreign-tenant ctx and confirming it still doesn't land.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRoutingRuleWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Routing Rule Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Routing Rule Write Other Tenant")

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("routing-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	channel := "email"
	newRule := func(id uuid.UUID, name string) *models.RoutingRule {
		return &models.RoutingRule{
			ID:         id,
			TenantID:   tenantOwn,
			Name:       name,
			Channel:    &channel,
			Conditions: []byte(`{"field":"subject","op":"contains","value":"urgent"}`),
			Actions:    []byte(`{"route_to_team":{"team_inbox_id":"` + uuid.New().String() + `"}}`),
			Priority:   50,
			IsActive:   true,
			CreatedBy:  userOwn,
		}
	}

	// Create — a foreign ctx inserting a row stamped with the real owning
	// tenantID must be rejected by the INSERT policy's WITH CHECK, not just
	// silently redirected.
	foreignAttempt := newRule(uuid.New(), "Foreign Insert Attempt")
	defer testutil.CleanupRow(t, pool, "routing_rules", foreignAttempt.ID)
	if err := repo.Create(ctxOther, foreignAttempt); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, testutil.WithSystemCtx(context.Background()), "routing_rules", foreignAttempt.ID, 0)

	rule := newRule(uuid.New(), "Auto-route "+uuid.New().String()[:6])
	if err := repo.Create(ctxOwn, rule); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "routing_rules", rule.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "routing_rules", rule.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "routing_rules", rule.ID, 0)

	// GetByID
	if _, err := repo.GetByID(ctxOther, tenantOwn, rule.ID); err != ErrRuleNotFound {
		t.Fatalf("GetByID (foreign ctx): expected ErrRuleNotFound, got %v", err)
	}
	got, err := repo.GetByID(ctxOwn, tenantOwn, rule.ID)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Name != rule.Name {
		t.Fatalf("GetByID (own ctx): name mismatch, got %q", got.Name)
	}
	if _, err := repo.GetByID(ctxOwn, tenantOwn, uuid.New()); err != ErrRuleNotFound {
		t.Fatalf("GetByID (unknown id): expected ErrRuleNotFound, got %v", err)
	}

	// Update — foreign ctx must not land even though the struct carries the
	// real owning tenantID; RLS (session tenant), not the WHERE literal,
	// must be what stops it.
	updateAttempt := newRule(rule.ID, "Hacked Name")
	updateAttempt.Priority = 999
	if err := repo.Update(ctxOther, updateAttempt); err != ErrRuleNotFound {
		t.Fatalf("Update (foreign ctx): expected ErrRuleNotFound, got %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, tenantOwn, rule.ID)
	if got.Name == "Hacked Name" {
		t.Fatalf("a foreign-tenant Update reached the row")
	}
	updateAttempt.Name = "Renamed Rule"
	if err := repo.Update(ctxOwn, updateAttempt); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, _ = repo.GetByID(ctxOwn, tenantOwn, rule.ID)
	if got.Name != "Renamed Rule" || got.Priority != 999 {
		t.Fatalf("own-tenant Update did not land: name=%q priority=%d", got.Name, got.Priority)
	}
	unknownUpdate := newRule(uuid.New(), "Ghost")
	if err := repo.Update(ctxOwn, unknownUpdate); err != ErrRuleNotFound {
		t.Fatalf("Update (unknown id): expected ErrRuleNotFound, got %v", err)
	}

	// A second, inactive rule on a different channel — used to prove
	// ListActive's channel filter and is_active exclusion, and ListAll's
	// inclusion of inactive rules.
	inactive := newRule(uuid.New(), "Inactive Chat Rule")
	inactive.IsActive = false
	inactive.Priority = 10
	chatChannel := "chat"
	inactive.Channel = &chatChannel
	if err := repo.Create(ctxOwn, inactive); err != nil {
		t.Fatalf("Create (inactive fixture): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "routing_rules", inactive.ID)

	// ListActive — foreign ctx sees nothing; own ctx sees only the active
	// email rule (priority ASC), filtered by channel excludes the chat one
	// even though it's inactive anyway.
	if list, err := repo.ListActive(ctxOther, tenantOwn, nil); err != nil || len(list) != 0 {
		t.Fatalf("ListActive (foreign ctx): expected empty/nil, got %d/%v", len(list), err)
	}
	if list, err := repo.ListActive(ctxOwn, tenantOwn, nil); err != nil || len(list) != 1 || list[0].ID != rule.ID {
		t.Fatalf("ListActive (own ctx, all channels): expected [rule]/nil, got %d entries/%v", len(list), err)
	}
	emailCh := "email"
	if list, err := repo.ListActive(ctxOwn, tenantOwn, &emailCh); err != nil || len(list) != 1 {
		t.Fatalf("ListActive (own ctx, email filter): expected 1/nil, got %d/%v", len(list), err)
	}
	if list, err := repo.ListActive(ctxOwn, tenantOwn, &chatChannel); err != nil || len(list) != 0 {
		t.Fatalf("ListActive (own ctx, chat filter): expected 0/nil (rule is inactive), got %d/%v", len(list), err)
	}

	// ListAll — includes the inactive rule, foreign ctx still sees nothing.
	if list, err := repo.ListAll(ctxOther, tenantOwn); err != nil || len(list) != 0 {
		t.Fatalf("ListAll (foreign ctx): expected empty/nil, got %d/%v", len(list), err)
	}
	if list, err := repo.ListAll(ctxOwn, tenantOwn); err != nil || len(list) != 2 {
		t.Fatalf("ListAll (own ctx): expected 2/nil, got %d/%v", len(list), err)
	}

	// Delete — foreign ctx must not land, own ctx must.
	if err := repo.Delete(ctxOther, tenantOwn, rule.ID); err != ErrRuleNotFound {
		t.Fatalf("Delete (foreign ctx): expected ErrRuleNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, ctxOwn, "routing_rules", rule.ID, 1)
	if err := repo.Delete(ctxOwn, tenantOwn, rule.ID); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, ctxOwn, "routing_rules", rule.ID, 0)
	if err := repo.Delete(ctxOwn, tenantOwn, rule.ID); err != ErrRuleNotFound {
		t.Fatalf("Delete (already deleted): expected ErrRuleNotFound, got %v", err)
	}
}
