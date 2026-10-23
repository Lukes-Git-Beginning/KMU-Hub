package gateway

// Work time entries as the desktop client reads them: `status` is the word the FE types
// (`active|completed|…`), not the enum number; the status filter and the project reach the
// HR service. A loopback gRPC server plays the service (see newHRWireRoutes).

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

const testHRProjectID = "66666666-6666-6666-6666-666666666666"

func (s *hrWireStub) ListWorkTimeEntries(_ context.Context, r *hrv1.ListWorkTimeEntriesReq) (*hrv1.ListWorkTimeEntriesResp, error) {
	s.listed = r
	return &hrv1.ListWorkTimeEntriesResp{Entries: []*hrv1.WorkTimeEntry{
		{Id: "e1", Status: hrv1.WorkTimeEntryStatus_WORK_TIME_ACTIVE},
		{Id: "e2", Status: hrv1.WorkTimeEntryStatus_WORK_TIME_COMPLETED},
		{Id: "e3", Status: hrv1.WorkTimeEntryStatus_WORK_TIME_CORRECTION_PENDING},
		{Id: "e4", Status: hrv1.WorkTimeEntryStatus_WORK_TIME_CORRECTION_APPROVED},
		{Id: "e5", Status: hrv1.WorkTimeEntryStatus_WORK_TIME_SUPERSEDED},
	}, Total: 5}, nil
}

func (s *hrWireStub) ClockIn(context.Context, *hrv1.ClockInReq) (*hrv1.ClockInResp, error) {
	return &hrv1.ClockInResp{Entry: &hrv1.WorkTimeEntry{Id: "e1", Status: hrv1.WorkTimeEntryStatus_WORK_TIME_ACTIVE}}, nil
}

func (s *hrWireStub) ClockOut(context.Context, *hrv1.ClockOutReq) (*hrv1.ClockOutResp, error) {
	return &hrv1.ClockOutResp{Entry: &hrv1.WorkTimeEntry{Id: "e1", Status: hrv1.WorkTimeEntryStatus_WORK_TIME_COMPLETED}}, nil
}

func (s *hrWireStub) GetActiveShift(context.Context, *hrv1.GetActiveShiftReq) (*hrv1.GetActiveShiftResp, error) {
	return &hrv1.GetActiveShiftResp{Entry: &hrv1.WorkTimeEntry{Id: "e1", Status: hrv1.WorkTimeEntryStatus_WORK_TIME_ACTIVE}}, nil
}

func (s *hrWireStub) CreateManualEntry(_ context.Context, r *hrv1.CreateManualEntryReq) (*hrv1.CreateManualEntryResp, error) {
	s.manual = r
	return &hrv1.CreateManualEntryResp{Entry: &hrv1.WorkTimeEntry{Id: "e6", Status: hrv1.WorkTimeEntryStatus_WORK_TIME_COMPLETED, ProjectId: r.GetProjectId()}}, nil
}

func (s *hrWireStub) SubmitTimeCorrection(context.Context, *hrv1.SubmitTimeCorrectionReq) (*hrv1.SubmitTimeCorrectionResp, error) {
	return &hrv1.SubmitTimeCorrectionResp{Correction: &hrv1.WorkTimeEntry{Id: "e7", Status: hrv1.WorkTimeEntryStatus_WORK_TIME_CORRECTION_PENDING}}, nil
}

func TestHRWire_TimeEntryStatusIsTheWordTheClientTypes(t *testing.T) {
	routes, _ := newHRWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListWorkTimeEntries(rec, hrRequest(t, http.MethodGet, nil, ""))
	assertStatus(t, rec, http.StatusOK)
	entries, ok := decodeBody(t, rec)["entries"].([]any)
	require.True(t, ok, rec.Body.String())
	got := make([]any, 0, len(entries))
	for _, e := range entries {
		got = append(got, e.(map[string]any)["status"])
	}
	assert.Equal(t, []any{"active", "completed", "correction_pending", "correction_approved", "superseded"}, got)
}

