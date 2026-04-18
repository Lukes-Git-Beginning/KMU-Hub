package berichte

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Fixtures
// ============================================================================

var testTenant = uuid.MustParse("11111111-1111-1111-1111-111111111111")

// ============================================================================
// Fixed clock
// ============================================================================

type fixedClock struct{ t time.Time }

func (f *fixedClock) Now() time.Time { return f.t }

func (f *fixedClock) advance(d time.Duration) { f.t = f.t.Add(d) }

// ============================================================================
// Mock executor
// ============================================================================

type mockExecutor struct {
	runResult   *ReportResult
	runErr      error
	runCalls    int
	lastDef     *Definition
	lastParams  json.RawMessage
	kpiResult   []KPI
	kpiErr      error
	kpiModules  []string
	kpiTenantID uuid.UUID
}

func (m *mockExecutor) Run(_ context.Context, def *Definition, params json.RawMessage) (*ReportResult, error) {
	m.runCalls++
	m.lastDef = def
	m.lastParams = append(json.RawMessage(nil), params...)
	if m.runErr != nil {
		return nil, m.runErr
	}
	if m.runResult != nil {
		return m.runResult, nil
	}
	return &ReportResult{
		Columns: []Column{{Name: "x", Label: "X", Kind: "number"}},
		Rows:    []map[string]any{{"x": 1}},
		Meta:    ReportMeta{RowCount: 1},
	}, nil
}

func (m *mockExecutor) DashboardKPIs(_ context.Context, tenantID uuid.UUID, modules []string) ([]KPI, error) {
	m.kpiTenantID = tenantID
	m.kpiModules = append([]string(nil), modules...)
	if m.kpiErr != nil {
		return nil, m.kpiErr
	}
	return m.kpiResult, nil
}

// ============================================================================
// Mock repository
// ============================================================================

type mockRepository struct {
	mu          sync.Mutex
	definitions map[uuid.UUID]*Definition
	cache       map[string]*CacheEntry // key = defID|hash
	schedules   map[uuid.UUID]*Schedule
	runs        []*Run

	createDefErr   error
	updateDefErr   error
	deleteDefErr   error
	getDefErr      error
	listDefErr     error
	getCacheErr    error
	upsertErr      error
	invalidateErr  error
	createSchedErr error
	updateSchedErr error
	getSchedErr    error
	insertRunErr   error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		definitions: map[uuid.UUID]*Definition{},
		cache:       map[string]*CacheEntry{},
		schedules:   map[uuid.UUID]*Schedule{},
	}
}

func cacheKey(defID uuid.UUID, hash string) string { return defID.String() + "|" + hash }

func (m *mockRepository) CreateDefinition(_ context.Context, def *Definition) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createDefErr != nil {
		return m.createDefErr
	}
	copy := *def
	m.definitions[def.ID] = &copy
	return nil
}

func (m *mockRepository) UpdateDefinition(_ context.Context, def *Definition) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateDefErr != nil {
		return m.updateDefErr
	}
	existing, ok := m.definitions[def.ID]
	if !ok || existing.TenantID != def.TenantID {
		return ErrDefinitionNotFound
	}
	copy := *def
	m.definitions[def.ID] = &copy
	return nil
}

func (m *mockRepository) DeleteDefinition(_ context.Context, tenantID, definitionID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteDefErr != nil {
		return m.deleteDefErr
	}
	existing, ok := m.definitions[definitionID]
	if !ok || existing.TenantID != tenantID {
		return ErrDefinitionNotFound
	}
	delete(m.definitions, definitionID)
	return nil
}

func (m *mockRepository) GetDefinition(_ context.Context, tenantID, definitionID uuid.UUID) (*Definition, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getDefErr != nil {
		return nil, m.getDefErr
	}
	def, ok := m.definitions[definitionID]
	if !ok || def.TenantID != tenantID {
		return nil, ErrDefinitionNotFound
	}
	copy := *def
	return &copy, nil
}

