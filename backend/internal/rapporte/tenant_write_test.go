package rapporte

// Exercises the real write path (PostgresRepository, not testutil.SeedRow) so a
// forgotten tenant_id column would show up as a genuine INSERT/constraint failure
// instead of being silently bypassed by a system-context fixture. Complements
// tenant_isolation_phase2_test.go, which only proves RLS SELECT filtering on
// hand-seeded rows.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestRapporteWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Rapporte Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Rapporte Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	report := &WorkReport{
		ID:         uuid.New(),
		TenantID:   tenantWrite,
		Title:      "Baustelle Musterweg",
		Status:     StatusDraft,
		AuthorID:   uuid.New(),
		ReportDate: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := repo.CreateReport(ctxA, report); err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "work_reports", report.ID)

	line := &ReportLine{
		ID:          uuid.New(),
		TenantID:    tenantWrite,
		ReportID:    report.ID,
		Position:    1,
		Description: "Montage",
		Quantity:    4,
		Unit:        "h",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.CreateLine(ctxA, line); err != nil {
		t.Fatalf("CreateLine: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "report_lines", line.ID)

	attachment := &ReportAttachment{
		ID:          uuid.New(),
		TenantID:    tenantWrite,
		ReportID:    report.ID,
		Filename:    "foto.jpg",
		ContentType: "image/jpeg",
		SizeBytes:   1024,
		ObjectKey:   "rapporte/" + uuid.New().String() + "/foto.jpg",
		UploadedBy:  uuid.New(),
		CreatedAt:   now,
	}
	if err := repo.CreateAttachment(ctxA, attachment); err != nil {
		t.Fatalf("CreateAttachment: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "report_attachments", attachment.ID)

	worker, err := repo.AddWorker(ctxA, tenantWrite, report.ID, "Max Muster", "Monteur", 8)
	if err != nil {
		t.Fatalf("AddWorker: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "report_workers", worker.ID)

	measurement, err := repo.CreateMeasurement(ctxA, tenantWrite, &report.ID, "Aufmass 1", "Halle A", "Max Muster", "2026-07-28", "")
	if err != nil {
		t.Fatalf("CreateMeasurement: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "measurements", measurement.ID)

	position, err := repo.AddMeasurementPosition(ctxA, tenantWrite, measurement.ID, 1, "Wandflaeche", "m2", 12.5, 9.9)
	if err != nil {
		t.Fatalf("AddMeasurementPosition: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "measurement_positions", position.ID)

	template, err := repo.CreateTemplate(ctxA, tenantWrite, "Standardbericht", "Vorlage", "montage", "[]")
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "report_templates", template.ID)

	rows := []struct {
		table string
		id    uuid.UUID
	}{
		{"work_reports", report.ID},
		{"report_lines", line.ID},
		{"report_attachments", attachment.ID},
		{"report_workers", worker.ID},
		{"measurements", measurement.ID},
		{"measurement_positions", position.ID},
		{"report_templates", template.ID},
	}

	for _, row := range rows {
		t.Run(row.table, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, row.table, row.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, row.table, row.id, 0)
		})
	}
}
