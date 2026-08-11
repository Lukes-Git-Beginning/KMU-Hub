package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/dialer"
	"github.com/kmuhub/kmuhub/internal/middleware"
	dialerv1 "github.com/kmuhub/kmuhub/proto/dialer/v1"
)

// ctxWithTenantAndUser builds a context carrying both tenant and caller
// identity, needed by CreateCampaign (which reads the caller from the
// x-user-id metadata to satisfy dialer_campaigns.created_by's FK to users).
func ctxWithTenantAndUser(tenantID, userID uuid.UUID) context.Context {
	ctx := ctxWithTenant(tenantID)
	return context.WithValue(ctx, middleware.UserIDKey, userID.String())
}

// ============================================================================
// CreateCampaign
// ============================================================================

func TestCreateCampaign_Happy(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	userID := uuid.New()
	ctx := ctxWithTenantAndUser(tenantID, userID)

	resp, err := srv.CreateCampaign(ctx, &dialerv1.CreateCampaignRequest{Name: "Q1 Outreach"})
	requireGRPCOK(t, err)
	if resp.GetStatus() != dialerv1.CampaignStatus_CAMPAIGN_STATUS_DRAFT {
		t.Errorf("expected draft status, got %v", resp.GetStatus())
	}
	if resp.GetCreatedBy() != userID.String() {
		t.Errorf("expected created_by %s, got %s", userID, resp.GetCreatedBy())
	}
	if len(cr.campaigns) != 1 {
		t.Errorf("expected 1 stored campaign, got %d", len(cr.campaigns))
	}
}

