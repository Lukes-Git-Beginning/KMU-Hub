package gateway

// HR responses the desktop client unwraps (`raw.employee`, `res.settings`, `raw.entries`,
// `{documents}`): a real gRPC server on loopback plays the HR service, so the tests see the
// bodies the client would read and the requests the service would receive.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

const (
	testHRUserID     = "33333333-3333-3333-3333-333333333333"
	testHRCategoryID = "44444444-4444-4444-4444-444444444444"
	testHRFileID     = "55555555-5555-5555-5555-555555555555"
)

type hrWireStub struct {
	hrv1.UnimplementedHRServiceServer
	settings        *hrv1.HRSettings // nil -> NotFound, like a tenant that never saved
	updated         *hrv1.UpdateHRSettingsReq
	uploaded        *hrv1.UploadEmployeeDocumentReq
	listed          *hrv1.ListWorkTimeEntriesReq
	manual          *hrv1.CreateManualEntryReq
	employeeUpdated *hrv1.UpdateEmployeeReq
}

func (s *hrWireStub) GetEmployee(_ context.Context, r *hrv1.GetEmployeeReq) (*hrv1.GetEmployeeResp, error) {
	return &hrv1.GetEmployeeResp{Employee: &hrv1.EmployeeProfile{Id: "p1", UserId: r.GetUserId(), Department: "Vertrieb"}}, nil
}

func (s *hrWireStub) CreateEmployee(_ context.Context, r *hrv1.CreateEmployeeReq) (*hrv1.CreateEmployeeResp, error) {
	return &hrv1.CreateEmployeeResp{Employee: &hrv1.EmployeeProfile{Id: "p1", UserId: r.GetUserId()}}, nil
}

func (s *hrWireStub) UpdateEmployee(_ context.Context, r *hrv1.UpdateEmployeeReq) (*hrv1.UpdateEmployeeResp, error) {
	s.employeeUpdated = r
	return &hrv1.UpdateEmployeeResp{Employee: &hrv1.EmployeeProfile{Id: "p1", UserId: r.GetUserId(), Department: r.GetDepartment()}}, nil
}

func (s *hrWireStub) UpdateSelfProfile(_ context.Context, r *hrv1.UpdateSelfProfileReq) (*hrv1.UpdateSelfProfileResp, error) {
	return &hrv1.UpdateSelfProfileResp{Employee: &hrv1.EmployeeProfile{Id: "p1", UserId: r.GetUserId()}}, nil
}

func (s *hrWireStub) GetAbsenceCalendar(context.Context, *hrv1.GetAbsenceCalendarReq) (*hrv1.GetAbsenceCalendarResp, error) {
	return &hrv1.GetAbsenceCalendarResp{}, nil
}

func (s *hrWireStub) ListEmployeeDocuments(context.Context, *hrv1.ListEmployeeDocumentsReq) (*hrv1.ListEmployeeDocumentsResp, error) {
	return &hrv1.ListEmployeeDocumentsResp{}, nil
}

func (s *hrWireStub) UploadEmployeeDocument(_ context.Context, r *hrv1.UploadEmployeeDocumentReq) (*hrv1.UploadEmployeeDocumentResp, error) {
	s.uploaded = r
	return &hrv1.UploadEmployeeDocumentResp{Document: &hrv1.EmployeeDocument{Id: "d1", EmployeeId: r.GetEmployeeId()}}, nil
}

func (s *hrWireStub) GetHRSettings(context.Context, *hrv1.GetHRSettingsReq) (*hrv1.GetHRSettingsResp, error) {
	if s.settings == nil {
		return nil, status.Error(codes.NotFound, "HR company settings not found")
	}
	return &hrv1.GetHRSettingsResp{Settings: s.settings}, nil
}

func (s *hrWireStub) UpdateHRSettings(_ context.Context, r *hrv1.UpdateHRSettingsReq) (*hrv1.UpdateHRSettingsResp, error) {
	s.updated = r
	s.settings = &hrv1.HRSettings{
		AuThresholdDays:        r.GetAuThresholdDays(),
		ShowAbsenceReason:      r.GetShowAbsenceReason(),
		DefaultAnnualLeaveDays: r.GetDefaultAnnualLeaveDays(),
		Timezone:               r.GetTimezone(),
		WorkHoursPerDay:        r.GetWorkHoursPerDay(),
		MaxDailyHours:          r.GetMaxDailyHours(),
		BreakAfterHours:        r.GetBreakAfterHours(),
	}
	return &hrv1.UpdateHRSettingsResp{Settings: s.settings}, nil
}

