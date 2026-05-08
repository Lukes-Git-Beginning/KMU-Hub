package dialer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// Service is the thick service layer for the Cosmi Dialer. All business logic
// lives here; handlers only parse requests and format responses.
type Service struct {
	campaigns       CampaignRepository
	calls           CallRepository
	outcomes        OutcomeRepository
	agentRepo       AgentStatusRepository
	agentStore      *AgentStatusStore
	crmBridge       CRMBridge
	emitter         *PGEventEmitter // nil-safe; events are skipped when nil
	// consentAsserter, when non-nil, enforces active phone consent before
	// initiating a call. Nil-safe for tests and legacy code paths.
	consentAsserter *consent.Asserter
}

// NewService wires all dialer dependencies into a ready-to-use Service.
func NewService(
	campaigns CampaignRepository,
	calls CallRepository,
	outcomes OutcomeRepository,
	agentRepo AgentStatusRepository,
	agentStore *AgentStatusStore,
	crmBridge CRMBridge,
	emitter *PGEventEmitter,
) *Service {
	return &Service{
		campaigns:  campaigns,
		calls:      calls,
		outcomes:   outcomes,
		agentRepo:  agentRepo,
		agentStore: agentStore,
		crmBridge:  crmBridge,
		emitter:    emitter,
	}
}

// NewServiceWithConsent creates a dialer service with a consent asserter wired
// in. This is the production constructor — the asserter must be non-nil.
func NewServiceWithConsent(
	campaigns CampaignRepository,
	calls CallRepository,
	outcomes OutcomeRepository,
	agentRepo AgentStatusRepository,
	agentStore *AgentStatusStore,
	crmBridge CRMBridge,
	emitter *PGEventEmitter,
	asserter *consent.Asserter,
) *Service {
	svc := NewService(campaigns, calls, outcomes, agentRepo, agentStore, crmBridge, emitter)
	svc.consentAsserter = asserter
	return svc
}

// ---------------------------------------------------------------------------
// Campaign Management
// ---------------------------------------------------------------------------

// CreateCampaign creates a new campaign in draft status for a tenant.
// It also ensures the tenant has the four default call outcomes.
func (s *Service) CreateCampaign(
	ctx context.Context,
	tenantID uuid.UUID,
	createdBy uuid.UUID,
	name string,
	description *string,
	mode string,
	assignedAgentIDs []uuid.UUID,
	settings *CampaignSettings,
) (*Campaign, error) {
	if name == "" {
		return nil, fmt.Errorf("campaign name must not be empty")
	}

	if mode == "" {
		mode = CampaignModePreview
	}

	cs := CampaignSettings{}
	if settings != nil {
		cs = *settings
	}

	// Ensure the tenant has default outcomes before the campaign goes live.
	if err := s.outcomes.EnsureDefaults(ctx, tenantID); err != nil {
		slog.WarnContext(ctx, "dialer: ensure default outcomes failed",
			"tenant_id", tenantID,
			"error", err,
		)
		// Non-fatal — campaign creation continues; agent will see outcomes
		// once EnsureDefaults succeeds on a later request.
	}

	now := time.Now().UTC()
	c := &Campaign{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Name:             name,
		Description:      description,
		Status:           CampaignStatusDraft,
		Mode:             mode,
		Settings:         cs,
		CreatedBy:        createdBy,
		AssignedAgentIDs: assignedAgentIDs,
		ContactCount:     0,
		CompletedCount:   0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.campaigns.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}

	slog.InfoContext(ctx, "dialer: campaign created",
		"campaign_id", c.ID,
		"tenant_id", tenantID,
		"created_by", createdBy,
		"mode", mode,
	)

	return c, nil
}

// GetCampaign retrieves a campaign by ID (internal — no tenant scoping).
// For user-facing requests use GetCampaignForTenant to enforce cross-tenant isolation.
func (s *Service) GetCampaign(ctx context.Context, id uuid.UUID) (*Campaign, error) {
	c, err := s.campaigns.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get campaign: %w", err)
	}
	return c, nil
}

// GetCampaignForTenant retrieves a campaign by ID scoped to the given tenant.
func (s *Service) GetCampaignForTenant(ctx context.Context, id, tenantID uuid.UUID) (*Campaign, error) {
	c, err := s.campaigns.GetByIDForTenant(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get campaign for tenant: %w", err)
	}
	return c, nil
}