func (m *mockRepository) ListDefinitions(_ context.Context, tenantID uuid.UUID, filter ListDefinitionsFilter, offset, limit int) ([]*Definition, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listDefErr != nil {
		return nil, 0, m.listDefErr
	}
	var all []*Definition
	for _, d := range m.definitions {
		if d.TenantID != tenantID {
			continue
		}
		if filter.Module != nil && d.Module != *filter.Module {
			continue
		}
		if filter.Kind != nil && d.Kind != *filter.Kind {
			continue
		}
		if filter.IsPublished != nil && d.IsPublished != *filter.IsPublished {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(d.Name), strings.ToLower(filter.Search)) {
			continue
		}
		copy := *d
		all = append(all, &copy)
	}
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *mockRepository) GetCacheEntry(_ context.Context, tenantID, definitionID uuid.UUID, paramsHash string) (*CacheEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getCacheErr != nil {
		return nil, m.getCacheErr
	}
	entry, ok := m.cache[cacheKey(definitionID, paramsHash)]
	if !ok || entry.TenantID != tenantID {
		return nil, ErrCacheMiss
	}
	copy := *entry
	return &copy, nil
}

func (m *mockRepository) UpsertCacheEntry(_ context.Context, entry *CacheEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.upsertErr != nil {
		return m.upsertErr
	}
	copy := *entry
	m.cache[cacheKey(entry.DefinitionID, entry.ParamsHash)] = &copy
	return nil
}

func (m *mockRepository) InvalidateCache(_ context.Context, tenantID, definitionID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.invalidateErr != nil {
		return 0, m.invalidateErr
	}
	evicted := 0
	for k, v := range m.cache {
		if v.TenantID == tenantID && v.DefinitionID == definitionID {
			delete(m.cache, k)
			evicted++
		}
	}
	return evicted, nil
}

func (m *mockRepository) DeleteExpiredCacheEntries(_ context.Context, before time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	evicted := 0
	for k, v := range m.cache {
		if v.ExpiresAt.Before(before) {
			delete(m.cache, k)
			evicted++
		}
	}
	return evicted, nil
}

func (m *mockRepository) CreateSchedule(_ context.Context, sch *Schedule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createSchedErr != nil {
		return m.createSchedErr
	}
	copy := *sch
	m.schedules[sch.ID] = &copy
	return nil
}

func (m *mockRepository) UpdateSchedule(_ context.Context, sch *Schedule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateSchedErr != nil {
		return m.updateSchedErr
	}
	existing, ok := m.schedules[sch.ID]
	if !ok || existing.TenantID != sch.TenantID {
		return ErrScheduleNotFound
	}
	copy := *sch
	m.schedules[sch.ID] = &copy
	return nil
}

func (m *mockRepository) DeleteSchedule(_ context.Context, tenantID, scheduleID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.schedules[scheduleID]
	if !ok || existing.TenantID != tenantID {
		return ErrScheduleNotFound
	}
	delete(m.schedules, scheduleID)
	return nil
}

func (m *mockRepository) GetSchedule(_ context.Context, tenantID, scheduleID uuid.UUID) (*Schedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getSchedErr != nil {
		return nil, m.getSchedErr
	}
	sch, ok := m.schedules[scheduleID]
	if !ok || sch.TenantID != tenantID {
		return nil, ErrScheduleNotFound
	}
	copy := *sch
	return &copy, nil
}

func (m *mockRepository) ListSchedules(_ context.Context, tenantID uuid.UUID, filter ListSchedulesFilter, offset, limit int) ([]*Schedule, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []*Schedule
	for _, s := range m.schedules {
		if s.TenantID != tenantID {
			continue
		}
		if filter.DefinitionID != nil && s.DefinitionID != *filter.DefinitionID {
			continue
		}
		if filter.Active != nil && s.Active != *filter.Active {
			continue
		}
		copy := *s
		all = append(all, &copy)
	}
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *mockRepository) ListDueSchedules(_ context.Context, _ time.Time) ([]*Schedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []*Schedule
	for _, s := range m.schedules {
		if !s.Active {
			continue
		}
		copy := *s
		all = append(all, &copy)
	}
	return all, nil
}

