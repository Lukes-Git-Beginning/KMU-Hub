package deal

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Valid ISO 4217 currency codes (subset for common currencies)
var validCurrencies = map[string]bool{
	"EUR": true, "USD": true, "GBP": true, "CHF": true,
	"CAD": true, "AUD": true, "JPY": true, "CNY": true,
	"SEK": true, "NOK": true, "DKK": true, "PLN": true,
	"CZK": true, "HUF": true, "RON": true, "BGN": true,
}

// EventEmitter is an optional interface for emitting notification events.
// If set, the deal service will emit events on stage changes and owner assignments.
type EventEmitter interface {
	EmitDealEvent(ctx context.Context, payload models.EventPayload) error
}

// Service handles deal business logic
type Service struct {
	repo         Repository
	eventEmitter EventEmitter
}

// NewService creates a new deal service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetEventEmitter sets the optional event emitter for notification events.
func (s *Service) SetEventEmitter(emitter EventEmitter) {
	s.eventEmitter = emitter
}

// CreateInput contains the data needed to create a deal
type CreateInput struct {
	Name              string
	Value             float64
	Currency          string
	StageID           uuid.UUID
	ContactID         *uuid.UUID
	CompanyID         *uuid.UUID
	OwnerID           *uuid.UUID
	ExpectedCloseDate *time.Time
	Notes             *string
	TagIDs            []uuid.UUID
	CustomFields      map[uuid.UUID]any // field_id -> value
	CreatedBy         uuid.UUID
}

// Create creates a new deal
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.DealWithRelations, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrNameRequired
	}

	// Validate currency
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = "EUR" // Default
	}
	if !validCurrencies[currency] {
		return nil, ErrInvalidCurrency
	}

	// Validate stage exists
	stage, err := s.repo.GetStage(ctx, input.StageID)
	if err != nil {
		return nil, ErrStageNotFound
	}

	// Validate contact exists if provided
	if input.ContactID != nil {
		exists, checkErr := s.repo.ContactExists(ctx, *input.ContactID)
		if checkErr != nil {
			return nil, checkErr
		}
		if !exists {
			return nil, ErrContactNotFound
		}
	}

	// Validate company exists if provided
	if input.CompanyID != nil {
		exists, checkErr := s.repo.CompanyExists(ctx, *input.CompanyID)
		if checkErr != nil {
			return nil, checkErr
		}
		if !exists {
			return nil, ErrCompanyNotFound
		}
	}

	// Validate owner exists if provided
	if input.OwnerID != nil {
		exists, checkErr := s.repo.OwnerExists(ctx, *input.OwnerID)
		if checkErr != nil {
			return nil, checkErr
		}
		if !exists {
			return nil, ErrOwnerNotFound
		}
	}

	// Validate tags are for deals
	for _, tagID := range input.TagIDs {
		exists, tagErr := s.repo.TagExists(ctx, tagID, models.EntityTypeDeal)
		if tagErr != nil {
			return nil, tagErr
		}
		if !exists {
			return nil, ErrTagNotFound
		}
	}

	now := time.Now()
	var closedAt *time.Time
	if stage.IsWon || stage.IsLost {
		closedAt = &now
	}

	deal := &models.Deal{
		ID:                uuid.New(),
		Name:              name,
		Value:             decimal.NewFromFloat(input.Value),
		Currency:          currency,
		StageID:           input.StageID,
		ContactID:         input.ContactID,
		CompanyID:         input.CompanyID,
		OwnerID:           input.OwnerID,
		ExpectedCloseDate: input.ExpectedCloseDate,
		Notes:             input.Notes,
		CreatedBy:         input.CreatedBy,
		CreatedAt:         now,
		UpdatedAt:         now,
		ClosedAt:          closedAt,
	}

	if createErr := s.repo.Create(ctx, deal); createErr != nil {
		return nil, createErr
	}

	// Add tags
	if len(input.TagIDs) > 0 {
		if tagErr := s.repo.AddTags(ctx, deal.ID, input.TagIDs); tagErr != nil {
			return nil, tagErr
		}
	}

	// Set custom field values
	if len(input.CustomFields) > 0 {
		if cfErr := s.repo.SetCustomFieldValues(ctx, deal.ID, input.CustomFields); cfErr != nil {
			return nil, cfErr
		}
	}

	slog.Info("deal created",
		"deal_id", deal.ID,
		"name", deal.Name,
		"value", deal.Value,
	)

	return s.getWithRelations(ctx, deal)
}

// GetByID retrieves a deal by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.DealWithRelations, error) {
	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.getWithRelations(ctx, deal)
}

// ListInput contains filtering options for listing deals
type ListInput struct {
	StageID   *uuid.UUID
	ContactID *uuid.UUID
	CompanyID *uuid.UUID
	OwnerID   *uuid.UUID
	TagIDs    []uuid.UUID
	Search    string
	Page      int
	PageSize  int
	SortBy    string
	SortDesc  bool
}

