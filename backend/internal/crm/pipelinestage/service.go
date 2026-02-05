package pipelinestage

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

var colorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Service handles pipeline stage business logic
type Service struct {
	repo Repository
}

// NewService creates a new pipeline stage service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains the data needed to create a pipeline stage
type CreateInput struct {
	Name        string
	Color       string
	IsWon       bool
	IsLost      bool
	Probability float64
}

// Create creates a new pipeline stage
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.PipelineStage, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrNameRequired
	}

	// Validate color
	color := strings.TrimSpace(input.Color)
	if color == "" {
		color = "#3b82f6" // Default blue
	}
	if !colorRegex.MatchString(color) {
		return nil, ErrInvalidColor
	}

	// Validate probability
	if input.Probability < 0 || input.Probability > 100 {
		return nil, ErrInvalidProbability
	}

	// Check is_won uniqueness
	if input.IsWon {
		hasWon, err := s.repo.HasWonStage(ctx, nil)
		if err != nil {
			return nil, err
		}
		if hasWon {
			return nil, ErrWonStageExists
		}
	}

	// Check is_lost uniqueness
	if input.IsLost {
		hasLost, err := s.repo.HasLostStage(ctx, nil)
		if err != nil {
			return nil, err
		}
		if hasLost {
			return nil, ErrLostStageExists
		}
	}

	// Get next sort order
	sortOrder, err := s.repo.GetNextSortOrder(ctx)
	if err != nil {
		return nil, err
	}

	stage := &models.PipelineStage{
		ID:          uuid.New(),
		Name:        name,
		Color:       color,
		SortOrder:   sortOrder,
		IsWon:       input.IsWon,
		IsLost:      input.IsLost,
		Probability: decimal.NewFromFloat(input.Probability),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if createErr := s.repo.Create(ctx, stage); createErr != nil {
		return nil, createErr
	}

	slog.Info("pipeline stage created",
		"stage_id", stage.ID,
		"name", stage.Name,
	)

	return stage, nil
}

// GetByID retrieves a pipeline stage by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.PipelineStage, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all pipeline stages ordered by sort_order
func (s *Service) List(ctx context.Context) ([]*models.PipelineStage, error) {
	return s.repo.List(ctx)
}

// ListWithStats retrieves all pipeline stages with deal counts and total values
func (s *Service) ListWithStats(ctx context.Context) ([]*models.PipelineStageWithStats, error) {
	return s.repo.ListWithStats(ctx)
}

// UpdateInput contains the data that can be updated on a pipeline stage
type UpdateInput struct {
	Name        *string
	Color       *string
	IsWon       *bool
	IsLost      *bool
	Probability *float64
}

// Update updates an existing pipeline stage
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*models.PipelineStage, error) {
	stage, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrNameRequired
		}
		stage.Name = name
	}

	if input.Color != nil {
		color := strings.TrimSpace(*input.Color)
		if !colorRegex.MatchString(color) {
			return nil, ErrInvalidColor
		}
		stage.Color = color
	}

	if input.Probability != nil {
		if *input.Probability < 0 || *input.Probability > 100 {
			return nil, ErrInvalidProbability
		}
		stage.Probability = decimal.NewFromFloat(*input.Probability)
	}

	if input.IsWon != nil && *input.IsWon && !stage.IsWon {
		hasWon, wonErr := s.repo.HasWonStage(ctx, &id)
		if wonErr != nil {
			return nil, wonErr
		}
		if hasWon {
			return nil, ErrWonStageExists
		}
		stage.IsWon = true
		stage.IsLost = false // Cannot be both
	} else if input.IsWon != nil && !*input.IsWon {
		stage.IsWon = false
	}

	if input.IsLost != nil && *input.IsLost && !stage.IsLost {
		hasLost, lostErr := s.repo.HasLostStage(ctx, &id)
		if lostErr != nil {
			return nil, lostErr
		}
		if hasLost {
			return nil, ErrLostStageExists
		}
		stage.IsLost = true
		stage.IsWon = false // Cannot be both
	} else if input.IsLost != nil && !*input.IsLost {
		stage.IsLost = false
	}

	stage.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, stage); updateErr != nil {
		return nil, updateErr
	}

	slog.Info("pipeline stage updated", "stage_id", stage.ID)

	return stage, nil
}

// Delete removes a pipeline stage
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	stage, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if stage has deals
	hasDeals, err := s.repo.HasDeals(ctx, id)
	if err != nil {
		return err
	}
	if hasDeals {
		return ErrStageHasDeals
	}

	if deleteErr := s.repo.Delete(ctx, id); deleteErr != nil {
		return deleteErr
	}

	slog.Info("pipeline stage deleted",
		"stage_id", stage.ID,
		"name", stage.Name,
	)

	return nil
}

// Reorder updates the sort order of all pipeline stages
// The stageIDs slice must contain all stage IDs exactly once
func (s *Service) Reorder(ctx context.Context, stageIDs []uuid.UUID) error {
	// Validate that all stages are provided
	count, err := s.repo.CountStages(ctx)
	if err != nil {
		return err
	}
	if len(stageIDs) != count {
		return ErrInvalidReorder
	}

	// Validate no duplicates
	seen := make(map[uuid.UUID]bool)
	for _, id := range stageIDs {
		if seen[id] {
			return ErrInvalidReorder
		}
		seen[id] = true

		// Verify stage exists
		_, getErr := s.repo.GetByID(ctx, id)
		if getErr != nil {
			return getErr
		}
	}

	if reorderErr := s.repo.Reorder(ctx, stageIDs); reorderErr != nil {
		return reorderErr
	}

	slog.Info("pipeline stages reordered", "count", len(stageIDs))

	return nil
}
