package berichte

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"
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
	documents   map[uuid.UUID]*Document
	shares      map[uuid.UUID]*ShareToken
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
		documents:   map[uuid.UUID]*Document{},
		shares:      map[uuid.UUID]*ShareToken{},
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

func (m *mockRepository) CreateDocument(_ context.Context, doc *Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *doc
	m.documents[doc.ID] = &copy
	return nil
}

func (m *mockRepository) UpdateDocument(_ context.Context, doc *Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.documents[doc.ID]
	if !ok || existing.TenantID != doc.TenantID {
		return ErrDocumentNotFound
	}
	copy := *doc
	m.documents[doc.ID] = &copy
	return nil
}

func (m *mockRepository) DeleteDocument(_ context.Context, tenantID, documentID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.documents[documentID]
	if !ok || existing.TenantID != tenantID {
		return ErrDocumentNotFound
	}
	delete(m.documents, documentID)
	return nil
}

func (m *mockRepository) GetDocument(_ context.Context, tenantID, documentID uuid.UUID) (*Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc, ok := m.documents[documentID]
	if !ok || doc.TenantID != tenantID {
		return nil, ErrDocumentNotFound
	}
	copy := *doc
	return &copy, nil
}

func (m *mockRepository) ListDocuments(_ context.Context, tenantID uuid.UUID, filter ListDocumentsFilter, offset, limit int) ([]*Document, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []*Document
	for _, d := range m.documents {
		if d.TenantID != tenantID {
			continue
		}
		if filter.Module != nil && d.Module != *filter.Module {
			continue
		}
		if filter.Status != nil && d.Status != *filter.Status {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(d.Title), strings.ToLower(filter.Search)) {
			continue
		}
		copy := *d
		all = append(all, &copy)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].UpdatedAt.After(all[j].UpdatedAt) })

	total := len(all)
	if offset >= total {
		return []*Document{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
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

// ============================================================================
// Documents (multi-page authoring)
// ============================================================================

func TestCreateDocumentAppliesDefaults(t *testing.T) {
	svc, _, clk := newServiceForTest(nil)

	doc, err := svc.CreateDocument(context.Background(), CreateDocumentInput{TenantID: testTenant})
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}

	if doc.Title != "Neuer Bericht" {
		t.Errorf("title = %q, want fallback title", doc.Title)
	}
	if doc.Module != "cross" {
		t.Errorf("module = %q, want cross", doc.Module)
	}
	if doc.Status != "draft" {
		t.Errorf("status = %q, want draft", doc.Status)
	}
	if string(doc.Rows) != "[]" {
		t.Errorf("rows = %q, want empty array", doc.Rows)
	}
	if string(doc.Settings) != "{}" {
		t.Errorf("settings = %q, want empty object", doc.Settings)
	}
	if doc.ReleasedAt != nil {
		t.Error("released_at set on a fresh draft")
	}
	if !doc.CreatedAt.Equal(clk.t) {
		t.Errorf("created_at = %v, want %v", doc.CreatedAt, clk.t)
	}
}

func TestCreateDocumentRejectsMalformedPayloads(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)

	tests := []struct {
		name  string
		input CreateDocumentInput
		want  error
	}{
		{
			name:  "rows must be an array",
			input: CreateDocumentInput{TenantID: testTenant, Rows: json.RawMessage(`{"id":"row-1"}`)},
			want:  ErrInvalidRows,
		},
		{
			name:  "rows must be valid JSON",
			input: CreateDocumentInput{TenantID: testTenant, Rows: json.RawMessage(`[{"id":`)},
			want:  ErrInvalidRows,
		},
		{
			name:  "settings must be an object",
			input: CreateDocumentInput{TenantID: testTenant, Settings: json.RawMessage(`[]`)},
			want:  ErrInvalidSettings,
		},
		{
			name:  "unknown module is rejected",
			input: CreateDocumentInput{TenantID: testTenant, Module: "marketing"},
			want:  ErrInvalidModule,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.CreateDocument(context.Background(), tc.input); !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCreateDocumentKeepsBlockTreeVerbatim(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)

	// camelCase block keys are frontend-owned and must survive untouched.
	rows := `[{"id":"row-1","columns":[{"id":"col-1","blocks":[{"id":"c1","type":"cover","title":"Q3","showDate":true}]}]}]`

	doc, err := svc.CreateDocument(context.Background(), CreateDocumentInput{
		TenantID: testTenant,
		Title:    "Quartalsbericht",
		Module:   "finanzen",
		Rows:     json.RawMessage(rows),
		Settings: json.RawMessage(`{"showPageNumbers":true}`),
	})
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	if string(doc.Rows) != rows {
		t.Errorf("rows were rewritten:\n got %s\nwant %s", doc.Rows, rows)
	}
	if string(doc.Settings) != `{"showPageNumbers":true}` {
		t.Errorf("settings were rewritten: %s", doc.Settings)
	}
}

func TestUpdateDocumentStampsReleasedAtOnce(t *testing.T) {
	svc, _, clk := newServiceForTest(nil)

	doc, err := svc.CreateDocument(context.Background(), CreateDocumentInput{TenantID: testTenant, Title: "Bericht"})
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}

	released := "released"
	updated, err := svc.UpdateDocument(context.Background(), UpdateDocumentInput{
		TenantID:   testTenant,
		DocumentID: doc.ID,
		Status:     &released,
	})
	if err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}
	if updated.ReleasedAt == nil || !updated.ReleasedAt.Equal(clk.t) {
		t.Fatalf("released_at = %v, want %v", updated.ReleasedAt, clk.t)
	}

	// A later edit must not move the release date.
	firstRelease := *updated.ReleasedAt
	clk.t = clk.t.Add(time.Hour)
	title := "Bericht (final)"
	again, err := svc.UpdateDocument(context.Background(), UpdateDocumentInput{
		TenantID:   testTenant,
		DocumentID: doc.ID,
		Title:      &title,
	})
	if err != nil {
		t.Fatalf("UpdateDocument (second): %v", err)
	}
	if again.ReleasedAt == nil || !again.ReleasedAt.Equal(firstRelease) {
		t.Errorf("released_at moved to %v, want %v", again.ReleasedAt, firstRelease)
	}
	if !again.UpdatedAt.Equal(clk.t) {
		t.Errorf("updated_at = %v, want %v", again.UpdatedAt, clk.t)
	}
}

