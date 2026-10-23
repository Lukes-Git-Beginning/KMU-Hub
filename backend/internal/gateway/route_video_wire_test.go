package gateway

// Wire shape of the video/meeting routes against the desktop types
// (api/video-client.ts, hooks/useVideo.ts, features/presence/PresenceProvider.tsx).
// A real gRPC server on loopback stands in for the work service, so the assertions
// run against what the handler writes and what it forwards over gRPC.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

type videoWireStub struct {
	videov1.UnimplementedVideoServiceServer
	gotCreate  *videov1.CreateMeetingRequest
	gotUpdate  *videov1.UpdateMeetingRequest
	recordings []*videov1.Recording
	items      []*videov1.ActionItem
	converted  []*videov1.ActionItem
	statuses   []*videov1.PresenceStatus
	coHosts    []*videov1.MeetingCoHost
	room       *videov1.BreakoutRoom
}

func (s *videoWireStub) CreateMeeting(_ context.Context, r *videov1.CreateMeetingRequest) (*videov1.Meeting, error) {
	s.gotCreate = r
	return &videov1.Meeting{Id: "m-1", Title: r.Title}, nil
}

func (s *videoWireStub) UpdateMeeting(_ context.Context, r *videov1.UpdateMeetingRequest) (*videov1.Meeting, error) {
	s.gotUpdate = r
	return &videov1.Meeting{Id: r.MeetingId}, nil
}

func (s *videoWireStub) ListRecordings(context.Context, *videov1.ListRecordingsRequest) (*videov1.ListRecordingsResponse, error) {
	return &videov1.ListRecordingsResponse{Recordings: s.recordings, Total: int32(len(s.recordings))}, nil
}

func (s *videoWireStub) ListActionItems(context.Context, *videov1.ListActionItemsRequest) (*videov1.ListActionItemsResponse, error) {
	return &videov1.ListActionItemsResponse{ActionItems: s.items}, nil
}

func (s *videoWireStub) ConvertActionItemsToTasks(context.Context, *videov1.ConvertActionItemsToTasksRequest) (*videov1.ConvertActionItemsToTasksResponse, error) {
	return &videov1.ConvertActionItemsToTasksResponse{ActionItems: s.converted, ConvertedCount: int32(len(s.converted))}, nil
}

func (s *videoWireStub) GetBulkPresence(context.Context, *videov1.GetBulkPresenceRequest) (*videov1.GetBulkPresenceResponse, error) {
	return &videov1.GetBulkPresenceResponse{Statuses: s.statuses}, nil
}

func (s *videoWireStub) GetPresence(context.Context, *videov1.GetPresenceRequest) (*videov1.PresenceStatus, error) {
	return s.statuses[0], nil
}

func (s *videoWireStub) ListCoHosts(context.Context, *videov1.ListCoHostsRequest) (*videov1.ListCoHostsResponse, error) {
	return &videov1.ListCoHostsResponse{CoHosts: s.coHosts}, nil
}

func (s *videoWireStub) GetBreakoutAssignment(context.Context, *videov1.GetBreakoutAssignmentRequest) (*videov1.GetBreakoutAssignmentResponse, error) {
	return &videov1.GetBreakoutAssignmentResponse{Room: s.room}, nil
}

func newVideoWireRoutes(t *testing.T) (*VideoRoutes, *videoWireStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &videoWireStub{}
	srv := grpc.NewServer()
	videov1.RegisterVideoServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("work", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewVideoRoutes(reg, "", ""), stub
}

func videoReq(t *testing.T, method, target string, body any, params map[string]string) *http.Request {
	t.Helper()
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, jsonBody(t, body))
		req.Header.Set("Content-Type", "application/json")
	}
	req = withAuth(req, uuid.New().String(), testTenantID)
	for k, v := range params {
		req = withChiURLParam(req, k, v)
	}
	return req
}

func TestParseTimestamp_ZonelessIsBerlinWallClock(t *testing.T) {
	winter, err := parseTimestamp("2026-01-15T09:30:00")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 1, 15, 8, 30, 0, 0, time.UTC), winter.AsTime(), "CET is UTC+1")

	summer, err := parseTimestamp("2026-07-15T09:30")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 7, 15, 7, 30, 0, 0, time.UTC), summer.AsTime(), "CEST is UTC+2")

	withZone, err := parseTimestamp("2026-07-15T09:30:00Z")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 7, 15, 9, 30, 0, 0, time.UTC), withZone.AsTime(), "explicit zone untouched")

	_, err = parseTimestamp("15.07.2026 09:30")
	assert.Error(t, err)
}

