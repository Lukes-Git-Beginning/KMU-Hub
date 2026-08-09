package contact

// Leads live on the contacts table as a lifecycle stage (migration 000259),
// never as a table of their own: the lead and the customer they become are the
// same person, and a second row would give duplicate detection two truths to
// reconcile. Everything here therefore reads and writes `contacts`.

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Lifecycle stages. Contacts that never came through the lead inbox are
// "customer" -- that is what the column defaults to for every existing row.
const (
	LifecycleLead      = "lead"
	LifecycleQualified = "qualified"
	LifecycleCustomer  = "customer"
)

// Lead statuses within the inbox.
const (
	LeadStatusNew          = "new"
	LeadStatusContacted    = "contacted"
	LeadStatusQualified    = "qualified"
	LeadStatusDisqualified = "disqualified"
)

// Lead intake sources.
const (
	LeadSourceManual = "manual"
	LeadSourceCSV    = "csv"
	LeadSourceDialer = "dialer"
)

// Lead temperatures.
const (
	LeadTemperatureHot  = "hot"
	LeadTemperatureWarm = "warm"
	LeadTemperatureCold = "cold"
)

// leadScoreSourceBase / leadScoreFieldPoints / lead temperature thresholds
// mirror DEFAULT_LEAD_SCORING in desktop/src/renderer/src/stores/leadScoring.ts
// so a score computed here matches what the inbox has been showing.
//
// lean: weights are constants; move them to tenant_settings
// (module_id='crm', key='leadScoring.*') once a tenant asks to tune them --
// the frontend store already anticipates that key.
var (
	leadScoreSourceBase = map[string]int16{
		LeadSourceDialer: 35,
		LeadSourceManual: 25,
		LeadSourceCSV:    10,
	}
	leadScoreFieldPoints = struct{ Email, Phone, Company, Notes int16 }{
		Email: 20, Phone: 15, Company: 20, Notes: 10,
	}
)

const (
	leadScoreUnknownSource = 15
	leadThresholdHot       = 66
	leadThresholdWarm      = 33
)

// LeadScoreInput carries the fields that feed the score.
type LeadScoreInput struct {
	Source  string
	Email   string
	Phone   string
	Company string
	Notes   string
}

// ComputeLeadScore derives a 0-100 score from intake source and data
// completeness. Rule-based on purpose: a lead score a salesperson cannot
// explain is a lead score they will not trust.
func ComputeLeadScore(in LeadScoreInput) int16 {
	score, ok := leadScoreSourceBase[in.Source]
	if !ok {
		score = leadScoreUnknownSource
	}
	if strings.TrimSpace(in.Email) != "" {
		score += leadScoreFieldPoints.Email
	}
	if strings.TrimSpace(in.Phone) != "" {
		score += leadScoreFieldPoints.Phone
	}
	if strings.TrimSpace(in.Company) != "" {
		score += leadScoreFieldPoints.Company
	}
	if strings.TrimSpace(in.Notes) != "" {
		score += leadScoreFieldPoints.Notes
	}
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

// ScoreToTemperature maps a score onto the three inbox buckets.
func ScoreToTemperature(score int16) string {
	switch {
	case score >= leadThresholdHot:
		return LeadTemperatureHot
	case score >= leadThresholdWarm:
		return LeadTemperatureWarm
	default:
		return LeadTemperatureCold
	}
}

// EffectiveTemperature resolves the temperature a client should display: a
// manual override wins over the score-derived one.
func EffectiveTemperature(score *int16, override *string) string {
	if override != nil && *override != "" {
		return *override
	}
	if score == nil {
		return LeadTemperatureCold
	}
	return ScoreToTemperature(*score)
}

func validLeadSource(s string) bool {
	_, ok := leadScoreSourceBase[s]
	return ok
}

func validLeadStatus(s string) bool {
	switch s {
	case LeadStatusNew, LeadStatusContacted, LeadStatusQualified, LeadStatusDisqualified:
		return true
	}
	return false
}

func validLeadTemperature(s string) bool {
	switch s {
	case LeadTemperatureHot, LeadTemperatureWarm, LeadTemperatureCold:
		return true
	}
	return false
}

// CreateLeadInput is the intake form: a raw, unqualified prospect.
type CreateLeadInput struct {
	FirstName string
	LastName  string
	Company   string
	Email     string
	Phone     string
	Notes     string
	Source    string
	CreatedBy uuid.UUID
	TenantID  uuid.UUID
}

// CreateLead records a lead as a contact at the "lead" lifecycle stage. The
// score is computed here rather than trusted from the client -- a client-sent
// score is a client-chosen priority.
func (s *Service) CreateLead(ctx context.Context, input CreateLeadInput) (*models.ContactWithRelations, error) {
	if input.TenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	source := strings.TrimSpace(input.Source)
	if !validLeadSource(source) {
		return nil, ErrInvalidLeadSource
	}

	score := ComputeLeadScore(LeadScoreInput{
		Source:  source,
		Email:   input.Email,
		Phone:   input.Phone,
		Company: input.Company,
		Notes:   input.Notes,
	})
	status := LeadStatusNew

	create := CreateInput{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     optionalString(input.Email),
		Phone:     optionalString(input.Phone),
		Notes:     optionalString(input.Notes),
		CreatedBy: input.CreatedBy,
		TenantID:  input.TenantID,
		Lead: &LeadIntake{
			Stage:   LifecycleLead,
			Source:  &source,
			Score:   &score,
			Status:  &status,
			Company: optionalString(input.Company),
		},
	}

	created, err := s.Create(ctx, create)
	if err != nil {
		return nil, err
	}

	slog.Info("lead created",
		"contact_id", created.ID,
		"source", source,
		"score", score,
	)
	return created, nil
}

// LeadListInput filters the lead inbox.
type LeadListInput struct {
	TenantID uuid.UUID
	// Stage defaults to the open inbox (lead + qualified); pass a single stage
	// to narrow it. "customer" is rejected -- that is the contact list.
	Stage    string
	Status   string
	Search   string
	Page     int
	PageSize int
}

// ListLeads returns the tenant's leads, newest first.
func (s *Service) ListLeads(ctx context.Context, input LeadListInput) ([]*models.ContactWithRelations, int, error) {
	if input.TenantID == uuid.Nil {
		return nil, 0, ErrInvalidTenant
	}
	if input.Stage != "" && input.Stage != LifecycleLead && input.Stage != LifecycleQualified {
		return nil, 0, ErrInvalidLifecycleStage
	}
	if input.Status != "" && !validLeadStatus(input.Status) {
		return nil, 0, ErrInvalidLeadStatus
	}

	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}
	offset := (input.Page - 1) * input.PageSize

	return s.repo.ListLeads(ctx, LeadFilter{
		TenantID: input.TenantID,
		Stage:    input.Stage,
		Status:   input.Status,
		Search:   strings.TrimSpace(input.Search),
	}, offset, input.PageSize)
}

