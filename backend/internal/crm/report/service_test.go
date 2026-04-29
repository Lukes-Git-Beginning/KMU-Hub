package report

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	pipelineReport   *models.PipelineReport
	conversionReport *models.ConversionReport
	activityReport   *models.ActivityReport
	pipelineErr      error
	conversionErr    error
	activityErr      error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{}
}

func (m *MockRepository) GetPipelineReport(ctx context.Context, filter PipelineFilter) (*models.PipelineReport, error) {
	if m.pipelineErr != nil {
		return nil, m.pipelineErr
	}
	if m.pipelineReport != nil {
		return m.pipelineReport, nil
	}
	return &models.PipelineReport{
		Stages: []*models.PipelineStageValue{
			{StageID: uuid.New(), StageName: "New", DealCount: 5, TotalValue: 50000, WeightedValue: 5000},
			{StageID: uuid.New(), StageName: "Qualified", DealCount: 3, TotalValue: 30000, WeightedValue: 9000},
			{StageID: uuid.New(), StageName: "Won", DealCount: 2, TotalValue: 20000, WeightedValue: 20000},
		},
		TotalDeals:         10,
		TotalValue:         100000,
		TotalWeightedValue: 34000,
	}, nil
}

func (m *MockRepository) GetConversionReport(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*models.ConversionReport, error) {
	if m.conversionErr != nil {
		return nil, m.conversionErr
	}
	if m.conversionReport != nil {
		return m.conversionReport, nil
	}
	return &models.ConversionReport{
		Metrics: []*models.ConversionMetric{
			{FromStage: "New", ToStage: "Qualified", ConvertedCount: 8, ConversionRate: 60, AverageDays: 5.5},
			{FromStage: "Qualified", ToStage: "Won", ConvertedCount: 4, ConversionRate: 30, AverageDays: 14.2},
		},
		OverallWinRate:   25.0,
		AverageDealCycle: 21.3,
	}, nil
}

func (m *MockRepository) GetActivityReport(ctx context.Context, filter ActivityFilter) (*models.ActivityReport, error) {
	if m.activityErr != nil {
		return nil, m.activityErr
	}
	if m.activityReport != nil {
		return m.activityReport, nil
	}
	return &models.ActivityReport{
		Metrics: []*models.ActivityMetric{
			{ActivityType: "call", TotalCount: 20, CompletedCount: 18, OverdueCount: 0},
			{ActivityType: "meeting", TotalCount: 10, CompletedCount: 8, OverdueCount: 1},
			{ActivityType: "task", TotalCount: 15, CompletedCount: 10, OverdueCount: 3},
		},
		TotalActivities: 45,
		CompletionRate:  80.0,
	}, nil
}

// ============================================================================
// Pipeline Report Tests
// ============================================================================

var testTenantID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

func TestService_GetPipelineReport_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := PipelineReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now().AddDate(0, -1, 0),
		EndDate:   time.Now(),
	}

	report, err := svc.GetPipelineReport(context.Background(), input)

	require.NoError(t, err)
	assert.Len(t, report.Stages, 3)
	assert.Equal(t, int32(10), report.TotalDeals)
	assert.Equal(t, 100000.0, report.TotalValue)
	assert.Equal(t, 34000.0, report.TotalWeightedValue)
}

func TestService_GetPipelineReport_WithOwnerFilter(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	input := PipelineReportInput{
		TenantID:  testTenantID,
		OwnerID:   &ownerID,
		StartDate: time.Now().AddDate(0, -1, 0),
		EndDate:   time.Now(),
	}

	report, err := svc.GetPipelineReport(context.Background(), input)

	require.NoError(t, err)
	assert.NotNil(t, report)
}

func TestService_GetPipelineReport_StartDateRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := PipelineReportInput{
		TenantID: testTenantID,
		EndDate:  time.Now(),
	}

	_, err := svc.GetPipelineReport(context.Background(), input)

	assert.ErrorIs(t, err, ErrStartDateRequired)
}

func TestService_GetPipelineReport_EndDateRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := PipelineReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now(),
	}

	_, err := svc.GetPipelineReport(context.Background(), input)

	assert.ErrorIs(t, err, ErrEndDateRequired)
}

func TestService_GetPipelineReport_InvalidDateRange(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := PipelineReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, -1, 0), // Before start
	}

	_, err := svc.GetPipelineReport(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidDateRange)
}

func TestService_GetPipelineReport_EmptyReport(t *testing.T) {
	repo := NewMockRepository()
	repo.pipelineReport = &models.PipelineReport{
		Stages:             []*models.PipelineStageValue{},
		TotalDeals:         0,
		TotalValue:         0,
		TotalWeightedValue: 0,
	}
	svc := NewService(repo)

	input := PipelineReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now().AddDate(-1, 0, 0),
		EndDate:   time.Now(),
	}

	report, err := svc.GetPipelineReport(context.Background(), input)

	require.NoError(t, err)
	assert.Empty(t, report.Stages)
	assert.Equal(t, int32(0), report.TotalDeals)
}

