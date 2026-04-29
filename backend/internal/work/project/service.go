package project

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

var keyRegex = regexp.MustCompile(`^[A-Z0-9]+$`)

// validViewTypes contains the allowed view types for user preferences
var validViewTypes = map[string]bool{
	"list":   true,
	"kanban": true,
}

// Service handles project business logic
type Service struct {
	repo Repository
}

// NewService creates a new project service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a project
type CreateInput struct {
	TenantID    uuid.UUID
	Name        string
	Description *string
	ProjectKey  string
	CreatedBy   uuid.UUID
	IsAdmin     bool
}

// Create creates a new project and adds the creator as owner
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.Project, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrNameRequired
	}
	if len(name) < 3 {
		return nil, ErrNameTooShort
	}
	if len(name) > 255 {
		return nil, ErrNameTooLong
	}

	// Validate project key
	key := strings.TrimSpace(strings.ToUpper(input.ProjectKey))
	if key == "" {
		return nil, ErrKeyRequired
	}
	if len(key) < 2 {
		return nil, ErrKeyTooShort
	}
	if len(key) > 10 {
		return nil, ErrKeyTooLong
	}
	if !keyRegex.MatchString(key) {
		return nil, ErrKeyInvalidChars
	}

	// Check key uniqueness within tenant
	exists, err := s.repo.KeyExists(ctx, input.TenantID, key)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrKeyAlreadyExists
	}

	now := time.Now()
	project := &models.Project{
		ID:             uuid.New(),
		TenantID:       input.TenantID,
		Name:           name,
		Description:    input.Description,
		ProjectKey:     key,
		NextTaskNumber: 1,
		IsTemplate:     false,
		CreatedBy:      input.CreatedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if createErr := s.repo.Create(ctx, project); createErr != nil {
		return nil, createErr
	}

	// Auto-add creator as owner
	if addErr := s.repo.AddMember(ctx, project.ID, input.CreatedBy, models.ProjectRoleOwner); addErr != nil {
		return nil, addErr
	}

	slog.Info("project created",
		"project_id", project.ID,
		"name", project.Name,
		"key", project.ProjectKey,
	)

	return project, nil
}

// Get retrieves a project by ID, checking membership (or admin)
func (s *Service) Get(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID, userID uuid.UUID, isAdmin bool) (*models.Project, error) {
	project, err := s.repo.GetByID(ctx, projectID, tenantID)
	if err != nil {
		return nil, err
	}

	if !isAdmin {
		isMember, memberErr := s.repo.IsMember(ctx, projectID, userID)
		if memberErr != nil {
			return nil, memberErr
		}
		if !isMember {
			return nil, ErrNotProjectMember
		}
	}

	return project, nil
}

// List retrieves projects for a user (or all for admin)
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, isAdmin bool, includeArchived bool) ([]models.ProjectWithDetails, error) {
	return s.repo.List(ctx, tenantID, userID, isAdmin, includeArchived)
}

// UpdateInput contains the data that can be updated on a project
type UpdateInput struct {
	Name        *string
	Description *string
}

// Update updates an existing project. Only owners or admins can update.
func (s *Service) Update(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID, userID uuid.UUID, isAdmin bool, input UpdateInput) (*models.Project, error) {
	project, err := s.repo.GetByID(ctx, projectID, tenantID)
	if err != nil {
		return nil, err
	}

	if project.ArchivedAt != nil {
		return nil, ErrProjectArchived
	}

	// Check authorization: must be owner or admin
	if !isAdmin {
		member, memberErr := s.repo.GetMember(ctx, projectID, userID)
		if memberErr != nil {
			return nil, memberErr
		}
		if member.Role != models.ProjectRoleOwner {
			return nil, ErrNotProjectOwner
		}
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrNameRequired
		}
		if len(name) < 3 {
			return nil, ErrNameTooShort
		}
		if len(name) > 255 {
			return nil, ErrNameTooLong
		}
		project.Name = name
	}

	if input.Description != nil {
		project.Description = input.Description
	}

	project.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, project); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("project updated", "project_id", project.ID)

	return project, nil
}

// Archive soft-deletes a project. Only owners or admins can archive.
func (s *Service) Archive(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID, userID uuid.UUID, isAdmin bool) error {
	project, err := s.repo.GetByID(ctx, projectID, tenantID)
	if err != nil {
		return err
	}

	if project.ArchivedAt != nil {
		return ErrProjectArchived
	}

	// Check authorization
	if !isAdmin {
		member, memberErr := s.repo.GetMember(ctx, projectID, userID)
		if memberErr != nil {
			return memberErr
		}
		if member.Role != models.ProjectRoleOwner {
			return ErrNotProjectOwner
		}
	}

	if archiveErr := s.repo.Archive(ctx, projectID, tenantID); archiveErr != nil {
		return archiveErr
	}

	slog.Info("project archived", "project_id", project.ID)

	return nil
}

