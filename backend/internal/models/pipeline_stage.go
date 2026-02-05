package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PipelineStage represents a stage in the sales pipeline
type PipelineStage struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Color       string          `json:"color"`
	SortOrder   int             `json:"sort_order"`
	IsWon       bool            `json:"is_won"`
	IsLost      bool            `json:"is_lost"`
	Probability decimal.Decimal `json:"probability"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// PipelineStageWithStats includes deal statistics for dashboard views
type PipelineStageWithStats struct {
	PipelineStage
	DealCount  int             `json:"deal_count"`
	TotalValue decimal.Decimal `json:"total_value"`
}
