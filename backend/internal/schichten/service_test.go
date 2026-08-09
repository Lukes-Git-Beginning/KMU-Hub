package schichten

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// MockRepository
// ============================================================================

type mockRepository struct {
	shifts       map[uuid.UUID]*Shift
	assignments  map[string]*ShiftAssignment // key: shiftID+":"+employeeID
	templates    map[uuid.UUID]*ShiftTemplate
	swapRequests map[uuid.UUID]*SwapRequest

	// per-employee latest shift end (for ArbZG check)
	employeeLatestEnd map[uuid.UUID]*time.Time

	// per-(tenant, employee) minor flag (for the JArbSchG check)
	minorEmployees map[string]bool

	createShiftErr      error
	createAssignmentErr error
	isMinorErr          error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		shifts:            make(map[uuid.UUID]*Shift),
		assignments:       make(map[string]*ShiftAssignment),
		templates:         make(map[uuid.UUID]*ShiftTemplate),
		swapRequests:      make(map[uuid.UUID]*SwapRequest),
		employeeLatestEnd: make(map[uuid.UUID]*time.Time),
		minorEmployees:    make(map[string]bool),
	}
}

func assignKey(shiftID, employeeID uuid.UUID) string {
	return shiftID.String() + ":" + employeeID.String()
}

// -- Shifts --

func (m *mockRepository) CreateShift(ctx context.Context, shift *Shift) error {
	if m.createShiftErr != nil {
		return m.createShiftErr
	}
	m.shifts[shift.ID] = shift
	return nil
}

func (m *mockRepository) UpdateShift(ctx context.Context, shift *Shift) error {
	if _, ok := m.shifts[shift.ID]; !ok || m.shifts[shift.ID].TenantID != shift.TenantID {
		return ErrShiftNotFound
	}
	m.shifts[shift.ID] = shift
	return nil
}

func (m *mockRepository) DeleteShift(ctx context.Context, tenantID, shiftID uuid.UUID) error {
	s, ok := m.shifts[shiftID]
	if !ok || s.TenantID != tenantID {
		return ErrShiftNotFound
	}
	delete(m.shifts, shiftID)
	return nil
}

func (m *mockRepository) GetShift(ctx context.Context, tenantID, shiftID uuid.UUID) (*Shift, error) {
	s, ok := m.shifts[shiftID]
	if !ok || s.TenantID != tenantID {
		return nil, ErrShiftNotFound
	}
	return s, nil
}

