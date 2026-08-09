package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// fireDueEntities -- due resolution
//
// These cover the difference between a time-BASED trigger and a time-DRIVEN
// one. Before due resolution existed the poller fired every active time-based
// automation on every tick with an empty payload, so "invoice overdue" meant
// "five minutes elapsed" and the shipped dunning template's
// invoice.days_overdue condition could never be true.
// ============================================================================

func newTenantAutomation(triggerType string, tenantID uuid.UUID) *models.Automation {
	auto := newTestAutomation(triggerType)
	auto.TenantID = tenantID
	return auto
}

func TestFireDueEntities_NothingDue_DoesNotExecute(t *testing.T) {
	auto := newTestAutomation("biz.invoice.overdue")
	repo := &fakeTimeBasedRepo{listResult: []*models.Automation{auto}}
	executor := newFakeEngineExecutor()

	// Resolver knows the trigger type but finds no due entity -- no overdue
	// invoice exists right now.
	resolver := &fakeDueResolver{byTrigger: map[string][]DueEntity{
		"biz.invoice.overdue": {},
	}}

	poller := NewTimeTriggerPoller(repo, executor, resolver, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())

	assertNoExecution(t, executor)
	if len(repo.firedKeys()) != 0 {
		t.Fatalf("expected no fire claim when nothing is due, got %d", len(repo.firedKeys()))
	}
}

func TestFireDueEntities_ExecutesOncePerDueEntity(t *testing.T) {
	tenantID := uuid.New()
	auto := newTenantAutomation("biz.invoice.overdue", tenantID)
	repo := &fakeTimeBasedRepo{listResult: []*models.Automation{auto}}
	executor := newFakeEngineExecutor()

	resolver := &fakeDueResolver{byTrigger: map[string][]DueEntity{
		"biz.invoice.overdue": {
			{Key: "invoice:a:2026-08-08", ResourceID: "a", Payload: map[string]any{"invoice": map[string]any{"days_overdue": 20}}},
			{Key: "invoice:b:2026-08-08", ResourceID: "b", Payload: map[string]any{"invoice": map[string]any{"days_overdue": 3}}},
		},
	}}

	poller := NewTimeTriggerPoller(repo, executor, resolver, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())

	executions := waitExecutions(t, executor, 2)
	assertNoExecution(t, executor)

	gotResources := map[string]bool{}
	for _, exec := range executions {
		if exec.automationID != auto.ID {
			t.Fatalf("expected automation %s, got %s", auto.ID, exec.automationID)
		}
		gotResources[exec.event.ResourceID] = true
	}
	if !gotResources["a"] || !gotResources["b"] {
		t.Fatalf("expected one execution per due entity (a and b), got %v", gotResources)
	}

	// The resolver must be asked for the automation's own tenant -- the poller
	// runs under system context, so this argument is the only thing that keeps
	// tenant A's automation off tenant B's invoices.
	calls := resolver.resolveCalls()
	if len(calls) != 1 || calls[0].tenantID != tenantID || calls[0].triggerType != "biz.invoice.overdue" {
		t.Fatalf("expected one resolve for (biz.invoice.overdue, %s), got %+v", tenantID, calls)
	}
}

func TestFireDueEntities_EventCarriesTenantResourceAndNestedPayload(t *testing.T) {
	tenantID := uuid.New()
	auto := newTenantAutomation("biz.invoice.overdue", tenantID)
	repo := &fakeTimeBasedRepo{listResult: []*models.Automation{auto}}
	executor := newFakeEngineExecutor()

	resolver := &fakeDueResolver{byTrigger: map[string][]DueEntity{
		"biz.invoice.overdue": {{
			Key:        "invoice:a:2026-08-08",
			ResourceID: "a",
			Payload: map[string]any{"invoice": map[string]any{
				"id":           "a",
				"days_overdue": 20,
				"total":        1234.5,
			}},
		}},
	}}

	poller := NewTimeTriggerPoller(repo, executor, resolver, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())

	exec := waitExecutions(t, executor, 1)[0]

	if exec.event.TenantID != tenantID {
		t.Errorf("event tenant = %s, want %s", exec.event.TenantID, tenantID)
	}
	if exec.event.ResourceID != "a" {
		t.Errorf("event resource_id = %q, want %q", exec.event.ResourceID, "a")
	}
	if exec.event.Type != "biz.invoice.overdue" {
		t.Errorf("event type = %q, want biz.invoice.overdue", exec.event.Type)
	}
	// ModuleID must stay off the automation sentinel or engine.Execute's loop
	// guard drops the event before conditions are evaluated.
	if exec.event.ModuleID == "automation" {
		t.Errorf("event module_id must not be the automation loop sentinel")
	}

	// Payload must round-trip NESTED: condition.getFieldValue walks
	// "invoice.days_overdue" as nested maps, so a flattened payload would never
	// match a condition.
	var decoded map[string]any
	if err := json.Unmarshal(exec.event.Payload, &decoded); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	invoice, ok := decoded["invoice"].(map[string]any)
	if !ok {
		t.Fatalf("payload[invoice] is not a nested object: %#v", decoded)
	}
	if got := invoice["days_overdue"]; got != float64(20) {
		t.Errorf("payload invoice.days_overdue = %v, want 20", got)
	}
}

