package activity

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles activity business logic
type Service struct {
	repo Repository
}

// NewService creates a new activity service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create an activity
type CreateInput struct {
	ActivityType models.ActivityType
	Subject      string
	Description  *string
	ContactID    *uuid.UUID
	CompanyID    *uuid.UUID
	DealID       *uuid.UUID
	AssignedTo   *uuid.UUID
	DueDate      *time.Time
	TagIDs       []uuid.UUID
	CustomFields map[uuid.UUID]any // field_id -> value
	CreatedBy    uuid.UUID
}

// Create creates a new activity
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.ActivityWithRelations, error) {
	// Validate activity type
	if !input.ActivityType.IsValid() {
		return nil, ErrInvalidActivityType
	}

	// Validate subject
	subject := strings.TrimSpace(input.Subject)
	if subject == "" {
		return nil, ErrSubjectRequired
	}

	// Validate contact exists if provided
	if input.ContactID != nil {
		exists, err := s.repo.ContactExists(ctx, *input.ContactID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrContactNotFound
		}
	}

	// Validate company exists if provided
	if input.CompanyID != nil {
		exists, err := s.repo.CompanyExists(ctx, *input.CompanyID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrCompanyNotFound
		}
	}

	// Validate deal exists if provided
	if input.DealID != nil {
		exists, err := s.repo.DealExists(ctx, *input.DealID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrDealNotFound
		}
	}

	// Validate assigned_to user exists if provided
	if input.AssignedTo != nil {
		exists, err := s.repo.UserExists(ctx, *input.AssignedTo)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrUserNotFound
		}
	}

	// Validate tags are for activities
	for _, tagID := range input.TagIDs {
		exists, err := s.repo.TagExists(ctx, tagID, models.EntityTypeActivity)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrTagNotFound
		}
	}

	now := time.Now()
	activity := &models.Activity{
		ID:           uuid.New(),
		ActivityType: input.ActivityType,
		Subject:      subject,
		Description:  input.Description,
		ContactID:    input.ContactID,
		CompanyID:    input.CompanyID,
		DealID:       input.DealID,
		AssignedTo:   input.AssignedTo,
		DueDate:      input.DueDate,
		IsCompleted:  false,
		CompletedAt:  nil,
		CreatedBy:    input.CreatedBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, activity); err != nil {
		return nil, err
	}

	// Add tags
	if len(input.TagIDs) > 0 {
		if err := s.repo.AddTags(ctx, activity.ID, input.TagIDs); err != nil {
			return nil, err
		}
	}

	// Set custom field values
	if len(input.CustomFields) > 0 {
		if err := s.repo.SetCustomFieldValues(ctx, activity.ID, input.CustomFields); err != nil {
			return nil, err
		}
	}

	slog.Info("activity created",
		"activity_id", activity.ID,
		"type", activity.ActivityType,
		"subject", activity.Subject,
	)

	return s.getWithRelations(ctx, activity)
}

// GetByID retrieves an activity by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.ActivityWithRelations, error) {
	activity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.getWithRelations(ctx, activity)
}

// ListInput contains filtering options for listing activities
type ListInput struct {
	ActivityType *models.ActivityType
	ContactID    *uuid.UUID
	CompanyID    *uuid.UUID
	DealID       *uuid.UUID
	AssignedTo   *uuid.UUID
	CreatedBy    *uuid.UUID
	IsCompleted  *bool
	TagIDs       []uuid.UUID
	Search       string
	Page         int
	PageSize     int
	SortBy       string
	SortDesc     bool
}

// List retrieves activities with optional filtering
func (s *Service) List(ctx context.Context, input ListInput) ([]*models.ActivityWithRelations, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := (input.Page - 1) * input.PageSize
	filter := ListFilter{
		ActivityType: input.ActivityType,
		ContactID:    input.ContactID,
		CompanyID:    input.CompanyID,
		DealID:       input.DealID,
		AssignedTo:   input.AssignedTo,
		CreatedBy:    input.CreatedBy,
		IsCompleted:  input.IsCompleted,
		TagIDs:       input.TagIDs,
		Search:       input.Search,
		SortBy:       input.SortBy,
		SortDesc:     input.SortDesc,
	}

	activities, total, err := s.repo.List(ctx, filter, offset, input.PageSize)
	if err != nil {
		return nil, 0, err
	}

	var results []*models.ActivityWithRelations
	for _, a := range activities {
		withRel, relErr := s.getWithRelations(ctx, a)
		if relErr != nil {
			return nil, 0, relErr
		}
		results = append(results, withRel)
	}

	return results, total, nil
}

// UpdateInput contains the data that can be updated on an activity
type UpdateInput struct {
	Subject      *string
	Description  *string
	ContactID    *uuid.UUID // use uuid.Nil to clear
	CompanyID    *uuid.UUID // use uuid.Nil to clear
	DealID       *uuid.UUID // use uuid.Nil to clear
	AssignedTo   *uuid.UUID // use uuid.Nil to clear
	DueDate      *time.Time
	CustomFields map[uuid.UUID]any
}

