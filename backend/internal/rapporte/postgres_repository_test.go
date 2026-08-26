package rapporte

// Covers the PostgresRepository methods left untested by tenant_write_test.go
// (plain INSERT paths), own_scope_list_test.go (ListReports via the service)
// and signature_test.go (SaveSignature): Update/SoftDelete/Get across
// reports, lines, attachments, workers, measurements and templates, plus the
// atomic approve/reject transitions and the GROUP BY stats query. Runs
// against the real schema, not a mock, so a forgotten WHERE clause or
// mismatched column shows up as a genuine query failure or wrong row count.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func setupRapporteRepo(t *testing.T) (*PostgresRepository, *pgxpool.Pool, context.Context, uuid.UUID) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Rapporte Repo Test Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	return repo, pool, ctx, tenantID
}

func newTestReport(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, title string, status ReportStatus) *WorkReport {
	t.Helper()
	now := time.Now().UTC()
	report := &WorkReport{
		ID: uuid.New(), TenantID: tenantID, Title: title, Status: status,
		AuthorID: uuid.New(), ReportDate: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateReport(ctx, report); err != nil {
		t.Fatalf("seed report %s: %v", title, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "work_reports", report.ID) })
	return report
}

// ============================================================================
// Reports
// ============================================================================

func TestUpdateReport_UpdatesFieldsAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle A", StatusDraft)

	report.Title = "Baustelle A (revidiert)"
	report.Status = StatusSubmitted
	report.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateReport(ctx, report); err != nil {
		t.Fatalf("UpdateReport: %v", err)
	}

	reloaded, err := repo.GetReport(ctx, tenantID, report.ID)
	if err != nil {
		t.Fatalf("GetReport after update: %v", err)
	}
	if reloaded.Title != "Baustelle A (revidiert)" || reloaded.Status != StatusSubmitted {
		t.Fatalf("update did not persist: got title=%q status=%q", reloaded.Title, reloaded.Status)
	}

	unknown := &WorkReport{ID: uuid.New(), TenantID: tenantID, Title: "x", UpdatedAt: time.Now().UTC()}
	if err := repo.UpdateReport(ctx, unknown); !errors.Is(err, ErrReportNotFound) {
		t.Fatalf("expected ErrReportNotFound for unknown report, got %v", err)
	}
}

func TestSoftDeleteReport_HidesRowAndRejectsDoubleDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle B", StatusDraft)

	if err := repo.SoftDeleteReport(ctx, tenantID, report.ID); err != nil {
		t.Fatalf("SoftDeleteReport: %v", err)
	}

	if _, err := repo.GetReport(ctx, tenantID, report.ID); !errors.Is(err, ErrReportNotFound) {
		t.Fatalf("expected ErrReportNotFound after soft delete, got %v", err)
	}

	if err := repo.SoftDeleteReport(ctx, tenantID, report.ID); !errors.Is(err, ErrReportNotFound) {
		t.Fatalf("expected ErrReportNotFound on double delete, got %v", err)
	}
}

func TestGetReport_LoadsWorkersAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle C", StatusDraft)
	worker, err := repo.AddWorker(ctx, tenantID, report.ID, "Max Muster", "Monteur", 6)
	if err != nil {
		t.Fatalf("AddWorker: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_workers", worker.ID) })

	reloaded, err := repo.GetReport(ctx, tenantID, report.ID)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if len(reloaded.Workers) != 1 || reloaded.Workers[0].Name != "Max Muster" {
		t.Fatalf("expected the seeded worker to be loaded, got %+v", reloaded.Workers)
	}

	if _, err := repo.GetReport(ctx, tenantID, uuid.New()); !errors.Is(err, ErrReportNotFound) {
		t.Fatalf("expected ErrReportNotFound for unknown report, got %v", err)
	}
}