func (m *mockRepository) ClaimSchedule(_ context.Context, scheduleID uuid.UUID, previousLastRunAt *time.Time, now time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sch, ok := m.schedules[scheduleID]
	if !ok {
		return false, nil
	}
	match := false
	if previousLastRunAt == nil {
		match = sch.LastRunAt == nil
	} else if sch.LastRunAt != nil && sch.LastRunAt.Equal(*previousLastRunAt) {
		match = true
	}
	if !match {
		return false, nil
	}
	sch.LastRunAt = &now
	sch.UpdatedAt = now
	return true, nil
}

func (m *mockRepository) UpdateScheduleLastRun(_ context.Context, scheduleID uuid.UUID, status string, runErr *string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sch, ok := m.schedules[scheduleID]
	if !ok {
		return ErrScheduleNotFound
	}
	sch.LastRunStatus = &status
	sch.LastRunError = runErr
	sch.LastRunAt = &at
	sch.UpdatedAt = at
	return nil
}

func (m *mockRepository) InsertRun(_ context.Context, run *Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.insertRunErr != nil {
		return m.insertRunErr
	}
	copy := *run
	m.runs = append(m.runs, &copy)
	return nil
}

// ============================================================================
// Helper
// ============================================================================

func newServiceForTest(exec Executor) (*Service, *mockRepository, *fixedClock) {
	repo := newMockRepository()
	clk := &fixedClock{t: time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)}
	svc := NewService(repo, exec, Options{Clock: clk, CacheTTL: 15 * time.Minute})
	return svc, repo, clk
}

func mustCreateDefinition(t *testing.T, svc *Service) *Definition {
	t.Helper()
	def, err := svc.CreateDefinition(context.Background(), CreateDefinitionInput{
		TenantID:      testTenant,
		Name:          "Pipeline",
		Module:        "crm",
		Kind:          "system",
		QueryConfig:   json.RawMessage(`{"kind":"pipeline"}`),
		DefaultFormat: "pdf",
		IsPublished:   true,
	})
	if err != nil {
		t.Fatalf("create definition: %v", err)
	}
	return def
}

// ============================================================================
// Tests: CreateDefinition
// ============================================================================

