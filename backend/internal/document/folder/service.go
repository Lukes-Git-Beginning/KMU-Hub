package folder

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Valid space types for folders.
var validSpaceTypes = map[string]bool{
	models.FolderSpacePersonal: true,
	models.FolderSpaceTeam:     true,
	models.FolderSpaceProject:  true,
}

// Service handles folder business logic.
type Service struct {
	repo Repository
}

// NewService creates a new folder service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a folder.
type CreateInput struct {
	TenantID  uuid.UUID
	Name      string
	ParentID  *uuid.UUID
	SpaceType string
	SpaceID   uuid.UUID
	Icon      string
	CreatedBy uuid.UUID
}

// Create creates a new folder with validation.
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.DocumentFolder, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrFolderNameRequired
	}
	if len(name) > 255 {
		return nil, ErrFolderNameTooLong
	}

	// Validate space type
	if !validSpaceTypes[input.SpaceType] {
		return nil, ErrInvalidSpaceType
	}

	// Validate parent exists if provided
	if input.ParentID != nil {
		_, err := s.repo.GetByID(ctx, input.TenantID, *input.ParentID)
		if err != nil {
			return nil, ErrFolderNotFound
		}
	}

	// Check for duplicate name in same parent
	siblings, _, err := s.repo.List(ctx, ListFilter{TenantID: input.TenantID, ParentID: input.ParentID, SpaceType: &input.SpaceType, SpaceID: &input.SpaceID})
	if err != nil {
		return nil, err
	}
	for _, sibling := range siblings {
		if strings.EqualFold(sibling.Name, name) {
			return nil, ErrFolderNameConflict
		}
	}

	icon := input.Icon
	if icon == "" {
		icon = "folder"
	}

	now := time.Now()
	folder := &models.DocumentFolder{
		ID:        uuid.New(),
		TenantID:  input.TenantID,
		Name:      name,
		ParentID:  input.ParentID,
		SpaceType: input.SpaceType,
		SpaceID:   input.SpaceID,
		IsSystem:  false,
		Icon:      icon,
		CreatedBy: input.CreatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, folder); err != nil {
		return nil, err
	}

	slog.Info("folder created",
		"folder_id", folder.ID,
		"name", folder.Name,
		"space_type", folder.SpaceType,
		"space_id", folder.SpaceID,
		"created_by", folder.CreatedBy,
	)

	return folder, nil
}

// GetByID retrieves a folder by ID, scoped to tenant.
func (s *Service) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*models.DocumentFolder, error) {
	folder, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return folder, nil
}

// List retrieves folders matching the given filter.
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*models.DocumentFolder, int, error) {
	return s.repo.List(ctx, filter)
}

// Update updates an existing folder with validation, scoped to tenant.
func (s *Service) Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, input UpdateInput) error {
	folder, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Check if trying to rename a system folder
	if input.Name != nil && folder.IsSystem {
		return ErrCannotRenameSystemFolder
	}

	// Validate new name if provided
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return ErrFolderNameRequired
		}
		if len(name) > 255 {
			return ErrFolderNameTooLong
		}

		// Check for duplicate name in same parent
		parentID := folder.ParentID
		if input.ParentID != nil {
			parentID = input.ParentID
		}
		siblings, _, listErr := s.repo.List(ctx, ListFilter{
			TenantID:  tenantID,
			ParentID:  parentID,
			SpaceType: &folder.SpaceType,
			SpaceID:   &folder.SpaceID,
		})
		if listErr != nil {
			return listErr
		}
		for _, sibling := range siblings {
			if sibling.ID != id && strings.EqualFold(sibling.Name, name) {
				return ErrFolderNameConflict
			}
		}
		input.Name = &name
	}

	// Check circular parent if moving
	if input.ParentID != nil {
		newParentID := *input.ParentID
		if newParentID == id {
			return ErrCannotMoveToSelf
		}

		// Check if new parent is a descendant of this folder (would create cycle)
		isDesc, descErr := s.repo.IsDescendant(ctx, newParentID, id)
		if descErr != nil {
			return descErr
		}
		if isDesc {
			return ErrCircularParent
		}
	}

	if err := s.repo.Update(ctx, tenantID, id, input); err != nil {
		return err
	}

	slog.Info("folder updated",
		"folder_id", id,
	)

	return nil
}

// Delete deletes a folder after validation, scoped to tenant.
func (s *Service) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	folder, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if folder.IsSystem {
		return ErrCannotDeleteSystemFolder
	}

	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return err
	}

	slog.Info("folder deleted",
		"folder_id", id,
		"name", folder.Name,
	)

	return nil
}

// GetPath returns the breadcrumb path from root to the given folder.
func (s *Service) GetPath(ctx context.Context, id uuid.UUID) ([]models.FolderPathSegment, error) {
	return s.repo.GetPath(ctx, id)
}

// InitializeUserSpace creates a personal root folder with default subfolders for a new user.
// Default subfolders: "Dokumente", "Bilder", "Vorlagen" (all system folders).
func (s *Service) InitializeUserSpace(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) error {
	now := time.Now()

	// Create personal root folder
	root := &models.DocumentFolder{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      "Meine Dateien",
		ParentID:  nil,
		SpaceType: models.FolderSpacePersonal,
		SpaceID:   userID,
		IsSystem:  true,
		Icon:      "home",
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, root); err != nil {
		return err
	}

	// Default subfolders for personal space
	defaults := []struct {
		name string
		icon string
	}{
		{"Dokumente", "file-text"},
		{"Bilder", "image"},
		{"Vorlagen", "layout-template"},
	}

	for _, d := range defaults {
		sub := &models.DocumentFolder{
			ID:        uuid.New(),
			TenantID:  tenantID,
			Name:      d.name,
			ParentID:  &root.ID,
			SpaceType: models.FolderSpacePersonal,
			SpaceID:   userID,
			IsSystem:  true,
			Icon:      d.icon,
			CreatedBy: userID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.repo.Create(ctx, sub); err != nil {
			return err
		}
	}

	slog.Info("user space initialized",
		"user_id", userID,
		"root_folder_id", root.ID,
	)

	return nil
}

// InitializeTeamSpace creates a team root folder with default subfolders for a new team.
// Default subfolders: "Allgemein", "Projekte", "Vorlagen" (all system folders).
func (s *Service) InitializeTeamSpace(ctx context.Context, tenantID uuid.UUID, teamID uuid.UUID, teamName string) error {
	now := time.Now()

	// Create team root folder
	root := &models.DocumentFolder{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      teamName,
		ParentID:  nil,
		SpaceType: models.FolderSpaceTeam,
		SpaceID:   teamID,
		IsSystem:  true,
		Icon:      "users",
		CreatedBy: teamID, // team as creator
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, root); err != nil {
		return err
	}

	// Default subfolders for team space
	defaults := []struct {
		name string
		icon string
	}{
		{"Allgemein", "folder"},
		{"Projekte", "briefcase"},
		{"Vorlagen", "layout-template"},
	}

	for _, d := range defaults {
		sub := &models.DocumentFolder{
			ID:        uuid.New(),
			TenantID:  tenantID,
			Name:      d.name,
			ParentID:  &root.ID,
			SpaceType: models.FolderSpaceTeam,
			SpaceID:   teamID,
			IsSystem:  true,
			Icon:      d.icon,
			CreatedBy: teamID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.repo.Create(ctx, sub); err != nil {
			return err
		}
	}

	slog.Info("team space initialized",
		"team_id", teamID,
		"team_name", teamName,
		"root_folder_id", root.ID,
	)

	return nil
}
