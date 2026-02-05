package report

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles report business logic
type Service struct {
	repo Repository
}

// NewService creates a new report service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// PipelineReportInput contains input parameters for pipeline report
type PipelineReportInput struct {
	OwnerID   *uuid.UUID
	StartDate time.Time
	EndDate   time.Time
}

// GetPipelineReport generates a pipeline report
func (s *Service) GetPipelineReport(ctx context.Context, input PipelineReportInput) (*models.PipelineReport, error) {
	if input.StartDate.IsZero() {
		return nil, ErrStartDateRequired
	}
	if input.EndDate.IsZero() {
		return nil, ErrEndDateRequired
	}
	if input.StartDate.After(input.EndDate) {
		return nil, ErrInvalidDateRange
	}

	report, err := s.repo.GetPipelineReport(ctx, PipelineFilter{
		OwnerID:   input.OwnerID,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
	})
	if err != nil {
		return nil, err
	}

	slog.Info("pipeline report generated",
		"start_date", input.StartDate,
		"end_date", input.EndDate,
		"total_deals", report.TotalDeals,
	)

	return report, nil
}

// ConversionReportInput contains input parameters for conversion report
type ConversionReportInput struct {
	StartDate time.Time
	EndDate   time.Time
}

// GetConversionReport generates a conversion report
func (s *Service) GetConversionReport(ctx context.Context, input ConversionReportInput) (*models.ConversionReport, error) {
	if input.StartDate.IsZero() {
		return nil, ErrStartDateRequired
	}
	if input.EndDate.IsZero() {
		return nil, ErrEndDateRequired
	}
	if input.StartDate.After(input.EndDate) {
		return nil, ErrInvalidDateRange
	}

	report, err := s.repo.GetConversionReport(ctx, input.StartDate, input.EndDate)
	if err != nil {
		return nil, err
	}

	slog.Info("conversion report generated",
		"start_date", input.StartDate,
		"end_date", input.EndDate,
		"metrics_count", len(report.Metrics),
	)

	return report, nil
}

// ActivityReportInput contains input parameters for activity report
type ActivityReportInput struct {
	UserID    *uuid.UUID
	StartDate time.Time
	EndDate   time.Time
}

// GetActivityReport generates an activity report
func (s *Service) GetActivityReport(ctx context.Context, input ActivityReportInput) (*models.ActivityReport, error) {
	if input.StartDate.IsZero() {
		return nil, ErrStartDateRequired
	}
	if input.EndDate.IsZero() {
		return nil, ErrEndDateRequired
	}
	if input.StartDate.After(input.EndDate) {
		return nil, ErrInvalidDateRange
	}

	report, err := s.repo.GetActivityReport(ctx, ActivityFilter{
		UserID:    input.UserID,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
	})
	if err != nil {
		return nil, err
	}

	slog.Info("activity report generated",
		"start_date", input.StartDate,
		"end_date", input.EndDate,
		"total_activities", report.TotalActivities,
	)

	return report, nil
}