func TestAtomicApproveReport_TransitionsOnlyFromSubmitted(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	submitted := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle D", StatusSubmitted)
	reviewer := uuid.New()

	ok, err := repo.AtomicApproveReport(ctx, tenantID, submitted.ID, reviewer, "passt")
	if err != nil {
		t.Fatalf("AtomicApproveReport: %v", err)
	}
	if !ok {
		t.Fatalf("expected AtomicApproveReport to transition a submitted report")
	}
	reloaded, err := repo.GetReport(ctx, tenantID, submitted.ID)
	if err != nil {
		t.Fatalf("GetReport after approve: %v", err)
	}
	if reloaded.Status != StatusApproved || reloaded.ReviewerID == nil || *reloaded.ReviewerID != reviewer {
		t.Fatalf("approve did not persist: status=%q reviewer=%v", reloaded.Status, reloaded.ReviewerID)
	}

	// Already approved: the atomic UPDATE must no-op instead of re-approving.
	ok, err = repo.AtomicApproveReport(ctx, tenantID, submitted.ID, reviewer, "nochmal")
	if err != nil {
		t.Fatalf("AtomicApproveReport (already approved): %v", err)
	}
	if ok {
		t.Fatalf("expected AtomicApproveReport to reject a report that is not in submitted state")
	}

	draft := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle E", StatusDraft)
	ok, err = repo.AtomicApproveReport(ctx, tenantID, draft.ID, reviewer, "zu frueh")
	if err != nil {
		t.Fatalf("AtomicApproveReport (draft): %v", err)
	}
	if ok {
		t.Fatalf("expected AtomicApproveReport to reject a draft report")
	}
}

func TestAtomicRejectReport_TransitionsOnlyFromSubmitted(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	submitted := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle F", StatusSubmitted)
	reviewer := uuid.New()

	ok, err := repo.AtomicRejectReport(ctx, tenantID, submitted.ID, reviewer, "unvollstaendig")
	if err != nil {
		t.Fatalf("AtomicRejectReport: %v", err)
	}
	if !ok {
		t.Fatalf("expected AtomicRejectReport to transition a submitted report")
	}
	reloaded, err := repo.GetReport(ctx, tenantID, submitted.ID)
	if err != nil {
		t.Fatalf("GetReport after reject: %v", err)
	}
	if reloaded.Status != StatusRejected {
		t.Fatalf("reject did not persist: status=%q", reloaded.Status)
	}

	ok, err = repo.AtomicRejectReport(ctx, tenantID, submitted.ID, reviewer, "nochmal")
	if err != nil {
		t.Fatalf("AtomicRejectReport (already rejected): %v", err)
	}
	if ok {
		t.Fatalf("expected AtomicRejectReport to reject a report that is not in submitted state")
	}
}

func TestGetReportStatsCounts_GroupsByStatus(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	newTestReport(t, repo, ctx, pool, tenantID, "Stats Draft 1", StatusDraft)
	newTestReport(t, repo, ctx, pool, tenantID, "Stats Draft 2", StatusDraft)
	newTestReport(t, repo, ctx, pool, tenantID, "Stats Submitted", StatusSubmitted)
	deleted := newTestReport(t, repo, ctx, pool, tenantID, "Stats Deleted", StatusApproved)
	if err := repo.SoftDeleteReport(ctx, tenantID, deleted.ID); err != nil {
		t.Fatalf("SoftDeleteReport: %v", err)
	}

	counts, err := repo.GetReportStatsCounts(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetReportStatsCounts: %v", err)
	}
	if counts[string(StatusDraft)] != 2 {
		t.Fatalf("expected 2 draft reports, got %d", counts[string(StatusDraft)])
	}
	if counts[string(StatusSubmitted)] != 1 {
		t.Fatalf("expected 1 submitted report, got %d", counts[string(StatusSubmitted)])
	}
	if _, deletedCounted := counts[string(StatusApproved)]; deletedCounted {
		t.Fatalf("soft-deleted report must not be counted, got approved=%d", counts[string(StatusApproved)])
	}
}

// ============================================================================
// Lines
// ============================================================================

