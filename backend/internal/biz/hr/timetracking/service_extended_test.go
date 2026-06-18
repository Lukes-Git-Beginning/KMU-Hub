package timetracking

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Mock extended repositories
// ============================================================================

type mockTimeCategoryRepo struct {
	categories map[uuid.UUID]*models.HRTimeCategory
}

func newMockTimeCategoryRepo() *mockTimeCategoryRepo {
	return &mockTimeCategoryRepo{categories: make(map[uuid.UUID]*models.HRTimeCategory)}
}

func (m *mockTimeCategoryRepo) List(_ context.Context, _ uuid.UUID) ([]*models.HRTimeCategory, error) {
	var result []*models.HRTimeCategory
	for _, c := range m.categories {
		result = append(result, c)
	}
	return result, nil
}

func (m *mockTimeCategoryRepo) GetByID(_ context.Context, id uuid.UUID) (*models.HRTimeCategory, error) {
	c, ok := m.categories[id]
	if !ok {
		return nil, ErrTimeCategoryNotFound
	}
	return c, nil
}

func (m *mockTimeCategoryRepo) Create(_ context.Context, cat *models.HRTimeCategory) error {
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockTimeCategoryRepo) Update(_ context.Context, cat *models.HRTimeCategory) error {
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockTimeCategoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.categories, id)
	return nil
}

type mockTimeTemplateRepo struct {
	templates map[uuid.UUID]*models.HRTimeTemplate
}

func newMockTimeTemplateRepo() *mockTimeTemplateRepo {
	return &mockTimeTemplateRepo{templates: make(map[uuid.UUID]*models.HRTimeTemplate)}
}

func (m *mockTimeTemplateRepo) List(_ context.Context, _ uuid.UUID) ([]*models.HRTimeTemplate, error) {
	var result []*models.HRTimeTemplate
	for _, t := range m.templates {
		result = append(result, t)
	}
	return result, nil
}

func (m *mockTimeTemplateRepo) Create(_ context.Context, t *models.HRTimeTemplate) error {
	m.templates[t.ID] = t
	return nil
}

func (m *mockTimeTemplateRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.templates, id)
	return nil
}

type mockTimeProjectRepo struct {
	projects map[uuid.UUID]*models.HRTimeProject
}

func newMockTimeProjectRepo() *mockTimeProjectRepo {
	return &mockTimeProjectRepo{projects: make(map[uuid.UUID]*models.HRTimeProject)}
}

func (m *mockTimeProjectRepo) List(_ context.Context, _ uuid.UUID) ([]*models.HRTimeProject, error) {
	var result []*models.HRTimeProject
	for _, p := range m.projects {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockTimeProjectRepo) Create(_ context.Context, p *models.HRTimeProject) error {
	m.projects[p.ID] = p
	return nil
}

type mockWeekApprovalRepo struct {
	approvals map[string]*models.HRWeekApproval // key: "employeeID:weekStart"
}

func newMockWeekApprovalRepo() *mockWeekApprovalRepo {
	return &mockWeekApprovalRepo{approvals: make(map[string]*models.HRWeekApproval)}
}

func (m *mockWeekApprovalRepo) GetByEmployeeWeek(_ context.Context, _ uuid.UUID, employeeID uuid.UUID, weekStart time.Time) (*models.HRWeekApproval, error) {
	key := employeeID.String() + ":" + weekStart.Format("2006-01-02")
	w, ok := m.approvals[key]
	if !ok {
		return nil, ErrWeekApprovalNotFound
	}
	return w, nil
}

func (m *mockWeekApprovalRepo) Upsert(_ context.Context, w *models.HRWeekApproval) error {
	key := w.EmployeeID.String() + ":" + w.WeekStart.Format("2006-01-02")
	m.approvals[key] = w
	return nil
}

func (m *mockWeekApprovalRepo) ListByWeek(_ context.Context, _ uuid.UUID, _ time.Time) ([]*models.HRWeekApproval, error) {
	var result []*models.HRWeekApproval
	for _, w := range m.approvals {
		result = append(result, w)
	}
	return result, nil
}

// ============================================================================
// Helper to build extended test service
// ============================================================================

func newExtendedTestService() (*Service, *mockWorkTimeRepo) {
	svc, workRepo, _ := newTestService()
	catRepo := newMockTimeCategoryRepo()
	tplRepo := newMockTimeTemplateRepo()
	projRepo := newMockTimeProjectRepo()
	waRepo := newMockWeekApprovalRepo()
	svc.SetExtendedRepos(catRepo, tplRepo, projRepo, waRepo)
	return svc, workRepo
}

// ============================================================================
// Tests: CreateManualEntry
// ============================================================================

func TestCreateManualEntry_HappyPath(t *testing.T) {
	svc, workRepo := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()

	clockIn := time.Now().Add(-2 * time.Hour)
	clockOut := time.Now()

	input := ManualEntryInput{
		TenantID:     tenantID,
		EmployeeID:   employeeID,
		ClockIn:      clockIn,
		ClockOut:     clockOut,
		BreakMinutes: 30,
	}

	entry, err := svc.CreateManualEntry(ctx, input)
	require.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, models.WorkTimeStatusCompleted, entry.Status)
	assert.NotNil(t, entry.NetWorkMinutes)
	// 120 min total - 30 break = 90 net
	assert.Equal(t, 90, *entry.NetWorkMinutes)

	// Verify stored
	_, ok := workRepo.entries[entry.ID]
	assert.True(t, ok)
}

