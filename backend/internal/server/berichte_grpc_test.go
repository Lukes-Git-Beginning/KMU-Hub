package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/berichte"
	berichtev1 "github.com/kmuhub/kmuhub/proto/berichte/v1"
)

// berichte_grpc.go uses *berichte.Service directly (not an interface).
// We test the server's UUID-validation paths by calling gRPC handlers
// directly with a nil service (UUID parse errors happen before any service call).

// newTestBerichteServer returns a server with a nil service for UUID-validation tests.
func newTestBerichteServer() *BerichteGRPCServer {
	return NewBerichteGRPCServer(nil, nil)
}

// newTestBerichteServerWithSvc returns a server with a real service backed by the given repo.
func newTestBerichteServerWithSvc(repo berichte.Repository, exec berichte.Executor) *BerichteGRPCServer {
	svc := berichte.NewService(repo, exec, berichte.Options{})
	return NewBerichteGRPCServer(svc, nil)
}

// ============================================================================
// Stub repository for happy-path tests
// ============================================================================

// stubBerichteRepo is a minimal in-memory repository for service-layer tests.
type stubBerichteRepo struct {
	def  *berichte.Definition
	sch  *berichte.Schedule
}

func (r *stubBerichteRepo) CreateDefinition(_ context.Context, def *berichte.Definition) error {
	r.def = def
	return nil
}
func (r *stubBerichteRepo) UpdateDefinition(_ context.Context, def *berichte.Definition) error {
	r.def = def
	return nil
}
func (r *stubBerichteRepo) DeleteDefinition(_ context.Context, _, _ uuid.UUID) error { return nil }
func (r *stubBerichteRepo) GetDefinition(_ context.Context, _, _ uuid.UUID) (*berichte.Definition, error) {
	if r.def == nil {
		return nil, berichte.ErrDefinitionNotFound
	}
	return r.def, nil
}
func (r *stubBerichteRepo) ListDefinitions(_ context.Context, _ uuid.UUID, _ berichte.ListDefinitionsFilter, _, _ int) ([]*berichte.Definition, int, error) {
	if r.def == nil {
		return nil, 0, nil
	}
	return []*berichte.Definition{r.def}, 1, nil
}
func (r *stubBerichteRepo) GetCacheEntry(_ context.Context, _, _ uuid.UUID, _ string) (*berichte.CacheEntry, error) {
	return nil, berichte.ErrCacheMiss
}
func (r *stubBerichteRepo) UpsertCacheEntry(_ context.Context, _ *berichte.CacheEntry) error { return nil }
func (r *stubBerichteRepo) InvalidateCache(_ context.Context, _, _ uuid.UUID) (int, error)   { return 3, nil }
func (r *stubBerichteRepo) DeleteExpiredCacheEntries(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (r *stubBerichteRepo) CreateSchedule(_ context.Context, sch *berichte.Schedule) error {
	r.sch = sch
	return nil
}
func (r *stubBerichteRepo) UpdateSchedule(_ context.Context, sch *berichte.Schedule) error {
	r.sch = sch
	return nil
}
func (r *stubBerichteRepo) DeleteSchedule(_ context.Context, _, _ uuid.UUID) error { return nil }
func (r *stubBerichteRepo) GetSchedule(_ context.Context, _, _ uuid.UUID) (*berichte.Schedule, error) {
	if r.sch == nil {
		return nil, berichte.ErrScheduleNotFound
	}
	return r.sch, nil
}
func (r *stubBerichteRepo) ListSchedules(_ context.Context, _ uuid.UUID, _ berichte.ListSchedulesFilter, _, _ int) ([]*berichte.Schedule, int, error) {
	if r.sch == nil {
		return nil, 0, nil
	}
	return []*berichte.Schedule{r.sch}, 1, nil
}
func (r *stubBerichteRepo) ListDueSchedules(_ context.Context, _ time.Time) ([]*berichte.Schedule, error) {
	return nil, nil
}
func (r *stubBerichteRepo) ClaimSchedule(_ context.Context, _ uuid.UUID, _ *time.Time, _ time.Time) (bool, error) {
	return false, nil
}
func (r *stubBerichteRepo) UpdateScheduleLastRun(_ context.Context, _ uuid.UUID, _ string, _ *string, _ time.Time) error {
	return nil
}
func (r *stubBerichteRepo) InsertRun(_ context.Context, _ *berichte.Run) error { return nil }

// ============================================================================
// UUID-Validation tests — these never reach the service layer
// ============================================================================

func TestBerichteGRPCServer_CreateDefinition_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.CreateDefinition(context.Background(), &berichtev1.CreateDefinitionRequest{
		TenantId: "not-a-uuid",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_CreateDefinition_EmptyTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.CreateDefinition(context.Background(), &berichtev1.CreateDefinitionRequest{
		TenantId: "",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_GetDefinition_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.GetDefinition(context.Background(), &berichtev1.GetDefinitionRequest{
		TenantId:     "bad",
		DefinitionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_GetDefinition_InvalidDefinitionID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.GetDefinition(context.Background(), &berichtev1.GetDefinitionRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: "not-valid",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_UpdateDefinition_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.UpdateDefinition(context.Background(), &berichtev1.UpdateDefinitionRequest{
		TenantId:     "oops",
		DefinitionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_UpdateDefinition_InvalidDefinitionID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.UpdateDefinition(context.Background(), &berichtev1.UpdateDefinitionRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: "oops",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_DeleteDefinition_InvalidIDs(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.DeleteDefinition(context.Background(), &berichtev1.DeleteDefinitionRequest{
		TenantId:     "bad",
		DefinitionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_ListDefinitions_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.ListDefinitions(context.Background(), &berichtev1.ListDefinitionsRequest{
		TenantId: "garbage",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_RunReport_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.RunReport(context.Background(), &berichtev1.RunReportRequest{
		TenantId:     "x",
		DefinitionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_RunReport_InvalidDefinitionID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.RunReport(context.Background(), &berichtev1.RunReportRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: "x",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_GetCachedResult_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.GetCachedResult(context.Background(), &berichtev1.GetCachedResultRequest{
		TenantId:     "bad",
		DefinitionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_InvalidateCache_InvalidIDs(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.InvalidateCache(context.Background(), &berichtev1.InvalidateCacheRequest{
		TenantId:     "bad",
		DefinitionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_ExportReport_NilFactory(t *testing.T) {
	srv := newTestBerichteServer() // exporterFactory == nil
	_, err := srv.ExportReport(context.Background(), &berichtev1.ExportReportRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: uuid.New().String(),
		Format:       "pdf",
	})
	assertGRPCCode(t, err, codes.Unimplemented)
}

func TestBerichteGRPCServer_ExportReport_InvalidTenantID(t *testing.T) {
	stub := func(format string) (BerichteExporter, error) {
		return nil, nil
	}
	srv := NewBerichteGRPCServer(nil, stub)
	_, err := srv.ExportReport(context.Background(), &berichtev1.ExportReportRequest{
		TenantId:     "bad",
		DefinitionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_ExportReport_UnsupportedFormat(t *testing.T) {
	stub := func(format string) (BerichteExporter, error) {
		return nil, errors.New("unsupported format")
	}
	srv := NewBerichteGRPCServer(nil, stub)
	_, err := srv.ExportReport(context.Background(), &berichtev1.ExportReportRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: uuid.New().String(),
		Format:       "docx",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_CreateSchedule_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.CreateSchedule(context.Background(), &berichtev1.CreateScheduleRequest{
		TenantId:     "bad",
		DefinitionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_CreateSchedule_InvalidDefinitionID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.CreateSchedule(context.Background(), &berichtev1.CreateScheduleRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: "bad",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_UpdateSchedule_InvalidScheduleID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.UpdateSchedule(context.Background(), &berichtev1.UpdateScheduleRequest{
		TenantId:   uuid.New().String(),
		ScheduleId: "bad",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_DeleteSchedule_InvalidIDs(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.DeleteSchedule(context.Background(), &berichtev1.DeleteScheduleRequest{
		TenantId:   "bad",
		ScheduleId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_ListSchedules_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.ListSchedules(context.Background(), &berichtev1.ListSchedulesRequest{
		TenantId: "bad",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_ListSchedules_InvalidDefinitionID(t *testing.T) {
	srv := newTestBerichteServer()
	badID := "not-a-uuid"
	_, err := srv.ListSchedules(context.Background(), &berichtev1.ListSchedulesRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: &badID,
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_ToggleSchedule_InvalidIDs(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.ToggleSchedule(context.Background(), &berichtev1.ToggleScheduleRequest{
		TenantId:   "bad",
		ScheduleId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_GetDashboardKPIs_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.GetDashboardKPIs(context.Background(), &berichtev1.DashboardKPIsRequest{
		TenantId: "bad",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Error-mapping tests
// ============================================================================

func TestBerichteMapError_Nil(t *testing.T) {
	if got := mapBerichteError(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestBerichteMapError_DefinitionNotFound(t *testing.T) {
	assertGRPCCode(t, mapBerichteError(berichte.ErrDefinitionNotFound), codes.NotFound)
}

func TestBerichteMapError_ScheduleNotFound(t *testing.T) {
	assertGRPCCode(t, mapBerichteError(berichte.ErrScheduleNotFound), codes.NotFound)
}

func TestBerichteMapError_CacheMiss(t *testing.T) {
	assertGRPCCode(t, mapBerichteError(berichte.ErrCacheMiss), codes.NotFound)
}

func TestBerichteMapError_InvalidQueryConfig(t *testing.T) {
	assertGRPCCode(t, mapBerichteError(berichte.ErrInvalidQueryConfig), codes.InvalidArgument)
}

func TestBerichteMapError_InvalidCron(t *testing.T) {
	assertGRPCCode(t, mapBerichteError(berichte.ErrInvalidCron), codes.InvalidArgument)
}

func TestBerichteMapError_InvalidFormat(t *testing.T) {
	assertGRPCCode(t, mapBerichteError(berichte.ErrInvalidFormat), codes.InvalidArgument)
}

func TestBerichteMapError_InvalidModule(t *testing.T) {
	assertGRPCCode(t, mapBerichteError(berichte.ErrInvalidModule), codes.InvalidArgument)
}

func TestBerichteMapError_InvalidKind(t *testing.T) {
	assertGRPCCode(t, mapBerichteError(berichte.ErrInvalidKind), codes.InvalidArgument)
}

func TestBerichteMapError_ExecutorUnavailable(t *testing.T) {
	assertGRPCCode(t, mapBerichteError(berichte.ErrExecutorUnavailable), codes.Unavailable)
}

func TestBerichteMapError_Unknown(t *testing.T) {
	assertGRPCCode(t, mapBerichteError(errors.New("some internal error")), codes.Internal)
}

// ============================================================================
// Conversion helper tests
// ============================================================================

func TestBerichteDefinitionToProto_Nil(t *testing.T) {
	if got := definitionToProto(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %v", got)
	}
}

func TestBerichteDefinitionToProto_Fields(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	createdBy := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	d := &berichte.Definition{
		ID:            id,
		TenantID:      tenantID,
		Name:          "Test Bericht",
		Description:   "desc",
		Module:        "finanzen",
		Kind:          "custom",
		QueryConfig:   []byte(`{"kind":"monthly_revenue"}`),
		DefaultFormat: "pdf",
		CreatedBy:     &createdBy,
		IsPublished:   true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	proto := definitionToProto(d)
	if proto.Id != id.String() {
		t.Errorf("id mismatch: got %s want %s", proto.Id, id.String())
	}
	if proto.TenantId != tenantID.String() {
		t.Errorf("tenant_id mismatch")
	}
	if proto.Name != "Test Bericht" {
		t.Errorf("name mismatch")
	}
	if proto.CreatedBy == nil || *proto.CreatedBy != createdBy.String() {
		t.Errorf("created_by mismatch")
	}
	if !proto.IsPublished {
		t.Errorf("is_published should be true")
	}
}

func TestBerichteDefinitionToProto_NilCreatedBy(t *testing.T) {
	d := &berichte.Definition{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "x",
	}
	proto := definitionToProto(d)
	if proto.CreatedBy != nil {
		t.Errorf("expected nil created_by")
	}
}

func TestBerichteScheduleToProto_Nil(t *testing.T) {
	if got := scheduleToProto(nil); got != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestBerichteScheduleToProto_LastRunAt(t *testing.T) {
	now := time.Now().UTC()
	sc := &berichte.Schedule{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		DefinitionID:   uuid.New(),
		Name:           "daily",
		CronExpression: "0 8 * * *",
		LastRunAt:      &now,
	}
	proto := scheduleToProto(sc)
	if proto.LastRunAt == nil {
		t.Errorf("expected LastRunAt to be set")
	}
}

func TestBerichteRunToProto_Nil(t *testing.T) {
	if got := runToProto(nil); got != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestBerichteRunToProto_OptionalFields(t *testing.T) {
	dur := 42
	rows := 7
	now := time.Now().UTC()
	r := &berichte.Run{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		DefinitionID: uuid.New(),
		Trigger:     "manual",
		Status:      "success",
		StartedAt:   now,
		DurationMs:  &dur,
		RowCount:    &rows,
		CompletedAt: &now,
	}
	proto := runToProto(r)
	if proto.DurationMs == nil || *proto.DurationMs != 42 {
		t.Errorf("duration_ms mismatch")
	}
	if proto.RowCount == nil || *proto.RowCount != 7 {
		t.Errorf("row_count mismatch")
	}
	if proto.CompletedAt == nil {
		t.Errorf("completed_at should be set")
	}
}

func TestBerichteResultToProto_Nil(t *testing.T) {
	if got := resultToProto(nil); got != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestBerichteResultToProto_FromCache(t *testing.T) {
	r := &berichte.ReportResult{
		Meta: berichte.ReportMeta{
			FromCache:   true,
			RowCount:    5,
			GeneratedAt: time.Now(),
		},
	}
	proto := resultToProto(r)
	if !proto.FromCache {
		t.Errorf("expected from_cache=true")
	}
	if proto.RowCount != 5 {
		t.Errorf("row_count mismatch: got %d want 5", proto.RowCount)
	}
}

func TestBerichteKpiToProto_Nil(t *testing.T) {
	if got := kpiToProto(nil); got != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestBerichteKpiToProto_WithChangePercent(t *testing.T) {
	pct := 12.5
	k := &berichte.KPI{
		ID:            "revenue_total",
		Label:         "Umsatz gesamt",
		Value:         "42.000 EUR",
		Unit:          "EUR",
		ChangePercent: &pct,
		ModuleID:      "finanzen",
	}
	proto := kpiToProto(k)
	if proto.ChangePercent == nil || *proto.ChangePercent != 12.5 {
		t.Errorf("change_percent mismatch")
	}
	if proto.ModuleId != "finanzen" {
		t.Errorf("module_id mismatch")
	}
}

// ============================================================================
// Happy-path tests using stub repository
// ============================================================================

func newStubServer() (*BerichteGRPCServer, *stubBerichteRepo) {
	repo := &stubBerichteRepo{}
	srv := newTestBerichteServerWithSvc(repo, nil)
	return srv, repo
}

func TestBerichteGRPCServer_CreateDefinition_HappyPath(t *testing.T) {
	srv, _ := newStubServer()
	tenantID := uuid.New()
	_, err := srv.CreateDefinition(context.Background(), &berichtev1.CreateDefinitionRequest{
		TenantId:    tenantID.String(),
		Name:        "Umsatzbericht",
		Module:      "finanzen",
		Kind:        "custom",
		QueryConfig: []byte(`{"kind":"monthly_revenue"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBerichteGRPCServer_GetDefinition_NotFound(t *testing.T) {
	srv, _ := newStubServer() // repo.def is nil → ErrDefinitionNotFound
	_, err := srv.GetDefinition(context.Background(), &berichtev1.GetDefinitionRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestBerichteGRPCServer_ListDefinitions_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	repo.def = &berichte.Definition{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "test",
		Module:   "crm",
		Kind:     "system",
	}
	resp, err := srv.ListDefinitions(context.Background(), &berichtev1.ListDefinitionsRequest{
		TenantId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Definitions) != 1 {
		t.Errorf("expected 1 definition, got %d", len(resp.Definitions))
	}
}

func TestBerichteGRPCServer_DeleteDefinition_HappyPath(t *testing.T) {
	srv, _ := newStubServer()
	_, err := srv.DeleteDefinition(context.Background(), &berichtev1.DeleteDefinitionRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBerichteGRPCServer_InvalidateCache_HappyPath(t *testing.T) {
	srv, _ := newStubServer()
	resp, err := srv.InvalidateCache(context.Background(), &berichtev1.InvalidateCacheRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Evicted != 3 {
		t.Errorf("expected 3 evicted, got %d", resp.Evicted)
	}
}

func TestBerichteGRPCServer_CreateSchedule_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	repo.def = &berichte.Definition{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "rev",
		Module:   "finanzen",
		Kind:     "system",
	}
	_, err := srv.CreateSchedule(context.Background(), &berichtev1.CreateScheduleRequest{
		TenantId:       repo.def.TenantID.String(),
		DefinitionId:   repo.def.ID.String(),
		Name:           "daily",
		CronExpression: "0 8 * * *",
		Format:         "pdf",
		Active:         true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBerichteGRPCServer_DeleteSchedule_HappyPath(t *testing.T) {
	srv, _ := newStubServer()
	_, err := srv.DeleteSchedule(context.Background(), &berichtev1.DeleteScheduleRequest{
		TenantId:   uuid.New().String(),
		ScheduleId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBerichteGRPCServer_ListSchedules_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	repo.sch = &berichte.Schedule{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		DefinitionID:   uuid.New(),
		Name:           "weekly",
		CronExpression: "0 9 * * 1",
		Format:         "csv",
	}
	resp, err := srv.ListSchedules(context.Background(), &berichtev1.ListSchedulesRequest{
		TenantId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Schedules) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(resp.Schedules))
	}
}

func TestBerichteGRPCServer_ToggleSchedule_NotFound(t *testing.T) {
	srv, _ := newStubServer() // repo.sch is nil → ErrScheduleNotFound
	_, err := srv.ToggleSchedule(context.Background(), &berichtev1.ToggleScheduleRequest{
		TenantId:   uuid.New().String(),
		ScheduleId: uuid.New().String(),
		Active:     true,
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestBerichteGRPCServer_UpdateSchedule_NotFound(t *testing.T) {
	srv, _ := newStubServer()
	_, err := srv.UpdateSchedule(context.Background(), &berichtev1.UpdateScheduleRequest{
		TenantId:   uuid.New().String(),
		ScheduleId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestBerichteGRPCServer_GetCachedResult_CacheMiss(t *testing.T) {
	srv, _ := newStubServer()
	_, err := srv.GetCachedResult(context.Background(), &berichtev1.GetCachedResultRequest{
		TenantId:     uuid.New().String(),
		DefinitionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestBerichteGRPCServer_GetDashboardKPIs_NoExecutor(t *testing.T) {
	srv, _ := newStubServer() // executor is nil → ErrExecutorUnavailable
	_, err := srv.GetDashboardKPIs(context.Background(), &berichtev1.DashboardKPIsRequest{
		TenantId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.Unavailable)
}

// ============================================================================
// Helper
// ============================================================================

func assertGRPCCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %v, got nil", want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != want {
		t.Errorf("gRPC code mismatch: got %v want %v (msg: %s)", st.Code(), want, st.Message())
	}
}