func TestHRWire_SingleTimeEntryRoutesWriteStatusAsWord(t *testing.T) {
	routes, _ := newHRWireRoutes(t)

	entryOf := func(rec *httptest.ResponseRecorder) map[string]any {
		body := decodeBody(t, rec)
		if nested, ok := body["entry"].(map[string]any); ok {
			return nested
		}
		return body
	}
	cases := []struct {
		name   string
		call   func(http.ResponseWriter)
		status string
	}{
		{"clock in", func(w http.ResponseWriter) {
			routes.HandleClockIn(w, hrRequest(t, http.MethodPost, nil, ""))
		}, "active"},
		{"clock out", func(w http.ResponseWriter) {
			routes.HandleClockOut(w, hrRequest(t, http.MethodPost, nil, ""))
		}, "completed"},
		{"active shift", func(w http.ResponseWriter) {
			routes.HandleGetActiveShift(w, hrRequest(t, http.MethodGet, nil, ""))
		}, "active"},
		{"manual entry", func(w http.ResponseWriter) {
			req := hrRequest(t, http.MethodPost, map[string]any{
				"clock_in": "2026-10-23T08:00:00Z", "clock_out": "2026-10-23T16:00:00Z",
			}, "")
			req.Header.Set("Idempotency-Key", "k1")
			routes.HandleCreateManualEntry(w, req)
		}, "completed"},
		{"correction", func(w http.ResponseWriter) {
			routes.HandleSubmitCorrection(w, hrRequest(t, http.MethodPost, map[string]any{
				"original_entry_id":   "77777777-7777-7777-7777-777777777777",
				"corrected_clock_in":  "2026-10-23T08:00:00Z",
				"corrected_clock_out": "2026-10-23T16:00:00Z",
				"reason":              "vergessen",
			}, ""))
		}, "correction_pending"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.call(rec)
			require.Less(t, rec.Code, 300, rec.Body.String())
			assert.Equal(t, tc.status, entryOf(rec)["status"], rec.Body.String())
		})
	}
}

func TestHRWire_TimeEntryListPassesStatusFilterToTheService(t *testing.T) {
	routes, stub := newHRWireRoutes(t)

	req := hrRequest(t, http.MethodGet, nil, "")
	req.URL.RawQuery = "status=correction_pending&employee_id=" + testHRUserID
	routes.HandleListWorkTimeEntries(httptest.NewRecorder(), req)

	require.NotNil(t, stub.listed)
	assert.Equal(t, "correction_pending", stub.listed.GetStatus())
	assert.Equal(t, testHRUserID, stub.listed.GetEmployeeId())
}

func TestHRWire_ManualEntryPassesProjectToTheService(t *testing.T) {
	routes, stub := newHRWireRoutes(t)

	req := hrRequest(t, http.MethodPost, map[string]any{
		"clock_in": "2026-10-23T08:00:00Z", "clock_out": "2026-10-23T16:00:00Z", "project_id": testHRProjectID,
	}, "")
	req.Header.Set("Idempotency-Key", "k1")
	rec := httptest.NewRecorder()
	routes.HandleCreateManualEntry(rec, req)

	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.manual)
	assert.Equal(t, testHRProjectID, stub.manual.GetProjectId())
	assert.Equal(t, testHRProjectID, decodeBody(t, rec)["project_id"])
}

func TestHRWire_ManualEntryRejectsProjectThatIsNoUUID(t *testing.T) {
	routes, stub := newHRWireRoutes(t)

	req := hrRequest(t, http.MethodPost, map[string]any{
		"clock_in": "2026-10-23T08:00:00Z", "clock_out": "2026-10-23T16:00:00Z", "project_id": "proj-1",
	}, "")
	req.Header.Set("Idempotency-Key", "k1")
	rec := httptest.NewRecorder()
	routes.HandleCreateManualEntry(rec, req)

	assertStatus(t, rec, http.StatusBadRequest)
	assert.Nil(t, stub.manual)
}