func newTestLine(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID, reportID uuid.UUID, position int, description string) *ReportLine {
	t.Helper()
	now := time.Now().UTC()
	line := &ReportLine{
		ID: uuid.New(), TenantID: tenantID, ReportID: reportID, Position: position,
		Description: description, Quantity: 1, Unit: "Stk", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateLine(ctx, line); err != nil {
		t.Fatalf("seed line %s: %v", description, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_lines", line.ID) })
	return line
}

func TestUpdateLine_UpdatesFieldsAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle G", StatusDraft)
	line := newTestLine(t, repo, ctx, pool, tenantID, report.ID, 1, "Montage")

	line.Description = "Montage (angepasst)"
	line.Quantity = 5
	line.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateLine(ctx, line); err != nil {
		t.Fatalf("UpdateLine: %v", err)
	}

	reloaded, err := repo.GetLine(ctx, tenantID, line.ID)
	if err != nil {
		t.Fatalf("GetLine after update: %v", err)
	}
	if reloaded.Description != "Montage (angepasst)" || reloaded.Quantity != 5 {
		t.Fatalf("update did not persist: got %+v", reloaded)
	}

	unknown := &ReportLine{ID: uuid.New(), TenantID: tenantID, UpdatedAt: time.Now().UTC()}
	if err := repo.UpdateLine(ctx, unknown); !errors.Is(err, ErrLineNotFound) {
		t.Fatalf("expected ErrLineNotFound for unknown line, got %v", err)
	}
}

func TestDeleteLine_RemovesRowAndRejectsDoubleDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle H", StatusDraft)
	line := newTestLine(t, repo, ctx, pool, tenantID, report.ID, 1, "Demontage")

	if err := repo.DeleteLine(ctx, tenantID, line.ID); err != nil {
		t.Fatalf("DeleteLine: %v", err)
	}
	if _, err := repo.GetLine(ctx, tenantID, line.ID); !errors.Is(err, ErrLineNotFound) {
		t.Fatalf("expected ErrLineNotFound after delete, got %v", err)
	}
	if err := repo.DeleteLine(ctx, tenantID, line.ID); !errors.Is(err, ErrLineNotFound) {
		t.Fatalf("expected ErrLineNotFound on double delete, got %v", err)
	}
}

func TestGetLine_ReturnsRowAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle I", StatusDraft)
	line := newTestLine(t, repo, ctx, pool, tenantID, report.ID, 1, "Verkabelung")

	reloaded, err := repo.GetLine(ctx, tenantID, line.ID)
	if err != nil {
		t.Fatalf("GetLine: %v", err)
	}
	if reloaded.ID != line.ID || reloaded.Description != "Verkabelung" {
		t.Fatalf("unexpected line: %+v", reloaded)
	}

	if _, err := repo.GetLine(ctx, tenantID, uuid.New()); !errors.Is(err, ErrLineNotFound) {
		t.Fatalf("expected ErrLineNotFound for unknown line, got %v", err)
	}
}

func TestListLines_OrdersByPositionAndScopesByReport(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	reportA := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle J", StatusDraft)
	reportB := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle K", StatusDraft)

	newTestLine(t, repo, ctx, pool, tenantID, reportA.ID, 2, "Zweite Position")
	newTestLine(t, repo, ctx, pool, tenantID, reportA.ID, 1, "Erste Position")
	newTestLine(t, repo, ctx, pool, tenantID, reportB.ID, 1, "Fremder Bericht")

	lines, err := repo.ListLines(ctx, tenantID, reportA.ID)
	if err != nil {
		t.Fatalf("ListLines: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines scoped to reportA, got %d", len(lines))
	}
	if lines[0].Position != 1 || lines[1].Position != 2 {
		t.Fatalf("expected lines ordered by position, got %d then %d", lines[0].Position, lines[1].Position)
	}
}

// ============================================================================
// Attachments
// ============================================================================

func newTestAttachment(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID, reportID uuid.UUID, lineID *uuid.UUID, filename string) *ReportAttachment {
	t.Helper()
	att := &ReportAttachment{
		ID: uuid.New(), TenantID: tenantID, ReportID: reportID, LineID: lineID,
		Filename: filename, ContentType: "image/jpeg", SizeBytes: 2048,
		ObjectKey: "rapporte/" + uuid.New().String() + "/" + filename, UploadedBy: uuid.New(),
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateAttachment(ctx, att); err != nil {
		t.Fatalf("seed attachment %s: %v", filename, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_attachments", att.ID) })
	return att
}

func TestGetAttachment_ReturnsRowAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle L", StatusDraft)
	att := newTestAttachment(t, repo, ctx, pool, tenantID, report.ID, nil, "foto.jpg")

	reloaded, err := repo.GetAttachment(ctx, tenantID, att.ID)
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if reloaded.Filename != "foto.jpg" {
		t.Fatalf("unexpected attachment: %+v", reloaded)
	}

	if _, err := repo.GetAttachment(ctx, tenantID, uuid.New()); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("expected ErrAttachmentNotFound for unknown attachment, got %v", err)
	}
}

func TestDeleteAttachment_RemovesRowAndRejectsDoubleDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle M", StatusDraft)
	att := newTestAttachment(t, repo, ctx, pool, tenantID, report.ID, nil, "plan.pdf")

	if err := repo.DeleteAttachment(ctx, tenantID, att.ID); err != nil {
		t.Fatalf("DeleteAttachment: %v", err)
	}
	if err := repo.DeleteAttachment(ctx, tenantID, att.ID); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("expected ErrAttachmentNotFound on double delete, got %v", err)
	}
}

func TestListAttachments_FiltersByLineIDAndScopesByReport(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	reportA := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle N", StatusDraft)
	reportB := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle O", StatusDraft)
	line := newTestLine(t, repo, ctx, pool, tenantID, reportA.ID, 1, "Position mit Foto")

	onLine := newTestAttachment(t, repo, ctx, pool, tenantID, reportA.ID, &line.ID, "linien-foto.jpg")
	newTestAttachment(t, repo, ctx, pool, tenantID, reportA.ID, nil, "bericht-foto.jpg")
	newTestAttachment(t, repo, ctx, pool, tenantID, reportB.ID, nil, "fremder-bericht.jpg")

	all, err := repo.ListAttachments(ctx, tenantID, reportA.ID, nil)
	if err != nil {
		t.Fatalf("ListAttachments (unfiltered): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 attachments scoped to reportA, got %d", len(all))
	}

	scoped, err := repo.ListAttachments(ctx, tenantID, reportA.ID, &line.ID)
	if err != nil {
		t.Fatalf("ListAttachments (line filter): %v", err)
	}
	if len(scoped) != 1 || scoped[0].ID != onLine.ID {
		t.Fatalf("expected exactly the line-scoped attachment, got %+v", scoped)
	}
}

// ============================================================================
// Workers
// ============================================================================

func TestRemoveWorker_RemovesRowAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle P", StatusDraft)

	worker, err := repo.AddWorker(ctx, tenantID, report.ID, "Erika Mustermann", "Elektrikerin", 4)
	if err != nil {
		t.Fatalf("AddWorker: %v", err)
	}

	before, err := repo.ListWorkers(ctx, tenantID, report.ID)
	if err != nil {
		t.Fatalf("ListWorkers before removal: %v", err)
	}
	if len(before) != 1 || before[0].Name != "Erika Mustermann" {
		t.Fatalf("expected exactly the seeded worker, got %+v", before)
	}

	if err := repo.RemoveWorker(ctx, tenantID, worker.ID); err != nil {
		t.Fatalf("RemoveWorker: %v", err)
	}

	after, err := repo.ListWorkers(ctx, tenantID, report.ID)
	if err != nil {
		t.Fatalf("ListWorkers after removal: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected no workers left, got %+v", after)
	}

	if err := repo.RemoveWorker(ctx, tenantID, worker.ID); !errors.Is(err, ErrWorkerNotFound) {
		t.Fatalf("expected ErrWorkerNotFound on double remove, got %v", err)
	}
}

// ============================================================================
// Measurements
// ============================================================================