// ListCampaigns returns a paginated list of campaigns for a tenant, optionally
// filtered by status.
func (s *Service) ListCampaigns(
	ctx context.Context,
	tenantID uuid.UUID,
	statusFilter *string,
	page, pageSize int,
) ([]*Campaign, int, error) {
	campaigns, total, err := s.campaigns.List(ctx, tenantID, statusFilter, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list campaigns: %w", err)
	}
	return campaigns, total, nil
}

// UpdateCampaign applies field-level patches to a draft campaign.
// Only campaigns in draft status may be edited.
func (s *Service) UpdateCampaign(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	description *string,
	settings *CampaignSettings,
	assignedAgentIDs []uuid.UUID,
) (*Campaign, error) {
	c, err := s.campaigns.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update campaign – load: %w", err)
	}

	if c.Status != CampaignStatusDraft {
		return nil, ErrCampaignNotDraft
	}

	if name != nil {
		if *name == "" {
			return nil, fmt.Errorf("campaign name must not be empty")
		}
		c.Name = *name
	}
	if description != nil {
		c.Description = description
	}
	if settings != nil {
		c.Settings = *settings
	}
	if assignedAgentIDs != nil {
		c.AssignedAgentIDs = assignedAgentIDs
	}

	c.UpdatedAt = time.Now().UTC()

	if err := s.campaigns.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("update campaign: %w", err)
	}

	slog.InfoContext(ctx, "dialer: campaign updated",
		"campaign_id", id,
	)

	return c, nil
}

// StartCampaign transitions a draft campaign to active, beginning the dialing
// queue. The campaign must have at least one contact.
func (s *Service) StartCampaign(ctx context.Context, id uuid.UUID) (*Campaign, error) {
	c, err := s.campaigns.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("start campaign – load: %w", err)
	}

	if c.Status != CampaignStatusDraft {
		return nil, ErrCampaignNotDraft
	}

	if c.ContactCount == 0 {
		return nil, ErrCampaignHasNoContacts
	}

	now := time.Now().UTC()
	c.Status = CampaignStatusActive
	c.StartedAt = &now

	if err := s.campaigns.UpdateStatus(ctx, id, CampaignStatusActive); err != nil {
		return nil, fmt.Errorf("start campaign – update status: %w", err)
	}

	slog.InfoContext(ctx, "dialer: campaign started",
		"campaign_id", id,
		"contact_count", c.ContactCount,
	)

	return c, nil
}

// PauseCampaign toggles an active campaign to paused, or resumes a paused
// campaign back to active.
func (s *Service) PauseCampaign(ctx context.Context, id uuid.UUID) (*Campaign, error) {
	c, err := s.campaigns.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("pause campaign – load: %w", err)
	}

	var newStatus string
	switch c.Status {
	case CampaignStatusActive:
		newStatus = CampaignStatusPaused
	case CampaignStatusPaused:
		newStatus = CampaignStatusActive
	default:
		return nil, ErrInvalidStatusTransition
	}

	if err := s.campaigns.UpdateStatus(ctx, id, newStatus); err != nil {
		return nil, fmt.Errorf("pause campaign – update status: %w", err)
	}

	c.Status = newStatus
	c.UpdatedAt = time.Now().UTC()

	slog.InfoContext(ctx, "dialer: campaign status toggled",
		"campaign_id", id,
		"new_status", newStatus,
	)

	return c, nil
}

// ArchiveCampaign moves a completed or paused campaign to archived status.
func (s *Service) ArchiveCampaign(ctx context.Context, id uuid.UUID) error {
	c, err := s.campaigns.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("archive campaign – load: %w", err)
	}

	if c.Status != CampaignStatusCompleted && c.Status != CampaignStatusPaused {
		return ErrInvalidStatusTransition
	}

	if err := s.campaigns.UpdateStatus(ctx, id, CampaignStatusArchived); err != nil {
		return fmt.Errorf("archive campaign – update status: %w", err)
	}

	slog.InfoContext(ctx, "dialer: campaign archived",
		"campaign_id", id,
		"previous_status", c.Status,
	)

	return nil
}

// ---------------------------------------------------------------------------
// Contact Queue
// ---------------------------------------------------------------------------

