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
	def   *berichte.Definition
	sch   *berichte.Schedule
	doc   *berichte.Document
	share *berichte.ShareToken
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
func (r *stubBerichteRepo) UpsertCacheEntry(_ context.Context, _ *berichte.CacheEntry) error {
	return nil
}
func (r *stubBerichteRepo) InvalidateCache(_ context.Context, _, _ uuid.UUID) (int, error) {
	return 3, nil
}
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

func (r *stubBerichteRepo) CreateDocument(_ context.Context, doc *berichte.Document) error {
	r.doc = doc
	return nil
}
func (r *stubBerichteRepo) UpdateDocument(_ context.Context, doc *berichte.Document) error {
	r.doc = doc
	return nil
}
func (r *stubBerichteRepo) DeleteDocument(_ context.Context, _, _ uuid.UUID) error { return nil }
func (r *stubBerichteRepo) GetDocument(_ context.Context, _, _ uuid.UUID) (*berichte.Document, error) {
	if r.doc == nil {
		return nil, berichte.ErrDocumentNotFound
	}
	return r.doc, nil
}
func (r *stubBerichteRepo) ListDocuments(_ context.Context, _ uuid.UUID, _ berichte.ListDocumentsFilter, _, _ int) ([]*berichte.Document, int, error) {
	if r.doc == nil {
		return nil, 0, nil
	}
	return []*berichte.Document{r.doc}, 1, nil
}

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
	// A nil exporter factory is a server misconfiguration (the factory is wired
	// in cmd/berichte since WP-3), so it surfaces as Internal, not Unimplemented.
	assertGRPCCode(t, err, codes.Internal)
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
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		DefinitionID: uuid.New(),
		Trigger:      "manual",
		Status:       "success",
		StartedAt:    now,
		DurationMs:   &dur,
		RowCount:     &rows,
		CompletedAt:  &now,
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