func (m *mockRepository) ListShifts(ctx context.Context, tenantID uuid.UUID, filter ListShiftsFilter, offset, limit int) ([]*Shift, int, error) {
	var result []*Shift
	for _, s := range m.shifts {
		if s.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && s.Status != *filter.Status {
			continue
		}
		if filter.From != nil && s.StartTime.Before(*filter.From) {
			continue
		}
		if filter.To != nil && s.EndTime.After(*filter.To) {
			continue
		}
		result = append(result, s)
	}
	total := len(result)
	if offset >= total {
		return []*Shift{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) PublishShifts(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (int64, error) {
	var count int64
	for _, s := range m.shifts {
		if s.TenantID != tenantID {
			continue
		}
		if s.Status == ShiftStatusDraft && !s.StartTime.Before(from) && !s.EndTime.After(to) {
			s.Status = ShiftStatusPublished
			count++
		}
	}
	return count, nil
}

// -- Assignments --

func (m *mockRepository) CreateAssignment(ctx context.Context, a *ShiftAssignment) error {
	if m.createAssignmentErr != nil {
		return m.createAssignmentErr
	}
	m.assignments[assignKey(a.ShiftID, a.EmployeeID)] = a
	return nil
}

func (m *mockRepository) DeleteAssignment(ctx context.Context, tenantID, shiftID, employeeID uuid.UUID) error {
	key := assignKey(shiftID, employeeID)
	if _, ok := m.assignments[key]; !ok {
		return ErrAssignmentNotFound
	}
	delete(m.assignments, key)
	return nil
}

func (m *mockRepository) GetAssignment(ctx context.Context, tenantID, shiftID, employeeID uuid.UUID) (*ShiftAssignment, error) {
	key := assignKey(shiftID, employeeID)
	a, ok := m.assignments[key]
	if !ok {
		return nil, ErrAssignmentNotFound
	}
	return a, nil
}

func (m *mockRepository) ListAssignments(ctx context.Context, tenantID, shiftID uuid.UUID) ([]*ShiftAssignment, error) {
	var result []*ShiftAssignment
	for _, a := range m.assignments {
		if a.ShiftID == shiftID && a.TenantID == tenantID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockRepository) CountAssignments(ctx context.Context, tenantID, shiftID uuid.UUID) (int, error) {
	count := 0
	for _, a := range m.assignments {
		if a.ShiftID == shiftID && a.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *mockRepository) LatestShiftEndBeforeForEmployee(ctx context.Context, tenantID, employeeID uuid.UUID, before time.Time) (*time.Time, error) {
	if t, ok := m.employeeLatestEnd[employeeID]; ok {
		return t, nil
	}
	return nil, nil
}

func (m *mockRepository) EarliestShiftStartAfterForEmployee(ctx context.Context, tenantID, employeeID uuid.UUID, after time.Time) (*time.Time, error) {
	// Find the earliest assigned shift that starts after 'after'
	var earliest *time.Time
	for _, a := range m.assignments {
		if a.TenantID != tenantID || a.EmployeeID != employeeID {
			continue
		}
		s, ok := m.shifts[a.ShiftID]
		if !ok {
			continue
		}
		if s.StartTime.After(after) {
			if earliest == nil || s.StartTime.Before(*earliest) {
				t := s.StartTime
				earliest = &t
			}
		}
	}
	return earliest, nil
}

func (m *mockRepository) IsMinorEmployee(ctx context.Context, tenantID, employeeID uuid.UUID) (bool, error) {
	if m.isMinorErr != nil {
		return false, m.isMinorErr
	}
	return m.minorEmployees[minorKey(tenantID, employeeID)], nil
}

func minorKey(tenantID, employeeID uuid.UUID) string {
	return tenantID.String() + ":" + employeeID.String()
}

func (m *mockRepository) ShiftExistsForTemplate(ctx context.Context, tenantID uuid.UUID, startTime, endTime time.Time, title string) (bool, error) {
	for _, s := range m.shifts {
		if s.TenantID == tenantID && s.StartTime.Equal(startTime) && s.EndTime.Equal(endTime) && s.Title == title {
			return true, nil
		}
	}
	return false, nil
}

// -- Templates --

func (m *mockRepository) CreateTemplate(ctx context.Context, t *ShiftTemplate) error {
	m.templates[t.ID] = t
	return nil
}

func (m *mockRepository) UpdateTemplate(ctx context.Context, t *ShiftTemplate) error {
	if _, ok := m.templates[t.ID]; !ok {
		return ErrTemplateNotFound
	}
	m.templates[t.ID] = t
	return nil
}

func (m *mockRepository) DeleteTemplate(ctx context.Context, tenantID, templateID uuid.UUID) error {
	t, ok := m.templates[templateID]
	if !ok || t.TenantID != tenantID {
		return ErrTemplateNotFound
	}
	delete(m.templates, templateID)
	return nil
}

func (m *mockRepository) GetTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*ShiftTemplate, error) {
	t, ok := m.templates[templateID]
	if !ok || t.TenantID != tenantID {
		return nil, ErrTemplateNotFound
	}
	return t, nil
}

func (m *mockRepository) ListTemplates(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*ShiftTemplate, int, error) {
	var result []*ShiftTemplate
	for _, t := range m.templates {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	total := len(result)
	if offset >= total {
		return []*ShiftTemplate{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) GetStats(ctx context.Context, tenantID uuid.UUID, from, to *time.Time) (*ShiftStats, error) {
	stats := &ShiftStats{}
	for _, s := range m.shifts {
		if s.TenantID != tenantID {
			continue
		}
		stats.TotalShifts++
		if s.Status == ShiftStatusPublished {
			stats.PublishedShifts++
		} else {
			stats.DraftShifts++
		}
	}
	return stats, nil
}

// -- SwapRequests --

func (m *mockRepository) CreateSwapRequest(ctx context.Context, req *SwapRequest) error {
	// Idempotency: check if key already exists.
	for _, existing := range m.swapRequests {
		if existing.IdempotencyKey == req.IdempotencyKey && existing.TenantID == req.TenantID {
			*req = *existing
			return nil
		}
	}
	m.swapRequests[req.ID] = req
	return nil
}

func (m *mockRepository) GetSwapRequest(ctx context.Context, tenantID, requestID uuid.UUID) (*SwapRequest, error) {
	req, ok := m.swapRequests[requestID]
	if !ok || req.TenantID != tenantID {
		return nil, ErrSwapRequestNotFound
	}
	return req, nil
}

func (m *mockRepository) ListSwapRequests(ctx context.Context, tenantID uuid.UUID, filter SwapRequestFilter, offset, limit int) ([]*SwapRequest, int, error) {
	var result []*SwapRequest
	for _, req := range m.swapRequests {
		if req.TenantID != tenantID {
			continue
		}
		if filter.ShiftID != nil && req.ShiftID != *filter.ShiftID {
			continue
		}
		if filter.Status != nil && req.Status != *filter.Status {
			continue
		}
		result = append(result, req)
	}
	total := len(result)
	if offset >= total {
		return []*SwapRequest{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) UpdateSwapRequestStatus(ctx context.Context, tenantID, requestID uuid.UUID, status SwapRequestStatus) error {
	req, ok := m.swapRequests[requestID]
	if !ok || req.TenantID != tenantID {
		return ErrSwapRequestNotFound
	}
	req.Status = status
	req.UpdatedAt = time.Now()
	return nil
}

func (m *mockRepository) SwapAssignmentsForRequest(ctx context.Context, req *SwapRequest) error {
	// Swap employee_id on the requester's assignment.
	assignKey1 := assignKey(req.ShiftID, req.RequestedByEmployeeID)
	assignKey2 := assignKey(req.ShiftID, req.SwapWithEmployeeID)

	a1, ok1 := m.assignments[assignKey1]
	a2, ok2 := m.assignments[assignKey2]

	// Best-effort: update what exists (mirrors DB ON CONFLICT DO NOTHING style).
	if ok1 {
		delete(m.assignments, assignKey1)
		a1.EmployeeID = req.SwapWithEmployeeID
		m.assignments[assignKey(req.ShiftID, req.SwapWithEmployeeID)] = a1
	}
	if ok2 {
		delete(m.assignments, assignKey2)
		a2.EmployeeID = req.RequestedByEmployeeID
		m.assignments[assignKey(req.ShiftID, req.RequestedByEmployeeID)] = a2
	}
	return nil
}

// compile-time interface check
var _ Repository = (*mockRepository)(nil)

// ============================================================================
// Helpers
// ============================================================================

func addShift(repo *mockRepository, tenantID uuid.UUID, title string, start, end time.Time, status ShiftStatus) *Shift {
	s := &Shift{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Title:     title,
		StartTime: start,
		EndTime:   end,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.shifts[s.ID] = s
	return s
}

func addTemplate(repo *mockRepository, tenantID uuid.UUID, name string, dow, startHour, startMin, dur int) *ShiftTemplate {
	t := &ShiftTemplate{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            name,
		DayOfWeek:       dow,
		StartHour:       startHour,
		StartMinute:     startMin,
		DurationMinutes: dur,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	repo.templates[t.ID] = t
	return t
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// ============================================================================
// CreateShift Tests
// ============================================================================

func TestService_CreateShift_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	start := time.Now().Add(24 * time.Hour)
	end := start.Add(8 * time.Hour)

	shift, err := svc.CreateShift(context.Background(), CreateShiftInput{
		TenantID:  tenantID,
		Title:     "Frühschicht",
		StartTime: start,
		EndTime:   end,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, shift.ID)
	assert.Equal(t, ShiftStatusDraft, shift.Status)
	assert.Equal(t, tenantID, shift.TenantID)
}

func TestService_CreateShift_EmptyTitle(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	start := time.Now()
	_, err := svc.CreateShift(context.Background(), CreateShiftInput{
		TenantID:  uuid.New(),
		Title:     "  ",
		StartTime: start,
		EndTime:   start.Add(8 * time.Hour),
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateShift_EndBeforeStart(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	start := time.Now()
	_, err := svc.CreateShift(context.Background(), CreateShiftInput{
		TenantID:  uuid.New(),
		Title:     "Shift",
		StartTime: start,
		EndTime:   start.Add(-1 * time.Hour),
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// UpdateShift Tests
// ============================================================================

func TestService_UpdateShift_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	start := time.Now()
	shift := addShift(repo, tenantID, "Alt", start, start.Add(8*time.Hour), ShiftStatusDraft)

	newTitle := "Neu"
	updated, err := svc.UpdateShift(context.Background(), UpdateShiftInput{
		TenantID: tenantID,
		ShiftID:  shift.ID,
		Title:    &newTitle,
	})

	require.NoError(t, err)
	assert.Equal(t, "Neu", updated.Title)
}

func TestService_UpdateShift_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateShift(context.Background(), UpdateShiftInput{
		TenantID: uuid.New(),
		ShiftID:  uuid.New(),
	})

	assert.ErrorIs(t, err, ErrShiftNotFound)
}

// ============================================================================
// DeleteShift Tests
// ============================================================================

func TestService_DeleteShift_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	start := time.Now()
	shift := addShift(repo, tenantID, "S", start, start.Add(4*time.Hour), ShiftStatusDraft)

	err := svc.DeleteShift(context.Background(), tenantID, shift.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.shifts, shift.ID)
}

func TestService_DeleteShift_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteShift(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrShiftNotFound)
}

// ============================================================================
// PublishShifts Tests
// ============================================================================

func TestService_PublishShifts_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	base := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)

	// 3 draft shifts in window
	s1 := addShift(repo, tenantID, "S1", base, base.Add(8*time.Hour), ShiftStatusDraft)
	s2 := addShift(repo, tenantID, "S2", base.AddDate(0, 0, 1), base.AddDate(0, 0, 1).Add(8*time.Hour), ShiftStatusDraft)
	// already published — should not be re-counted
	_ = addShift(repo, tenantID, "S3", base.AddDate(0, 0, 2), base.AddDate(0, 0, 2).Add(8*time.Hour), ShiftStatusPublished)

	from := base.AddDate(0, -1, 0)
	to := base.AddDate(0, 1, 0)

	count, err := svc.PublishShifts(context.Background(), tenantID, from, to)

	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Equal(t, ShiftStatusPublished, repo.shifts[s1.ID].Status)
	assert.Equal(t, ShiftStatusPublished, repo.shifts[s2.ID].Status)
}

// Guard: idempotent publish — calling twice does not double-count
func TestService_PublishShifts_Idempotent(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	base := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	addShift(repo, tenantID, "S1", base, base.Add(8*time.Hour), ShiftStatusDraft)

	from := base.AddDate(0, -1, 0)
	to := base.AddDate(0, 1, 0)

	count1, err := svc.PublishShifts(context.Background(), tenantID, from, to)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count1)

	// Second call: shift already published — no draft left to publish
	count2, err := svc.PublishShifts(context.Background(), tenantID, from, to)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count2)
}

// ============================================================================
// ArbZG §5 Rest-Period Tests
// ============================================================================

// Happy path: 12h rest → OK
func TestService_ArbZG_HappyPath_12h(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	employeeID := uuid.New()

	prevEnd := mustTime("2026-05-01T22:00:00Z")
	repo.employeeLatestEnd[employeeID] = &prevEnd // prev shift ended 22:00

	// New shift starts 10:00 next day → 12h rest
	newStart := mustTime("2026-05-02T10:00:00Z")
	newEnd := newStart.Add(8 * time.Hour)

	shift := addShift(repo, tenantID, "Schicht", newStart, newEnd, ShiftStatusDraft)

	_, err := svc.AssignEmployee(context.Background(), AssignEmployeeInput{
		TenantID:   tenantID,
		ShiftID:    shift.ID,
		EmployeeID: employeeID,
	})

	require.NoError(t, err)
}

// PublishShifts with invalid range (end not after start) → ErrInvalidInput
func TestService_PublishShifts_InvalidRange(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	base := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	_, err := svc.PublishShifts(context.Background(), uuid.New(), base, base.Add(-time.Hour))
	assert.ErrorIs(t, err, ErrInvalidInput)
}

// GetShift success
func TestService_GetShift_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	start := time.Now()
	shift := addShift(repo, tenantID, "S", start, start.Add(8*time.Hour), ShiftStatusDraft)

	result, err := svc.GetShift(context.Background(), tenantID, shift.ID)
	require.NoError(t, err)
	assert.Equal(t, shift.ID, result.ID)
}

// ListAssignments: shift not found → error
func TestService_ListAssignments_ShiftNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.ListAssignments(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrShiftNotFound)
}

// UpdateTemplate success
func TestService_UpdateTemplate_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	tmpl := addTemplate(repo, tenantID, "Orig", 1, 8, 0, 480)

	newName := "Updated"
	updated, err := svc.UpdateTemplate(context.Background(), UpdateTemplateInput{
		TenantID:   tenantID,
		TemplateID: tmpl.ID,
		Name:       &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
}

// UpdateTemplate not found
func TestService_UpdateTemplate_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateTemplate(context.Background(), UpdateTemplateInput{
		TenantID:   uuid.New(),
		TemplateID: uuid.New(),
	})
	assert.ErrorIs(t, err, ErrTemplateNotFound)
}

// Violation: 10h59m rest → ErrArbzgViolation
func TestService_ArbZG_Violation_10h59m(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	employeeID := uuid.New()

	prevEnd := mustTime("2026-05-01T22:00:00Z")
	repo.employeeLatestEnd[employeeID] = &prevEnd // prev shift ended 22:00

	// New shift starts 08:59 next day → 10h59m rest (< 11h)
	newStart := mustTime("2026-05-02T08:59:00Z")
	newEnd := newStart.Add(8 * time.Hour)

	shift := addShift(repo, tenantID, "Schicht", newStart, newEnd, ShiftStatusDraft)

	_, err := svc.AssignEmployee(context.Background(), AssignEmployeeInput{
		TenantID:   tenantID,
		ShiftID:    shift.ID,
		EmployeeID: employeeID,
	})

	assert.ErrorIs(t, err, ErrArbzgViolation)
}

// Edge case: exactly 11h00m00s rest → OK
func TestService_ArbZG_EdgeCase_ExactlyElevenHours(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	employeeID := uuid.New()

	prevEnd := mustTime("2026-05-01T22:00:00Z")
	repo.employeeLatestEnd[employeeID] = &prevEnd

	// New shift starts exactly 11h later
	newStart := prevEnd.Add(11 * time.Hour) // 09:00:00
	newEnd := newStart.Add(8 * time.Hour)

	shift := addShift(repo, tenantID, "Schicht", newStart, newEnd, ShiftStatusDraft)

	_, err := svc.AssignEmployee(context.Background(), AssignEmployeeInput{
		TenantID:   tenantID,
		ShiftID:    shift.ID,
		EmployeeID: employeeID,
	})

	require.NoError(t, err, "exactly 11h rest must be accepted")
}

// DST Spring-Forward: use Europe/Berlin, clocks go from 02:00 → 03:00
// On the spring-forward night (wall-clock only 23h between 01:00 and 01:00 next day),
// the absolute time distance is still >= 11h.
func TestService_ArbZG_DST_SpringForward(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	// 2026 DST change: last Sunday in March = 2026-03-29 02:00 → 03:00
	// Prev shift ends at 22:00 CET (=21:00 UTC)
	prevEnd := time.Date(2026, 3, 28, 22, 0, 0, 0, loc)

	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	employeeID := uuid.New()
	repo.employeeLatestEnd[employeeID] = &prevEnd

	// New shift starts at 10:00 CEST (=08:00 UTC) the next day.
	// Wall-clock distance: 12h, but DST springs forward → actual elapsed = 11h.
	// 22:00 CET + 11h actual = 09:00 CEST (10:00 local display).
	newStart := time.Date(2026, 3, 29, 10, 0, 0, 0, loc)
	newEnd := newStart.Add(8 * time.Hour)

	shift := addShift(repo, tenantID, "DST-Schicht", newStart, newEnd, ShiftStatusDraft)

	_, err = svc.AssignEmployee(context.Background(), AssignEmployeeInput{
		TenantID:   tenantID,
		ShiftID:    shift.ID,
		EmployeeID: employeeID,
	})

	require.NoError(t, err, "11h actual rest over DST spring-forward must be accepted")
}

// No prior shift → no constraint
func TestService_ArbZG_NoPriorShift(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	employeeID := uuid.New()
	// No entry in employeeLatestEnd → nil returned

	start := time.Now()
	shift := addShift(repo, tenantID, "S", start, start.Add(8*time.Hour), ShiftStatusDraft)

	_, err := svc.AssignEmployee(context.Background(), AssignEmployeeInput{
		TenantID:   tenantID,
		ShiftID:    shift.ID,
		EmployeeID: employeeID,
	})

	require.NoError(t, err)
}

// ============================================================================
// Cross-Tenant Security Tests
// ============================================================================

// Guard: User A (different tenant) tries to read Shift of User B → ErrShiftNotFound
func TestService_GetShift_CrossTenant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantB := uuid.New()
	tenantA := uuid.New()

	start := time.Now()
	shiftB := addShift(repo, tenantB, "TenantB Shift", start, start.Add(8*time.Hour), ShiftStatusDraft)

	// Tenant A tries to read Tenant B's shift
	_, err := svc.GetShift(context.Background(), tenantA, shiftB.ID)

	assert.ErrorIs(t, err, ErrShiftNotFound)
}

// ============================================================================
// Template Tests
// ============================================================================

func TestService_CreateTemplate_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	tmpl, err := svc.CreateTemplate(context.Background(), CreateTemplateInput{
		TenantID:        tenantID,
		Name:            "Mo-Fr Frühschicht",
		DayOfWeek:       1, // Monday
		StartHour:       8,
		StartMinute:     0,
		DurationMinutes: 540, // 9h
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tmpl.ID)
	assert.Equal(t, "Mo-Fr Frühschicht", tmpl.Name)
}

func TestService_CreateTemplate_InvalidDayOfWeek(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateTemplate(context.Background(), CreateTemplateInput{
		TenantID:        uuid.New(),
		Name:            "Bad",
		DayOfWeek:       7, // invalid
		StartHour:       8,
		StartMinute:     0,
		DurationMinutes: 480,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateTemplate_ZeroDuration(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateTemplate(context.Background(), CreateTemplateInput{
		TenantID:        uuid.New(),
		Name:            "Bad",
		DayOfWeek:       1,
		StartHour:       8,
		StartMinute:     0,
		DurationMinutes: 0,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_DeleteTemplate_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteTemplate(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrTemplateNotFound)
}

// ============================================================================
// ApplyTemplate Tests
// ============================================================================

func TestService_ApplyTemplate_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	// Monday=1 template, 08:00, 8h
	tmpl := addTemplate(repo, tenantID, "Montag", 1, 8, 0, 480)

	// Range: 2026-04-27 (Mon) to 2026-05-04 (Mon) → 2 Mondays
	rangeStart := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 5, 4, 23, 59, 0, 0, time.UTC)

	shifts, err := svc.ApplyTemplate(context.Background(), ApplyTemplateInput{
		TenantID:   tenantID,
		TemplateID: tmpl.ID,
		RangeStart: rangeStart,
		RangeEnd:   rangeEnd,
	})

	require.NoError(t, err)
	assert.Len(t, shifts, 2)
	for _, s := range shifts {
		assert.Equal(t, ShiftStatusDraft, s.Status)
	}
}

func TestService_ApplyTemplate_InvalidRange(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	tmpl := addTemplate(repo, tenantID, "T", 1, 8, 0, 480)

	start := time.Now()
	_, err := svc.ApplyTemplate(context.Background(), ApplyTemplateInput{
		TenantID:   tenantID,
		TemplateID: tmpl.ID,
		RangeStart: start,
		RangeEnd:   start.Add(-1 * time.Hour), // end before start
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// UnassignEmployee Tests
// ============================================================================

func TestService_UnassignEmployee_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	employeeID := uuid.New()

	start := time.Now()
	shift := addShift(repo, tenantID, "S", start, start.Add(8*time.Hour), ShiftStatusDraft)

	// Pre-create assignment
	key := assignKey(shift.ID, employeeID)
	repo.assignments[key] = &ShiftAssignment{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ShiftID:    shift.ID,
		EmployeeID: employeeID,
		AssignedAt: time.Now(),
	}

	err := svc.UnassignEmployee(context.Background(), tenantID, shift.ID, employeeID)

	require.NoError(t, err)
	assert.NotContains(t, repo.assignments, key)
}

func TestService_UnassignEmployee_ShiftNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.UnassignEmployee(context.Background(), uuid.New(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrShiftNotFound)
}

// ============================================================================
// GetShiftStats Tests
// ============================================================================

func TestService_GetShiftStats_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	now := time.Now()
	addShift(repo, tenantID, "Draft", now, now.Add(8*time.Hour), ShiftStatusDraft)
	addShift(repo, tenantID, "Pub", now.AddDate(0, 0, 1), now.AddDate(0, 0, 1).Add(8*time.Hour), ShiftStatusPublished)

	stats, err := svc.GetShiftStats(context.Background(), GetShiftStatsInput{TenantID: tenantID})

	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalShifts)
	assert.Equal(t, 1, stats.PublishedShifts)
	assert.Equal(t, 1, stats.DraftShifts)
}

// ============================================================================
// CheckArbzgCompliance Tests
// ============================================================================

func TestService_CheckArbzgCompliance_Compliant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	employeeID := uuid.New()

	prevEnd := mustTime("2026-05-01T20:00:00Z")
	repo.employeeLatestEnd[employeeID] = &prevEnd

	newStart := mustTime("2026-05-02T08:00:00Z") // 12h rest
	compliant, reason, err := svc.CheckArbzgCompliance(context.Background(), tenantID, employeeID, newStart, newStart.Add(8*time.Hour))

	require.NoError(t, err)
	assert.True(t, compliant)
	assert.Empty(t, reason)
}

func TestService_CheckArbzgCompliance_Violation(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	employeeID := uuid.New()

	prevEnd := mustTime("2026-05-01T22:00:00Z")
	repo.employeeLatestEnd[employeeID] = &prevEnd

	newStart := mustTime("2026-05-02T07:00:00Z") // 9h rest — violation
	compliant, reason, err := svc.CheckArbzgCompliance(context.Background(), tenantID, employeeID, newStart, newStart.Add(8*time.Hour))

	require.NoError(t, err)
	assert.False(t, compliant)
	assert.NotEmpty(t, reason)
}

// ============================================================================
// JArbSchG (minor protection) Tests
//
// All timestamps are UTC; Europe/Berlin runs at UTC+2 in May, so 06:00Z is
// 08:00 local. The hour limits are evaluated in local time, which is the whole
// point of the conversion — comparing UTC hours would let a 21:00 local night
// shift pass as 19:00.
// ============================================================================

// markMinor flags the employee and returns a Monday morning shift that breaks
// none of the three rules, so each test only has to move the one dimension it
// is about.
func markMinor(repo *mockRepository, tenantID, employeeID uuid.UUID) (start, end time.Time) {
	repo.minorEmployees[minorKey(tenantID, employeeID)] = true
	// Monday 08:00–14:00 local
	return mustTime("2026-05-04T06:00:00Z"), mustTime("2026-05-04T12:00:00Z")
}

func TestService_CheckArbzgCompliance_Minor_Compliant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID, employeeID := uuid.New(), uuid.New()
	start, end := markMinor(repo, tenantID, employeeID)

	compliant, reason, err := svc.CheckArbzgCompliance(context.Background(), tenantID, employeeID, start, end)

	require.NoError(t, err)
	assert.True(t, compliant)
	assert.Empty(t, reason)
}

func TestService_CheckArbzgCompliance_Minor_NightWork(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID, employeeID := uuid.New(), uuid.New()
	markMinor(repo, tenantID, employeeID)

	// Monday 20:00–24:00 local — starts exactly at the §14 boundary and runs
	// past it.
	start := mustTime("2026-05-04T18:00:00Z")
	end := mustTime("2026-05-04T22:00:00Z")

	compliant, reason, err := svc.CheckArbzgCompliance(context.Background(), tenantID, employeeID, start, end)

	require.NoError(t, err)
	assert.False(t, compliant)
	assert.Equal(t, ErrJArbSchGNightWork.Error(), reason)
}

func TestService_CheckArbzgCompliance_Minor_DailyHours(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID, employeeID := uuid.New(), uuid.New()
	markMinor(repo, tenantID, employeeID)

	// Monday 08:00–17:00 local: inside the day window, but nine hours long.
	start := mustTime("2026-05-04T06:00:00Z")
	end := mustTime("2026-05-04T15:00:00Z")

	compliant, reason, err := svc.CheckArbzgCompliance(context.Background(), tenantID, employeeID, start, end)

	require.NoError(t, err)
	assert.False(t, compliant)
	assert.Equal(t, ErrJArbSchGDailyHours.Error(), reason)
}

func TestService_CheckArbzgCompliance_Minor_Weekend(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID, employeeID := uuid.New(), uuid.New()
	markMinor(repo, tenantID, employeeID)

	// Saturday 10:00–14:00 local: legal on any weekday, forbidden here.
	start := mustTime("2026-05-02T08:00:00Z")
	end := mustTime("2026-05-02T12:00:00Z")

	compliant, reason, err := svc.CheckArbzgCompliance(context.Background(), tenantID, employeeID, start, end)

	require.NoError(t, err)
	assert.False(t, compliant)
	assert.Equal(t, ErrJArbSchGWeekend.Error(), reason)
}

// The flag is what makes the difference: the exact same shift is fine for an
// adult. Without this the three tests above would also pass if the check
// simply applied to everyone.
func TestService_CheckArbzgCompliance_Adult_NightShiftAllowed(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID, employeeID := uuid.New(), uuid.New()

	start := mustTime("2026-05-04T18:00:00Z")
	end := mustTime("2026-05-04T22:00:00Z")

	compliant, reason, err := svc.CheckArbzgCompliance(context.Background(), tenantID, employeeID, start, end)

	require.NoError(t, err)
	assert.True(t, compliant)
	assert.Empty(t, reason)
}

// A failed lookup must surface as an error, not as a clean "not compliant"
// verdict with an empty reason.
func TestService_CheckArbzgCompliance_MinorLookupFails(t *testing.T) {
	repo := newMockRepository()
	repo.isMinorErr = errors.New("connection refused")
	svc := NewService(repo)
	tenantID, employeeID := uuid.New(), uuid.New()

	start := mustTime("2026-05-04T06:00:00Z")
	compliant, reason, err := svc.CheckArbzgCompliance(context.Background(), tenantID, employeeID, start, start.Add(6*time.Hour))

	require.Error(t, err)
	assert.False(t, compliant)
	assert.Empty(t, reason)
}

// The compliance endpoint reporting a violation is not enough on its own — the
// assignment that creates it has to be refused as well.
func TestService_AssignEmployee_MinorNightShiftRejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID, employeeID := uuid.New(), uuid.New()
	markMinor(repo, tenantID, employeeID)

	shift := &Shift{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Title:     "Late shift",
		StartTime: mustTime("2026-05-04T18:00:00Z"), // 20:00 local
		EndTime:   mustTime("2026-05-04T22:00:00Z"), // 24:00 local
		Status:    ShiftStatusDraft,
	}
	repo.shifts[shift.ID] = shift

	_, err := svc.AssignEmployee(context.Background(), AssignEmployeeInput{
		TenantID:   tenantID,
		ShiftID:    shift.ID,
		EmployeeID: employeeID,
	})

	require.ErrorIs(t, err, ErrJArbSchGNightWork)
	assert.Empty(t, repo.assignments)
}

// Bug #5: ArbZG bidirectional check — new shift sandwiched between two close shifts
func TestService_ArbZG_ViolationBothDirections(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	employeeID := uuid.New()

	// Existing shift A: ends at 22:00
	shiftA := addShift(repo, tenantID, "Shift A", mustTime("2026-05-01T14:00:00Z"), mustTime("2026-05-01T22:00:00Z"), ShiftStatusPublished)
	repo.assignments[assignKey(shiftA.ID, employeeID)] = &ShiftAssignment{
		ID: uuid.New(), TenantID: tenantID, ShiftID: shiftA.ID, EmployeeID: employeeID, AssignedAt: time.Now(),
	}

	// Existing shift B: starts at 04:00 next day (6h after A ends)
	shiftB := addShift(repo, tenantID, "Shift B", mustTime("2026-05-02T04:00:00Z"), mustTime("2026-05-02T12:00:00Z"), ShiftStatusPublished)
	repo.assignments[assignKey(shiftB.ID, employeeID)] = &ShiftAssignment{
		ID: uuid.New(), TenantID: tenantID, ShiftID: shiftB.ID, EmployeeID: employeeID, AssignedAt: time.Now(),
	}

	// New shift: 23:30 to 03:30 — only 1.5h after A ends, 0.5h before B starts
	newShift := addShift(repo, tenantID, "New Middle", mustTime("2026-05-01T23:30:00Z"), mustTime("2026-05-02T03:30:00Z"), ShiftStatusDraft)

	// Should fail: 1.5h rest before (from A) violates ArbZG
	prevEnd := mustTime("2026-05-01T22:00:00Z")
	repo.employeeLatestEnd[employeeID] = &prevEnd

	_, err := svc.AssignEmployee(context.Background(), AssignEmployeeInput{
		TenantID: tenantID, ShiftID: newShift.ID, EmployeeID: employeeID,
	})
	assert.ErrorIs(t, err, ErrArbzgViolation, "shift sandwiched with < 11h rest must be rejected")
}

// Bug #6: ApplyTemplate idempotency — duplicate application must not create duplicate shifts
func TestService_ApplyTemplate_Idempotent(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	// Monday=1 template
	tmpl := addTemplate(repo, tenantID, "Mon-Schicht", 1, 8, 0, 480)

	rangeStart := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 5, 3, 23, 59, 0, 0, time.UTC)

	// Apply once
	shifts1, err := svc.ApplyTemplate(context.Background(), ApplyTemplateInput{
		TenantID: tenantID, TemplateID: tmpl.ID, RangeStart: rangeStart, RangeEnd: rangeEnd,
	})
	require.NoError(t, err)
	count1 := len(shifts1)

	// Apply again — must not create duplicates
	shifts2, err := svc.ApplyTemplate(context.Background(), ApplyTemplateInput{
		TenantID: tenantID, TemplateID: tmpl.ID, RangeStart: rangeStart, RangeEnd: rangeEnd,
	})
	require.NoError(t, err)
	assert.Empty(t, shifts2, "second apply must produce 0 new shifts (all already exist)")
	assert.Equal(t, count1, len(repo.shifts), "total shifts in repo must not grow on second apply")
}

// ============================================================================
// ListShifts Tests
// ============================================================================

func TestService_ListShifts_DefaultPagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	now := time.Now()
	for i := range 60 {
		addShift(repo, tenantID, "Schicht", now.Add(time.Duration(i)*time.Hour), now.Add(time.Duration(i)*time.Hour+8*time.Hour), ShiftStatusDraft)
	}

	shifts, total, err := svc.ListShifts(context.Background(), ListShiftsInput{
		TenantID: tenantID,
	})

	require.NoError(t, err)
	assert.Equal(t, 60, total)
	assert.Len(t, shifts, 50) // default page size
}

// ============================================================================
// Capacity Tests (Bug #8)
// ============================================================================

// A shift with capacity=2 should reject the third assignment.
func TestService_AssignEmployee_CapacityExceeded_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	start := time.Now().Add(48 * time.Hour)
	end := start.Add(8 * time.Hour)
	cap := 2

	shift := &Shift{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Title:     "Full Shift",
		StartTime: start,
		EndTime:   end,
		Status:    ShiftStatusDraft,
		Capacity:  &cap,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.shifts[shift.ID] = shift

	emp1, emp2, emp3 := uuid.New(), uuid.New(), uuid.New()

	// First two assignments succeed
	_, err := svc.AssignEmployee(context.Background(), AssignEmployeeInput{
		TenantID: tenantID, ShiftID: shift.ID, EmployeeID: emp1,
	})
	require.NoError(t, err)

	_, err = svc.AssignEmployee(context.Background(), AssignEmployeeInput{
		TenantID: tenantID, ShiftID: shift.ID, EmployeeID: emp2,
	})
	require.NoError(t, err)

	// Third assignment must be rejected
	_, err = svc.AssignEmployee(context.Background(), AssignEmployeeInput{
		TenantID: tenantID, ShiftID: shift.ID, EmployeeID: emp3,
	})
	assert.ErrorIs(t, err, ErrShiftFull)
}

// A shift without capacity (nil) must accept any number of assignments.
func TestService_AssignEmployee_NoCapacity_Unlimited(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	start := time.Now().Add(48 * time.Hour)
	end := start.Add(8 * time.Hour)

	shift := addShift(repo, tenantID, "Unlimited", start, end, ShiftStatusDraft)
	// shift.Capacity is nil by default

	for i := range 5 {
		emp := uuid.New()
		_, err := svc.AssignEmployee(context.Background(), AssignEmployeeInput{
			TenantID: tenantID, ShiftID: shift.ID, EmployeeID: emp,
		})
		require.NoError(t, err, "assignment %d must succeed when capacity is unlimited", i+1)
	}
}

// ============================================================================
// SwapRequest Tests
// ============================================================================

func TestService_CreateSwapRequest_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	shiftID := uuid.New()
	empA := uuid.New()
	empB := uuid.New()
	assignID := uuid.New()

	req, err := svc.CreateSwapRequest(context.Background(), CreateSwapRequestInput{
		TenantID:              tenantID,
		AssignmentID:          assignID,
		RequestedByEmployeeID: empA,
		SwapWithEmployeeID:    empB,
		ShiftID:               shiftID,
		Reason:                "Urlaub",
		IdempotencyKey:        "idem-key-001",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, req.ID)
	assert.Equal(t, SwapRequestStatusPending, req.Status)
	assert.Equal(t, tenantID, req.TenantID)
}

func TestService_CreateSwapRequest_MissingIdempotencyKey(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateSwapRequest(context.Background(), CreateSwapRequestInput{
		TenantID:              uuid.New(),
		AssignmentID:          uuid.New(),
		RequestedByEmployeeID: uuid.New(),
		SwapWithEmployeeID:    uuid.New(),
		ShiftID:               uuid.New(),
		IdempotencyKey:        "", // missing
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateSwapRequest_SameEmployee(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	empID := uuid.New()
	_, err := svc.CreateSwapRequest(context.Background(), CreateSwapRequestInput{
		TenantID:              uuid.New(),
		AssignmentID:          uuid.New(),
		RequestedByEmployeeID: empID,
		SwapWithEmployeeID:    empID, // same as requester
		ShiftID:               uuid.New(),
		IdempotencyKey:        "idem-key-002",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateSwapRequest_Idempotent(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	input := CreateSwapRequestInput{
		TenantID:              tenantID,
		AssignmentID:          uuid.New(),
		RequestedByEmployeeID: uuid.New(),
		SwapWithEmployeeID:    uuid.New(),
		ShiftID:               uuid.New(),
		IdempotencyKey:        "idem-key-003",
	}

	req1, err := svc.CreateSwapRequest(context.Background(), input)
	require.NoError(t, err)

	req2, err := svc.CreateSwapRequest(context.Background(), input)
	require.NoError(t, err)

	assert.Equal(t, req1.ID, req2.ID, "idempotent call must return same ID")
}

func TestService_ListSwapRequests_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	shiftID := uuid.New()

	for i := range 3 {
		repo.swapRequests[uuid.New()] = &SwapRequest{
			ID:                    uuid.New(),
			TenantID:              tenantID,
			ShiftID:               shiftID,
			RequestedByEmployeeID: uuid.New(),
			SwapWithEmployeeID:    uuid.New(),
			AssignmentID:          uuid.New(),
			Status:                SwapRequestStatusPending,
			IdempotencyKey:        "key-" + string(rune('A'+i)),
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		}
	}

	reqs, total, err := svc.ListSwapRequests(context.Background(), ListSwapRequestsInput{
		TenantID: tenantID,
		ShiftID:  &shiftID,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, reqs, 3)
}

func TestService_ApproveSwapRequest_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	shiftID := uuid.New()
	empA := uuid.New()
	empB := uuid.New()
	assignID := uuid.New()

	// Pre-seed assignments so swap has something to work with.
	repo.assignments[assignKey(shiftID, empA)] = &ShiftAssignment{
		ID: assignID, TenantID: tenantID, ShiftID: shiftID, EmployeeID: empA, AssignedAt: time.Now(),
	}
	repo.assignments[assignKey(shiftID, empB)] = &ShiftAssignment{
		ID: uuid.New(), TenantID: tenantID, ShiftID: shiftID, EmployeeID: empB, AssignedAt: time.Now(),
	}

	reqID := uuid.New()
	repo.swapRequests[reqID] = &SwapRequest{
		ID:                    reqID,
		TenantID:              tenantID,
		AssignmentID:          assignID,
		RequestedByEmployeeID: empA,
		SwapWithEmployeeID:    empB,
		ShiftID:               shiftID,
		Status:                SwapRequestStatusPending,
		IdempotencyKey:        "idem-key-approve",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	approved, err := svc.ApproveSwapRequest(context.Background(), tenantID, reqID)

	require.NoError(t, err)
	assert.Equal(t, SwapRequestStatusApproved, approved.Status)
}

func TestService_ApproveSwapRequest_AlreadyProcessed(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	reqID := uuid.New()
	repo.swapRequests[reqID] = &SwapRequest{
		ID:                    reqID,
		TenantID:              tenantID,
		AssignmentID:          uuid.New(),
		RequestedByEmployeeID: uuid.New(),
		SwapWithEmployeeID:    uuid.New(),
		ShiftID:               uuid.New(),
		Status:                SwapRequestStatusApproved, // already processed
		IdempotencyKey:        "idem-key-already",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	_, err := svc.ApproveSwapRequest(context.Background(), tenantID, reqID)
	assert.ErrorIs(t, err, ErrSwapAlreadyProcessed)
}

func TestService_RejectSwapRequest_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	reqID := uuid.New()
	repo.swapRequests[reqID] = &SwapRequest{
		ID:                    reqID,
		TenantID:              tenantID,
		AssignmentID:          uuid.New(),
		RequestedByEmployeeID: uuid.New(),
		SwapWithEmployeeID:    uuid.New(),
		ShiftID:               uuid.New(),
		Status:                SwapRequestStatusPending,
		IdempotencyKey:        "idem-key-reject",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	rejected, err := svc.RejectSwapRequest(context.Background(), tenantID, reqID)

	require.NoError(t, err)
	assert.Equal(t, SwapRequestStatusRejected, rejected.Status)
}

func TestService_RejectSwapRequest_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.RejectSwapRequest(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrSwapRequestNotFound)
}

// ============================================================================
// SwapRequest Tenant-Isolation Tests
// ============================================================================

// Tenant A cannot read Tenant B's swap request → ErrSwapRequestNotFound.
func TestService_SwapRequest_CrossTenant_NotVisible(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()

	reqID := uuid.New()
	repo.swapRequests[reqID] = &SwapRequest{
		ID:                    reqID,
		TenantID:              tenantB,
		AssignmentID:          uuid.New(),
		RequestedByEmployeeID: uuid.New(),
		SwapWithEmployeeID:    uuid.New(),
		ShiftID:               uuid.New(),
		Status:                SwapRequestStatusPending,
		IdempotencyKey:        "idem-key-b",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Tenant A tries to approve Tenant B's swap request.
	_, err := svc.ApproveSwapRequest(context.Background(), tenantA, reqID)
	assert.ErrorIs(t, err, ErrSwapRequestNotFound, "tenant A must not see tenant B's swap requests")
}

// ListSwapRequests only returns requests for the requesting tenant.
func TestService_ListSwapRequests_TenantIsolation(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Seed 2 requests for A, 1 for B.
	for i := range 2 {
		id := uuid.New()
		repo.swapRequests[id] = &SwapRequest{
			ID:                    id,
			TenantID:              tenantA,
			AssignmentID:          uuid.New(),
			RequestedByEmployeeID: uuid.New(),
			SwapWithEmployeeID:    uuid.New(),
			ShiftID:               uuid.New(),
			Status:                SwapRequestStatusPending,
			IdempotencyKey:        "key-a-" + string(rune('0'+i)),
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		}
	}
	idB := uuid.New()
	repo.swapRequests[idB] = &SwapRequest{
		ID:                    idB,
		TenantID:              tenantB,
		AssignmentID:          uuid.New(),
		RequestedByEmployeeID: uuid.New(),
		SwapWithEmployeeID:    uuid.New(),
		ShiftID:               uuid.New(),
		Status:                SwapRequestStatusPending,
		IdempotencyKey:        "key-b-0",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	reqs, total, err := svc.ListSwapRequests(context.Background(), ListSwapRequestsInput{
		TenantID: tenantA,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, total, "tenant A must only see its own 2 swap requests")
	assert.Len(t, reqs, 2)
	for _, r := range reqs {
		assert.Equal(t, tenantA, r.TenantID)
	}
}