func TestCreateCampaign_MissingCallerIdentity(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	// Tenant set, but no UserIDKey — createdBy would be uuid.Nil, which the DB
	// FK on dialer_campaigns.created_by would reject.
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.CreateCampaign(ctx, &dialerv1.CreateCampaignRequest{Name: "No Caller"})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateCampaign_InvalidAgentID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenantAndUser(uuid.New(), uuid.New())

	_, err := srv.CreateCampaign(ctx, &dialerv1.CreateCampaignRequest{
		Name:             "Bad Agent",
		AssignedAgentIds: []string{"not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_InvalidSettingsJSON(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenantAndUser(uuid.New(), uuid.New())

	badSettings := "{not json"
	_, err := srv.CreateCampaign(ctx, &dialerv1.CreateCampaignRequest{
		Name:     "Bad Settings",
		Settings: &badSettings,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateCampaign_InvalidTenantIDInRequest(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenantAndUser(uuid.New(), uuid.New())

	_, err := srv.CreateCampaign(ctx, &dialerv1.CreateCampaignRequest{
		Name:     "Bad Tenant",
		TenantId: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// GetCampaign
// ============================================================================

func TestGetCampaign_Happy(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Existing", Status: dialer.CampaignStatusDraft, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	resp, err := srv.GetCampaign(ctx, &dialerv1.GetCampaignRequest{CampaignId: id.String()})
	requireGRPCOK(t, err)
	if resp.GetId() != id.String() {
		t.Errorf("expected id %s, got %s", id, resp.GetId())
	}
}

func TestGetCampaign_InvalidCampaignID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.GetCampaign(ctx, &dialerv1.GetCampaignRequest{CampaignId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetCampaign_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()

	_, err := srv.GetCampaign(context.Background(), &dialerv1.GetCampaignRequest{CampaignId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetCampaign_NotFound(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.GetCampaign(ctx, &dialerv1.GetCampaignRequest{CampaignId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// ListCampaigns
// ============================================================================

func TestListCampaigns_EmptyWireShape(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	resp, err := srv.ListCampaigns(ctx, &dialerv1.ListCampaignsRequest{})
	requireGRPCOK(t, err)
	if resp.GetCampaigns() == nil {
		t.Error("expected empty slice, got nil (wire-shape must be [] not null)")
	}
	if len(resp.GetCampaigns()) != 0 {
		t.Errorf("expected 0 campaigns, got %d", len(resp.GetCampaigns()))
	}
}

func TestListCampaigns_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()

	_, err := srv.ListCampaigns(context.Background(), &dialerv1.ListCampaignsRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// ============================================================================
// UpdateCampaign
// ============================================================================

func TestUpdateCampaign_Happy(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Old Name", Status: dialer.CampaignStatusDraft, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	newName := "New Name"
	resp, err := srv.UpdateCampaign(ctx, &dialerv1.UpdateCampaignRequest{CampaignId: id.String(), Name: &newName})
	requireGRPCOK(t, err)
	if resp.GetName() != newName {
		t.Errorf("expected name %q, got %q", newName, resp.GetName())
	}
}

func TestUpdateCampaign_NotDraft(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Active One", Status: dialer.CampaignStatusActive, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	newName := "Attempted Edit"
	_, err := srv.UpdateCampaign(ctx, &dialerv1.UpdateCampaignRequest{CampaignId: id.String(), Name: &newName})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestUpdateCampaign_InvalidCampaignID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.UpdateCampaign(ctx, &dialerv1.UpdateCampaignRequest{CampaignId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateCampaign_InvalidSettingsJSON(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Draft", Status: dialer.CampaignStatusDraft, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	badSettings := "{not json"
	_, err := srv.UpdateCampaign(ctx, &dialerv1.UpdateCampaignRequest{CampaignId: id.String(), Settings: &badSettings})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// StartCampaign
// ============================================================================

func TestStartCampaign_Happy(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Ready", Status: dialer.CampaignStatusDraft, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New(), ContactCount: 3}

	resp, err := srv.StartCampaign(ctx, &dialerv1.StartCampaignRequest{CampaignId: id.String()})
	requireGRPCOK(t, err)
	if resp.GetStatus() != dialerv1.CampaignStatus_CAMPAIGN_STATUS_ACTIVE {
		t.Errorf("expected active status, got %v", resp.GetStatus())
	}
}

func TestStartCampaign_NotDraft(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Already Active", Status: dialer.CampaignStatusActive, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New(), ContactCount: 3}

	_, err := srv.StartCampaign(ctx, &dialerv1.StartCampaignRequest{CampaignId: id.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestStartCampaign_NoContacts(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Empty", Status: dialer.CampaignStatusDraft, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New(), ContactCount: 0}

	_, err := srv.StartCampaign(ctx, &dialerv1.StartCampaignRequest{CampaignId: id.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestStartCampaign_InvalidCampaignID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.StartCampaign(ctx, &dialerv1.StartCampaignRequest{CampaignId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// PauseCampaign
// ============================================================================

func TestPauseCampaign_ActiveToPaused(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Running", Status: dialer.CampaignStatusActive, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	resp, err := srv.PauseCampaign(ctx, &dialerv1.PauseCampaignRequest{CampaignId: id.String()})
	requireGRPCOK(t, err)
	if resp.GetStatus() != dialerv1.CampaignStatus_CAMPAIGN_STATUS_PAUSED {
		t.Errorf("expected paused status, got %v", resp.GetStatus())
	}
}

func TestPauseCampaign_PausedToActive(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Paused", Status: dialer.CampaignStatusPaused, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	resp, err := srv.PauseCampaign(ctx, &dialerv1.PauseCampaignRequest{CampaignId: id.String()})
	requireGRPCOK(t, err)
	if resp.GetStatus() != dialerv1.CampaignStatus_CAMPAIGN_STATUS_ACTIVE {
		t.Errorf("expected active status, got %v", resp.GetStatus())
	}
}

func TestPauseCampaign_InvalidTransition(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Draft", Status: dialer.CampaignStatusDraft, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	_, err := srv.PauseCampaign(ctx, &dialerv1.PauseCampaignRequest{CampaignId: id.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestPauseCampaign_InvalidCampaignID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.PauseCampaign(ctx, &dialerv1.PauseCampaignRequest{CampaignId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// ArchiveCampaign
// ============================================================================

func TestArchiveCampaign_FromCompleted(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Done", Status: dialer.CampaignStatusCompleted, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	_, err := srv.ArchiveCampaign(ctx, &dialerv1.ArchiveCampaignRequest{CampaignId: id.String()})
	requireGRPCOK(t, err)
}

func TestArchiveCampaign_InvalidTransition(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	id := uuid.New()
	cr.campaigns[id] = &dialer.Campaign{ID: id, TenantID: tenantID, Name: "Draft", Status: dialer.CampaignStatusDraft, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	_, err := srv.ArchiveCampaign(ctx, &dialerv1.ArchiveCampaignRequest{CampaignId: id.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestArchiveCampaign_InvalidCampaignID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.ArchiveCampaign(ctx, &dialerv1.ArchiveCampaignRequest{CampaignId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// AddContactsToCampaign
// ============================================================================

func TestAddContactsToCampaign_Happy(t *testing.T) {
	srv, cr, _, _, br := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	campaignID := uuid.New()
	cr.campaigns[campaignID] = &dialer.Campaign{ID: campaignID, TenantID: tenantID, Name: "Fresh", Status: dialer.CampaignStatusDraft, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	contactID := uuid.New()
	br.contactDetails[contactID] = &dialer.ContactDetails{ID: contactID, Name: "Max Mustermann", Phone: "+491701234567"}

	resp, err := srv.AddContactsToCampaign(ctx, &dialerv1.AddContactsToCampaignRequest{
		CampaignId: campaignID.String(),
		ContactIds: []string{contactID.String()},
	})
	requireGRPCOK(t, err)
	if resp.GetAddedCount() != 1 {
		t.Errorf("expected added_count 1, got %d", resp.GetAddedCount())
	}
}

func TestAddContactsToCampaign_NotDraft(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	campaignID := uuid.New()
	cr.campaigns[campaignID] = &dialer.Campaign{ID: campaignID, TenantID: tenantID, Name: "Live", Status: dialer.CampaignStatusActive, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	_, err := srv.AddContactsToCampaign(ctx, &dialerv1.AddContactsToCampaignRequest{
		CampaignId: campaignID.String(),
		ContactIds: []string{uuid.New().String()},
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestAddContactsToCampaign_InvalidContactID(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	campaignID := uuid.New()
	cr.campaigns[campaignID] = &dialer.Campaign{ID: campaignID, TenantID: tenantID, Name: "Fresh", Status: dialer.CampaignStatusDraft, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	_, err := srv.AddContactsToCampaign(ctx, &dialerv1.AddContactsToCampaignRequest{
		CampaignId: campaignID.String(),
		ContactIds: []string{"not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAddContactsToCampaign_InvalidCampaignID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.AddContactsToCampaign(ctx, &dialerv1.AddContactsToCampaignRequest{CampaignId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// GetNextContact
// ============================================================================

func TestGetNextContact_Happy(t *testing.T) {
	srv, cr, _, _, br := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	campaignID := uuid.New()
	cr.campaigns[campaignID] = &dialer.Campaign{ID: campaignID, TenantID: tenantID, Name: "Active", Status: dialer.CampaignStatusActive, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	contactID := uuid.New()
	ccID := uuid.New()
	cr.nextPending[campaignID] = &dialer.CampaignContact{ID: ccID, CampaignID: campaignID, ContactID: contactID, Status: dialer.ContactStatusPending}
	br.contactDetails[contactID] = &dialer.ContactDetails{ID: contactID, Name: "Erika Musterfrau", Phone: "+491701234567"}

	resp, err := srv.GetNextContact(ctx, &dialerv1.GetNextContactRequest{CampaignId: campaignID.String()})
	requireGRPCOK(t, err)
	if resp.GetContactId() != contactID.String() {
		t.Errorf("expected contact_id %s, got %s", contactID, resp.GetContactId())
	}
	if resp.GetContactName() != "Erika Musterfrau" {
		t.Errorf("expected enriched contact name, got %q", resp.GetContactName())
	}
}

func TestGetNextContact_NotActive(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	campaignID := uuid.New()
	cr.campaigns[campaignID] = &dialer.Campaign{ID: campaignID, TenantID: tenantID, Name: "Draft", Status: dialer.CampaignStatusDraft, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	_, err := srv.GetNextContact(ctx, &dialerv1.GetNextContactRequest{CampaignId: campaignID.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestGetNextContact_QueueExhausted(t *testing.T) {
	srv, cr, _, _, _ := newDialerTestServer()
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	campaignID := uuid.New()
	cr.campaigns[campaignID] = &dialer.Campaign{ID: campaignID, TenantID: tenantID, Name: "Active", Status: dialer.CampaignStatusActive, Mode: dialer.CampaignModePreview, CreatedBy: uuid.New()}

	// stubCampaignRepo.GetNextPendingContact always returns
	// ErrNoContactsAvailable — this exercises the auto-complete branch in
	// Service.GetNextContact (campaign transitions to completed).
	_, err := srv.GetNextContact(ctx, &dialerv1.GetNextContactRequest{CampaignId: campaignID.String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetNextContact_InvalidCampaignID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.GetNextContact(ctx, &dialerv1.GetNextContactRequest{CampaignId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// SkipContact
// ============================================================================

func TestSkipContact_Happy(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.SkipContact(ctx, &dialerv1.SkipContactRequest{CampaignContactId: uuid.New().String()})
	requireGRPCOK(t, err)
}

func TestSkipContact_InvalidID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.SkipContact(ctx, &dialerv1.SkipContactRequest{CampaignContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSkipContact_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()

	_, err := srv.SkipContact(context.Background(), &dialerv1.SkipContactRequest{CampaignContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// ============================================================================
// RequeueContact
// ============================================================================

func TestRequeueContact_Happy(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.RequeueContact(ctx, &dialerv1.RequeueContactRequest{CampaignContactId: uuid.New().String()})
	requireGRPCOK(t, err)
}

func TestRequeueContact_InvalidID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.RequeueContact(ctx, &dialerv1.RequeueContactRequest{CampaignContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRequeueContact_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()

	_, err := srv.RequeueContact(context.Background(), &dialerv1.RequeueContactRequest{CampaignContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// ============================================================================
// ListCampaignContacts
// ============================================================================

func TestListCampaignContacts_EmptyWireShape(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	resp, err := srv.ListCampaignContacts(ctx, &dialerv1.ListCampaignContactsRequest{CampaignId: uuid.New().String()})
	requireGRPCOK(t, err)
	if resp.GetContacts() == nil {
		t.Error("expected empty slice, got nil (wire-shape must be [] not null)")
	}
	if len(resp.GetContacts()) != 0 {
		t.Errorf("expected 0 contacts, got %d", len(resp.GetContacts()))
	}
}

func TestListCampaignContacts_InvalidCampaignID(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.ListCampaignContacts(ctx, &dialerv1.ListCampaignContactsRequest{CampaignId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListCampaignContacts_MissingTenant(t *testing.T) {
	srv, _, _, _, _ := newDialerTestServer()

	_, err := srv.ListCampaignContacts(context.Background(), &dialerv1.ListCampaignContactsRequest{CampaignId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}