func TestHandleCreateMeeting_ZonelessStartAndCrmLinks(t *testing.T) {
	routes, stub := newVideoWireRoutes(t)
	contact, deal := uuid.New().String(), uuid.New().String()
	rec := httptest.NewRecorder()
	routes.HandleCreateMeeting(rec, videoReq(t, http.MethodPost, "/api/v1/meetings", map[string]any{
		"title":           "Kickoff",
		"scheduled_start": "2026-11-02T10:00:00",
		"scheduled_end":   "2026-11-02T11:00:00",
		"contact_id":      contact,
		"deal_id":         deal,
	}, nil))
	assertStatus(t, rec, http.StatusCreated)

	require.NotNil(t, stub.gotCreate)
	assert.Equal(t, time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC), stub.gotCreate.ScheduledStart.AsTime())
	assert.Equal(t, time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC), stub.gotCreate.ScheduledEnd.AsTime())
	require.NotNil(t, stub.gotCreate.ContactId)
	assert.Equal(t, contact, *stub.gotCreate.ContactId)
	require.NotNil(t, stub.gotCreate.DealId)
	assert.Equal(t, deal, *stub.gotCreate.DealId)
}

func TestHandleCreateMeeting_RejectsBadCrmLink(t *testing.T) {
	routes, _ := newVideoWireRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleCreateMeeting(rec, videoReq(t, http.MethodPost, "/api/v1/meetings", map[string]any{
		"title":           "Kickoff",
		"scheduled_start": "2026-11-02T10:00:00",
		"scheduled_end":   "2026-11-02T11:00:00",
		"contact_id":      "not-a-uuid",
	}, nil))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateMeeting_ZonelessStartAndCrmLinks(t *testing.T) {
	routes, stub := newVideoWireRoutes(t)
	id, contact := uuid.New().String(), uuid.New().String()
	rec := httptest.NewRecorder()
	routes.HandleUpdateMeeting(rec, videoReq(t, http.MethodPut, "/api/v1/meetings/"+id, map[string]any{
		"scheduled_start": "2026-11-02T10:15",
		"contact_id":      contact,
	}, map[string]string{"id": id}))
	assertStatus(t, rec, http.StatusOK)

	require.NotNil(t, stub.gotUpdate)
	assert.Equal(t, time.Date(2026, 11, 2, 9, 15, 0, 0, time.UTC), stub.gotUpdate.ScheduledStart.AsTime())
	require.NotNil(t, stub.gotUpdate.ContactId)
	assert.Equal(t, contact, *stub.gotUpdate.ContactId)
	assert.Nil(t, stub.gotUpdate.DealId, "unsent link stays untouched")
}

func TestHandleListRecordings_BareArray(t *testing.T) {
	routes, stub := newVideoWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListRecordings(rec, videoReq(t, http.MethodGet, "/api/v1/video/recordings?meeting_id=x", nil, nil))
	assertStatus(t, rec, http.StatusOK)
	assert.JSONEq(t, `[]`, rec.Body.String(), "empty list is [] not {}")

	stub.recordings = []*videov1.Recording{{Id: "r-1"}}
	rec = httptest.NewRecorder()
	routes.HandleListRecordings(rec, videoReq(t, http.MethodGet, "/api/v1/video/recordings", nil, nil))
	var got []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "r-1", got[0]["id"])
}

func TestWriteMeetingRecordings_EnvelopeWithArray(t *testing.T) {
	rec := httptest.NewRecorder()
	writeMeetingRecordings(rec, &videov1.ListRecordingsByMeetingResponse{})
	assert.JSONEq(t, `{"recordings":[],"total":0}`, rec.Body.String())

	rec = httptest.NewRecorder()
	writeMeetingRecordings(rec, &videov1.ListRecordingsByMeetingResponse{
		Recordings: []*videov1.Recording{{Id: "r-1"}}, Total: 7,
	})
	var got struct {
		Recordings []map[string]any `json:"recordings"`
		Total      int              `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Recordings, 1)
	assert.Equal(t, "r-1", got.Recordings[0]["id"])
	assert.Equal(t, 7, got.Total)
}

func TestHandleListActionItems_BareArray(t *testing.T) {
	routes, stub := newVideoWireRoutes(t)
	meeting := uuid.New().String()
	params := map[string]string{"id": meeting}

	rec := httptest.NewRecorder()
	routes.HandleListActionItems(rec, videoReq(t, http.MethodGet, "/api/v1/meetings/"+meeting+"/action-items", nil, params))
	assertStatus(t, rec, http.StatusOK)
	assert.JSONEq(t, `[]`, rec.Body.String(), "empty list used to be {}")

	stub.items = []*videov1.ActionItem{{Id: "a-1", Description: "Angebot senden"}}
	rec = httptest.NewRecorder()
	routes.HandleListActionItems(rec, videoReq(t, http.MethodGet, "/api/v1/meetings/"+meeting+"/action-items", nil, params))
	var got []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "a-1", got[0]["id"])
}

func TestHandleConvertActionItems_ReturnsTaskIDs(t *testing.T) {
	routes, stub := newVideoWireRoutes(t)
	task := "t-1"
	stub.converted = []*videov1.ActionItem{{Id: "a-1", TaskId: &task}, {Id: "a-2"}}
	meeting := uuid.New().String()
	body := map[string]any{"action_item_ids": []string{uuid.New().String()}, "project_id": uuid.New().String()}
	rec := httptest.NewRecorder()
	routes.HandleConvertActionItemsToTasks(rec, videoReq(t, http.MethodPost, "/api/v1/meetings/"+meeting+"/action-items/convert",
		body, map[string]string{"id": meeting}))
	assertStatus(t, rec, http.StatusOK)

	var got struct {
		ActionItems    []map[string]any `json:"action_items"`
		ConvertedCount int              `json:"converted_count"`
		TaskIDs        []string         `json:"task_ids"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, []string{"t-1"}, got.TaskIDs)
	assert.Len(t, got.ActionItems, 2)
	assert.Equal(t, 2, got.ConvertedCount)

	stub.converted = nil
	rec = httptest.NewRecorder()
	routes.HandleConvertActionItemsToTasks(rec, videoReq(t, http.MethodPost, "/x", body, map[string]string{"id": meeting}))
	var empty map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &empty))
	assert.Equal(t, []any{}, empty["task_ids"], "task_ids is [] so .length never throws")
}