func TestService_CreateDefinition_HappyPath(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def, err := svc.CreateDefinition(context.Background(), CreateDefinitionInput{
		TenantID:    testTenant,
		Name:        "Umsatz",
		Module:      "finanzen",
		Kind:        "system",
		QueryConfig: json.RawMessage(`{"kind":"revenue_by_month"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if def.DefaultFormat != "pdf" {
		t.Errorf("expected default format pdf, got %q", def.DefaultFormat)
	}
}

func TestService_CreateDefinition_EmptyName(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	_, err := svc.CreateDefinition(context.Background(), CreateDefinitionInput{
		TenantID: testTenant,
		Name:     "   ",
		Module:   "finanzen",
	})
	if !errors.Is(err, ErrInvalidQueryConfig) {
		t.Errorf("expected ErrInvalidQueryConfig, got %v", err)
	}
}

func TestService_CreateDefinition_InvalidModule(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	_, err := svc.CreateDefinition(context.Background(), CreateDefinitionInput{
		TenantID: testTenant,
		Name:     "X",
		Module:   "xyz",
	})
	if !errors.Is(err, ErrInvalidModule) {
		t.Errorf("expected ErrInvalidModule, got %v", err)
	}
}

func TestService_CreateDefinition_InvalidKind(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	_, err := svc.CreateDefinition(context.Background(), CreateDefinitionInput{
		TenantID: testTenant,
		Name:     "X",
		Module:   "crm",
		Kind:     "weird",
	})
	if !errors.Is(err, ErrInvalidKind) {
		t.Errorf("expected ErrInvalidKind, got %v", err)
	}
}

func TestService_CreateDefinition_InvalidFormat(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	_, err := svc.CreateDefinition(context.Background(), CreateDefinitionInput{
		TenantID:      testTenant,
		Name:          "X",
		Module:        "crm",
		DefaultFormat: "docx",
	})
	if !errors.Is(err, ErrInvalidFormat) {
		t.Errorf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestService_CreateDefinition_InvalidJSON(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	_, err := svc.CreateDefinition(context.Background(), CreateDefinitionInput{
		TenantID:    testTenant,
		Name:        "X",
		Module:      "crm",
		QueryConfig: json.RawMessage(`not json`),
	})
	if !errors.Is(err, ErrInvalidQueryConfig) {
		t.Errorf("expected ErrInvalidQueryConfig, got %v", err)
	}
}

func TestService_CreateDefinition_DefaultsApplied(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def, err := svc.CreateDefinition(context.Background(), CreateDefinitionInput{
		TenantID: testTenant,
		Name:     "X",
		Module:   "crm",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.Kind != "custom" {
		t.Errorf("expected kind custom, got %q", def.Kind)
	}
	if def.DefaultFormat != "pdf" {
		t.Errorf("expected format pdf, got %q", def.DefaultFormat)
	}
	if string(def.QueryConfig) != "{}" {
		t.Errorf("expected empty JSON object, got %s", def.QueryConfig)
	}
}

// ============================================================================
// Tests: UpdateDefinition
// ============================================================================

func TestService_UpdateDefinition_HappyPath(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)

	newName := "Neue Pipeline"
	newPublished := false
	updated, err := svc.UpdateDefinition(context.Background(), UpdateDefinitionInput{
		TenantID:     testTenant,
		DefinitionID: def.ID,
		Name:         &newName,
		IsPublished:  &newPublished,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Neue Pipeline" {
		t.Errorf("expected name updated, got %q", updated.Name)
	}
	if updated.IsPublished {
		t.Error("expected is_published false")
	}
}

func TestService_UpdateDefinition_NotFound(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	_, err := svc.UpdateDefinition(context.Background(), UpdateDefinitionInput{
		TenantID:     testTenant,
		DefinitionID: uuid.New(),
	})
	if !errors.Is(err, ErrDefinitionNotFound) {
		t.Errorf("expected ErrDefinitionNotFound, got %v", err)
	}
}

func TestService_UpdateDefinition_InvalidName(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	empty := "   "
	_, err := svc.UpdateDefinition(context.Background(), UpdateDefinitionInput{
		TenantID:     testTenant,
		DefinitionID: def.ID,
		Name:         &empty,
	})
	if !errors.Is(err, ErrInvalidQueryConfig) {
		t.Errorf("expected ErrInvalidQueryConfig, got %v", err)
	}
}

func TestService_UpdateDefinition_InvalidModule(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	bad := "xxx"
	_, err := svc.UpdateDefinition(context.Background(), UpdateDefinitionInput{
		TenantID:     testTenant,
		DefinitionID: def.ID,
		Module:       &bad,
	})
	if !errors.Is(err, ErrInvalidModule) {
		t.Errorf("expected ErrInvalidModule, got %v", err)
	}
}

func TestService_UpdateDefinition_InvalidFormat(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	bad := "docx"
	_, err := svc.UpdateDefinition(context.Background(), UpdateDefinitionInput{
		TenantID:      testTenant,
		DefinitionID:  def.ID,
		DefaultFormat: &bad,
	})
	if !errors.Is(err, ErrInvalidFormat) {
		t.Errorf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestService_UpdateDefinition_InvalidJSON(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	_, err := svc.UpdateDefinition(context.Background(), UpdateDefinitionInput{
		TenantID:     testTenant,
		DefinitionID: def.ID,
		QueryConfig:  json.RawMessage(`not-json`),
	})
	if !errors.Is(err, ErrInvalidQueryConfig) {
		t.Errorf("expected ErrInvalidQueryConfig, got %v", err)
	}
}

// ============================================================================
// Tests: DeleteDefinition / GetDefinition / ListDefinitions
// ============================================================================

func TestService_DeleteDefinition_HappyPath(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	if err := svc.DeleteDefinition(context.Background(), testTenant, def.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.GetDefinition(context.Background(), testTenant, def.ID); !errors.Is(err, ErrDefinitionNotFound) {
		t.Errorf("expected ErrDefinitionNotFound, got %v", err)
	}
}

func TestService_DeleteDefinition_NotFound(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	err := svc.DeleteDefinition(context.Background(), testTenant, uuid.New())
	if !errors.Is(err, ErrDefinitionNotFound) {
		t.Errorf("expected ErrDefinitionNotFound, got %v", err)
	}
}

func TestService_ListDefinitions(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	mustCreateDefinition(t, svc) // crm system
	_, _ = svc.CreateDefinition(context.Background(), CreateDefinitionInput{
		TenantID: testTenant, Name: "Other", Module: "finanzen", Kind: "custom",
	})

	crm := "crm"
	defs, total, err := svc.ListDefinitions(context.Background(), ListDefinitionsInput{
		TenantID: testTenant,
		Module:   &crm,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(defs) != 1 {
		t.Errorf("expected 1 CRM definition, got total=%d len=%d", total, len(defs))
	}
}

// ============================================================================
// Tests: RunReport
// ============================================================================

func TestService_RunReport_NoExecutor(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	_, _, err := svc.RunReport(context.Background(), RunReportInput{
		TenantID:     testTenant,
		DefinitionID: uuid.New(),
	})
	if !errors.Is(err, ErrExecutorUnavailable) {
		t.Errorf("expected ErrExecutorUnavailable, got %v", err)
	}
}

func TestService_RunReport_DefinitionNotFound(t *testing.T) {
	svc, _, _ := newServiceForTest(&mockExecutor{})
	_, _, err := svc.RunReport(context.Background(), RunReportInput{
		TenantID:     testTenant,
		DefinitionID: uuid.New(),
	})
	if !errors.Is(err, ErrDefinitionNotFound) {
		t.Errorf("expected ErrDefinitionNotFound, got %v", err)
	}
}

func TestService_RunReport_CacheMissCallsExecutor(t *testing.T) {
	exec := &mockExecutor{}
	svc, repo, _ := newServiceForTest(exec)
	def := mustCreateDefinition(t, svc)

	result, run, err := svc.RunReport(context.Background(), RunReportInput{
		TenantID:     testTenant,
		DefinitionID: def.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.runCalls != 1 {
		t.Errorf("expected executor called once, got %d", exec.runCalls)
	}
	if result.Meta.FromCache {
		t.Error("expected FromCache=false on miss")
	}
	if run.Status != "success" {
		t.Errorf("expected success run, got %q", run.Status)
	}
	if len(repo.cache) != 1 {
		t.Errorf("expected cache populated, got %d entries", len(repo.cache))
	}
	if len(repo.runs) != 1 {
		t.Errorf("expected 1 run recorded, got %d", len(repo.runs))
	}
}

func TestService_RunReport_CacheHit(t *testing.T) {
	exec := &mockExecutor{}
	svc, _, _ := newServiceForTest(exec)
	def := mustCreateDefinition(t, svc)

	// First call populates the cache.
	if _, _, err := svc.RunReport(context.Background(), RunReportInput{
		TenantID: testTenant, DefinitionID: def.ID,
	}); err != nil {
		t.Fatalf("first run: %v", err)
	}
	firstCalls := exec.runCalls

	// Second call hits cache.
	result, _, err := svc.RunReport(context.Background(), RunReportInput{
		TenantID: testTenant, DefinitionID: def.ID,
	})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if exec.runCalls != firstCalls {
		t.Errorf("expected no additional executor call on cache hit, got %d extra",
			exec.runCalls-firstCalls)
	}
	if !result.Meta.FromCache {
		t.Error("expected FromCache=true on cache hit")
	}
}

func TestService_RunReport_ForceRefresh(t *testing.T) {
	exec := &mockExecutor{}
	svc, _, _ := newServiceForTest(exec)
	def := mustCreateDefinition(t, svc)

	_, _, _ = svc.RunReport(context.Background(), RunReportInput{
		TenantID: testTenant, DefinitionID: def.ID,
	})
	firstCalls := exec.runCalls

	_, _, err := svc.RunReport(context.Background(), RunReportInput{
		TenantID: testTenant, DefinitionID: def.ID, ForceRefresh: true,
	})
	if err != nil {
		t.Fatalf("force refresh: %v", err)
	}
	if exec.runCalls != firstCalls+1 {
		t.Errorf("expected executor re-called on force refresh, got %d calls after %d",
			exec.runCalls, firstCalls)
	}
}

func TestService_RunReport_CacheExpiredRefetches(t *testing.T) {
	exec := &mockExecutor{}
	svc, _, clk := newServiceForTest(exec)
	def := mustCreateDefinition(t, svc)

	_, _, _ = svc.RunReport(context.Background(), RunReportInput{
		TenantID: testTenant, DefinitionID: def.ID,
	})
	firstCalls := exec.runCalls

	clk.advance(20 * time.Minute) // past 15-min default TTL

	_, _, err := svc.RunReport(context.Background(), RunReportInput{
		TenantID: testTenant, DefinitionID: def.ID,
	})
	if err != nil {
		t.Fatalf("post-expiry run: %v", err)
	}
	if exec.runCalls != firstCalls+1 {
		t.Errorf("expected executor re-called after TTL, got %d", exec.runCalls)
	}
}

func TestService_RunReport_ExecutorError(t *testing.T) {
	exec := &mockExecutor{runErr: errors.New("boom")}
	svc, repo, _ := newServiceForTest(exec)
	def := mustCreateDefinition(t, svc)

	_, _, err := svc.RunReport(context.Background(), RunReportInput{
		TenantID: testTenant, DefinitionID: def.ID,
	})
	if err == nil {
		t.Fatal("expected error from executor")
	}
	if len(repo.runs) != 1 || repo.runs[0].Status != "failed" {
		t.Errorf("expected failed run recorded, got runs=%d", len(repo.runs))
	}
}

// ============================================================================
// Tests: GetCachedResult / InvalidateCache
// ============================================================================

func TestService_GetCachedResult_Miss(t *testing.T) {
	svc, _, _ := newServiceForTest(&mockExecutor{})
	_, err := svc.GetCachedResult(context.Background(), testTenant, uuid.New(), nil)
	if !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}
}

func TestService_GetCachedResult_Hit(t *testing.T) {
	exec := &mockExecutor{}
	svc, _, _ := newServiceForTest(exec)
	def := mustCreateDefinition(t, svc)

	_, _, _ = svc.RunReport(context.Background(), RunReportInput{
		TenantID: testTenant, DefinitionID: def.ID,
	})

	result, err := svc.GetCachedResult(context.Background(), testTenant, def.ID, nil)
	if err != nil {
		t.Fatalf("cached result: %v", err)
	}
	if !result.Meta.FromCache {
		t.Error("expected FromCache=true")
	}
}

func TestService_GetCachedResult_Expired(t *testing.T) {
	exec := &mockExecutor{}
	svc, _, clk := newServiceForTest(exec)
	def := mustCreateDefinition(t, svc)
	_, _, _ = svc.RunReport(context.Background(), RunReportInput{
		TenantID: testTenant, DefinitionID: def.ID,
	})
	clk.advance(1 * time.Hour)
	_, err := svc.GetCachedResult(context.Background(), testTenant, def.ID, nil)
	if !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss after expiry, got %v", err)
	}
}

func TestService_InvalidateCache(t *testing.T) {
	exec := &mockExecutor{}
	svc, _, _ := newServiceForTest(exec)
	def := mustCreateDefinition(t, svc)
	_, _, _ = svc.RunReport(context.Background(), RunReportInput{
		TenantID: testTenant, DefinitionID: def.ID,
	})
	evicted, err := svc.InvalidateCache(context.Background(), testTenant, def.ID)
	if err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	if evicted != 1 {
		t.Errorf("expected 1 evicted, got %d", evicted)
	}
}

// ============================================================================
// Tests: Schedules
// ============================================================================

func TestService_CreateSchedule_HappyPath(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	sch, err := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID:       testTenant,
		DefinitionID:   def.ID,
		Name:           "Weekly",
		CronExpression: "0 8 * * MON",
		Recipients:     []string{"a@example.com", "  ", "b@example.com"},
		Format:         "pdf",
		Active:         true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sch.Recipients) != 2 {
		t.Errorf("expected 2 sanitized recipients, got %d", len(sch.Recipients))
	}
}

func TestService_CreateSchedule_InvalidCron(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	_, err := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID:       testTenant,
		DefinitionID:   def.ID,
		Name:           "Broken",
		CronExpression: "not-a-cron",
		Format:         "pdf",
	})
	if !errors.Is(err, ErrInvalidCron) {
		t.Errorf("expected ErrInvalidCron, got %v", err)
	}
}

func TestService_CreateSchedule_InvalidFormat(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	_, err := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID:       testTenant,
		DefinitionID:   def.ID,
		Name:           "X",
		CronExpression: "* * * * *",
		Format:         "docx",
	})
	if !errors.Is(err, ErrInvalidFormat) {
		t.Errorf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestService_CreateSchedule_DefinitionNotFound(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	_, err := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID:       testTenant,
		DefinitionID:   uuid.New(),
		Name:           "X",
		CronExpression: "* * * * *",
	})
	if !errors.Is(err, ErrDefinitionNotFound) {
		t.Errorf("expected ErrDefinitionNotFound, got %v", err)
	}
}

func TestService_CreateSchedule_DefaultFormat(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	sch, err := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID:       testTenant,
		DefinitionID:   def.ID,
		Name:           "X",
		CronExpression: "* * * * *",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sch.Format != "pdf" {
		t.Errorf("expected default format pdf, got %q", sch.Format)
	}
}

func TestService_CreateSchedule_EmptyName(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	_, err := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID:       testTenant,
		DefinitionID:   def.ID,
		Name:           "  ",
		CronExpression: "* * * * *",
	})
	if !errors.Is(err, ErrInvalidQueryConfig) {
		t.Errorf("expected ErrInvalidQueryConfig, got %v", err)
	}
}

func TestService_UpdateSchedule_HappyPath(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	sch, _ := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID:       testTenant,
		DefinitionID:   def.ID,
		Name:           "Weekly",
		CronExpression: "0 8 * * MON",
		Format:         "pdf",
	})

	newName := "Daily"
	newCron := "0 8 * * *"
	newFormat := "csv"
	active := false
	updated, err := svc.UpdateSchedule(context.Background(), UpdateScheduleInput{
		TenantID:       testTenant,
		ScheduleID:     sch.ID,
		Name:           &newName,
		CronExpression: &newCron,
		Format:         &newFormat,
		Recipients:     []string{"x@y.com"},
		RecipientsSet:  true,
		Active:         &active,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Daily" || updated.Format != "csv" || updated.Active {
		t.Errorf("update not applied: %+v", updated)
	}
	if len(updated.Recipients) != 1 {
		t.Errorf("expected 1 recipient, got %d", len(updated.Recipients))
	}
}

func TestService_UpdateSchedule_NotFound(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	_, err := svc.UpdateSchedule(context.Background(), UpdateScheduleInput{
		TenantID:   testTenant,
		ScheduleID: uuid.New(),
	})
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound, got %v", err)
	}
}

func TestService_UpdateSchedule_InvalidCron(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	sch, _ := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID: testTenant, DefinitionID: def.ID, Name: "X",
		CronExpression: "* * * * *", Format: "pdf",
	})
	bad := "bad cron"
	_, err := svc.UpdateSchedule(context.Background(), UpdateScheduleInput{
		TenantID:       testTenant,
		ScheduleID:     sch.ID,
		CronExpression: &bad,
	})
	if !errors.Is(err, ErrInvalidCron) {
		t.Errorf("expected ErrInvalidCron, got %v", err)
	}
}

func TestService_UpdateSchedule_InvalidFormat(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	sch, _ := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID: testTenant, DefinitionID: def.ID, Name: "X",
		CronExpression: "* * * * *", Format: "pdf",
	})
	bad := "docx"
	_, err := svc.UpdateSchedule(context.Background(), UpdateScheduleInput{
		TenantID:   testTenant,
		ScheduleID: sch.ID,
		Format:     &bad,
	})
	if !errors.Is(err, ErrInvalidFormat) {
		t.Errorf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestService_DeleteSchedule_HappyPath(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	sch, _ := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID: testTenant, DefinitionID: def.ID, Name: "X",
		CronExpression: "* * * * *", Format: "pdf",
	})
	if err := svc.DeleteSchedule(context.Background(), testTenant, sch.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := svc.DeleteSchedule(context.Background(), testTenant, sch.ID); !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound on second delete, got %v", err)
	}
}

func TestService_ListSchedules(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	_, _ = svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID: testTenant, DefinitionID: def.ID, Name: "A",
		CronExpression: "* * * * *", Format: "pdf", Active: true,
	})
	_, _ = svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID: testTenant, DefinitionID: def.ID, Name: "B",
		CronExpression: "* * * * *", Format: "pdf", Active: false,
	})
	active := true
	out, total, err := svc.ListSchedules(context.Background(), ListSchedulesInput{
		TenantID: testTenant, Active: &active,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(out) != 1 {
		t.Errorf("expected 1 active schedule, got total=%d len=%d", total, len(out))
	}
}

func TestService_ToggleSchedule(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	def := mustCreateDefinition(t, svc)
	sch, _ := svc.CreateSchedule(context.Background(), CreateScheduleInput{
		TenantID: testTenant, DefinitionID: def.ID, Name: "X",
		CronExpression: "* * * * *", Format: "pdf", Active: true,
	})
	toggled, err := svc.ToggleSchedule(context.Background(), testTenant, sch.ID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if toggled.Active {
		t.Error("expected active=false")
	}
}

// ============================================================================
// Tests: ValidateCron / KPIs
// ============================================================================

func TestService_ValidateCron(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	if err := svc.ValidateCron("0 8 * * *"); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
	if err := svc.ValidateCron("xxx"); !errors.Is(err, ErrInvalidCron) {
		t.Errorf("expected ErrInvalidCron, got %v", err)
	}
}

func TestService_GetDashboardKPIs_HappyPath(t *testing.T) {
	exec := &mockExecutor{
		kpiResult: []KPI{{ID: "rev", Label: "Revenue", Value: "1.2M", ModuleID: "finanzen"}},
	}
	svc, _, _ := newServiceForTest(exec)
	kpis, err := svc.GetDashboardKPIs(context.Background(), testTenant, []string{"finanzen", "xxx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kpis) != 1 {
		t.Errorf("expected 1 KPI, got %d", len(kpis))
	}
	if len(exec.kpiModules) != 1 || exec.kpiModules[0] != "finanzen" {
		t.Errorf("expected invalid module filtered, got %v", exec.kpiModules)
	}
}

func TestService_GetDashboardKPIs_NoExecutor(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	_, err := svc.GetDashboardKPIs(context.Background(), testTenant, nil)
	if !errors.Is(err, ErrExecutorUnavailable) {
		t.Errorf("expected ErrExecutorUnavailable, got %v", err)
	}
}

// ============================================================================
// Tests: hash/normalize helpers
// ============================================================================

func TestHashParams_StableForEmpty(t *testing.T) {
	h1 := hashParams(nil)
	h2 := hashParams([]byte(""))
	h3 := hashParams([]byte("{}"))
	if h1 != h2 || h2 != h3 {
		t.Errorf("empty variants produced different hashes: %s / %s / %s", h1, h2, h3)
	}
}

func TestSanitizeRecipients_TrimsAndDrops(t *testing.T) {
	got := sanitizeRecipients([]string{" a@b.com ", "", "c@d.com"})
	if len(got) != 2 || got[0] != "a@b.com" || got[1] != "c@d.com" {
		t.Errorf("unexpected output: %v", got)
	}
}

func TestSanitizeModules_FiltersAndSorts(t *testing.T) {
	got := sanitizeModules([]string{"finanzen", "xxx", " crm "})
	if len(got) != 2 || got[0] != "crm" || got[1] != "finanzen" {
		t.Errorf("unexpected output: %v", got)
	}
}

// compile-time interface check for mockRepository
var _ Repository = (*mockRepository)(nil)