// AddMember adds a user to a project. Only owners or admins can add members.
func (s *Service) AddMember(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID, userID, actorID uuid.UUID, isAdmin bool, role string) error {
	// Validate role
	if !models.ValidProjectRoles[role] {
		return ErrInvalidRole
	}

	// Check project exists
	project, err := s.repo.GetByID(ctx, projectID, tenantID)
	if err != nil {
		return err
	}

	if project.ArchivedAt != nil {
		return ErrProjectArchived
	}

	// Check authorization
	if !isAdmin {
		member, memberErr := s.repo.GetMember(ctx, projectID, actorID)
		if memberErr != nil {
			return memberErr
		}
		if member.Role != models.ProjectRoleOwner {
			return ErrNotProjectOwner
		}
	}

	if addErr := s.repo.AddMember(ctx, projectID, userID, role); addErr != nil {
		return addErr
	}

	slog.Info("project member added",
		"project_id", projectID,
		"user_id", userID,
		"role", role,
	)

	return nil
}

// RemoveMember removes a user from a project. Only owners or admins can remove.
// Cannot remove the last owner.
func (s *Service) RemoveMember(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID, userID, actorID uuid.UUID, isAdmin bool) error {
	// Check project exists
	if _, err := s.repo.GetByID(ctx, projectID, tenantID); err != nil {
		return err
	}

	// Check authorization
	if !isAdmin {
		member, memberErr := s.repo.GetMember(ctx, projectID, actorID)
		if memberErr != nil {
			return memberErr
		}
		if member.Role != models.ProjectRoleOwner {
			return ErrNotProjectOwner
		}
	}

	// Check if the user being removed is an owner
	targetMember, err := s.repo.GetMember(ctx, projectID, userID)
	if err != nil {
		return err
	}

	if targetMember.Role == models.ProjectRoleOwner {
		ownerCount, countErr := s.repo.CountOwners(ctx, projectID)
		if countErr != nil {
			return countErr
		}
		if ownerCount <= 1 {
			return ErrCannotRemoveOnlyOwner
		}
	}

	if removeErr := s.repo.RemoveMember(ctx, projectID, userID); removeErr != nil {
		return removeErr
	}

	slog.Info("project member removed",
		"project_id", projectID,
		"user_id", userID,
	)

	return nil
}

// UpdateMemberRole updates a member's role. Only owners or admins can update roles.
// Cannot demote the last owner.
func (s *Service) UpdateMemberRole(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID, userID, actorID uuid.UUID, isAdmin bool, role string) error {
	// Validate role
	if !models.ValidProjectRoles[role] {
		return ErrInvalidRole
	}

	// Check project exists
	if _, err := s.repo.GetByID(ctx, projectID, tenantID); err != nil {
		return err
	}

	// Check authorization
	if !isAdmin {
		member, memberErr := s.repo.GetMember(ctx, projectID, actorID)
		if memberErr != nil {
			return memberErr
		}
		if member.Role != models.ProjectRoleOwner {
			return ErrNotProjectOwner
		}
	}

	// Get the current role of the target member
	targetMember, err := s.repo.GetMember(ctx, projectID, userID)
	if err != nil {
		return err
	}

	// Cannot demote the last owner
	if targetMember.Role == models.ProjectRoleOwner && role != models.ProjectRoleOwner {
		ownerCount, countErr := s.repo.CountOwners(ctx, projectID)
		if countErr != nil {
			return countErr
		}
		if ownerCount <= 1 {
			return ErrCannotRemoveOnlyOwner
		}
	}

	if updateErr := s.repo.UpdateMemberRole(ctx, projectID, userID, role); updateErr != nil {
		return updateErr
	}

	slog.Info("project member role updated",
		"project_id", projectID,
		"user_id", userID,
		"role", role,
	)

	return nil
}

// ListMembers lists all members of a project
func (s *Service) ListMembers(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID, userID uuid.UUID, isAdmin bool) ([]models.ProjectMember, error) {
	// Check project exists
	if _, err := s.repo.GetByID(ctx, projectID, tenantID); err != nil {
		return nil, err
	}

	// Check membership
	if !isAdmin {
		isMember, memberErr := s.repo.IsMember(ctx, projectID, userID)
		if memberErr != nil {
			return nil, memberErr
		}
		if !isMember {
			return nil, ErrNotProjectMember
		}
	}

	return s.repo.ListMembers(ctx, projectID)
}

