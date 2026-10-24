package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/models"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// listRecordingRepo answers List with one entry per status and remembers the filter; the
// embedded nil interface panics on any other repository call.
type listRecordingRepo struct {
	timetracking.WorkTimeRepository
	filter timetracking.WorkTimeFilter
}

func (r *listRecordingRepo) List(_ context.Context, f timetracking.WorkTimeFilter) ([]*models.HRWorkTimeEntry, int, error) {
	r.filter = f
	return []*models.HRWorkTimeEntry{{ID: uuid.New(), Status: models.WorkTimeStatusCorrectionPending}}, 1, nil
}

func newListEntriesServer(repo timetracking.WorkTimeRepository) *HRGRPCServer {
	svc := timetracking.NewService(repo, nil, nil, nil, nil)
	return &HRGRPCServer{timetrackingService: svc}
}

func TestListWorkTimeEntries_StatusReachesTheRepositoryFilter(t *testing.T) {
	repo := &listRecordingRepo{}
	srv := newListEntriesServer(repo)

	resp, err := srv.ListWorkTimeEntries(ctxWithTenant(uuid.New()), &hrv1.ListWorkTimeEntriesReq{Status: "correction_pending"})

	require.NoError(t, err)
	require.NotNil(t, repo.filter.Status)
	assert.Equal(t, "correction_pending", *repo.filter.Status)
	require.Len(t, resp.GetEntries(), 1)
	assert.Equal(t, hrv1.WorkTimeEntryStatus_WORK_TIME_CORRECTION_PENDING, resp.GetEntries()[0].GetStatus())
}

func TestListWorkTimeEntries_NoStatusLeavesTheFilterOpen(t *testing.T) {
	repo := &listRecordingRepo{}
	srv := newListEntriesServer(repo)

	_, err := srv.ListWorkTimeEntries(ctxWithTenant(uuid.New()), &hrv1.ListWorkTimeEntriesReq{})

	require.NoError(t, err)
	assert.Nil(t, repo.filter.Status)
}

func TestListWorkTimeEntries_UnknownStatusIsInvalidArgument(t *testing.T) {
	repo := &listRecordingRepo{}
	srv := newListEntriesServer(repo)

	_, err := srv.ListWorkTimeEntries(ctxWithTenant(uuid.New()), &hrv1.ListWorkTimeEntriesReq{Status: "everything"})

	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Nil(t, repo.filter.Status, "the repository must not be reached")
}
