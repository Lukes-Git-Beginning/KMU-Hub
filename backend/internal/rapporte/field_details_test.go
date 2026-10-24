package rapporte

// The site-diary fields of a report (weather, temperature, working hours, break, project)
// travel through the service into work_reports and back. The round trip runs against the real
// schema because work_start/work_end are TIME columns read into *string.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func strp(s string) *string { return &s }

func TestFieldDetails_CreateAndUpdateRoundTripThroughTheDatabase(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRapporteRepo(t)
	svc := NewService(repo)

	zero, brk := 0.0, 45
	rep, err := svc.CreateReport(ctx, CreateReportInput{
		TenantID: tenantID, Title: "Baustelle", AuthorID: uuid.New(),
		FieldDetails: FieldDetails{
			Weather: strp("rainy"), Temperature: &zero, WorkStart: strp("07:30"), WorkEnd: strp("16:00"),
			BreakMinutes: &brk, ProjectName: strp("Neubau"),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "work_reports", rep.ID) })

	got, err := svc.GetReport(ctx, tenantID, rep.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Weather)
	assert.Equal(t, "rainy", *got.Weather)
	require.NotNil(t, got.Temperature, "0 degrees is stored, not dropped")
	assert.Equal(t, 0.0, *got.Temperature)
	require.NotNil(t, got.WorkStart)
	assert.Equal(t, "07:30", (*got.WorkStart)[:5])
	require.NotNil(t, got.WorkEnd)
	assert.Equal(t, "16:00", (*got.WorkEnd)[:5])
	require.NotNil(t, got.BreakMinutes)
	assert.Equal(t, 45, *got.BreakMinutes)
	require.NotNil(t, got.ProjectName)
	assert.Equal(t, "Neubau", *got.ProjectName)

	// An update that names only the weather keeps everything else.
	_, err = svc.UpdateReport(ctx, UpdateReportInput{
		TenantID: tenantID, ReportID: rep.ID, FieldDetails: FieldDetails{Weather: strp("sunny")},
	})
	require.NoError(t, err)
	got, err = svc.GetReport(ctx, tenantID, rep.ID)
	require.NoError(t, err)
	assert.Equal(t, "sunny", *got.Weather)
	assert.Equal(t, "Neubau", *got.ProjectName)
	assert.Equal(t, "07:30", (*got.WorkStart)[:5])
	assert.Equal(t, 45, *got.BreakMinutes)
}

func TestFieldDetails_RejectsValuesTheColumnsCannotHold(t *testing.T) {
	svc := NewService(newMockRepository())
	neg, hot, tooLong := -1, 150.0, 25*60

	for name, d := range map[string]FieldDetails{
		"negative break":    {BreakMinutes: &neg},
		"break over 24 h":   {BreakMinutes: &tooLong},
		"temperature range": {Temperature: &hot},
		"bad work start":    {WorkStart: strp("25:99")},
		"bad work end":      {WorkEnd: strp("abends")},
	} {
		_, err := svc.CreateReport(context.Background(), CreateReportInput{
			TenantID: uuid.New(), Title: "x", AuthorID: uuid.New(), FieldDetails: d,
		})
		assert.ErrorIs(t, err, ErrInvalidInput, name)
	}
}