// UpdateLeadInput patches the two fields the inbox lets a user change by hand.
type UpdateLeadInput struct {
	Status *string
	// Temperature is the manual override. An empty string clears it and hands
	// the display back to the computed score.
	Temperature *string
}

// UpdateLead moves a lead through the inbox or pins its temperature.
func (s *Service) UpdateLead(ctx context.Context, id uuid.UUID, input UpdateLeadInput, tenantID uuid.UUID) (*models.ContactWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	if input.Status == nil && input.Temperature == nil {
		return nil, ErrNoLeadChanges
	}

	patch := LeadPatch{}
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if !validLeadStatus(status) {
			return nil, ErrInvalidLeadStatus
		}
		patch.Status = &status
		// Qualifying from the inbox is the same act as converting, so it moves
		// the lifecycle stage too. Disqualified stays at stage "lead" -- the
		// person is still on file, just not being worked.
		if status == LeadStatusQualified {
			stage := LifecycleQualified
			patch.Stage = &stage
		}
	}
	if input.Temperature != nil {
		temperature := strings.TrimSpace(*input.Temperature)
		if temperature != "" && !validLeadTemperature(temperature) {
			return nil, ErrInvalidLeadTemperature
		}
		patch.ClearTemperature = temperature == ""
		if temperature != "" {
			patch.Temperature = &temperature
		}
	}

	return s.repo.UpdateLead(ctx, id, tenantID, patch)
}

// ConvertLead qualifies a lead. The contact row is already the contact -- so
// conversion is a stage change, not a copy into a second table.
func (s *Service) ConvertLead(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.ContactWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}

	stage := LifecycleQualified
	status := LeadStatusQualified
	converted, err := s.repo.UpdateLead(ctx, id, tenantID, LeadPatch{
		Stage:  &stage,
		Status: &status,
	})
	if err != nil {
		return nil, err
	}

	slog.Info("lead converted", "contact_id", id)
	return converted, nil
}

// PromoteFromDialerCallback lifts a contact into the lead funnel when a
// dialer call outcome carries a callback request. It never moves a contact
// backwards: one already at "qualified" or "customer" is returned unchanged
// -- a callback request is a data point, not grounds to undo a stage the
// contact already earned (existing contacts default to "customer" at
// creation, see models.Contact; this guard also protects those). The score
// recompute is an absolute SET rather than an increment, so replaying the
// same outcome twice leaves lifecycle_stage/lead_source/lead_score
// unchanged the second time.
func (s *Service) PromoteFromDialerCallback(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.ContactWithRelations, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}

	current, err := s.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if current.LifecycleStage == LifecycleQualified || current.LifecycleStage == LifecycleCustomer {
		return current, nil
	}

	company := ""
	if current.LeadCompany != nil {
		company = *current.LeadCompany
	} else if current.CompanyName != nil {
		company = *current.CompanyName
	}
	email := ""
	if current.Email != nil {
		email = *current.Email
	}
	phone := ""
	if current.Phone != nil {
		phone = *current.Phone
	}
	notes := ""
	if current.Notes != nil {
		notes = *current.Notes
	}

	score := ComputeLeadScore(LeadScoreInput{
		Source:  LeadSourceDialer,
		Email:   email,
		Phone:   phone,
		Company: company,
		Notes:   notes,
	})
	source := LeadSourceDialer
	stage := LifecycleLead

	updated, err := s.repo.UpdateLead(ctx, id, tenantID, LeadPatch{
		Stage:  &stage,
		Source: &source,
		Score:  &score,
	})
	if err != nil {
		return nil, err
	}

	slog.Info("contact promoted to lead from dialer callback",
		"contact_id", id,
		"score", score,
	)
	return updated, nil
}

func optionalString(s string) *string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