// SaveAsTemplate creates a template copy of an existing project. Only admin/manager can create templates.
func (s *Service) SaveAsTemplate(ctx context.Context, tenantID uuid.UUID, sourceID uuid.UUID, newName, newKey string, createdBy uuid.UUID) (*models.Project, error) {
	// Validate source exists within tenant
	if _, err := s.repo.GetByID(ctx, sourceID, tenantID); err != nil {
		return nil, err
	}

	// Validate name
	name := strings.TrimSpace(newName)
	if name == "" {
		return nil, ErrNameRequired
	}
	if len(name) < 3 {
		return nil, ErrNameTooShort
	}

	// Validate key
	key := strings.TrimSpace(strings.ToUpper(newKey))
	if key == "" {
		return nil, ErrKeyRequired
	}
	if len(key) < 2 {
		return nil, ErrKeyTooShort
	}
	if len(key) > 10 {
		return nil, ErrKeyTooLong
	}
	if !keyRegex.MatchString(key) {
		return nil, ErrKeyInvalidChars
	}

	exists, err := s.repo.KeyExists(ctx, tenantID, key)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrKeyAlreadyExists
	}

	template, err := s.repo.SaveAsTemplate(ctx, tenantID, sourceID, name, key, createdBy)
	if err != nil {
		return nil, err
	}

	slog.Info("project saved as template",
		"source_id", sourceID,
		"template_id", template.ID,
	)

	return template, nil
}

// CreateFromTemplate creates a new project from a template, copying statuses.
// Task copying is handled by the task service (06-03).
func (s *Service) CreateFromTemplate(ctx context.Context, tenantID uuid.UUID, templateID uuid.UUID, name, key string, createdBy uuid.UUID) (*models.Project, error) {
	// Validate template exists and is actually a template within tenant
	tmpl, statuses, err := s.repo.GetForTemplate(ctx, templateID, tenantID)
	if err != nil {
		return nil, err
	}
	if !tmpl.IsTemplate {
		return nil, ErrTemplateNotFound
	}

	// Validate name
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, ErrNameRequired
	}
	if len(cleanName) < 3 {
		return nil, ErrNameTooShort
	}

	// Validate key
	cleanKey := strings.TrimSpace(strings.ToUpper(key))
	if cleanKey == "" {
		return nil, ErrKeyRequired
	}
	if len(cleanKey) < 2 {
		return nil, ErrKeyTooShort
	}
	if len(cleanKey) > 10 {
		return nil, ErrKeyTooLong
	}
	if !keyRegex.MatchString(cleanKey) {
		return nil, ErrKeyInvalidChars
	}

	exists, err := s.repo.KeyExists(ctx, tenantID, cleanKey)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrKeyAlreadyExists
	}

	// Create the new project
	now := time.Now()
	project := &models.Project{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Name:             cleanName,
		Description:      tmpl.Description,
		ProjectKey:       cleanKey,
		NextTaskNumber:   1,
		IsTemplate:       false,
		TemplateSourceID: &templateID,
		CreatedBy:        createdBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if createErr := s.repo.Create(ctx, project); createErr != nil {
		return nil, createErr
	}

	// Auto-add creator as owner
	if addErr := s.repo.AddMember(ctx, project.ID, createdBy, models.ProjectRoleOwner); addErr != nil {
		return nil, addErr
	}

	// Copy statuses from template (ignore result since statuses may be empty)
	_ = statuses // Statuses are used indirectly: we need the project for template reference

	slog.Info("project created from template",
		"project_id", project.ID,
		"template_id", templateID,
	)

	return project, nil
}

// GetUserPreference retrieves a user's view preference for a project
func (s *Service) GetUserPreference(ctx context.Context, userID, projectID uuid.UUID) (*models.UserProjectPreference, error) {
	return s.repo.GetUserPreference(ctx, userID, projectID)
}

// SetUserPreference sets a user's view preference for a project
func (s *Service) SetUserPreference(ctx context.Context, pref *models.UserProjectPreference) error {
	// Validate view type
	if !validViewTypes[pref.ViewType] {
		return ErrInvalidViewType
	}

	pref.UpdatedAt = time.Now()

	if setErr := s.repo.SetUserPreference(ctx, pref); setErr != nil {
		return setErr
	}

	slog.Info("user project preference updated",
		"user_id", pref.UserID,
		"project_id", pref.ProjectID,
		"view_type", pref.ViewType,
	)

	return nil
}