func newHRWireRoutes(t *testing.T) (*HRRoutes, *hrWireStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &hrWireStub{}
	srv := grpc.NewServer()
	hrv1.RegisterHRServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewHRRoutes(reg, nil), stub
}

func hrRequest(t *testing.T, method string, body any, idParam string) *http.Request {
	t.Helper()
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, "/x", nil)
	} else {
		req = httptest.NewRequest(method, "/x", jsonBody(t, body))
	}
	req = withAuth(req, testHRUserID, testTenantID)
	if idParam != "" {
		req = withChiURLParam(req, "id", idParam)
	}
	return req
}

func TestHRWire_EmployeeResponsesAreWrapped(t *testing.T) {
	routes, _ := newHRWireRoutes(t)

	cases := map[string]func(http.ResponseWriter){
		"get self": func(w http.ResponseWriter) {
			routes.HandleGetSelfProfile(w, hrRequest(t, http.MethodGet, nil, ""))
		},
		"get by id": func(w http.ResponseWriter) {
			routes.HandleGetEmployee(w, hrRequest(t, http.MethodGet, nil, testHRUserID))
		},
		"update by id": func(w http.ResponseWriter) {
			routes.HandleUpdateEmployee(w, hrRequest(t, http.MethodPut, map[string]any{"department": "Vertrieb"}, testHRUserID))
		},
		"update self": func(w http.ResponseWriter) {
			routes.HandleUpdateSelfProfile(w, hrRequest(t, http.MethodPut, map[string]any{"address_city": "Bern"}, ""))
		},
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			call(rec)
			assertStatus(t, rec, http.StatusOK)
			emp, ok := decodeBody(t, rec)["employee"].(map[string]any)
			require.True(t, ok, rec.Body.String())
			assert.Equal(t, "p1", emp["id"])
			assert.Equal(t, testHRUserID, emp["user_id"])
		})
	}
}

// hr-client.ts create reads raw.employee like every other employee route; a bare
// profile made adaptEmployee throw after the row was already saved.
func TestHRWire_CreateEmployeeIsWrapped(t *testing.T) {
	routes, _ := newHRWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreateEmployee(rec, hrRequest(t, http.MethodPost,
		map[string]any{"user_id": testHRUserID, "start_date": "2026-11-01"}, ""))

	assertStatus(t, rec, http.StatusCreated)
	emp, ok := decodeBody(t, rec)["employee"].(map[string]any)
	require.True(t, ok, rec.Body.String())
	assert.Equal(t, "p1", emp["id"])
	assert.Equal(t, testHRUserID, emp["user_id"])
}

func TestHRWire_EmptyCalendarAndDocumentsAreWrappedEmptyLists(t *testing.T) {
	routes, _ := newHRWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleGetAbsenceCalendar(rec, hrRequest(t, http.MethodGet, nil, ""))
	assertStatus(t, rec, http.StatusOK)
	entries, ok := decodeBody(t, rec)["entries"].([]any)
	require.True(t, ok, rec.Body.String())
	assert.Empty(t, entries)

	rec = httptest.NewRecorder()
	routes.HandleListEmployeeDocuments(rec, hrRequest(t, http.MethodGet, nil, testHRUserID))
	assertStatus(t, rec, http.StatusOK)
	docs, ok := decodeBody(t, rec)["documents"].([]any)
	require.True(t, ok, rec.Body.String())
	assert.Empty(t, docs)
}

func TestHRWire_UploadDocumentReadsCamelCaseBodyAndWrapsAnswer(t *testing.T) {
	routes, stub := newHRWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUploadEmployeeDocument(rec, hrRequest(t, http.MethodPost, map[string]any{
		"categoryId": testHRCategoryID,
		"fileId":     testHRFileID,
		"fileName":   "vertrag.pdf",
		"fileSize":   "12 KB",
		"notes":      "unterschrieben",
	}, testHRUserID))

	assertStatus(t, rec, http.StatusCreated)
	doc, ok := decodeBody(t, rec)["document"].(map[string]any)
	require.True(t, ok, rec.Body.String())
	assert.Equal(t, "d1", doc["id"])
	require.NotNil(t, stub.uploaded)
	assert.Equal(t, testHRCategoryID, stub.uploaded.GetCategoryId())
	assert.Equal(t, testHRFileID, stub.uploaded.GetFileId())
	assert.Equal(t, "vertrag.pdf", stub.uploaded.GetFileName())
	assert.Equal(t, "12 KB", stub.uploaded.GetFileSize())
	assert.Equal(t, testHRUserID, stub.uploaded.GetEmployeeId())
}