func TestGetMeasurement_LoadsPositionsAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle Q", StatusDraft)

	measurement, err := repo.CreateMeasurement(ctx, tenantID, &report.ID, "Aufmass Halle", "Halle B", "Max Muster", "2026-08-01", "")
	if err != nil {
		t.Fatalf("CreateMeasurement: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurements", measurement.ID) })

	position, err := repo.AddMeasurementPosition(ctx, tenantID, measurement.ID, 1, "Wandflaeche", "m2", 10, 8.5)
	if err != nil {
		t.Fatalf("AddMeasurementPosition: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurement_positions", position.ID) })

	reloaded, err := repo.GetMeasurement(ctx, tenantID, measurement.ID)
	if err != nil {
		t.Fatalf("GetMeasurement: %v", err)
	}
	if len(reloaded.Positions) != 1 || reloaded.Positions[0].Description != "Wandflaeche" {
		t.Fatalf("expected the seeded position to be loaded, got %+v", reloaded.Positions)
	}

	if _, err := repo.GetMeasurement(ctx, tenantID, uuid.New()); !errors.Is(err, ErrMeasurementNotFound) {
		t.Fatalf("expected ErrMeasurementNotFound for unknown measurement, got %v", err)
	}
}

func TestListMeasurements_FiltersByReportAndPaginates(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	report := newTestReport(t, repo, ctx, pool, tenantID, "Baustelle R", StatusDraft)

	inReport, err := repo.CreateMeasurement(ctx, tenantID, &report.ID, "Aufmass 1", "", "", "", "")
	if err != nil {
		t.Fatalf("CreateMeasurement (in report): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurements", inReport.ID) })

	standalone, err := repo.CreateMeasurement(ctx, tenantID, nil, "Aufmass ohne Bericht", "", "", "", "")
	if err != nil {
		t.Fatalf("CreateMeasurement (standalone): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurements", standalone.ID) })

	scoped, total, err := repo.ListMeasurements(ctx, tenantID, &report.ID, 1, 50)
	if err != nil {
		t.Fatalf("ListMeasurements (report filter): %v", err)
	}
	if total != 1 || len(scoped) != 1 || scoped[0].ID != inReport.ID {
		t.Fatalf("expected exactly the report-scoped measurement, got total=%d rows=%+v", total, scoped)
	}

	all, total, err := repo.ListMeasurements(ctx, tenantID, nil, 1, 50)
	if err != nil {
		t.Fatalf("ListMeasurements (unfiltered): %v", err)
	}
	if total != 2 || len(all) != 2 {
		t.Fatalf("expected 2 measurements tenant-wide, got total=%d rows=%d", total, len(all))
	}

	page, total, err := repo.ListMeasurements(ctx, tenantID, nil, 1, 1)
	if err != nil {
		t.Fatalf("ListMeasurements (page 1, size 1): %v", err)
	}
	if total != 2 || len(page) != 1 {
		t.Fatalf("expected total=2 with 1 row on a size-1 page, got total=%d rows=%d", total, len(page))
	}
}

func TestUpdateMeasurement_UpdatesFieldsAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	measurement, err := repo.CreateMeasurement(ctx, tenantID, nil, "Aufmass S", "Halle C", "", "", "")
	if err != nil {
		t.Fatalf("CreateMeasurement: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurements", measurement.ID) })

	updated, err := repo.UpdateMeasurement(ctx, tenantID, measurement.ID, "Aufmass S (final)", "Halle C1", "Erika", "2026-08-05", "geprueft")
	if err != nil {
		t.Fatalf("UpdateMeasurement: %v", err)
	}
	if updated.Title != "Aufmass S (final)" || updated.Location.String != "Halle C1" {
		t.Fatalf("update did not persist: %+v", updated)
	}

	if _, err := repo.UpdateMeasurement(ctx, tenantID, uuid.New(), "x", "", "", "", ""); !errors.Is(err, ErrMeasurementNotFound) {
		t.Fatalf("expected ErrMeasurementNotFound for unknown measurement, got %v", err)
	}
}

