package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/models"
)

// DashboardService handles dashboard layout business logic.
type DashboardService struct {
	repo DashboardRepository
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(repo DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

// hardcodedDefaultLayout returns a fallback layout when neither user layout
// nor role default exists in the database.
func hardcodedDefaultLayout() *models.DashboardLayoutResponse {
	layout := json.RawMessage(`[
		{"i":"recent-contacts","x":0,"y":0,"w":4,"h":3},
		{"i":"deal-pipeline","x":4,"y":0,"w":4,"h":4},
		{"i":"unread-messages","x":8,"y":0,"w":4,"h":3},
		{"i":"activity-feed","x":0,"y":3,"w":4,"h":4},
		{"i":"quick-actions","x":4,"y":4,"w":4,"h":2},
		{"i":"notification-summary","x":8,"y":3,"w":4,"h":3}
	]`)
	activeWidgets := json.RawMessage(`["recent-contacts","deal-pipeline","unread-messages","activity-feed","quick-actions","notification-summary"]`)

	return &models.DashboardLayoutResponse{
		Layout:        layout,
		ActiveWidgets: activeWidgets,
		IsCustom:      false,
	}
}

// GetDashboard returns the effective dashboard layout for a user.
// Priority: user override > role default > hardcoded default.
func (s *DashboardService) GetDashboard(ctx context.Context, tenantID uuid.UUID, userID, role string) (*models.DashboardLayoutResponse, error) {
	// 1. Try user-specific layout
	userLayout, err := s.repo.GetUserLayout(ctx, tenantID, userID)
	if err == nil {
		return &models.DashboardLayoutResponse{
			Layout:        userLayout.Layout,
			ActiveWidgets: userLayout.ActiveWidgets,
			IsCustom:      true,
			UpdatedAt:     userLayout.UpdatedAt,
		}, nil
	}
	if !errors.Is(err, ErrDashboardNotFound) {
		slog.Error("failed to get user dashboard layout",
			"user_id", userID,
			"error", err,
		)
		return nil, err
	}

	// 2. Fall back to role default
	roleDefault, err := s.repo.GetDefaultLayout(ctx, tenantID, role)
	if err == nil {
		return &models.DashboardLayoutResponse{
			Layout:        roleDefault.Layout,
			ActiveWidgets: roleDefault.ActiveWidgets,
			IsCustom:      false,
			UpdatedAt:     roleDefault.UpdatedAt,
		}, nil
	}
	if !errors.Is(err, ErrDashboardNotFound) {
		slog.Error("failed to get role default layout",
			"role", role,
			"error", err,
		)
		return nil, err
	}

	// 3. Hardcoded fallback
	slog.Info("using hardcoded default layout",
		"user_id", userID,
		"role", role,
	)
	return hardcodedDefaultLayout(), nil
}

// SaveDashboard saves a user's personal dashboard layout.
func (s *DashboardService) SaveDashboard(ctx context.Context, tenantID uuid.UUID, userID string, layout, activeWidgets json.RawMessage) (*models.DashboardLayoutResponse, error) {
	saved, err := s.repo.UpsertUserLayout(ctx, &models.UserDashboardLayout{
		TenantID:      tenantID,
		UserID:        userID,
		Layout:        layout,
		ActiveWidgets: activeWidgets,
	})
	if err != nil {
		slog.Error("failed to save user dashboard layout",
			"user_id", userID,
			"error", err,
		)
		return nil, err
	}

	return &models.DashboardLayoutResponse{
		Layout:        saved.Layout,
		ActiveWidgets: saved.ActiveWidgets,
		IsCustom:      true,
		UpdatedAt:     saved.UpdatedAt,
	}, nil
}

// ResetToDefaults removes a user's personal layout override.
func (s *DashboardService) ResetToDefaults(ctx context.Context, tenantID uuid.UUID, userID string) error {
	err := s.repo.DeleteUserLayout(ctx, tenantID, userID)
	if err != nil && !errors.Is(err, ErrDashboardNotFound) {
		slog.Error("failed to delete user dashboard layout",
			"user_id", userID,
			"error", err,
		)
		return err
	}
	return nil
}

// GetDefaults returns the tenant's default layout for a role.
func (s *DashboardService) GetDefaults(ctx context.Context, tenantID uuid.UUID, role string) (*models.DashboardDefault, error) {
	def, err := s.repo.GetDefaultLayout(ctx, tenantID, role)
	if err != nil {
		if errors.Is(err, ErrDashboardNotFound) {
			return nil, ErrDashboardNotFound
		}
		slog.Error("failed to get role default layout",
			"role", role,
			"error", err,
		)
		return nil, err
	}
	return def, nil
}

// SaveDefaults saves or updates the tenant's default layout for a role.
func (s *DashboardService) SaveDefaults(ctx context.Context, tenantID uuid.UUID, role string, layout, activeWidgets json.RawMessage) (*models.DashboardDefault, error) {
	saved, err := s.repo.UpsertDefaultLayout(ctx, tenantID, &models.DashboardDefault{
		Role:          role,
		Layout:        layout,
		ActiveWidgets: activeWidgets,
	})
	if err != nil {
		slog.Error("failed to save role default layout",
			"role", role,
			"error", err,
		)
		return nil, err
	}
	return saved, nil
}
