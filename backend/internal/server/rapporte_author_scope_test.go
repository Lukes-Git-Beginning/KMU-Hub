package server

// The gateway flags a request "restrict_to_author" when the caller's grant is
// scoped to "own". The server turns that flag plus the propagated x-user-id into
// the author-only context of the service — and refuses to widen when the caller is
// missing. Measurements go straight to the repository, so they are checked here.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/rapporte"
	rapportev1 "github.com/kmuhub/kmuhub/proto/rapporte/v1"
)

func rapporteCtxAs(tenantID uuid.UUID, userID string) context.Context {
	return context.WithValue(rapporteCtxWithTenant(tenantID), middleware.UserIDKey, userID)
}

func TestRapporteGRPC_RestrictToAuthor(t *testing.T) {
	tenantID := uuid.New()
	me, colleague := uuid.New(), uuid.New()

	repo := newStubRapporteRepo()
	srv := newRapporteTestServer(repo)

	now := time.Now()
	mine := &rapporte.WorkReport{ID: uuid.New(), TenantID: tenantID, Title: "mine", Status: rapporte.StatusDraft, AuthorID: me, ReportDate: now}
	theirs := &rapporte.WorkReport{ID: uuid.New(), TenantID: tenantID, Title: "theirs", Status: rapporte.StatusDraft, AuthorID: colleague, ReportDate: now}
	repo.seedReport(mine)
	repo.seedReport(theirs)

	code := func(err error) codes.Code { return status.Code(err) }

	t.Run("an author-only request without a caller is refused, not widened", func(t *testing.T) {
		_, err := srv.GetReport(rapporteCtxWithTenant(tenantID), &rapportev1.GetReportRequest{ReportId: mine.ID.String(), RestrictToAuthor: true})
		assert.Equal(t, codes.PermissionDenied, code(err))
		_, err = srv.AddLine(rapporteCtxAs(tenantID, "not-a-uuid"), &rapportev1.AddLineRequest{ReportId: mine.ID.String(), Description: "x", Quantity: 1, RestrictToAuthor: true})
		assert.Equal(t, codes.PermissionDenied, code(err))
	})

	t.Run("author-only sees only their own report", func(t *testing.T) {
		ctx := rapporteCtxAs(tenantID, me.String())
		_, err := srv.GetReport(ctx, &rapportev1.GetReportRequest{ReportId: mine.ID.String(), RestrictToAuthor: true})
		require.NoError(t, err)
		_, err = srv.GetReport(ctx, &rapportev1.GetReportRequest{ReportId: theirs.ID.String(), RestrictToAuthor: true})
		assert.Equal(t, codes.NotFound, code(err))
		// Without the flag the same caller is not narrowed (manager/admin scope).
		_, err = srv.GetReport(ctx, &rapportev1.GetReportRequest{ReportId: theirs.ID.String()})
		require.NoError(t, err)
	})

	t.Run("author-only fills and submits their report, not a colleague's", func(t *testing.T) {
		ctx := rapporteCtxAs(tenantID, me.String())
		_, err := srv.AddLine(ctx, &rapportev1.AddLineRequest{ReportId: mine.ID.String(), Description: "Kabel", Quantity: 1, RestrictToAuthor: true})
		require.NoError(t, err)
		_, err = srv.AddLine(ctx, &rapportev1.AddLineRequest{ReportId: theirs.ID.String(), Description: "Kabel", Quantity: 1, RestrictToAuthor: true})
		assert.Equal(t, codes.NotFound, code(err))
		_, err = srv.SubmitReport(ctx, &rapportev1.SubmitReportRequest{ReportId: theirs.ID.String(), RestrictToAuthor: true})
		assert.Equal(t, codes.NotFound, code(err))
		_, err = srv.SubmitReport(ctx, &rapportev1.SubmitReportRequest{ReportId: mine.ID.String(), RestrictToAuthor: true})
		require.NoError(t, err)
		// Submitted: the author's content is locked.
		_, err = srv.AddLine(ctx, &rapportev1.AddLineRequest{ReportId: mine.ID.String(), Description: "spät", Quantity: 1, RestrictToAuthor: true})
		assert.Equal(t, codes.FailedPrecondition, code(err))
	})

	t.Run("measurements follow the report author", func(t *testing.T) {
		myDraft := &rapporte.WorkReport{ID: uuid.New(), TenantID: tenantID, Title: "draft", Status: rapporte.StatusDraft, AuthorID: me, ReportDate: now}
		repo.seedReport(myDraft)
		ctx := rapporteCtxAs(tenantID, me.String())

		own, err := srv.CreateMeasurement(ctx, &rapportev1.CreateMeasurementRequest{ReportId: myDraft.ID.String(), Title: "Aufmaß", RestrictToAuthor: true})
		require.NoError(t, err)
		_, err = srv.CreateMeasurement(ctx, &rapportev1.CreateMeasurementRequest{ReportId: theirs.ID.String(), Title: "fremd", RestrictToAuthor: true})
		assert.Equal(t, codes.NotFound, code(err))

		pos, err := srv.AddMeasurementPosition(ctx, &rapportev1.AddMeasurementPositionRequest{MeasurementId: own.GetMeasurement().GetId(), Description: "Wand", RestrictToAuthor: true})
		require.NoError(t, err)

		// A measurement of a colleague's report, planted directly.
		foreign, err := repo.CreateMeasurement(context.Background(), tenantID, &theirs.ID, "fremd", "", "", "", "")
		require.NoError(t, err)
		foreignPos, err := repo.AddMeasurementPosition(context.Background(), tenantID, foreign.ID, 1, "x", "", 1, 1)
		require.NoError(t, err)

		_, err = srv.UpdateMeasurement(ctx, &rapportev1.UpdateMeasurementRequest{MeasurementId: foreign.ID.String(), Title: "x", RestrictToAuthor: true})
		assert.Equal(t, codes.NotFound, code(err))
		_, err = srv.DeleteMeasurement(ctx, &rapportev1.DeleteMeasurementRequest{MeasurementId: foreign.ID.String(), RestrictToAuthor: true})
		assert.Equal(t, codes.NotFound, code(err))
		_, err = srv.AddMeasurementPosition(ctx, &rapportev1.AddMeasurementPositionRequest{MeasurementId: foreign.ID.String(), Description: "x", RestrictToAuthor: true})
		assert.Equal(t, codes.NotFound, code(err))
		_, err = srv.DeleteMeasurementPosition(ctx, &rapportev1.DeleteMeasurementPositionRequest{PositionId: foreignPos.ID.String(), RestrictToAuthor: true})
		assert.Equal(t, codes.NotFound, code(err))

		// Own ones work, and an unrestricted caller reaches the foreign one.
		_, err = srv.DeleteMeasurementPosition(ctx, &rapportev1.DeleteMeasurementPositionRequest{PositionId: pos.GetPosition().GetId(), RestrictToAuthor: true})
		require.NoError(t, err)
		_, err = srv.UpdateMeasurement(ctx, &rapportev1.UpdateMeasurementRequest{MeasurementId: foreign.ID.String(), Title: "Chef"})
		require.NoError(t, err)
	})
}