func TestDeleteMeasurement_RemovesRowAndRejectsDoubleDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	measurement, err := repo.CreateMeasurement(ctx, tenantID, nil, "Aufmass T", "", "", "", "")
	if err != nil {
		t.Fatalf("CreateMeasurement: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurements", measurement.ID) })

	if err := repo.DeleteMeasurement(ctx, tenantID, measurement.ID); err != nil {
		t.Fatalf("DeleteMeasurement: %v", err)
	}
	if _, err := repo.GetMeasurement(ctx, tenantID, measurement.ID); !errors.Is(err, ErrMeasurementNotFound) {
		t.Fatalf("expected ErrMeasurementNotFound after delete, got %v", err)
	}
	if err := repo.DeleteMeasurement(ctx, tenantID, measurement.ID); !errors.Is(err, ErrMeasurementNotFound) {
		t.Fatalf("expected ErrMeasurementNotFound on double delete, got %v", err)
	}
}

func TestDeleteMeasurementPosition_RemovesRowAndRejectsDoubleDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	measurement, err := repo.CreateMeasurement(ctx, tenantID, nil, "Aufmass U", "", "", "", "")
	if err != nil {
		t.Fatalf("CreateMeasurement: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurements", measurement.ID) })

	position, err := repo.AddMeasurementPosition(ctx, tenantID, measurement.ID, 1, "Bodenflaeche", "m2", 20, 5)
	if err != nil {
		t.Fatalf("AddMeasurementPosition: %v", err)
	}

	if err := repo.DeleteMeasurementPosition(ctx, tenantID, position.ID); err != nil {
		t.Fatalf("DeleteMeasurementPosition: %v", err)
	}

	reloaded, err := repo.GetMeasurement(ctx, tenantID, measurement.ID)
	if err != nil {
		t.Fatalf("GetMeasurement after position delete: %v", err)
	}
	if len(reloaded.Positions) != 0 {
		t.Fatalf("expected no positions left, got %+v", reloaded.Positions)
	}

	if err := repo.DeleteMeasurementPosition(ctx, tenantID, position.ID); !errors.Is(err, ErrPositionNotFound) {
		t.Fatalf("expected ErrPositionNotFound on double delete, got %v", err)
	}
}

// TestDeleteMeasurement_CascadesToPositions belegs point (2) from the unit
// scope: no orphaned measurement_positions rows survive a deleted parent.
// The cascade is a DB-level FK (ON DELETE CASCADE, migration 000163), not
// application code — this test proves it actually fires against the real
// schema rather than trusting the migration file.
func TestDeleteMeasurement_CascadesToPositions(t *testing.T) {
	t.Parallel()
	repo, _, ctx, tenantID := setupRapporteRepo(t)

	measurement, err := repo.CreateMeasurement(ctx, tenantID, nil, "Aufmass V", "", "", "", "")
	if err != nil {
		t.Fatalf("CreateMeasurement: %v", err)
	}

	position, err := repo.AddMeasurementPosition(ctx, tenantID, measurement.ID, 1, "Deckenflaeche", "m2", 12, 9)
	if err != nil {
		t.Fatalf("AddMeasurementPosition: %v", err)
	}

	if err := repo.DeleteMeasurement(ctx, tenantID, measurement.ID); err != nil {
		t.Fatalf("DeleteMeasurement: %v", err)
	}

	if err := repo.DeleteMeasurementPosition(ctx, tenantID, position.ID); !errors.Is(err, ErrPositionNotFound) {
		t.Fatalf("expected the cascade to have already removed the position, got %v", err)
	}
}