func TestBerichteGRPCServer_ListTemplates(t *testing.T) {
	srv := newTestBerichteServerWithSvc(&stubBerichteRepo{}, nil)

	resp, err := srv.ListTemplates(context.Background(), &berichtev1.ListTemplatesRequest{})
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(resp.GetTemplates()) == 0 {
		t.Fatal("ListTemplates returned no templates")
	}
	for _, tpl := range resp.GetTemplates() {
		if tpl.GetId() == "" || tpl.GetModule() == "" {
			t.Fatalf("template missing id/module: %+v", tpl)
		}
		if len(tpl.GetRows()) == 0 {
			t.Fatalf("template %s: empty rows", tpl.GetId())
		}
	}
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

func (r *stubBerichteRepo) CreateShareToken(_ context.Context, t *berichte.ShareToken) error {
	r.share = t
	return nil
}

func (r *stubBerichteRepo) ListShareTokens(_ context.Context, _, _ uuid.UUID) ([]*berichte.ShareToken, error) {
	if r.share == nil {
		return nil, nil
	}
	return []*berichte.ShareToken{r.share}, nil
}

func (r *stubBerichteRepo) RevokeShareToken(_ context.Context, _, _ uuid.UUID, at time.Time) error {
	if r.share == nil || r.share.RevokedAt != nil {
		return berichte.ErrShareNotFound
	}
	r.share.RevokedAt = &at
	return nil
}

func (r *stubBerichteRepo) GetShareTokenBySecret(_ context.Context, secret string) (*berichte.ShareToken, error) {
	if r.share == nil || r.share.Token != secret {
		return nil, berichte.ErrShareNotFound
	}
	return r.share, nil
}

func (r *stubBerichteRepo) IncrementShareView(_ context.Context, _, _ uuid.UUID) error {
	if r.share == nil {
		return berichte.ErrShareNotFound
	}
	r.share.ViewCount++
	return nil
}

func TestBerichteGRPCServer_ExportDocumentPDF_HappyPath(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	repo := &stubBerichteRepo{doc: &berichte.Document{
		ID:       uuid.New(),
		TenantID: tenantID,
		Title:    "Q3 Bericht",
		Module:   "finanzen",
		Status:   "draft",
		Rows: []byte(`[{"columns":[{"width":1,"blocks":[
			{"id":"b1","type":"heading","level":1,"text":"Kennzahlen"},
			{"id":"b2","type":"kpi","label":"Umsatz","value":"12345"}
		]}]}]`),
		Settings: []byte(`{}`),
	}}
	s := newTestBerichteServerWithSvc(repo, nil)

	resp, err := s.ExportDocumentPDF(context.Background(), &berichtev1.ExportDocumentPDFRequest{
		TenantId:   tenantID.String(),
		DocumentId: repo.doc.ID.String(),
	})
	if err != nil {
		t.Fatalf("ExportDocumentPDF: %v", err)
	}
	if len(resp.GetPdfData()) < 4 || string(resp.GetPdfData()[:4]) != "%PDF" {
		t.Fatalf("expected a %%PDF-prefixed payload, got %d bytes", len(resp.GetPdfData()))
	}
	if resp.GetFilename() == "" {
		t.Error("expected a non-empty filename")
	}
}

func TestBerichteGRPCServer_ExportDocumentPDF_NotFound(t *testing.T) {
	t.Parallel()

	repo := &stubBerichteRepo{}
	s := newTestBerichteServerWithSvc(repo, nil)

	_, err := s.ExportDocumentPDF(context.Background(), &berichtev1.ExportDocumentPDFRequest{
		TenantId:   uuid.New().String(),
		DocumentId: uuid.New().String(),
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestBerichteGRPCServer_ExportDocumentPDF_InvalidTenantID(t *testing.T) {
	t.Parallel()

	s := newTestBerichteServer()
	_, err := s.ExportDocumentPDF(context.Background(), &berichtev1.ExportDocumentPDFRequest{
		TenantId:   "not-a-uuid",
		DocumentId: uuid.New().String(),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

// ============================================================================
// Document RPCs (multi-page authoring) — were 0% covered
// ============================================================================

func TestBerichteGRPCServer_CreateDocument_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	tenantID := uuid.New()
	resp, err := srv.CreateDocument(context.Background(), &berichtev1.CreateDocumentRequest{
		TenantId: tenantID.String(),
		Title:    "Quartalsbericht",
		Module:   "finanzen",
		Rows:     []byte(`[]`),
		Settings: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Document.TenantId != tenantID.String() {
		t.Errorf("tenant_id mismatch: got %s", resp.Document.TenantId)
	}
	if repo.doc == nil {
		t.Fatal("CreateDocument did not persist through the repository")
	}
}

func TestBerichteGRPCServer_CreateDocument_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.CreateDocument(context.Background(), &berichtev1.CreateDocumentRequest{
		TenantId: "not-a-uuid",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_CreateDocument_InvalidCreatedBy(t *testing.T) {
	srv := newTestBerichteServer()
	bad := "not-a-uuid"
	_, err := srv.CreateDocument(context.Background(), &berichtev1.CreateDocumentRequest{
		TenantId:  uuid.New().String(),
		Title:     "x",
		CreatedBy: &bad,
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_GetDocument_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	repo.doc = &berichte.Document{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Title:    "Bestehender Bericht",
		Module:   "crm",
		Status:   "draft",
		Rows:     []byte(`[]`),
		Settings: []byte(`{}`),
	}
	resp, err := srv.GetDocument(context.Background(), &berichtev1.GetDocumentRequest{
		TenantId:   repo.doc.TenantID.String(),
		DocumentId: repo.doc.ID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Document.Title != "Bestehender Bericht" {
		t.Errorf("title mismatch: got %q", resp.Document.Title)
	}
}

func TestBerichteGRPCServer_GetDocument_NotFound(t *testing.T) {
	srv, _ := newStubServer() // repo.doc is nil → ErrDocumentNotFound
	_, err := srv.GetDocument(context.Background(), &berichtev1.GetDocumentRequest{
		TenantId:   uuid.New().String(),
		DocumentId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestBerichteGRPCServer_GetDocument_InvalidIDs(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.GetDocument(context.Background(), &berichtev1.GetDocumentRequest{
		TenantId:   "bad",
		DocumentId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.GetDocument(context.Background(), &berichtev1.GetDocumentRequest{
		TenantId:   uuid.New().String(),
		DocumentId: "bad",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_UpdateDocument_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	repo.doc = &berichte.Document{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Title:    "Alt",
		Module:   "crm",
		Status:   "draft",
		Rows:     []byte(`[]`),
		Settings: []byte(`{}`),
	}
	newTitle := "Neu"
	resp, err := srv.UpdateDocument(context.Background(), &berichtev1.UpdateDocumentRequest{
		TenantId:   repo.doc.TenantID.String(),
		DocumentId: repo.doc.ID.String(),
		Title:      &newTitle,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Document.Title != "Neu" {
		t.Errorf("title not updated: got %q", resp.Document.Title)
	}
}

func TestBerichteGRPCServer_UpdateDocument_InvalidIDs(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.UpdateDocument(context.Background(), &berichtev1.UpdateDocumentRequest{
		TenantId:   "bad",
		DocumentId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_DeleteDocument_HappyPath(t *testing.T) {
	srv, _ := newStubServer()
	_, err := srv.DeleteDocument(context.Background(), &berichtev1.DeleteDocumentRequest{
		TenantId:   uuid.New().String(),
		DocumentId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBerichteGRPCServer_DeleteDocument_InvalidIDs(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.DeleteDocument(context.Background(), &berichtev1.DeleteDocumentRequest{
		TenantId:   uuid.New().String(),
		DocumentId: "bad",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_ListDocuments_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	repo.doc = &berichte.Document{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Title:    "test",
		Module:   "crm",
		Status:   "draft",
		Rows:     []byte(`[]`),
		Settings: []byte(`{}`),
	}
	resp, err := srv.ListDocuments(context.Background(), &berichtev1.ListDocumentsRequest{
		TenantId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Errorf("expected 1 document, got %d", len(resp.Documents))
	}
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
}

func TestBerichteGRPCServer_ListDocuments_Empty(t *testing.T) {
	srv, _ := newStubServer() // repo.doc is nil
	resp, err := srv.ListDocuments(context.Background(), &berichtev1.ListDocumentsRequest{
		TenantId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Wire-shape contract: an empty listing must serialize as [], not omit the
	// field / carry nil, which some JSON encodings would render as null.
	if resp.Documents == nil {
		t.Error("expected an empty (non-nil) slice, got nil")
	}
	if len(resp.Documents) != 0 {
		t.Errorf("expected 0 documents, got %d", len(resp.Documents))
	}
}

func TestBerichteGRPCServer_ListDocuments_InvalidTenantID(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.ListDocuments(context.Background(), &berichtev1.ListDocumentsRequest{
		TenantId: "bad",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteDocumentToProto_Nil(t *testing.T) {
	if got := documentToProto(nil); got != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestBerichteDocumentToProto_Fields(t *testing.T) {
	id, tenantID, createdBy := uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	released := now.Add(time.Hour)
	tmplID := "tmpl-1"
	d := &berichte.Document{
		ID: id, TenantID: tenantID, Title: "Titel", Module: "cross", Status: "released",
		Rows: []byte(`[]`), Settings: []byte(`{}`), TemplateID: &tmplID,
		CreatedBy: &createdBy, CreatedAt: now, UpdatedAt: now, ReleasedAt: &released,
	}
	proto := documentToProto(d)
	if proto.Id != id.String() || proto.TenantId != tenantID.String() {
		t.Errorf("id/tenant_id mismatch")
	}
	if proto.CreatedBy == nil || *proto.CreatedBy != createdBy.String() {
		t.Errorf("created_by mismatch")
	}
	if proto.TemplateId == nil || *proto.TemplateId != tmplID {
		t.Errorf("template_id mismatch")
	}
	if proto.ReleasedAt == nil {
		t.Errorf("released_at should be set")
	}
}

// ============================================================================
// Share token RPCs — were 0% covered
// ============================================================================

func TestBerichteGRPCServer_CreateShareToken_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	repo.doc = &berichte.Document{ID: uuid.New(), TenantID: uuid.New(), Title: "x", Module: "cross", Status: "final"}
	resp, err := srv.CreateShareToken(context.Background(), &berichtev1.CreateShareTokenRequest{
		TenantId:   repo.doc.TenantID.String(),
		DocumentId: repo.doc.ID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Share.Token == "" {
		t.Error("expected a non-empty token")
	}
	if resp.Share.HasPassword {
		t.Error("expected has_password=false, no password was requested")
	}
}

func TestBerichteGRPCServer_CreateShareToken_InvalidIDs(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.CreateShareToken(context.Background(), &berichtev1.CreateShareTokenRequest{
		TenantId:   "bad",
		DocumentId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_CreateShareToken_InvalidCreatedBy(t *testing.T) {
	srv, repo := newStubServer()
	repo.doc = &berichte.Document{ID: uuid.New(), TenantID: uuid.New(), Title: "x", Module: "cross", Status: "final"}
	_, err := srv.CreateShareToken(context.Background(), &berichtev1.CreateShareTokenRequest{
		TenantId:   repo.doc.TenantID.String(),
		DocumentId: repo.doc.ID.String(),
		CreatedBy:  ptrString("not-a-uuid"),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_CreateShareToken_DocumentNotFound(t *testing.T) {
	srv, _ := newStubServer() // repo.doc is nil
	_, err := srv.CreateShareToken(context.Background(), &berichtev1.CreateShareTokenRequest{
		TenantId:   uuid.New().String(),
		DocumentId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestBerichteGRPCServer_ListShareTokens_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	tenantID, docID := uuid.New(), uuid.New()
	repo.doc = &berichte.Document{ID: docID, TenantID: tenantID, Title: "x", Module: "cross", Status: "final"}
	repo.share = &berichte.ShareToken{ID: uuid.New(), TenantID: tenantID, DocumentID: docID, Token: "secret-token"}

	resp, err := srv.ListShareTokens(context.Background(), &berichtev1.ListShareTokensRequest{
		TenantId:   tenantID.String(),
		DocumentId: docID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(resp.Tokens))
	}
	if resp.Tokens[0].Token != "secret-token" {
		t.Errorf("token mismatch")
	}
}

func TestBerichteGRPCServer_ListShareTokens_InvalidIDs(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.ListShareTokens(context.Background(), &berichtev1.ListShareTokensRequest{
		TenantId:   uuid.New().String(),
		DocumentId: "bad",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_RevokeShareToken_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	tenantID := uuid.New()
	repo.share = &berichte.ShareToken{ID: uuid.New(), TenantID: tenantID, DocumentID: uuid.New(), Token: "secret-token"}

	_, err := srv.RevokeShareToken(context.Background(), &berichtev1.RevokeShareTokenRequest{
		TenantId: tenantID.String(),
		ShareId:  repo.share.ID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.share.RevokedAt == nil {
		t.Error("expected the share token to be revoked")
	}
}

func TestBerichteGRPCServer_RevokeShareToken_NotFound(t *testing.T) {
	srv, _ := newStubServer() // repo.share is nil
	_, err := srv.RevokeShareToken(context.Background(), &berichtev1.RevokeShareTokenRequest{
		TenantId: uuid.New().String(),
		ShareId:  uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestBerichteGRPCServer_RevokeShareToken_InvalidIDs(t *testing.T) {
	srv := newTestBerichteServer()
	_, err := srv.RevokeShareToken(context.Background(), &berichtev1.RevokeShareTokenRequest{
		TenantId: "bad",
		ShareId:  uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestBerichteGRPCServer_GetSharedDocument_HappyPath(t *testing.T) {
	srv, repo := newStubServer()
	tenantID, docID := uuid.New(), uuid.New()
	repo.doc = &berichte.Document{ID: docID, TenantID: tenantID, Title: "Geteilt", Module: "cross", Status: "final"}
	repo.share = &berichte.ShareToken{ID: uuid.New(), TenantID: tenantID, DocumentID: docID, Token: "public-secret"}

	resp, err := srv.GetSharedDocument(context.Background(), &berichtev1.GetSharedDocumentRequest{
		Token: "public-secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Document.Title != "Geteilt" {
		t.Errorf("title mismatch: got %q", resp.Document.Title)
	}
}

func TestBerichteGRPCServer_GetSharedDocument_NotFound(t *testing.T) {
	srv, _ := newStubServer() // repo.share is nil
	_, err := srv.GetSharedDocument(context.Background(), &berichtev1.GetSharedDocumentRequest{
		Token: "unknown-token",
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestBerichteReportShareTokenToProto_Nil(t *testing.T) {
	if got := reportShareTokenToProto(nil); got != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestBerichteReportShareTokenToProto_NeverLeaksPasswordHash(t *testing.T) {
	hash := "$2a$12$somebcrypthashvalue"
	expires := time.Now().UTC().Add(24 * time.Hour)
	tok := &berichte.ShareToken{
		ID: uuid.New(), DocumentID: uuid.New(), Token: "abc123",
		PasswordHash: &hash, ExpiresAt: &expires, ViewCount: 5, CreatedAt: time.Now().UTC(),
	}
	proto := reportShareTokenToProto(tok)
	if !proto.HasPassword {
		t.Error("expected has_password=true")
	}
	if proto.ViewCount != 5 {
		t.Errorf("view_count mismatch: got %d", proto.ViewCount)
	}
	if proto.ExpiresAt == nil {
		t.Error("expected expires_at to be set")
	}
	// The generated proto has no field to carry a password hash at all — this
	// test exists to catch the day someone adds one and forgets to leave it unset.
}