// List retrieves deals with optional filtering
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.DealWithRelations, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := (input.Page - 1) * input.PageSize
	filter := ListFilter{
		StageID:   input.StageID,
		ContactID: input.ContactID,
		CompanyID: input.CompanyID,
		OwnerID:   input.OwnerID,
		TagIDs:    input.TagIDs,
		Search:    input.Search,
		SortBy:    input.SortBy,
		SortDesc:  input.SortDesc,
	}

	deals, total, err := s.repo.List(ctx, filter, offset, input.PageSize)
	if err != nil {
		return nil, 0, err
	}

	var results []*models.DealWithRelations
	for _, d := range deals {
		withRel, relErr := s.getWithRelations(ctx, d)
		if relErr != nil {
			return nil, 0, relErr
		}
		results = append(results, withRel)
	}

	return results, total, nil
}

// UpdateInput contains the data that can be updated on a deal
type UpdateInput struct {
	Name              *string
	Value             *float64
	Currency          *string
	ContactID         *uuid.UUID // use uuid.Nil to clear
	CompanyID         *uuid.UUID // use uuid.Nil to clear
	OwnerID           *uuid.UUID // use uuid.Nil to clear
	ExpectedCloseDate *time.Time
	Notes             *string
	CustomFields      map[uuid.UUID]any
}

// Update updates an existing deal
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*models.DealWithRelations, error) {
	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrNameRequired
		}
		deal.Name = name
	}

	if input.Value != nil {
		deal.Value = decimal.NewFromFloat(*input.Value)
	}

	if input.Currency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*input.Currency))
		if !validCurrencies[currency] {
			return nil, ErrInvalidCurrency
		}
		deal.Currency = currency
	}

	if input.ContactID != nil {
		if *input.ContactID == uuid.Nil {
			deal.ContactID = nil
		} else {
			exists, contactErr := s.repo.ContactExists(ctx, *input.ContactID)
			if contactErr != nil {
				return nil, contactErr
			}
			if !exists {
				return nil, ErrContactNotFound
			}
			deal.ContactID = input.ContactID
		}
	}

	if input.CompanyID != nil {
		if *input.CompanyID == uuid.Nil {
			deal.CompanyID = nil
		} else {
			exists, companyErr := s.repo.CompanyExists(ctx, *input.CompanyID)
			if companyErr != nil {
				return nil, companyErr
			}
			if !exists {
				return nil, ErrCompanyNotFound
			}
			deal.CompanyID = input.CompanyID
		}
	}

	var ownerChanged bool
	var newOwnerID uuid.UUID
	if input.OwnerID != nil {
		if *input.OwnerID == uuid.Nil {
			deal.OwnerID = nil
		} else {
			// Track owner change for event emission
			if deal.OwnerID == nil || *deal.OwnerID != *input.OwnerID {
				ownerChanged = true
				newOwnerID = *input.OwnerID
			}
			exists, ownerErr := s.repo.OwnerExists(ctx, *input.OwnerID)
			if ownerErr != nil {
				return nil, ownerErr
			}
			if !exists {
				return nil, ErrOwnerNotFound
			}
			deal.OwnerID = input.OwnerID
		}
	}

	if input.ExpectedCloseDate != nil {
		deal.ExpectedCloseDate = input.ExpectedCloseDate
	}

	if input.Notes != nil {
		deal.Notes = input.Notes
	}

	deal.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, deal); updateErr != nil {
		return nil, updateErr
	}

	// Update custom fields if provided
	if len(input.CustomFields) > 0 {
		if cfErr := s.repo.SetCustomFieldValues(ctx, deal.ID, input.CustomFields); cfErr != nil {
			return nil, cfErr
		}
	}

	slog.Info("deal updated", "deal_id", deal.ID)

	// Emit notification event for deal assignment change
	if s.eventEmitter != nil && ownerChanged {
		_ = s.eventEmitter.EmitDealEvent(ctx, models.EventPayload{
			Type:          "crm.deal.assigned",
			Priority:      models.PriorityNormal,
			ResourceID:    deal.ID.String(),
			ModuleID:      "crm",
			Title:         "You were assigned to deal: " + deal.Name,
			Body:          "Deal assignment changed",
			DeepLink:      "/crm/deals/" + deal.ID.String(),
			TargetUserIDs: []string{newOwnerID.String()},
			Timestamp:     time.Now(),
		})
	}

	return s.getWithRelations(ctx, deal)
}

// Delete removes a deal
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if deleteErr := s.repo.Delete(ctx, id); deleteErr != nil {
		return deleteErr
	}

	slog.Info("deal deleted",
		"deal_id", deal.ID,
		"name", deal.Name,
	)

	return nil
}