// TestAddMeasurementPosition_PreservesFractionalQuantityAndRoundsUnitPrice
// runs with a krumme Menge (the "1,00" trap from the GoBD grouping bug in
// Lauf 11): quantity uses all four NUMERIC(12,4) decimal places, unit_price
// crosses a NUMERIC(12,2) rounding boundary. Both are checked against the
// actual round-tripped value, not just "no error".
func TestAddMeasurementPosition_PreservesFractionalQuantityAndRoundsUnitPrice(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	measurement, err := repo.CreateMeasurement(ctx, tenantID, nil, "Aufmass W", "", "", "", "")
	if err != nil {
		t.Fatalf("CreateMeasurement: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurements", measurement.ID) })

	const quantity = 12.3456   // exact at NUMERIC(12,4)
	const unitPrice = 45.999   // NUMERIC(12,2) must round this, not truncate to 45.99
	position, err := repo.AddMeasurementPosition(ctx, tenantID, measurement.ID, 1, "Fensterflaeche", "m2", quantity, unitPrice)
	if err != nil {
		t.Fatalf("AddMeasurementPosition: %v", err)
	}

	reloaded, err := repo.GetMeasurement(ctx, tenantID, measurement.ID)
	if err != nil {
		t.Fatalf("GetMeasurement: %v", err)
	}
	if len(reloaded.Positions) != 1 {
		t.Fatalf("expected exactly one position, got %+v", reloaded.Positions)
	}
	got := reloaded.Positions[0]
	if got.ID != position.ID {
		t.Fatalf("expected the seeded position, got %+v", got)
	}
	if diff := got.Quantity.Float64 - quantity; diff < -0.00005 || diff > 0.00005 {
		t.Fatalf("expected quantity %.4f preserved at 4 decimals, got %.4f", quantity, got.Quantity.Float64)
	}
	if diff := got.UnitPrice.Float64 - 46.00; diff < -0.005 || diff > 0.005 {
		t.Fatalf("expected unit_price rounded to 46.00 at NUMERIC(12,2), got %.4f", got.UnitPrice.Float64)
	}
}

// TestAddMeasurementPosition_AcceptsMeasurementIDFromAnotherTenant documents
// a VERIFIED tenant-isolation gap (reported as a fix-unit, not fixed here —
// a coverage unit changes no behaviour): AddMeasurementPosition never checks
// that measurementID belongs to tenantID before inserting. The DB migration
// only references measurements(id) (ON DELETE CASCADE, no tenant match), and
// enable_tenant_rls('measurement_positions') checks the new row's own
// tenant_id column, not the parent measurement's tenant — so the INSERT's
// RLS WITH CHECK passes as long as the position's tenant_id equals the
// caller's own tenant, regardless of who owns the referenced measurement.
// Net effect: tenant A can attach a position (billing-relevant: quantity +
// unit_price) to any measurement UUID it can obtain, even one owned by
// tenant B, without ever passing tenant B's ownership check.
func TestAddMeasurementPosition_AcceptsMeasurementIDFromAnotherTenant(t *testing.T) {
	t.Parallel()
	repo, pool, ctxA, tenantA := setupRapporteRepo(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Rapporte Repo Test Tenant B")
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	measurementB, err := repo.CreateMeasurement(ctxB, tenantB, nil, "Aufmass Tenant B", "", "", "", "")
	if err != nil {
		t.Fatalf("CreateMeasurement (tenant B): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurements", measurementB.ID) })

	position, err := repo.AddMeasurementPosition(ctxA, tenantA, measurementB.ID, 1, "Fremde Flaeche", "m2", 5, 10)
	if err != nil {
		t.Fatalf("expected the cross-tenant insert to currently SUCCEED (documenting the gap), got error: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurement_positions", position.ID) })

	if _, err := repo.GetMeasurement(ctxA, tenantA, measurementB.ID); !errors.Is(err, ErrMeasurementNotFound) {
		t.Fatalf("tenant A must not be able to read tenant B's measurement via GetMeasurement, got %v", err)
	}
}

// ============================================================================
// Templates
// ============================================================================

func TestGetTemplate_ReturnsRowAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	template, err := repo.CreateTemplate(ctx, tenantID, "Standardbericht", "Vorlage", "montage", `[{"description":"Montage"}]`)
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_templates", template.ID) })

	reloaded, err := repo.GetTemplate(ctx, tenantID, template.ID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if reloaded.Name != "Standardbericht" || !reloaded.DefaultLinesJSON.Valid {
		t.Fatalf("unexpected template: %+v", reloaded)
	}

	if _, err := repo.GetTemplate(ctx, tenantID, uuid.New()); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound for unknown template, got %v", err)
	}
}