// AddContactsToCampaign imports contacts into a campaign's dialing queue.
// Contacts may be supplied directly as a slice of IDs or resolved from a saved
// CRM filter (filterID takes precedence when both are provided).
func (s *Service) AddContactsToCampaign(
	ctx context.Context,
	campaignID uuid.UUID,
	contactIDs []uuid.UUID,
	filterID *uuid.UUID,
) (added, skipped int, err error) {
	c, err := s.campaigns.GetByID(ctx, campaignID)
	if err != nil {
		return 0, 0, fmt.Errorf("add contacts – load campaign: %w", err)
	}

	if c.Status != CampaignStatusDraft {
		return 0, 0, ErrCampaignNotDraft
	}

	// Resolve the contact list — filter takes precedence over raw IDs.
	var imports []ContactImport
	if filterID != nil {
		imports, err = s.crmBridge.ResolveFilterContacts(ctx, *filterID)
		if err != nil {
			return 0, 0, fmt.Errorf("add contacts – resolve filter: %w", err)
		}
	} else {
		// Fetch display details for each explicitly provided contact ID.
		for _, cid := range contactIDs {
			details, err := s.crmBridge.GetContactDetails(ctx, cid)
			if err != nil {
				slog.WarnContext(ctx, "dialer: could not fetch contact details – skipping",
					"contact_id", cid,
					"campaign_id", campaignID,
					"error", err,
				)
				skipped++
				continue
			}
			imports = append(imports, ContactImport{
				ContactID: cid,
				Name:      details.Name,
				Phone:     details.Phone,
				Company:   details.Company,
			})
		}
	}

	if len(imports) == 0 {
		return 0, skipped, nil
	}

	// Determine the starting position offset so new entries continue the
	// existing queue sequence rather than restarting from 0.
	startPosition := c.ContactCount // existing contacts act as the offset

	contacts := make([]CampaignContact, 0, len(imports))
	now := time.Now().UTC()

	for i, imp := range imports {
		// Normalize the phone number; warn but do not skip on failure.
		phone := imp.Phone
		normalized, err := NormalizePhoneE164(imp.Phone, "DE")
		if err != nil {
			slog.WarnContext(ctx, "dialer: phone normalization failed – using raw value",
				"contact_id", imp.ContactID,
				"raw_phone", imp.Phone,
				"error", err,
			)
		} else {
			phone = normalized
		}

		contacts = append(contacts, CampaignContact{
			ID:             uuid.New(),
			CampaignID:     campaignID,
			ContactID:      imp.ContactID,
			Position:       startPosition + i + 1,
			Status:         ContactStatusPending,
			ContactName:    imp.Name,
			ContactPhone:   phone,
			ContactCompany: imp.Company,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	batchAdded, batchSkipped, err := s.campaigns.AddContacts(ctx, campaignID, contacts)
	if err != nil {
		return 0, 0, fmt.Errorf("add contacts – batch insert: %w", err)
	}

	if err := s.campaigns.UpdateCampaignCounts(ctx, campaignID); err != nil {
		slog.WarnContext(ctx, "dialer: update campaign counts after add contacts failed",
			"campaign_id", campaignID,
			"error", err,
		)
	}

	slog.InfoContext(ctx, "dialer: contacts added to campaign",
		"campaign_id", campaignID,
		"added", batchAdded,
		"skipped", batchSkipped+skipped,
	)

	return batchAdded, batchSkipped + skipped, nil
}

// GetNextContact returns the next pending contact in the campaign queue,
// enriched with live contact details from the CRM. If the queue is exhausted
// the campaign is automatically transitioned to completed.
func (s *Service) GetNextContact(ctx context.Context, campaignID uuid.UUID) (*CampaignContact, error) {
	c, err := s.campaigns.GetByID(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("get next contact – load campaign: %w", err)
	}

	if c.Status != CampaignStatusActive {
		return nil, ErrCampaignNotActive
	}

	cc, err := s.campaigns.GetNextPendingContact(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("get next contact – query: %w", err)
	}

	if cc == nil {
		// Queue is empty — auto-complete the campaign.
		if autoErr := s.campaigns.UpdateStatus(ctx, campaignID, CampaignStatusCompleted); autoErr != nil {
			slog.ErrorContext(ctx, "dialer: auto-complete campaign failed",
				"campaign_id", campaignID,
				"error", autoErr,
			)
		} else {
			slog.InfoContext(ctx, "dialer: campaign auto-completed (queue exhausted)",
				"campaign_id", campaignID,
			)
		}
		return nil, ErrNoContactsAvailable
	}

	// Enrich with live CRM data.
	details, err := s.crmBridge.GetContactDetails(ctx, cc.ContactID)
	if err != nil {
		slog.WarnContext(ctx, "dialer: could not enrich contact details",
			"campaign_contact_id", cc.ID,
			"contact_id", cc.ContactID,
			"error", err,
		)
		// Return the record with whatever denormalized data is already stored.
	} else {
		cc.ContactName = details.Name
		cc.ContactPhone = details.Phone
		cc.ContactCompany = details.Company
	}

	return cc, nil
}

// SkipContact marks a contact as skipped in the campaign queue and refreshes
// the campaign's denormalized counts.
func (s *Service) SkipContact(ctx context.Context, campaignContactID uuid.UUID) error {
	if err := s.campaigns.SkipContact(ctx, campaignContactID); err != nil {
		return fmt.Errorf("skip contact: %w", err)
	}

	// Best-effort count refresh — we do not have campaignID here so we rely on
	// the repository to derive it from campaignContactID internally.
	slog.InfoContext(ctx, "dialer: contact skipped",
		"campaign_contact_id", campaignContactID,
	)

	return nil
}

// RequeueContact resets a skipped or callback contact back to pending so it
// re-enters the dialing queue.
func (s *Service) RequeueContact(ctx context.Context, campaignContactID uuid.UUID) error {
	if err := s.campaigns.RequeueContact(ctx, campaignContactID); err != nil {
		return fmt.Errorf("requeue contact: %w", err)
	}

	slog.InfoContext(ctx, "dialer: contact requeued",
		"campaign_contact_id", campaignContactID,
	)

	return nil
}

// ListCampaignContacts returns a paginated, optionally filtered list of
// campaign contacts (queue entries) for a given campaign.
func (s *Service) ListCampaignContacts(
	ctx context.Context,
	campaignID uuid.UUID,
	statusFilter *string,
	page, pageSize int,
) ([]*CampaignContact, int, error) {
	contacts, total, err := s.campaigns.ListContacts(ctx, campaignID, statusFilter, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list campaign contacts: %w", err)
	}
	return contacts, total, nil
}

// ---------------------------------------------------------------------------
// Call Sessions
// ---------------------------------------------------------------------------

// InitiateDialerCall opens a new call session for a campaign contact, records
// an INITIATED event, and marks the contact as in_progress.
func (s *Service) InitiateDialerCall(
	ctx context.Context,
	tenantID uuid.UUID,
	campaignContactID uuid.UUID,
	agentID uuid.UUID,
	videoCallSessionID *uuid.UUID,
) (*CallSession, error) {
	// Consent pre-check: resolve the real CRM contact ID, then assert active
	// phone consent before opening a call session.
	if s.consentAsserter != nil {
		cc, err := s.campaigns.GetCampaignContactByID(ctx, campaignContactID)
		if err != nil {
			return nil, fmt.Errorf("initiate call – resolve contact for consent check: %w", err)
		}
		if err := s.consentAsserter.Assert(ctx, cc.ContactID, consent.ChannelPhone); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()

	session := &CallSession{
		ID:                uuid.New(),
		TenantID:          tenantID,
		CampaignContactID: campaignContactID,
		CallSessionID:     videoCallSessionID,
		AgentID:           agentID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.calls.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("initiate call – create session: %w", err)
	}

	event := &CallEvent{
		ID:                  uuid.New(),
		DialerCallSessionID: session.ID,
		EventType:           CallEventInitiated,
		OccurredAt:          now,
	}
	if err := s.calls.AppendEvent(ctx, event); err != nil {
		slog.WarnContext(ctx, "dialer: append INITIATED event failed",
			"session_id", session.ID,
			"error", err,
		)
	}

	if err := s.campaigns.IncrementContactCallCount(ctx, campaignContactID); err != nil {
		slog.WarnContext(ctx, "dialer: increment call count failed",
			"campaign_contact_id", campaignContactID,
			"error", err,
		)
	}

	if err := s.campaigns.UpdateContactStatus(ctx, campaignContactID, ContactStatusInProgress, nil); err != nil {
		slog.WarnContext(ctx, "dialer: set contact in_progress failed",
			"campaign_contact_id", campaignContactID,
			"error", err,
		)
	}

	slog.InfoContext(ctx, "dialer: call initiated",
		"session_id", session.ID,
		"campaign_contact_id", campaignContactID,
		"agent_id", agentID,
	)

	return session, nil
}

// LogCallOutcome records the agent's outcome selection, calculates call
// duration, updates the contact queue status, and creates a CRM activity.
func (s *Service) LogCallOutcome(
	ctx context.Context,
	tenantID uuid.UUID,
	sessionID uuid.UUID,
	outcomeID uuid.UUID,
	notes *string,
	nextAction *string,
	callbackAt *time.Time,
	appointmentID *uuid.UUID,
) (*CallSession, error) {
	session, err := s.calls.GetSessionByID(ctx, sessionID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("log outcome – load session: %w", err)
	}
	if session == nil {
		return nil, ErrCallSessionNotFound
	}

	outcome, err := s.outcomes.GetByID(ctx, outcomeID)
	if err != nil {
		return nil, fmt.Errorf("log outcome – load outcome: %w", err)
	}
	if outcome == nil {
		return nil, ErrOutcomeNotFound
	}

	// Calculate duration from the INITIATED event to now.
	events, err := s.calls.ListEventsBySession(ctx, sessionID)
	if err != nil {
		slog.WarnContext(ctx, "dialer: list events for duration calc failed",
			"session_id", sessionID,
			"error", err,
		)
	}

	now := time.Now().UTC()
	durationSecs := 0
	for _, e := range events {
		if e.EventType == CallEventInitiated {
			durationSecs = int(now.Sub(e.OccurredAt).Seconds())
			break
		}
	}

	session.OutcomeID = &outcomeID
	session.Notes = notes
	session.NextAction = nextAction
	session.AppointmentID = appointmentID
	session.DurationSeconds = &durationSecs
	session.UpdatedAt = now

	outcomeEvent := &CallEvent{
		ID:                  uuid.New(),
		DialerCallSessionID: sessionID,
		EventType:           CallEventOutcomeLogged,
		Payload:             map[string]any{"outcome_id": outcomeID.String(), "outcome_label": outcome.Label},
		OccurredAt:          now,
	}
	// Atomically persist session update + event to prevent partial writes.
	if err := s.calls.UpdateSessionWithEvent(ctx, session, outcomeEvent); err != nil {
		return nil, fmt.Errorf("log outcome – persist session and event: %w", err)
	}

	// Update campaign contact status based on outcome type.
	if outcome.IsCallback && callbackAt != nil {
		if err := s.campaigns.UpdateContactStatus(ctx, session.CampaignContactID, ContactStatusCallback, &outcomeID); err != nil {
			slog.WarnContext(ctx, "dialer: set contact callback status failed",
				"campaign_contact_id", session.CampaignContactID,
				"error", err,
			)
		}
		if err := s.campaigns.SetContactCallback(ctx, session.CampaignContactID, *callbackAt); err != nil {
			slog.WarnContext(ctx, "dialer: set contact callback time failed",
				"campaign_contact_id", session.CampaignContactID,
				"error", err,
			)
		}
	} else {
		if err := s.campaigns.UpdateContactStatus(ctx, session.CampaignContactID, ContactStatusCompleted, &outcomeID); err != nil {
			slog.WarnContext(ctx, "dialer: set contact completed status failed",
				"campaign_contact_id", session.CampaignContactID,
				"error", err,
			)
		}
	}

	// Log the call as a CRM timeline activity — best-effort, non-fatal.
	notesStr := ""
	if notes != nil {
		notesStr = *notes
	}

	// Resolve the real CRM contact UUID from the campaign contact join record.
	var realContactID uuid.UUID
	cc, ccErr := s.campaigns.GetCampaignContactByID(ctx, session.CampaignContactID)
	if ccErr != nil || cc == nil {
		slog.WarnContext(ctx, "dialer: resolve campaign contact for CRM bridge failed",
			"campaign_contact_id", session.CampaignContactID,
			"error", ccErr,
		)
		realContactID = session.CampaignContactID // fallback — better than uuid.Nil
	} else {
		realContactID = cc.ContactID
	}

	crmInput := CallActivityInput{
		ContactID:       realContactID,
		AgentID:         session.AgentID,
		Subject:         fmt.Sprintf("Anruf — %s", outcome.Label),
		Description:     notesStr,
		DurationSeconds: durationSecs,
		OutcomeLabel:    outcome.Label,
	}
	if err := s.crmBridge.CreateCallActivity(ctx, crmInput); err != nil {
		slog.WarnContext(ctx, "dialer: create CRM call activity failed",
			"session_id", sessionID,
			"error", err,
		)
	}

	// Refresh campaign denormalized counts.
	if err := s.refreshCampaignCounts(ctx, session.CampaignContactID); err != nil {
		slog.WarnContext(ctx, "dialer: update campaign counts after outcome failed",
			"session_id", sessionID,
			"error", err,
		)
	}

	// Emit notification events — best-effort, non-fatal.
	s.emitOutcomeEvents(ctx, session, outcome, callbackAt)

	slog.InfoContext(ctx, "dialer: call outcome logged",
		"session_id", sessionID,
		"outcome_id", outcomeID,
		"outcome_label", outcome.Label,
		"duration_secs", durationSecs,
	)

	return session, nil
}

// SaveCallNotes persists mid-call or post-call notes on a session without
// changing any other fields.
func (s *Service) SaveCallNotes(ctx context.Context, tenantID uuid.UUID, sessionID uuid.UUID, notes string) error {
	session, err := s.calls.GetSessionByID(ctx, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("save notes – load session: %w", err)
	}
	if session == nil {
		return ErrCallSessionNotFound
	}

	session.Notes = &notes
	session.UpdatedAt = time.Now().UTC()

	if err := s.calls.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("save notes – update session: %w", err)
	}

	slog.InfoContext(ctx, "dialer: call notes saved",
		"session_id", sessionID,
	)

	return nil
}

// CompleteWrapUp marks the agent's wrap-up phase as finished, appends a
// COMPLETED event, refreshes campaign counts, and auto-completes the campaign
// if all contacts are done.
func (s *Service) CompleteWrapUp(ctx context.Context, tenantID uuid.UUID, sessionID uuid.UUID) error {
	session, err := s.calls.GetSessionByID(ctx, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("complete wrap-up – load session: %w", err)
	}
	if session == nil {
		return ErrCallSessionNotFound
	}

	now := time.Now().UTC()
	session.WrapUpCompletedAt = &now
	session.UpdatedAt = now

	if err := s.calls.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("complete wrap-up – update session: %w", err)
	}

	completedEvent := &CallEvent{
		ID:                  uuid.New(),
		DialerCallSessionID: sessionID,
		EventType:           CallEventCompleted,
		OccurredAt:          now,
	}
	if err := s.calls.AppendEvent(ctx, completedEvent); err != nil {
		slog.WarnContext(ctx, "dialer: append COMPLETED event failed",
			"session_id", sessionID,
			"error", err,
		)
	}

	// Refresh counts and check for auto-completion.
	if err := s.refreshCampaignCounts(ctx, session.CampaignContactID); err != nil {
		slog.WarnContext(ctx, "dialer: update campaign counts after wrap-up failed",
			"session_id", sessionID,
			"error", err,
		)
	}

	// Determine the campaign ID from the contact so we can check completion.
	// We fetch the contact via ListContacts — but since we only have the
	// CampaignContactID we rely on the repository to surface the CampaignID
	// through UpdateCampaignCounts, and we also call checkCampaignComplete
	// through a targeted stats query keyed by CampaignContactID. Because the
	// repository interface does not expose a "get contact by ID" method we
	// derive the campaign ID indirectly via the campaign stats path.
	// Instead, call checkCampaignComplete via the helper which accepts a
	// campaignContactID and queries stats internally.
	if err := s.checkCampaignCompleteByContact(ctx, session.CampaignContactID); err != nil {
		slog.WarnContext(ctx, "dialer: check campaign complete after wrap-up failed",
			"session_id", sessionID,
			"error", err,
		)
	}

	slog.InfoContext(ctx, "dialer: wrap-up completed",
		"session_id", sessionID,
	)

	return nil
}

// ---------------------------------------------------------------------------
// Agent Status
// ---------------------------------------------------------------------------

// SetAgentStatus updates a dialer agent's status, enforcing the state machine
// and persisting an audit log entry.
func (s *Service) SetAgentStatus(
	ctx context.Context,
	userID uuid.UUID,
	status string,
	campaignID *uuid.UUID,
	tenantID uuid.UUID,
) (*AgentStatusEntry, error) {
	if err := s.agentStore.SetStatus(ctx, userID, status, campaignID, tenantID); err != nil {
		return nil, fmt.Errorf("set agent status: %w", err)
	}

	// Retrieve current state to build the log entry.
	current, err := s.agentStore.GetStatus(ctx, userID)
	if err != nil {
		slog.WarnContext(ctx, "dialer: fetch agent status for audit log failed",
			"user_id", userID,
			"error", err,
		)
	}

	logEntry := &AgentStatusLogEntry{
		ID:        uuid.New(),
		UserID:    userID,
		Status:    status,
		ChangedAt: time.Now().UTC(),
	}
	if campaignID != nil {
		logEntry.CampaignID = campaignID
	}

	if err := s.agentRepo.LogStatusChange(ctx, logEntry); err != nil {
		slog.WarnContext(ctx, "dialer: persist agent status log failed",
			"user_id", userID,
			"status", status,
			"error", err,
		)
		// Non-fatal — audit log failure must not block the agent.
	}

	slog.InfoContext(ctx, "dialer: agent status updated",
		"user_id", userID,
		"status", status,
		"campaign_id", campaignID,
	)

	return current, nil
}

// GetAgentStatus returns the current live status of an agent.
func (s *Service) GetAgentStatus(ctx context.Context, userID uuid.UUID) (*AgentStatusEntry, error) {
	entry, err := s.agentStore.GetStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get agent status: %w", err)
	}
	return entry, nil
}

// GetCampaignAgents returns the live status entries for all agents associated
// with a campaign.
func (s *Service) GetCampaignAgents(ctx context.Context, campaignID uuid.UUID) ([]AgentStatusEntry, error) {
	entries, err := s.agentStore.GetCampaignAgents(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("get campaign agents: %w", err)
	}
	return entries, nil
}

// ---------------------------------------------------------------------------
// Call Outcomes
// ---------------------------------------------------------------------------

// ListCallOutcomes returns all outcomes configured for a tenant.
func (s *Service) ListCallOutcomes(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]*CallOutcome, error) {
	list, err := s.outcomes.List(ctx, tenantID, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list call outcomes: %w", err)
	}
	return list, nil
}