// MoveToStage moves a deal to a different pipeline stage
func (s *Service) MoveToStage(ctx context.Context, dealID, stageID uuid.UUID) (*models.DealWithRelations, error) {
	deal, err := s.repo.GetByID(ctx, dealID)
	if err != nil {
		return nil, err
	}

	stage, err := s.repo.GetStage(ctx, stageID)
	if err != nil {
		return nil, ErrStageNotFound
	}

	oldStageID := deal.StageID
	deal.StageID = stageID
	deal.UpdatedAt = time.Now()

	// Set or clear closed_at based on stage type
	if stage.IsWon || stage.IsLost {
		if deal.ClosedAt == nil {
			now := time.Now()
			deal.ClosedAt = &now
		}
	} else {
		deal.ClosedAt = nil
	}

	if updateErr := s.repo.Update(ctx, deal); updateErr != nil {
		return nil, updateErr
	}

	// Update closed_at separately to handle the NULL case
	if setErr := s.repo.SetClosedAt(ctx, dealID, deal.ClosedAt); setErr != nil {
		return nil, setErr
	}

	slog.Info("deal moved to stage",
		"deal_id", deal.ID,
		"old_stage_id", oldStageID,
		"new_stage_id", stageID,
		"is_won", stage.IsWon,
		"is_lost", stage.IsLost,
	)

	// Emit notification event for deal stage change
	if s.eventEmitter != nil && deal.OwnerID != nil {
		// Find who triggered this operation (not available in current context,
		// so we use empty actor_id which means the notification service will
		// deliver to the owner unconditionally). In a proper implementation,
		// the actorID would be passed through the context or as a parameter.
		ownerIDStr := deal.OwnerID.String()
		_ = s.eventEmitter.EmitDealEvent(ctx, models.EventPayload{
			Type:       "crm.deal.stage_changed",
			Priority:   models.PriorityNormal,
			ResourceID: deal.ID.String(),
			ModuleID:   "crm",
			Title:      deal.Name + " moved to " + stage.Name,
			Body:       "Deal stage changed",
			DeepLink:   "/crm/deals/" + deal.ID.String(),
			GroupKey:   "deal-stage:" + deal.ID.String(),
			TargetUserIDs: []string{ownerIDStr},
			Timestamp:  time.Now(),
		})
	}

	return s.getWithRelations(ctx, deal)
}

// AddTags adds tags to a deal
func (s *Service) AddTags(ctx context.Context, dealID uuid.UUID, tagIDs []uuid.UUID) (*models.DealWithRelations, error) {
	deal, err := s.repo.GetByID(ctx, dealID)
	if err != nil {
		return nil, err
	}

	// Validate tags
	for _, tagID := range tagIDs {
		exists, tagErr := s.repo.TagExists(ctx, tagID, models.EntityTypeDeal)
		if tagErr != nil {
			return nil, tagErr
		}
		if !exists {
			return nil, ErrTagNotFound
		}
	}

	if addErr := s.repo.AddTags(ctx, dealID, tagIDs); addErr != nil {
		return nil, addErr
	}

	return s.getWithRelations(ctx, deal)
}

// RemoveTags removes tags from a deal
func (s *Service) RemoveTags(ctx context.Context, dealID uuid.UUID, tagIDs []uuid.UUID) (*models.DealWithRelations, error) {
	deal, err := s.repo.GetByID(ctx, dealID)
	if err != nil {
		return nil, err
	}

	if removeErr := s.repo.RemoveTags(ctx, dealID, tagIDs); removeErr != nil {
		return nil, removeErr
	}

	return s.getWithRelations(ctx, deal)
}

// getWithRelations loads all relations for a deal
func (s *Service) getWithRelations(ctx context.Context, deal *models.Deal) (*models.DealWithRelations, error) {
	result := &models.DealWithRelations{
		Deal: *deal,
	}

	// Load stage name
	stageName, _ := s.repo.GetStageName(ctx, deal.StageID)
	result.StageName = stageName

	// Load contact name
	if deal.ContactID != nil {
		name, _ := s.repo.GetContactName(ctx, *deal.ContactID)
		if name != "" {
			result.ContactName = &name
		}
	}

	// Load company name
	if deal.CompanyID != nil {
		name, _ := s.repo.GetCompanyName(ctx, *deal.CompanyID)
		if name != "" {
			result.CompanyName = &name
		}
	}

	// Load owner name
	if deal.OwnerID != nil {
		name, _ := s.repo.GetOwnerName(ctx, *deal.OwnerID)
		if name != "" {
			result.OwnerName = &name
		}
	}

	// Load tags
	tags, _ := s.repo.GetTags(ctx, deal.ID)
	result.Tags = tags

	// Load custom field values
	values, _ := s.repo.GetCustomFieldValues(ctx, deal.ID)
	if len(values) > 0 {
		result.CustomFields = make(map[string]any)
		for _, v := range values {
			result.CustomFields[v.FieldName] = v.Value
		}
	}

	return result, nil
}