func TestUpdateDocumentRejectsUnknownStatus(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)

	doc, err := svc.CreateDocument(context.Background(), CreateDocumentInput{TenantID: testTenant})
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}

	bogus := "approved"
	if _, err := svc.UpdateDocument(context.Background(), UpdateDocumentInput{
		TenantID:   testTenant,
		DocumentID: doc.ID,
		Status:     &bogus,
	}); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("error = %v, want ErrInvalidStatus", err)
	}
}

func TestDocumentAccessIsTenantScoped(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	otherTenant := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	doc, err := svc.CreateDocument(context.Background(), CreateDocumentInput{TenantID: testTenant, Title: "Intern"})
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}

	if _, err := svc.GetDocument(context.Background(), otherTenant, doc.ID); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("Get across tenants: %v, want ErrDocumentNotFound", err)
	}
	title := "Übernommen"
	if _, err := svc.UpdateDocument(context.Background(), UpdateDocumentInput{
		TenantID:   otherTenant,
		DocumentID: doc.ID,
		Title:      &title,
	}); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("Update across tenants: %v, want ErrDocumentNotFound", err)
	}
	if err := svc.DeleteDocument(context.Background(), otherTenant, doc.ID); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("Delete across tenants: %v, want ErrDocumentNotFound", err)
	}

	docs, total, err := svc.ListDocuments(context.Background(), ListDocumentsInput{TenantID: otherTenant})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if total != 0 || len(docs) != 0 {
		t.Errorf("foreign tenant sees %d documents (total %d), want 0", len(docs), total)
	}
}

func TestListDocumentsFilters(t *testing.T) {
	svc, _, clk := newServiceForTest(nil)
	ctx := context.Background()

	for _, spec := range []struct{ title, module string }{
		{"Umsatzbericht", "finanzen"},
		{"Pipeline-Report", "crm"},
	} {
		clk.t = clk.t.Add(time.Minute)
		if _, err := svc.CreateDocument(ctx, CreateDocumentInput{
			TenantID: testTenant,
			Title:    spec.title,
			Module:   spec.module,
		}); err != nil {
			t.Fatalf("CreateDocument: %v", err)
		}
	}

	module := "crm"
	docs, total, err := svc.ListDocuments(ctx, ListDocumentsInput{TenantID: testTenant, Module: &module})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if total != 1 || len(docs) != 1 || docs[0].Title != "Pipeline-Report" {
		t.Fatalf("module filter returned %d docs (total %d)", len(docs), total)
	}

	docs, total, err = svc.ListDocuments(ctx, ListDocumentsInput{TenantID: testTenant, Search: "umsatz"})
	if err != nil {
		t.Fatalf("ListDocuments (search): %v", err)
	}
	if total != 1 || len(docs) != 1 || docs[0].Title != "Umsatzbericht" {
		t.Fatalf("search returned %d docs (total %d)", len(docs), total)
	}

	bogus := "approved"
	if _, _, err := svc.ListDocuments(ctx, ListDocumentsInput{TenantID: testTenant, Status: &bogus}); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("status filter error = %v, want ErrInvalidStatus", err)
	}
}

