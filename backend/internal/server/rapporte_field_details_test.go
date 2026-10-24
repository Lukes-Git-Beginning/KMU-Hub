package server

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	rapportev1 "github.com/kmuhub/kmuhub/proto/rapporte/v1"
)

// The site-diary fields travel from the proto request through the service into the stored
// report, and an update that names one field keeps the others.
func TestRapporteGRPC_SiteDiaryFieldsReachTheStoredReport(t *testing.T) {
	repo := newStubRapporteRepo()
	s := newRapporteTestServer(repo)
	tenantID := uuid.New()
	ctx := rapporteCtxWithTenant(tenantID)

	created, err := s.CreateReport(ctx, &rapportev1.CreateReportRequest{
		Title: "Baustelle", AuthorId: uuid.NewString(),
		Weather: "rainy", Temperature: proto.Float64(0), WorkStart: "07:30", WorkEnd: "16:00",
		BreakMinutes: proto.Int32(45), ProjectName: "Neubau",
	})
	require.NoError(t, err)
	r := created.GetReport()
	require.Equal(t, "rainy", r.GetWeather())
	require.Equal(t, "07:30", r.GetWorkStart())
	require.Equal(t, "16:00", r.GetWorkEnd())
	require.Equal(t, int32(45), r.GetBreakMinutes())
	require.Equal(t, "Neubau", r.GetProjectName())

	updated, err := s.UpdateReport(ctx, &rapportev1.UpdateReportRequest{ReportId: r.GetId(), Weather: proto.String("sunny")})
	require.NoError(t, err)
	require.Equal(t, "sunny", updated.GetReport().GetWeather())
	require.Equal(t, "Neubau", updated.GetReport().GetProjectName(), "an update without the field keeps the stored value")
	require.Equal(t, int32(45), updated.GetReport().GetBreakMinutes())

	_, err = s.CreateReport(ctx, &rapportev1.CreateReportRequest{
		Title: "x", AuthorId: uuid.NewString(), WorkStart: "abends",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err), "an unparsable time is a client error")
}