func TestCreateAndUpdateTemplate_EmptyDefaultLinesJSONDefaultsToEmptyArray(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	template, err := repo.CreateTemplate(ctx, tenantID, "Leere Vorlage", "", "", "")
	if err != nil {
		t.Fatalf("CreateTemplate with empty default_lines_json: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_templates", template.ID) })

	if !template.DefaultLinesJSON.Valid || template.DefaultLinesJSON.String != "[]" {
		t.Fatalf("expected default_lines_json '[]', got %+v", template.DefaultLinesJSON)
	}

	reloaded, err := repo.GetTemplate(ctx, tenantID, template.ID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if !reloaded.DefaultLinesJSON.Valid || reloaded.DefaultLinesJSON.String != "[]" {
		t.Fatalf("expected reloaded default_lines_json '[]', got %+v", reloaded.DefaultLinesJSON)
	}

	updated, err := repo.UpdateTemplate(ctx, tenantID, template.ID, "Leere Vorlage", "", "", "", true)
	if err != nil {
		t.Fatalf("UpdateTemplate with empty default_lines_json: %v", err)
	}
	if !updated.DefaultLinesJSON.Valid || updated.DefaultLinesJSON.String != "[]" {
		t.Fatalf("expected updated default_lines_json '[]', got %+v", updated.DefaultLinesJSON)
	}
}

func TestListTemplates_FiltersActiveOnlyAndPaginates(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	active, err := repo.CreateTemplate(ctx, tenantID, "Aktive Vorlage", "", "", "[]")
	if err != nil {
		t.Fatalf("CreateTemplate (active): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_templates", active.ID) })

	inactive, err := repo.CreateTemplate(ctx, tenantID, "Inaktive Vorlage", "", "", "[]")
	if err != nil {
		t.Fatalf("CreateTemplate (inactive): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_templates", inactive.ID) })
	if _, err := repo.UpdateTemplate(ctx, tenantID, inactive.ID, "Inaktive Vorlage", "", "", "[]", false); err != nil {
		t.Fatalf("deactivate template: %v", err)
	}

	activeOnly, total, err := repo.ListTemplates(ctx, tenantID, true, 1, 50)
	if err != nil {
		t.Fatalf("ListTemplates (activeOnly): %v", err)
	}
	if total != 1 || len(activeOnly) != 1 || activeOnly[0].ID != active.ID {
		t.Fatalf("expected exactly the active template, got total=%d rows=%+v", total, activeOnly)
	}

	all, total, err := repo.ListTemplates(ctx, tenantID, false, 1, 50)
	if err != nil {
		t.Fatalf("ListTemplates (all): %v", err)
	}
	if total != 2 || len(all) != 2 {
		t.Fatalf("expected 2 templates tenant-wide, got total=%d rows=%d", total, len(all))
	}
}

func TestUpdateTemplate_UpdatesFieldsAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	template, err := repo.CreateTemplate(ctx, tenantID, "Vorlage V", "", "", "[]")
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_templates", template.ID) })

	updated, err := repo.UpdateTemplate(ctx, tenantID, template.ID, "Vorlage V (final)", "Beschreibung", "wartung", `[]`, false)
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if updated.Name != "Vorlage V (final)" || updated.IsActive {
		t.Fatalf("update did not persist: %+v", updated)
	}

	if _, err := repo.UpdateTemplate(ctx, tenantID, uuid.New(), "x", "", "", "", true); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound for unknown template, got %v", err)
	}
}

func TestDeleteTemplate_RemovesRowAndRejectsDoubleDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)

	template, err := repo.CreateTemplate(ctx, tenantID, "Vorlage W", "", "", "[]")
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_templates", template.ID) })

	if err := repo.DeleteTemplate(ctx, tenantID, template.ID); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	if _, err := repo.GetTemplate(ctx, tenantID, template.ID); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound after delete, got %v", err)
	}
	if err := repo.DeleteTemplate(ctx, tenantID, template.ID); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound on double delete, got %v", err)
	}
}
