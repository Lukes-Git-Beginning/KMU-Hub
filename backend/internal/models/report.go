package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PipelineStageValue represents aggregated deal data for a pipeline stage.
// Monetary fields use decimal (ADR-0007); the gRPC layer converts to the proto
// double representation at the wire boundary.
type PipelineStageValue struct {
	StageID       uuid.UUID       `json:"stage_id"`
	StageName     string          `json:"stage_name"`
	DealCount     int32           `json:"deal_count"`
	TotalValue    decimal.Decimal `json:"total_value"`
	WeightedValue decimal.Decimal `json:"weighted_value"`
}

// PipelineReport represents a pipeline report with aggregated deal data
type PipelineReport struct {
	Stages             []*PipelineStageValue `json:"stages"`
	TotalDeals         int32                 `json:"total_deals"`
	TotalValue         decimal.Decimal       `json:"total_value"`
	TotalWeightedValue decimal.Decimal       `json:"total_weighted_value"`
}

// ConversionMetric represents stage-to-stage conversion data
type ConversionMetric struct {
	FromStage      string  `json:"from_stage"`
	ToStage        string  `json:"to_stage"`
	ConvertedCount int32   `json:"converted_count"`
	ConversionRate float64 `json:"conversion_rate"`
	AverageDays    float64 `json:"average_days"`
}

// ConversionReport represents a deal conversion report
type ConversionReport struct {
	Metrics          []*ConversionMetric `json:"metrics"`
	OverallWinRate   float64             `json:"overall_win_rate"`
	AverageDealCycle float64             `json:"average_deal_cycle"`
}

// ActivityMetric represents aggregated activity data by type
type ActivityMetric struct {
	ActivityType   string `json:"activity_type"`
	TotalCount     int32  `json:"total_count"`
	CompletedCount int32  `json:"completed_count"`
	OverdueCount   int32  `json:"overdue_count"`
}

// ActivityReport represents an activity report with metrics
type ActivityReport struct {
	Metrics         []*ActivityMetric `json:"metrics"`
	TotalActivities int32             `json:"total_activities"`
	CompletionRate  float64           `json:"completion_rate"`
}