func TestHandleGetBulkPresence_MapKeyedByUserWithStringLevels(t *testing.T) {
	routes, stub := newVideoWireRoutes(t)
	stub.statuses = []*videov1.PresenceStatus{
		{UserId: "u-1", Status: videov1.PresenceLevel_PRESENCE_IN_CALL, LastActivity: timestamppb.New(time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))},
		{UserId: "u-2", Status: videov1.PresenceLevel_PRESENCE_UNSPECIFIED},
	}
	rec := httptest.NewRecorder()
	routes.HandleGetBulkPresence(rec, videoReq(t, http.MethodPost, "/api/v1/video/presence/bulk",
		map[string]any{"user_ids": []string{"u-1", "u-2"}}, nil))
	assertStatus(t, rec, http.StatusOK)

	var got map[string]map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 2)
	assert.NotContains(t, got, "statuses")
	assert.Equal(t, "in_call", got["u-1"]["status"])
	assert.Equal(t, "2026-03-02T09:00:00Z", got["u-1"]["last_activity"])
	assert.Equal(t, "offline", got["u-2"]["status"])
	assert.Nil(t, got["u-2"]["last_activity"])
}

func TestHandleGetPresence_StringLevel(t *testing.T) {
	routes, stub := newVideoWireRoutes(t)
	stub.statuses = []*videov1.PresenceStatus{{UserId: "u-1", Status: videov1.PresenceLevel_PRESENCE_DND, ManualOverride: true}}
	rec := httptest.NewRecorder()
	routes.HandleGetPresence(rec, videoReq(t, http.MethodGet, "/api/v1/video/presence/u-1", nil, map[string]string{"userId": "u-1"}))
	assertStatus(t, rec, http.StatusOK)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "dnd", got["status"])
	assert.Equal(t, true, got["manual_override"])
}

func TestHandleListCoHosts_AddsUserIDs(t *testing.T) {
	routes, stub := newVideoWireRoutes(t)
	meeting := uuid.New().String()
	params := map[string]string{"meetingId": meeting}

	rec := httptest.NewRecorder()
	routes.HandleListCoHosts(rec, videoReq(t, http.MethodGet, "/x", nil, params))
	var empty map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &empty))
	assert.Equal(t, []any{}, empty["user_ids"])
	assert.Equal(t, []any{}, empty["co_hosts"])

	stub.coHosts = []*videov1.MeetingCoHost{{Id: "c-1", UserId: "u-1"}, {Id: "c-2", UserId: "u-2"}}
	rec = httptest.NewRecorder()
	routes.HandleListCoHosts(rec, videoReq(t, http.MethodGet, "/x", nil, params))
	var got struct {
		UserIDs []string         `json:"user_ids"`
		CoHosts []map[string]any `json:"co_hosts"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, []string{"u-1", "u-2"}, got.UserIDs)
	assert.Len(t, got.CoHosts, 2)
}

func TestHandleGetBreakoutAssignment_MainRoomIsExplicitNull(t *testing.T) {
	routes, stub := newVideoWireRoutes(t)
	meeting := uuid.New().String()
	params := map[string]string{"id": meeting}

	rec := httptest.NewRecorder()
	routes.HandleGetBreakoutAssignment(rec, videoReq(t, http.MethodGet, "/x", nil, params))
	assert.JSONEq(t, `{"room":null}`, rec.Body.String())

	stub.room = &videov1.BreakoutRoom{Id: "b-1", Label: "Gruppe 1"}
	rec = httptest.NewRecorder()
	routes.HandleGetBreakoutAssignment(rec, videoReq(t, http.MethodGet, "/x", nil, params))
	var got struct {
		Room map[string]any `json:"room"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "b-1", got.Room["id"])
}