// ============================================================================
// Conversion Report Tests
// ============================================================================

func TestService_GetConversionReport_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := ConversionReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now().AddDate(0, -1, 0),
		EndDate:   time.Now(),
	}

	report, err := svc.GetConversionReport(context.Background(), input)

	require.NoError(t, err)
	assert.Len(t, report.Metrics, 2)
	assert.Equal(t, 25.0, report.OverallWinRate)
	assert.Equal(t, 21.3, report.AverageDealCycle)
}

func TestService_GetConversionReport_StartDateRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := ConversionReportInput{
		TenantID: testTenantID,
		EndDate:  time.Now(),
	}

	_, err := svc.GetConversionReport(context.Background(), input)

	assert.ErrorIs(t, err, ErrStartDateRequired)
}

func TestService_GetConversionReport_EndDateRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := ConversionReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now(),
	}

	_, err := svc.GetConversionReport(context.Background(), input)

	assert.ErrorIs(t, err, ErrEndDateRequired)
}

func TestService_GetConversionReport_InvalidDateRange(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := ConversionReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, -1, 0), // Before start
	}

	_, err := svc.GetConversionReport(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidDateRange)
}

func TestService_GetConversionReport_EmptyReport(t *testing.T) {
	repo := NewMockRepository()
	repo.conversionReport = &models.ConversionReport{
		Metrics:          []*models.ConversionMetric{},
		OverallWinRate:   0,
		AverageDealCycle: 0,
	}
	svc := NewService(repo)

	input := ConversionReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now().AddDate(-1, 0, 0),
		EndDate:   time.Now(),
	}

	report, err := svc.GetConversionReport(context.Background(), input)

	require.NoError(t, err)
	assert.Empty(t, report.Metrics)
	assert.Equal(t, 0.0, report.OverallWinRate)
}

// ============================================================================
// Activity Report Tests
// ============================================================================

func TestService_GetActivityReport_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := ActivityReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now().AddDate(0, -1, 0),
		EndDate:   time.Now(),
	}

	report, err := svc.GetActivityReport(context.Background(), input)

	require.NoError(t, err)
	assert.Len(t, report.Metrics, 3)
	assert.Equal(t, int32(45), report.TotalActivities)
	assert.Equal(t, 80.0, report.CompletionRate)
}

func TestService_GetActivityReport_WithUserFilter(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	input := ActivityReportInput{
		TenantID:  testTenantID,
		UserID:    &userID,
		StartDate: time.Now().AddDate(0, -1, 0),
		EndDate:   time.Now(),
	}

	report, err := svc.GetActivityReport(context.Background(), input)

	require.NoError(t, err)
	assert.NotNil(t, report)
}

func TestService_GetActivityReport_StartDateRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := ActivityReportInput{
		TenantID: testTenantID,
		EndDate:  time.Now(),
	}

	_, err := svc.GetActivityReport(context.Background(), input)

	assert.ErrorIs(t, err, ErrStartDateRequired)
}

func TestService_GetActivityReport_EndDateRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := ActivityReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now(),
	}

	_, err := svc.GetActivityReport(context.Background(), input)

	assert.ErrorIs(t, err, ErrEndDateRequired)
}

func TestService_GetActivityReport_InvalidDateRange(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := ActivityReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, -1, 0), // Before start
	}

	_, err := svc.GetActivityReport(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidDateRange)
}

func TestService_GetActivityReport_EmptyReport(t *testing.T) {
	repo := NewMockRepository()
	repo.activityReport = &models.ActivityReport{
		Metrics:         []*models.ActivityMetric{},
		TotalActivities: 0,
		CompletionRate:  0,
	}
	svc := NewService(repo)

	input := ActivityReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now().AddDate(-1, 0, 0),
		EndDate:   time.Now(),
	}

	report, err := svc.GetActivityReport(context.Background(), input)

	require.NoError(t, err)
	assert.Empty(t, report.Metrics)
	assert.Equal(t, int32(0), report.TotalActivities)
	assert.Equal(t, 0.0, report.CompletionRate)
}

func TestService_GetActivityReport_ChecksAllMetricTypes(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := ActivityReportInput{
		TenantID:  testTenantID,
		StartDate: time.Now().AddDate(0, -1, 0),
		EndDate:   time.Now(),
	}

	report, err := svc.GetActivityReport(context.Background(), input)

	require.NoError(t, err)

	// Verify metrics contain expected types
	typeSet := make(map[string]bool)
	for _, m := range report.Metrics {
		typeSet[m.ActivityType] = true
		// Each metric should have valid counts
		assert.GreaterOrEqual(t, m.TotalCount, int32(0))
		assert.GreaterOrEqual(t, m.CompletedCount, int32(0))
		assert.GreaterOrEqual(t, m.OverdueCount, int32(0))
		assert.LessOrEqual(t, m.CompletedCount, m.TotalCount)
	}

	assert.Contains(t, typeSet, "call")
	assert.Contains(t, typeSet, "meeting")
	assert.Contains(t, typeSet, "task")
}