func TestCreateManualEntry_ClockOutBeforeClockIn_ReturnsError(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()

	input := ManualEntryInput{
		TenantID:   uuid.New(),
		EmployeeID: uuid.New(),
		ClockIn:    time.Now(),
		ClockOut:   time.Now().Add(-1 * time.Hour), // clock_out before clock_in
	}

	_, err := svc.CreateManualEntry(ctx, input)
	assert.ErrorIs(t, err, ErrInvalidManualEntry)
}

// ============================================================================
// Tests: Time Categories
// ============================================================================

func TestCreateTimeCategory_HappyPath(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()

	cat, err := svc.CreateTimeCategory(ctx, tenantID, "Meeting", "#3b82f6", "Users", false)
	require.NoError(t, err)
	assert.NotNil(t, cat)
	assert.Equal(t, "Meeting", cat.Name)
	assert.Equal(t, tenantID, cat.TenantID)
}

func TestDeleteTimeCategory_RemovesCategory(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()

	cat, err := svc.CreateTimeCategory(ctx, tenantID, "To Delete", "#000", "X", false)
	require.NoError(t, err)

	err = svc.DeleteTimeCategory(ctx, cat.ID)
	require.NoError(t, err)

	cats, listErr := svc.ListTimeCategories(ctx, tenantID)
	require.NoError(t, listErr)
	assert.Empty(t, cats)
}

// ============================================================================
// Tests: Time Templates
// ============================================================================

func TestCreateTimeTemplate_HappyPath(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()

	t1, err := svc.CreateTimeTemplate(ctx, tenantID, "Stand-up", nil, "Daily stand-up meeting", 15)
	require.NoError(t, err)
	assert.Equal(t, "Stand-up", t1.Name)
	assert.Equal(t, 15, t1.EstimatedMinutes)
}

// ============================================================================
// Tests: Time Projects
// ============================================================================

func TestCreateTimeProject_HappyPath(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()

	p, err := svc.CreateTimeProject(ctx, tenantID, "Website Relaunch", "ACME GmbH", "#22c55e", true)
	require.NoError(t, err)
	assert.Equal(t, "Website Relaunch", p.Name)
	assert.Equal(t, "ACME GmbH", p.CustomerName)
	assert.True(t, p.BillableDefault)
}

// ============================================================================
// Tests: Week Approvals
// ============================================================================

func TestSubmitWeek_HappyPath(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	weekStart := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) // Monday

	wa, err := svc.SubmitWeek(ctx, tenantID, employeeID, weekStart)
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalSubmitted, wa.Status)
	assert.NotNil(t, wa.SubmittedAt)
}

func TestSubmitWeek_AlreadySubmitted_ReturnsError(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	weekStart := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) // Monday

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, weekStart)
	require.NoError(t, err)

	_, err = svc.SubmitWeek(ctx, tenantID, employeeID, weekStart)
	assert.ErrorIs(t, err, ErrWeekAlreadySubmitted)
}

func TestApproveWeek_HappyPath(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()
	weekStart := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) // Monday

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, weekStart)
	require.NoError(t, err)

	wa, err := svc.ApproveWeek(ctx, tenantID, approverID, employeeID, weekStart)
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalApproved, wa.Status)
	assert.NotNil(t, wa.ApprovedBy)
	assert.Equal(t, approverID, *wa.ApprovedBy)
}

func TestApproveWeek_NotSubmitted_ReturnsError(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()
	// June 15 2026 is a Monday
	weekStart := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	// Pre-seed with open status — must use the same Monday key that ApproveWeek will compute
	wa := &models.HRWeekApproval{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EmployeeID: employeeID,
		WeekStart:  weekStart,
		Status:     models.WeekApprovalOpen,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	_ = svc.weekApprovalRepo.Upsert(ctx, wa)

	_, err := svc.ApproveWeek(ctx, tenantID, approverID, employeeID, weekStart)
	assert.ErrorIs(t, err, ErrWeekNotSubmitted)
}

func TestRejectWeek_HappyPath(t *testing.T) {
	svc, _ := newExtendedTestService()
	ctx := context.Background()
	tenantID := uuid.New()
	employeeID := uuid.New()
	approverID := uuid.New()
	weekStart := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) // Monday

	_, err := svc.SubmitWeek(ctx, tenantID, employeeID, weekStart)
	require.NoError(t, err)

	wa, err := svc.RejectWeek(ctx, tenantID, approverID, employeeID, weekStart, "Missing project codes")
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalRejected, wa.Status)
	require.NotNil(t, wa.RejectionReason)
	assert.Equal(t, "Missing project codes", *wa.RejectionReason)
}

// ============================================================================
// Tests: Tenant isolation — manual entries stay in tenant scope
// ============================================================================

func TestCreateManualEntry_TenantIsolation(t *testing.T) {
	svc, workRepo := newExtendedTestService()
	ctx := context.Background()
	tenantA := uuid.New()
	tenantB := uuid.New()
	empA := uuid.New()
	empB := uuid.New()

	now := time.Now()
	for _, tc := range []struct {
		tenant uuid.UUID
		emp    uuid.UUID
	}{
		{tenantA, empA},
		{tenantB, empB},
	} {
		_, err := svc.CreateManualEntry(ctx, ManualEntryInput{
			TenantID:   tc.tenant,
			EmployeeID: tc.emp,
			ClockIn:    now.Add(-1 * time.Hour),
			ClockOut:   now,
		})
		require.NoError(t, err)
	}

	// Each tenant has exactly 1 entry in the store
	var countA, countB int
	for _, e := range workRepo.entries {
		switch e.TenantID {
		case tenantA:
			countA++
		case tenantB:
			countB++
		}
	}
	assert.Equal(t, 1, countA)
	assert.Equal(t, 1, countB)
}