func TestListTemplates(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)

	templates := svc.ListTemplates()
	if len(templates) == 0 {
		t.Fatal("ListTemplates returned no templates")
	}

	seenIDs := make(map[string]bool, len(templates))
	for _, tpl := range templates {
		if tpl.ID == "" || seenIDs[tpl.ID] {
			t.Fatalf("template id %q is empty or duplicated", tpl.ID)
		}
		seenIDs[tpl.ID] = true

		if tpl.Title == "" {
			t.Fatalf("template %s: empty title", tpl.ID)
		}
		// Every template's module must satisfy the same enum the
		// report_documents CHECK constraint enforces, so creating a document
		// from any template never fails module validation (regression guard
		// for the frontend's "work" module, which has no backend equivalent).
		if !isValidModule(tpl.Module) {
			t.Fatalf("template %s: module %q is not a valid backend module", tpl.ID, tpl.Module)
		}

		trimmedRows := bytes.TrimSpace(tpl.Rows)
		if len(trimmedRows) == 0 || trimmedRows[0] != '[' || !json.Valid(trimmedRows) {
			t.Fatalf("template %s: rows is not a valid JSON array", tpl.ID)
		}
		trimmedSettings := bytes.TrimSpace(tpl.Settings)
		if len(trimmedSettings) == 0 || trimmedSettings[0] != '{' || !json.Valid(trimmedSettings) {
			t.Fatalf("template %s: settings is not a valid JSON object", tpl.ID)
		}
	}
}

func TestCreateDocumentFromEveryTemplateSucceeds(t *testing.T) {
	svc, _, _ := newServiceForTest(nil)
	ctx := context.Background()

	for _, tpl := range svc.ListTemplates() {
		templateID := tpl.ID
		doc, err := svc.CreateDocument(ctx, CreateDocumentInput{
			TenantID:   testTenant,
			Title:      tpl.Title,
			Module:     tpl.Module,
			TemplateID: &templateID,
			Rows:       json.RawMessage(tpl.Rows),
			Settings:   json.RawMessage(tpl.Settings),
		})
		if err != nil {
			t.Fatalf("CreateDocument from template %s: %v", tpl.ID, err)
		}
		if string(doc.Rows) != string(tpl.Rows) {
			t.Errorf("template %s: rows not stored verbatim", tpl.ID)
		}
	}
}

// compile-time interface check for mockRepository
var _ Repository = (*mockRepository)(nil)

// --- share tokens -----------------------------------------------------------

func (m *mockRepository) CreateShareToken(_ context.Context, t *ShareToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *t
	m.shares[t.ID] = &copy
	return nil
}

func (m *mockRepository) ListShareTokens(_ context.Context, tenantID, documentID uuid.UUID) ([]*ShareToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*ShareToken, 0, len(m.shares))
	for _, t := range m.shares {
		if t.TenantID != tenantID || t.DocumentID != documentID || t.RevokedAt != nil {
			continue
		}
		copy := *t
		out = append(out, &copy)
	}
	return out, nil
}

func (m *mockRepository) RevokeShareToken(_ context.Context, tenantID, shareID uuid.UUID, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.shares[shareID]
	if !ok || t.TenantID != tenantID || t.RevokedAt != nil {
		return ErrShareNotFound
	}
	t.RevokedAt = &at
	return nil
}

// GetShareTokenBySecret mirrors the production query: an exact match on the
// secret, across all tenants, with no tenant filter — that is the whole point
// of the system-context read it stands in for.
func (m *mockRepository) GetShareTokenBySecret(_ context.Context, secret string) (*ShareToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if secret == "" {
		return nil, ErrShareNotFound
	}
	for _, t := range m.shares {
		if t.Token == secret {
			copy := *t
			return &copy, nil
		}
	}
	return nil, ErrShareNotFound
}

func (m *mockRepository) IncrementShareView(_ context.Context, tenantID, shareID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.shares[shareID]
	if !ok || t.TenantID != tenantID {
		return ErrShareNotFound
	}
	t.ViewCount++
	return nil
}