// Update updates an existing activity
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*models.ActivityWithRelations, error) {
	activity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Subject != nil {
		subject := strings.TrimSpace(*input.Subject)
		if subject == "" {
			return nil, ErrSubjectRequired
		}
		activity.Subject = subject
	}

	if input.Description != nil {
		activity.Description = input.Description
	}

	if input.ContactID != nil {
		if *input.ContactID == uuid.Nil {
			activity.ContactID = nil
		} else {
			exists, contactErr := s.repo.ContactExists(ctx, *input.ContactID)
			if contactErr != nil {
				return nil, contactErr
			}
			if !exists {
				return nil, ErrContactNotFound
			}
			activity.ContactID = input.ContactID
		}
	}

	if input.CompanyID != nil {
		if *input.CompanyID == uuid.Nil {
			activity.CompanyID = nil
		} else {
			exists, companyErr := s.repo.CompanyExists(ctx, *input.CompanyID)
			if companyErr != nil {
				return nil, companyErr
			}
			if !exists {
				return nil, ErrCompanyNotFound
			}
			activity.CompanyID = input.CompanyID
		}
	}

	if input.DealID != nil {
		if *input.DealID == uuid.Nil {
			activity.DealID = nil
		} else {
			exists, dealErr := s.repo.DealExists(ctx, *input.DealID)
			if dealErr != nil {
				return nil, dealErr
			}
			if !exists {
				return nil, ErrDealNotFound
			}
			activity.DealID = input.DealID
		}
	}

	if input.AssignedTo != nil {
		if *input.AssignedTo == uuid.Nil {
			activity.AssignedTo = nil
		} else {
			exists, userErr := s.repo.UserExists(ctx, *input.AssignedTo)
			if userErr != nil {
				return nil, userErr
			}
			if !exists {
				return nil, ErrUserNotFound
			}
			activity.AssignedTo = input.AssignedTo
		}
	}

	if input.DueDate != nil {
		activity.DueDate = input.DueDate
	}

	activity.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, activity); updateErr != nil {
		return nil, updateErr
	}

	// Update custom fields if provided
	if len(input.CustomFields) > 0 {
		if cfErr := s.repo.SetCustomFieldValues(ctx, activity.ID, input.CustomFields); cfErr != nil {
			return nil, cfErr
		}
	}

	slog.Info("activity updated", "activity_id", activity.ID)

	return s.getWithRelations(ctx, activity)
}

// Delete removes an activity
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	activity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if deleteErr := s.repo.Delete(ctx, id); deleteErr != nil {
		return deleteErr
	}

	slog.Info("activity deleted",
		"activity_id", activity.ID,
		"subject", activity.Subject,
	)

	return nil
}

// Complete marks an activity as completed
func (s *Service) Complete(ctx context.Context, id uuid.UUID) (*models.ActivityWithRelations, error) {
	activity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Idempotent - if already completed, just return
	if activity.IsCompleted {
		return s.getWithRelations(ctx, activity)
	}

	now := time.Now()
	activity.IsCompleted = true
	activity.CompletedAt = &now
	activity.UpdatedAt = now

	if updateErr := s.repo.Update(ctx, activity); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("activity completed", "activity_id", activity.ID)

	return s.getWithRelations(ctx, activity)
}

// Uncomplete marks an activity as not completed
func (s *Service) Uncomplete(ctx context.Context, id uuid.UUID) (*models.ActivityWithRelations, error) {
	activity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Idempotent - if already not completed, just return
	if !activity.IsCompleted {
		return s.getWithRelations(ctx, activity)
	}

	activity.IsCompleted = false
	activity.CompletedAt = nil
	activity.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, activity); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("activity uncompleted", "activity_id", activity.ID)

	return s.getWithRelations(ctx, activity)
}

// AddTags adds tags to an activity
func (s *Service) AddTags(ctx context.Context, activityID uuid.UUID, tagIDs []uuid.UUID) (*models.ActivityWithRelations, error) {
	activity, err := s.repo.GetByID(ctx, activityID)
	if err != nil {
		return nil, err
	}

	// Validate tags
	for _, tagID := range tagIDs {
		exists, tagErr := s.repo.TagExists(ctx, tagID, models.EntityTypeActivity)
		if tagErr != nil {
			return nil, tagErr
		}
		if !exists {
			return nil, ErrTagNotFound
		}
	}

	if addErr := s.repo.AddTags(ctx, activityID, tagIDs); addErr != nil {
		return nil, addErr
	}

	return s.getWithRelations(ctx, activity)
}

// RemoveTags removes tags from an activity
func (s *Service) RemoveTags(ctx context.Context, activityID uuid.UUID, tagIDs []uuid.UUID) (*models.ActivityWithRelations, error) {
	activity, err := s.repo.GetByID(ctx, activityID)
	if err != nil {
		return nil, err
	}

	if removeErr := s.repo.RemoveTags(ctx, activityID, tagIDs); removeErr != nil {
		return nil, removeErr
	}

	return s.getWithRelations(ctx, activity)
}

// getWithRelations loads all relations for an activity
func (s *Service) getWithRelations(ctx context.Context, activity *models.Activity) (*models.ActivityWithRelations, error) {
	result := &models.ActivityWithRelations{
		Activity: *activity,
	}

	// Load contact name
	if activity.ContactID != nil {
		name, _ := s.repo.GetContactName(ctx, *activity.ContactID)
		if name != "" {
			result.ContactName = &name
		}
	}

	// Load company name
	if activity.CompanyID != nil {
		name, _ := s.repo.GetCompanyName(ctx, *activity.CompanyID)
		if name != "" {
			result.CompanyName = &name
		}
	}

	// Load deal name
	if activity.DealID != nil {
		name, _ := s.repo.GetDealName(ctx, *activity.DealID)
		if name != "" {
			result.DealName = &name
		}
	}

	// Load assigned_to name
	if activity.AssignedTo != nil {
		name, _ := s.repo.GetUserName(ctx, *activity.AssignedTo)
		if name != "" {
			result.AssignedToName = &name
		}
	}

	// Load tags
	tags, _ := s.repo.GetTags(ctx, activity.ID)
	result.Tags = tags

	// Load custom field values
	values, _ := s.repo.GetCustomFieldValues(ctx, activity.ID)
	if len(values) > 0 {
		result.CustomFields = make(map[string]any)
		for _, v := range values {
			result.CustomFields[v.FieldName] = v.Value
		}
	}

	return result, nil
}
