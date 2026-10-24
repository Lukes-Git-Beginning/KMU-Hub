package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/biz/dashboard"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

func strp(s string) *string { return &s }

func TestDashboardRange(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 30, 0, 0, time.UTC)

	t.Run("no bounds keeps today's behaviour: January 1st until now", func(t *testing.T) {
		from, to, err := dashboardRange(now, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), from)
		assert.Equal(t, now, to)
	})

	t.Run("past range: date_to is the inclusive end of that day", func(t *testing.T) {
		from, to, err := dashboardRange(now, strp("2026-03-01"), strp("2026-03-31"))
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), from)
		assert.Equal(t, time.Date(2026, 3, 31, 23, 59, 59, 999999999, time.UTC), to)
	})

	t.Run("the whole-year preset ending 31 December is clamped to now", func(t *testing.T) {
		_, to, err := dashboardRange(now, strp("2026-01-01"), strp("2026-12-31"))
		require.NoError(t, err)
		assert.Equal(t, now, to)
	})

	t.Run("only date_to: the range starts January 1st of that year", func(t *testing.T) {
		from, _, err := dashboardRange(now, nil, strp("2025-12-31"))
		require.NoError(t, err)
		assert.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), from)
	})

	t.Run("a range in the future is valid and empty", func(t *testing.T) {
		from, to, err := dashboardRange(now, strp("2027-01-01"), strp("2027-01-31"))
		require.NoError(t, err)
		assert.True(t, to.Before(from))
	})

	for name, c := range map[string]struct{ from, to *string }{
		"garbage date_from": {strp("yesterday"), nil},
		"garbage date_to":   {nil, strp("2026-13-45")},
		"inverted range":    {strp("2026-04-01"), strp("2026-03-01")},
	} {
		t.Run(name+" is InvalidArgument", func(t *testing.T) {
			_, _, err := dashboardRange(now, c.from, c.to)
			requireGRPCCode(t, err, codes.InvalidArgument)
		})
	}
}

type rangeRecordingRepo struct {
	stubDashboardRepo
	from, to time.Time
}

func (r *rangeRecordingRepo) GetDashboardMetrics(ctx context.Context, id uuid.UUID, from, to time.Time) (*dashboard.Metrics, error) {
	r.from, r.to = from, to
	return r.stubDashboardRepo.GetDashboardMetrics(ctx, id, from, to)
}

func TestGetFinanceDashboard_PassesTheRangeToTheRepository(t *testing.T) {
	repo := &rangeRecordingRepo{stubDashboardRepo: stubDashboardRepo{metrics: &dashboard.Metrics{}}}
	srv := newDashboardTestServer(repo)

	_, err := srv.GetFinanceDashboard(context.Background(), &bizv1.GetFinanceDashboardRequest{
		TenantId: uuid.NewString(), DateFrom: strp("2025-04-01"), DateTo: strp("2025-06-30"),
	})
	require.NoError(t, err)
	assert.Equal(t, time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), repo.from)
	assert.Equal(t, time.Date(2025, 6, 30, 23, 59, 59, 999999999, time.UTC), repo.to)

	_, err = srv.GetFinanceDashboard(context.Background(), &bizv1.GetFinanceDashboardRequest{
		TenantId: uuid.NewString(), DateFrom: strp("nope"),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}