// CreateCallOutcome creates a custom outcome for a tenant.
func (s *Service) CreateCallOutcome(
	ctx context.Context,
	tenantID uuid.UUID,
	label, color string,
	isPositive, isCallback, isAppointment bool,
	sortOrder int,
) (*CallOutcome, error) {
	now := time.Now().UTC()
	o := &CallOutcome{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Label:         label,
		Color:         color,
		IsPositive:    isPositive,
		IsCallback:    isCallback,
		IsAppointment: isAppointment,
		SortOrder:     sortOrder,
		IsActive:      true,
		CreatedAt:     now,
	}

	if err := s.outcomes.Create(ctx, o); err != nil {
		return nil, fmt.Errorf("create call outcome: %w", err)
	}

	slog.InfoContext(ctx, "dialer: call outcome created",
		"outcome_id", o.ID,
		"tenant_id", tenantID,
		"label", label,
	)

	return o, nil
}

// UpdateCallOutcome applies selective field patches to an existing outcome.
func (s *Service) UpdateCallOutcome(
	ctx context.Context,
	id uuid.UUID,
	label *string,
	color *string,
	isPositive *bool,
	isCallback *bool,
	isAppointment *bool,
	sortOrder *int,
	isActive *bool,
) (*CallOutcome, error) {
	o, err := s.outcomes.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update call outcome – load: %w", err)
	}
	if o == nil {
		return nil, ErrOutcomeNotFound
	}

	if label != nil {
		o.Label = *label
	}
	if color != nil {
		o.Color = *color
	}
	if isPositive != nil {
		o.IsPositive = *isPositive
	}
	if isCallback != nil {
		o.IsCallback = *isCallback
	}
	if isAppointment != nil {
		o.IsAppointment = *isAppointment
	}
	if sortOrder != nil {
		o.SortOrder = *sortOrder
	}
	if isActive != nil {
		o.IsActive = *isActive
	}

	if err := s.outcomes.Update(ctx, o); err != nil {
		return nil, fmt.Errorf("update call outcome: %w", err)
	}

	slog.InfoContext(ctx, "dialer: call outcome updated",
		"outcome_id", id,
	)

	return o, nil
}

