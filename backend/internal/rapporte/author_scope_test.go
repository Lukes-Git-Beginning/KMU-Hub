package rapporte

// A caller whose grant is scoped to "own" reaches a report only as its author and only
// while it is a draft (or was sent back). Everything runs against the real database:
// the point is that the author check sits behind every entry the member routes now
// open (lines, attachments, signature, submit, measurements), not just the report CRUD.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAuthorOnly_ReportContentFollowsTheAuthor(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Rapporte Author Scope Tenant")
	svc := NewService(NewPostgresRepository(pool))
	base := testutil.WithTenantCtx(context.Background(), tenant)

	me, colleague := uuid.New(), uuid.New()
	asMe := WithAuthorOnly(base, me)
	asColleague := WithAuthorOnly(base, colleague)

	report, err := svc.CreateReport(asMe, CreateReportInput{TenantID: tenant, Title: "Baustelle", AuthorID: me})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "work_reports", report.ID)

	// --- the author fills the report ---
	line, err := svc.AddLine(asMe, AddLineInput{TenantID: tenant, ReportID: report.ID, Description: "Kabel", Quantity: 2})
	if err != nil {
		t.Fatalf("author AddLine: %v", err)
	}
	if _, err := svc.UpdateLine(asMe, UpdateLineInput{TenantID: tenant, LineID: line.ID, Notes: ptr("NYM-J")}); err != nil {
		t.Fatalf("author UpdateLine: %v", err)
	}
	att, err := svc.UploadAttachment(asMe, UploadAttachmentInput{
		TenantID: tenant, ReportID: report.ID, Filename: "a.jpg", SizeBytes: 10,
		ObjectKey: "tenants/" + tenant.String() + "/a.jpg", UploadedBy: me,
	})
	if err != nil {
		t.Fatalf("author UploadAttachment: %v", err)
	}
	if _, err := svc.SaveSignature(asMe, tenant, report.ID, "data:image/png;base64,AAAA", "Kunde"); err != nil {
		t.Fatalf("author SaveSignature: %v", err)
	}
	if lines, err := svc.ListLines(asMe, tenant, report.ID); err != nil || len(lines) != 1 {
		t.Fatalf("author ListLines: %v (%d lines)", err, len(lines))
	}

	// --- a colleague with the same grant reaches nothing of it, and learns nothing ---
	notFound := func(name string, err error, want error) {
		t.Helper()
		if !errors.Is(err, want) {
			t.Errorf("%s as a colleague: got %v, want %v", name, err, want)
		}
	}
	_, err = svc.GetReport(asColleague, tenant, report.ID)
	notFound("GetReport", err, ErrReportNotFound)
	_, err = svc.UpdateReport(asColleague, UpdateReportInput{TenantID: tenant, ReportID: report.ID, Title: ptr("fremd")})
	notFound("UpdateReport", err, ErrReportNotFound)
	notFound("DeleteReport", svc.DeleteReport(asColleague, tenant, report.ID), ErrReportNotFound)
	_, err = svc.SubmitReport(asColleague, SubmitReportInput{TenantID: tenant, ReportID: report.ID})
	notFound("SubmitReport", err, ErrReportNotFound)
	_, err = svc.AddLine(asColleague, AddLineInput{TenantID: tenant, ReportID: report.ID, Description: "x", Quantity: 1})
	notFound("AddLine", err, ErrReportNotFound)
	_, err = svc.UpdateLine(asColleague, UpdateLineInput{TenantID: tenant, LineID: line.ID, Notes: ptr("x")})
	notFound("UpdateLine", err, ErrLineNotFound)
	notFound("DeleteLine", svc.DeleteLine(asColleague, tenant, line.ID), ErrLineNotFound)
	_, err = svc.ListLines(asColleague, tenant, report.ID)
	notFound("ListLines", err, ErrReportNotFound)
	_, err = svc.UploadAttachment(asColleague, UploadAttachmentInput{
		TenantID: tenant, ReportID: report.ID, Filename: "b.jpg", SizeBytes: 1,
		ObjectKey: "tenants/" + tenant.String() + "/b.jpg", UploadedBy: colleague,
	})
	notFound("UploadAttachment", err, ErrReportNotFound)
	_, err = svc.ListAttachments(asColleague, tenant, report.ID, nil)
	notFound("ListAttachments", err, ErrReportNotFound)
	notFound("DeleteAttachment", svc.DeleteAttachment(asColleague, tenant, att.ID), ErrAttachmentNotFound)
	_, err = svc.SaveSignature(asColleague, tenant, report.ID, "data:image/png;base64,BBBB", "x")
	notFound("SaveSignature", err, ErrReportNotFound)
	_, err = svc.ExportPDF(asColleague, tenant, report.ID)
	notFound("ExportPDF", err, ErrReportNotFound)
	notFound("AuthorizeReport", svc.AuthorizeReport(asColleague, tenant, report.ID, true), ErrReportNotFound)

	// Nothing the colleague tried changed the report.
	if got, err := svc.GetReport(base, tenant, report.ID); err != nil || got.Title != "Baustelle" || got.Status != StatusDraft {
		t.Fatalf("report was touched: %+v, %v", got, err)
	}

	// --- a caller without the restriction (manager/admin scope "all") is not narrowed ---
	if _, err := svc.GetReport(base, tenant, report.ID); err != nil {
		t.Fatalf("unrestricted GetReport: %v", err)
	}
	if _, err := svc.AddLine(base, AddLineInput{TenantID: tenant, ReportID: report.ID, Description: "Chef", Quantity: 1}); err != nil {
		t.Fatalf("unrestricted AddLine: %v", err)
	}

	// --- the author submits; from then on the content is closed to them ---
	if _, err := svc.SubmitReport(asMe, SubmitReportInput{TenantID: tenant, ReportID: report.ID}); err != nil {
		t.Fatalf("author SubmitReport: %v", err)
	}
	locked := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, ErrInvalidStateTransition) {
			t.Errorf("%s after submit: got %v, want ErrInvalidStateTransition", name, err)
		}
	}
	_, err = svc.AddLine(asMe, AddLineInput{TenantID: tenant, ReportID: report.ID, Description: "spät", Quantity: 1})
	locked("AddLine", err)
	_, err = svc.UpdateLine(asMe, UpdateLineInput{TenantID: tenant, LineID: line.ID, Notes: ptr("spät")})
	locked("UpdateLine", err)
	locked("DeleteLine", svc.DeleteLine(asMe, tenant, line.ID))
	_, err = svc.UploadAttachment(asMe, UploadAttachmentInput{
		TenantID: tenant, ReportID: report.ID, Filename: "c.jpg", SizeBytes: 1,
		ObjectKey: "tenants/" + tenant.String() + "/c.jpg", UploadedBy: me,
	})
	locked("UploadAttachment", err)
	locked("DeleteAttachment", svc.DeleteAttachment(asMe, tenant, att.ID))
	_, err = svc.UpdateReport(asMe, UpdateReportInput{TenantID: tenant, ReportID: report.ID, Title: ptr("spät")})
	locked("UpdateReport", err)
	locked("DeleteReport", svc.DeleteReport(asMe, tenant, report.ID))
	_, err = svc.SaveSignature(asMe, tenant, report.ID, "data:image/png;base64,CCCC", "x")
	locked("SaveSignature", err)

	// Reading stays open to the author, and the reviewer (unrestricted) can still correct.
	if _, err := svc.ListLines(asMe, tenant, report.ID); err != nil {
		t.Fatalf("author ListLines after submit: %v", err)
	}
	if _, err := svc.UpdateLine(base, UpdateLineInput{TenantID: tenant, LineID: line.ID, Notes: ptr("Prüfer")}); err != nil {
		t.Fatalf("unrestricted UpdateLine after submit: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }
