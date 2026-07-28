package helpdesk

// Iteration 50 gave UpdateTicket, DeleteTicket, ReassignMessages, MergeTicketTx,
// and the equivalent Update/Delete methods for TicketQueue, CannedResponse,
// SLAPolicy, KBArticle, and RoutingRule an explicit tenant_id predicate --
// before that, they trusted RLS alone (mig 000122). This file proves the fix
// against a real repository call, not just a raw-SQL row count: each write is
// exercised once from a foreign-tenant ctx, passing the row's *real* tenantID
// as the explicit parameter -- so only RLS, not the WHERE clause, could be
// stopping it -- before repeating the same call from the owning ctx and
// confirming it lands.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTicketWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Ticket Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk Ticket Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	ticket := &Ticket{
		ID: uuid.New(), TenantID: tenantOwn, Subject: "Write Test Ticket",
		Status: TicketStatusOpen, Priority: TicketPriorityNormal,
		RequesterID: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctxOwn, ticket); err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", ticket.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "tickets", ticket.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "tickets", ticket.ID, 0)

	// UpdateTicket
	foreign := *ticket
	foreign.Subject = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateTicket(ctxOther, &foreign); err == nil {
		t.Fatalf("UpdateTicket (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetTicketByID(sysCtx, ticket.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if got.Subject == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the ticket")
	}
	foreign.Subject = "Updated"
	if err := repo.UpdateTicket(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateTicket (own ctx): %v", err)
	}
	got, _ = repo.GetTicketByID(ctxOwn, ticket.ID, tenantOwn)
	if got.Subject != "Updated" {
		t.Fatalf("own-tenant write did not land: subject=%q", got.Subject)
	}

	// CreateMessage + ReassignMessages need a second ticket as the reassign target.
	other := &Ticket{
		ID: uuid.New(), TenantID: tenantOwn, Subject: "Reassign Target",
		Status: TicketStatusOpen, Priority: TicketPriorityNormal,
		RequesterID: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctxOwn, other); err != nil {
		t.Fatalf("CreateTicket (target): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", other.ID)

	msg := &TicketMessage{
		ID: uuid.New(), TenantID: tenantOwn, TicketID: ticket.ID,
		AuthorID: ticket.RequesterID, Body: "hello", Attachments: []string{},
		CreatedAt: now,
	}
	if err := repo.CreateMessage(ctxOwn, msg); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "ticket_messages", msg.ID)

	// ReassignMessages carries an explicit tenant_id predicate but no
	// RowsAffected guard, so a foreign-ctx call returns nil and silently
	// touches zero rows -- verify via a read instead of the returned error.
	if err := repo.ReassignMessages(ctxOther, ticket.ID, other.ID, tenantOwn); err != nil {
		t.Fatalf("ReassignMessages (foreign ctx): %v", err)
	}
	msgs, err := repo.ListMessagesByTicket(sysCtx, ticket.ID, tenantOwn)
	if err != nil {
		t.Fatalf("ListMessagesByTicket: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("a foreign-tenant ReassignMessages moved the message: source now has %d", len(msgs))
	}
	if err := repo.ReassignMessages(ctxOwn, ticket.ID, other.ID, tenantOwn); err != nil {
		t.Fatalf("ReassignMessages (own ctx): %v", err)
	}
	msgs, _ = repo.ListMessagesByTicket(ctxOwn, ticket.ID, tenantOwn)
	if len(msgs) != 0 {
		t.Fatalf("own-tenant ReassignMessages did not move the message out of the source")
	}
	msgs, _ = repo.ListMessagesByTicket(ctxOwn, other.ID, tenantOwn)
	if len(msgs) != 1 {
		t.Fatalf("own-tenant ReassignMessages did not land on the target")
	}

	// DeleteTicket
	if err := repo.DeleteTicket(ctxOther, ticket.ID, tenantOwn); err == nil {
		t.Fatalf("DeleteTicket (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "tickets", ticket.ID, 1)
	if err := repo.DeleteTicket(ctxOwn, ticket.ID, tenantOwn); err != nil {
		t.Fatalf("DeleteTicket (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "tickets", ticket.ID, 0)
}

func TestMergeTicketTx_RespectsTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Merge Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk Merge Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	source := &Ticket{
		ID: uuid.New(), TenantID: tenantOwn, Subject: "Merge Source",
		Status: TicketStatusOpen, Priority: TicketPriorityNormal,
		RequesterID: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctxOwn, source); err != nil {
		t.Fatalf("CreateTicket (source): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", source.ID)

	target := &Ticket{
		ID: uuid.New(), TenantID: tenantOwn, Subject: "Merge Target",
		Status: TicketStatusOpen, Priority: TicketPriorityNormal,
		RequesterID: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTicket(ctxOwn, target); err != nil {
		t.Fatalf("CreateTicket (target): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", target.ID)

	mergedInto := target.ID
	foreign := *source
	foreign.Status = TicketStatusMerged
	foreign.MergedIntoID = &mergedInto
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.MergeTicketTx(ctxOther, &foreign, target.ID); err == nil {
		t.Fatalf("MergeTicketTx (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetTicketByID(sysCtx, source.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if got.Status == TicketStatusMerged {
		t.Fatalf("a foreign-tenant MergeTicketTx merged the ticket")
	}

	if err := repo.MergeTicketTx(ctxOwn, &foreign, target.ID); err != nil {
		t.Fatalf("MergeTicketTx (own ctx): %v", err)
	}
	got, _ = repo.GetTicketByID(ctxOwn, source.ID, tenantOwn)
	if got.Status != TicketStatusMerged {
		t.Fatalf("own-tenant MergeTicketTx did not land: status=%q", got.Status)
	}
}

func TestQueueWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Queue Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk Queue Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	q := &TicketQueue{
		ID: uuid.New(), TenantID: tenantOwn, Name: "Write Test Queue",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateQueue(ctxOwn, q); err != nil {
		t.Fatalf("CreateQueue: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "ticket_queues", q.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "ticket_queues", q.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "ticket_queues", q.ID, 0)

	foreign := *q
	foreign.Name = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateQueue(ctxOther, &foreign); err == nil {
		t.Fatalf("UpdateQueue (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetQueueByID(sysCtx, q.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetQueueByID: %v", err)
	}
	if got.Name == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the queue")
	}
	foreign.Name = "Renamed"
	if err := repo.UpdateQueue(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateQueue (own ctx): %v", err)
	}
	got, _ = repo.GetQueueByID(ctxOwn, q.ID, tenantOwn)
	if got.Name != "Renamed" {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	if err := repo.DeleteQueue(ctxOther, q.ID, tenantOwn); err == nil {
		t.Fatalf("DeleteQueue (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "ticket_queues", q.ID, 1)
	if err := repo.DeleteQueue(ctxOwn, q.ID, tenantOwn); err != nil {
		t.Fatalf("DeleteQueue (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "ticket_queues", q.ID, 0)
}

func TestCannedResponseWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Canned Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk Canned Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	cr := &CannedResponse{
		ID: uuid.New(), TenantID: tenantOwn, Name: "Write Test Reply",
		Body: "Thanks!", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateCannedResponse(ctxOwn, cr); err != nil {
		t.Fatalf("CreateCannedResponse: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "canned_responses", cr.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "canned_responses", cr.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "canned_responses", cr.ID, 0)

	foreign := *cr
	foreign.Body = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateCannedResponse(ctxOther, &foreign); err == nil {
		t.Fatalf("UpdateCannedResponse (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetCannedResponseByID(sysCtx, cr.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetCannedResponseByID: %v", err)
	}
	if got.Body == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the canned response")
	}
	foreign.Body = "Updated"
	if err := repo.UpdateCannedResponse(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateCannedResponse (own ctx): %v", err)
	}
	got, _ = repo.GetCannedResponseByID(ctxOwn, cr.ID, tenantOwn)
	if got.Body != "Updated" {
		t.Fatalf("own-tenant write did not land: body=%q", got.Body)
	}

	if err := repo.DeleteCannedResponse(ctxOther, cr.ID, tenantOwn); err == nil {
		t.Fatalf("DeleteCannedResponse (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "canned_responses", cr.ID, 1)
	if err := repo.DeleteCannedResponse(ctxOwn, cr.ID, tenantOwn); err != nil {
		t.Fatalf("DeleteCannedResponse (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "canned_responses", cr.ID, 0)
}

func TestSLAPolicyWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk SLA Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk SLA Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	p := &SLAPolicy{
		ID: uuid.New(), TenantID: tenantOwn, Name: "Write Test SLA",
		FirstResponseMins: 60, ResolutionMins: 480,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSLAPolicy(ctxOwn, p); err != nil {
		t.Fatalf("CreateSLAPolicy: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "sla_policies", p.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "sla_policies", p.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "sla_policies", p.ID, 0)

	foreign := *p
	foreign.ResolutionMins = 999
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateSLAPolicy(ctxOther, &foreign); err == nil {
		t.Fatalf("UpdateSLAPolicy (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetSLAPolicyByID(sysCtx, p.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetSLAPolicyByID: %v", err)
	}
	if got.ResolutionMins == 999 {
		t.Fatalf("a foreign-tenant write reached the SLA policy")
	}
	foreign.ResolutionMins = 720
	if err := repo.UpdateSLAPolicy(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateSLAPolicy (own ctx): %v", err)
	}
	got, _ = repo.GetSLAPolicyByID(ctxOwn, p.ID, tenantOwn)
	if got.ResolutionMins != 720 {
		t.Fatalf("own-tenant write did not land: resolution_mins=%d", got.ResolutionMins)
	}

	if err := repo.DeleteSLAPolicy(ctxOther, p.ID, tenantOwn); err == nil {
		t.Fatalf("DeleteSLAPolicy (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "sla_policies", p.ID, 1)
	if err := repo.DeleteSLAPolicy(ctxOwn, p.ID, tenantOwn); err != nil {
		t.Fatalf("DeleteSLAPolicy (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "sla_policies", p.ID, 0)
}

func TestKBArticleWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk KB Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk KB Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	a := &KBArticle{
		ID: uuid.New(), TenantID: tenantOwn, Title: "Write Test Article",
		Content: "body", Category: "general", Status: string(KBArticleStatusDraft),
		AuthorID: uuid.New(), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateKBArticle(ctxOwn, a); err != nil {
		t.Fatalf("CreateKBArticle: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "helpdesk_kb_articles", a.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "helpdesk_kb_articles", a.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "helpdesk_kb_articles", a.ID, 0)

	foreign := *a
	foreign.Title = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateKBArticle(ctxOther, &foreign); err == nil {
		t.Fatalf("UpdateKBArticle (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetKBArticleByID(sysCtx, a.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetKBArticleByID: %v", err)
	}
	if got.Title == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the KB article")
	}
	foreign.Title = "Updated"
	if err := repo.UpdateKBArticle(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateKBArticle (own ctx): %v", err)
	}
	got, _ = repo.GetKBArticleByID(ctxOwn, a.ID, tenantOwn)
	if got.Title != "Updated" {
		t.Fatalf("own-tenant write did not land: title=%q", got.Title)
	}

	if err := repo.DeleteKBArticle(ctxOther, a.ID, tenantOwn); err == nil {
		t.Fatalf("DeleteKBArticle (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "helpdesk_kb_articles", a.ID, 1)
	if err := repo.DeleteKBArticle(ctxOwn, a.ID, tenantOwn); err != nil {
		t.Fatalf("DeleteKBArticle (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "helpdesk_kb_articles", a.ID, 0)
}

func TestRoutingRuleWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Helpdesk Routing Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Helpdesk Routing Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	rr := &RoutingRule{
		ID: uuid.New(), TenantID: tenantOwn, Name: "Write Test Rule",
		Conditions: map[string]any{"channel": "email"}, Priority: 1, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateRoutingRule(ctxOwn, rr); err != nil {
		t.Fatalf("CreateRoutingRule: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "helpdesk_routing_rules", rr.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "helpdesk_routing_rules", rr.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "helpdesk_routing_rules", rr.ID, 0)

	foreign := *rr
	foreign.Name = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateRoutingRule(ctxOther, &foreign); err == nil {
		t.Fatalf("UpdateRoutingRule (foreign ctx): expected an error, got nil")
	}
	got, err := repo.GetRoutingRuleByID(sysCtx, rr.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetRoutingRuleByID: %v", err)
	}
	if got.Name == "Hacked" {
		t.Fatalf("a foreign-tenant write reached the routing rule")
	}
	foreign.Name = "Renamed"
	if err := repo.UpdateRoutingRule(ctxOwn, &foreign); err != nil {
		t.Fatalf("UpdateRoutingRule (own ctx): %v", err)
	}
	got, _ = repo.GetRoutingRuleByID(ctxOwn, rr.ID, tenantOwn)
	if got.Name != "Renamed" {
		t.Fatalf("own-tenant write did not land: name=%q", got.Name)
	}

	if err := repo.DeleteRoutingRule(ctxOther, rr.ID, tenantOwn); err == nil {
		t.Fatalf("DeleteRoutingRule (foreign ctx): expected an error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "helpdesk_routing_rules", rr.ID, 1)
	if err := repo.DeleteRoutingRule(ctxOwn, rr.ID, tenantOwn); err != nil {
		t.Fatalf("DeleteRoutingRule (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "helpdesk_routing_rules", rr.ID, 0)
}