func TestHRWire_UploadDocumentStillReadsSnakeCaseBody(t *testing.T) {
	routes, stub := newHRWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUploadEmployeeDocument(rec, hrRequest(t, http.MethodPost, map[string]any{
		"category_id": testHRCategoryID,
		"file_id":     testHRFileID,
	}, testHRUserID))

	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.uploaded)
	assert.Equal(t, testHRCategoryID, stub.uploaded.GetCategoryId())
	assert.Equal(t, testHRFileID, stub.uploaded.GetFileId())
}

func TestHRWire_SettingsGetIsWrapped(t *testing.T) {
	routes, stub := newHRWireRoutes(t)
	stub.settings = &hrv1.HRSettings{Timezone: "Europe/Zurich", WorkHoursPerDay: 8}

	rec := httptest.NewRecorder()
	routes.HandleGetHRSettings(rec, hrRequest(t, http.MethodGet, nil, ""))

	assertStatus(t, rec, http.StatusOK)
	got, ok := decodeBody(t, rec)["settings"].(map[string]any)
	require.True(t, ok, rec.Body.String())
	assert.Equal(t, "Europe/Zurich", got["timezone"])
	assert.EqualValues(t, 8, got["work_hours_per_day"])
}

func TestHRWire_SettingsGetWithoutStoredRowIsEmptyObject(t *testing.T) {
	routes, _ := newHRWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleGetHRSettings(rec, hrRequest(t, http.MethodGet, nil, ""))

	assertStatus(t, rec, http.StatusOK)
	got, ok := decodeBody(t, rec)["settings"].(map[string]any)
	require.True(t, ok, rec.Body.String())
	assert.Empty(t, got)
}

func TestHRWire_SettingsPutWithFourFieldsKeepsTheOtherValues(t *testing.T) {
	routes, stub := newHRWireRoutes(t)
	stub.settings = &hrv1.HRSettings{
		AuThresholdDays: 3, ShowAbsenceReason: true, DefaultAnnualLeaveDays: 28, Timezone: "Europe/Zurich",
		WorkHoursPerDay: 8, MaxDailyHours: 10, BreakAfterHours: 6,
	}

	// The desktop sends exactly these four.
	rec := httptest.NewRecorder()
	routes.HandleUpdateHRSettings(rec, hrRequest(t, http.MethodPut, map[string]any{
		"au_threshold_days":         5,
		"show_absence_reason":       false,
		"default_annual_leave_days": 30,
		"timezone":                  "Europe/Berlin",
	}, ""))

	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.updated)
	assert.EqualValues(t, 5, stub.updated.GetAuThresholdDays())
	assert.False(t, stub.updated.GetShowAbsenceReason(), "an explicit false must overwrite true")
	assert.EqualValues(t, 30, stub.updated.GetDefaultAnnualLeaveDays())
	assert.Equal(t, "Europe/Berlin", stub.updated.GetTimezone())
	assert.EqualValues(t, 8, stub.updated.GetWorkHoursPerDay())
	assert.EqualValues(t, 10, stub.updated.GetMaxDailyHours())
	assert.EqualValues(t, 6, stub.updated.GetBreakAfterHours())
	got, ok := decodeBody(t, rec)["settings"].(map[string]any)
	require.True(t, ok, rec.Body.String())
	assert.EqualValues(t, 8, got["work_hours_per_day"])
}

func TestHRWire_SettingsFirstSaveWithoutStoredRow(t *testing.T) {
	routes, stub := newHRWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUpdateHRSettings(rec, hrRequest(t, http.MethodPut, map[string]any{"timezone": "Europe/Berlin", "work_hours_per_day": 8}, ""))

	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.updated)
	assert.Equal(t, "Europe/Berlin", stub.updated.GetTimezone())
	assert.EqualValues(t, 8, stub.updated.GetWorkHoursPerDay())
}

func TestHRWire_SettingsPutStillValidates(t *testing.T) {
	routes, _ := newHRWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUpdateHRSettings(rec, hrRequest(t, http.MethodPut, map[string]any{"work_hours_per_day": 30}, ""))

	assertValidationError(t, rec, "work_hours_per_day")
}