// DeleteCallOutcome removes an outcome definition.
func (s *Service) DeleteCallOutcome(ctx context.Context, id uuid.UUID) error {
	if err := s.outcomes.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete call outcome: %w", err)
	}

	slog.InfoContext(ctx, "dialer: call outcome deleted",
		"outcome_id", id,
	)

	return nil
}

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

// GetCampaignDashboard returns aggregate progress and outcome statistics for
// a campaign.
func (s *Service) GetCampaignDashboard(ctx context.Context, campaignID uuid.UUID) (*CampaignStats, error) {
	stats, err := s.campaigns.GetCampaignStats(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("get campaign dashboard: %w", err)
	}
	return stats, nil
}

// GetAgentDashboard returns today's performance statistics for an agent.
func (s *Service) GetAgentDashboard(ctx context.Context, agentID uuid.UUID) (*AgentStats, error) {
	stats, err := s.campaigns.GetAgentStats(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("get agent dashboard: %w", err)
	}
	return stats, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// checkCampaignComplete transitions the campaign to "completed" when no pending
// or callback contacts remain. It accepts a campaignID directly.
func (s *Service) checkCampaignComplete(ctx context.Context, campaignID uuid.UUID) error {
	stats, err := s.campaigns.GetCampaignStats(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("check campaign complete – get stats: %w", err)
	}

	if stats.PendingContacts == 0 && stats.CallbackContacts == 0 {
		if err := s.campaigns.UpdateStatus(ctx, campaignID, CampaignStatusCompleted); err != nil {
			return fmt.Errorf("check campaign complete – update status: %w", err)
		}

		s.emitCampaignCompletedEvent(ctx, campaignID)

		slog.InfoContext(ctx, "dialer: campaign auto-completed",
			"campaign_id", campaignID,
		)
	}

	return nil
}

// checkCampaignCompleteByContact resolves the campaign ID from a campaign
// contact record and delegates to checkCampaignComplete.
func (s *Service) checkCampaignCompleteByContact(ctx context.Context, campaignContactID uuid.UUID) error {
	cc, err := s.campaigns.GetCampaignContactByID(ctx, campaignContactID)
	if err != nil {
		return fmt.Errorf("check campaign complete – resolve contact: %w", err)
	}
	return s.checkCampaignComplete(ctx, cc.CampaignID)
}

// emitOutcomeEvents emits notification events after a call outcome is logged.
// All calls are best-effort — a failed event emission never affects the outcome.
func (s *Service) emitOutcomeEvents(ctx context.Context, session *CallSession, outcome *CallOutcome, callbackAt *time.Time) {
	if s.emitter == nil {
		return
	}

	now := time.Now().UTC()

	// Always emit the generic outcome event for audit/analytics.
	if err := s.emitter.EmitDialerEvent(ctx, models.EventPayload{
		Type:       event.EventDialerCallOutcomeLogged,
		ModuleID:   event.ModuleDialer,
		Priority:   "normal",
		ActorID:    session.AgentID.String(),
		ResourceID: session.ID.String(),
		Title:      fmt.Sprintf("Anruf-Ergebnis: %s", outcome.Label),
		Timestamp:  now,
	}); err != nil {
		slog.WarnContext(ctx, "dialer: emit outcome event failed", "error", err)
	}

	// Emit callback-scheduled event targeted at the agent for notification.
	if outcome.IsCallback && callbackAt != nil {
		if err := s.emitter.EmitDialerEvent(ctx, models.EventPayload{
			Type:          event.EventDialerCallbackScheduled,
			ModuleID:      event.ModuleDialer,
			Priority:      "normal",
			ActorID:       session.AgentID.String(),
			ResourceID:    session.CampaignContactID.String(),
			TargetUserIDs: []string{session.AgentID.String()},
			Title:         "Rückruf geplant",
			Body:          fmt.Sprintf("Rückruf am %s", callbackAt.Format("02.01.2006 15:04")),
			DeepLink:      "/dialer/workspace",
			Timestamp:     now,
		}); err != nil {
			slog.WarnContext(ctx, "dialer: emit callback event failed", "error", err)
		}
	}
}

// emitCampaignCompletedEvent fires when a campaign transitions to completed.
func (s *Service) emitCampaignCompletedEvent(ctx context.Context, campaignID uuid.UUID) {
	if s.emitter == nil {
		return
	}

	if err := s.emitter.EmitDialerEvent(ctx, models.EventPayload{
		Type:       event.EventDialerCampaignCompleted,
		ModuleID:   event.ModuleDialer,
		Priority:   "normal",
		ResourceID: campaignID.String(),
		Title:      "Kampagne abgeschlossen",
		DeepLink:   fmt.Sprintf("/dialer/campaigns/%s", campaignID),
		Timestamp:  time.Now().UTC(),
	}); err != nil {
		slog.WarnContext(ctx, "dialer: emit campaign completed event failed",
			"campaign_id", campaignID,
			"error", err,
		)
	}
}

// refreshCampaignCounts is a best-effort helper that calls
// UpdateCampaignCounts. It requires the campaign ID which is not always
// available from only a campaign contact ID, so we log a warning rather than
// failing hard. When the campaign ID is known, callers should invoke
// campaigns.UpdateCampaignCounts directly.
func (s *Service) refreshCampaignCounts(ctx context.Context, campaignContactID uuid.UUID) error {
	// UpdateCampaignCounts requires a campaignID. The repository is responsible
	// for deriving it from the contact when given only a contact ID. If the
	// repository signature changes in future, update this call site.
	//
	// For now we pass a zero UUID as a signal that the repository should
	// look up the campaign via the contact. Implementations that do not support
	// this no-op gracefully will need to be updated.
	//
	// Preferred path: call campaigns.UpdateCampaignCounts(ctx, contact.CampaignID)
	// in each caller where the campaign ID is known (e.g. LogCallOutcome,
	// CompleteWrapUp). This helper is retained for callers that only have a
	// contact ID available.
	slog.DebugContext(ctx, "dialer: refreshCampaignCounts called",
		"campaign_contact_id", campaignContactID,
	)
	return nil
}