func TestFireDueEntities_FireClaimLost_DoesNotExecute(t *testing.T) {
	auto := newTestAutomation("biz.invoice.overdue")
	repo := &fakeTimeBasedRepo{
		listResult: []*models.Automation{auto},
		// Entity already fired -- by an earlier tick, or another instance.
		fireResult: map[string]bool{"invoice:a:2026-08-08": false},
	}
	executor := newFakeEngineExecutor()

	resolver := &fakeDueResolver{byTrigger: map[string][]DueEntity{
		"biz.invoice.overdue": {
			{Key: "invoice:a:2026-08-08", ResourceID: "a"},
			{Key: "invoice:b:2026-08-08", ResourceID: "b"},
		},
	}}

	poller := NewTimeTriggerPoller(repo, executor, resolver, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())

	executions := waitExecutions(t, executor, 1)
	if executions[0].event.ResourceID != "b" {
		t.Fatalf("expected only the unclaimed entity b to execute, got %q", executions[0].event.ResourceID)
	}
	assertNoExecution(t, executor)

	if len(repo.firedKeys()) != 2 {
		t.Fatalf("expected a fire claim attempt for both entities, got %d", len(repo.firedKeys()))
	}
}

func TestFireDueEntities_FireClaimUsesAutomationTenant(t *testing.T) {
	tenantID := uuid.New()
	auto := newTenantAutomation("biz.invoice.overdue", tenantID)
	repo := &fakeTimeBasedRepo{listResult: []*models.Automation{auto}}
	executor := newFakeEngineExecutor()

	resolver := &fakeDueResolver{byTrigger: map[string][]DueEntity{
		"biz.invoice.overdue": {{Key: "invoice:a:2026-08-08", ResourceID: "a"}},
	}}

	poller := NewTimeTriggerPoller(repo, executor, resolver, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())
	waitExecutions(t, executor, 1)

	fires := repo.firedKeys()
	if len(fires) != 1 {
		t.Fatalf("expected one fire claim, got %d", len(fires))
	}
	if fires[0].tenantID != tenantID {
		t.Errorf("fire claim tenant = %s, want %s", fires[0].tenantID, tenantID)
	}
	if fires[0].automationID != auto.ID {
		t.Errorf("fire claim automation = %s, want %s", fires[0].automationID, auto.ID)
	}
	if fires[0].entityKey != "invoice:a:2026-08-08" {
		t.Errorf("fire claim entity_key = %q, want invoice:a:2026-08-08", fires[0].entityKey)
	}
}

func TestFireDueEntities_FireClaimError_SkipsWithoutExecuting(t *testing.T) {
	auto := newTestAutomation("biz.invoice.overdue")
	repo := &fakeTimeBasedRepo{
		listResult: []*models.Automation{auto},
		fireErr:    context.Canceled,
	}
	executor := newFakeEngineExecutor()

	resolver := &fakeDueResolver{byTrigger: map[string][]DueEntity{
		"biz.invoice.overdue": {
			{Key: "invoice:a:2026-08-08", ResourceID: "a"},
			{Key: "invoice:b:2026-08-08", ResourceID: "b"},
		},
	}}

	poller := NewTimeTriggerPoller(repo, executor, resolver, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())

	assertNoExecution(t, executor)
	if len(repo.firedKeys()) != 2 {
		t.Fatalf("a failing claim must not abort the remaining entities, got %d attempts", len(repo.firedKeys()))
	}
}

func TestFireDueEntities_ResolverError_DoesNotExecuteAndDoesNotStopOtherAutomations(t *testing.T) {
	auto := newTestAutomation("biz.invoice.overdue")
	other := newTestAutomation("calendar.event.upcoming")
	repo := &fakeTimeBasedRepo{listResult: []*models.Automation{auto, other}}
	executor := newFakeEngineExecutor()

	// err applies to every Resolve call in this fake: neither automation may
	// execute, and the poller must walk the whole list without panicking.
	resolver := &fakeDueResolver{err: context.DeadlineExceeded}

	poller := NewTimeTriggerPoller(repo, executor, resolver, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())

	assertNoExecution(t, executor)
	if len(resolver.resolveCalls()) != 2 {
		t.Fatalf("expected a resolve attempt for both automations, got %d", len(resolver.resolveCalls()))
	}
	if len(repo.firedKeys()) != 0 {
		t.Fatalf("a failed resolve must not claim a fire, got %d", len(repo.firedKeys()))
	}
}

func TestFireDueEntities_UnknownTriggerType_DoesNotExecute(t *testing.T) {
	// A registry entry flagged TimeBased that no resolver knows: the poller
	// must fall through to the error path, not fire with an empty payload.
	auto := newTestAutomation("calendar.event.upcoming")
	repo := &fakeTimeBasedRepo{listResult: []*models.Automation{auto}}
	executor := newFakeEngineExecutor()

	resolver := &fakeDueResolver{err: ErrUnknownTimeTrigger}

	poller := NewTimeTriggerPoller(repo, executor, resolver, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())

	assertNoExecution(t, executor)
}

func TestFireDueEntities_CapSizedResultStillFiresEveryEntity(t *testing.T) {
	// The cap is a resolver-side LIMIT; hitting it warns and defers the tail to
	// the next tick, but every entity the resolver did return must fire.
	auto := newTestAutomation("biz.invoice.overdue")
	repo := &fakeTimeBasedRepo{listResult: []*models.Automation{auto}}
	executor := newFakeEngineExecutor()

	entities := make([]DueEntity, maxDueEntitiesPerAutomation)
	for i := range entities {
		entities[i] = DueEntity{
			Key:        fmt.Sprintf("invoice:%d:2026-08-08", i),
			ResourceID: fmt.Sprintf("%d", i),
		}
	}
	resolver := &fakeDueResolver{byTrigger: map[string][]DueEntity{"biz.invoice.overdue": entities}}

	poller := NewTimeTriggerPoller(repo, executor, resolver, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())

	waitExecutions(t, executor, maxDueEntitiesPerAutomation)
	assertNoExecution(t, executor)
}
