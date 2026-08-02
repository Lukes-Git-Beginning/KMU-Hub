package gateway

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// ============================================================================
// Time Tracking Handlers
// ============================================================================

func (w *WorkRoutes) HandleStartTimer(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	taskID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	resp, err := client.StartTimer(r.Context(), &workv1.StartTimerRequest{
		TaskId: taskID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusCreated, resp)
}

func (w *WorkRoutes) HandleStopTimer(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.StopTimer(r.Context(), &workv1.StopTimerRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleGetActiveTimer(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.GetActiveTimer(r.Context(), &workv1.GetActiveTimerRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

type addManualTimeEntryRequest struct {
	StartedAt       string  `json:"started_at" validate:"required"`
	DurationSeconds int     `json:"duration_seconds" validate:"gt=0"`
	Description     *string `json:"description,omitempty"`
}

func (w *WorkRoutes) HandleAddManualTimeEntry(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	taskID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[addManualTimeEntryRequest](wr, r)
	if !ok {
		return
	}

	startedAt, parseErr := parseTimestamp(req.StartedAt)
	if parseErr != nil {
		response.Error(wr, http.StatusBadRequest, "invalid started_at format")
		return
	}

	grpcReq := &workv1.AddManualTimeEntryRequest{
		TaskId:          taskID,
		UserId:          userID,
		StartedAt:       startedAt,
		DurationSeconds: int32(req.DurationSeconds),
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}

	resp, err := client.AddManualTimeEntry(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusCreated, resp)
}

type updateTimeEntryRequest struct {
	StartedAt       *string `json:"started_at,omitempty"`
	DurationSeconds *int    `json:"duration_seconds,omitempty" validate:"omitempty,gt=0"`
	Description     *string `json:"description,omitempty"`
}

func (w *WorkRoutes) HandleUpdateTimeEntry(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	entryID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateTimeEntryRequest](wr, r)
	if !ok {
		return
	}

	grpcReq := &workv1.UpdateTimeEntryRequest{
		Id:     entryID,
		UserId: userID,
	}
	if req.StartedAt != nil {
		t, parseErr := parseTimestamp(*req.StartedAt)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid started_at format")
			return
		}
		grpcReq.StartedAt = t
	}
	if req.DurationSeconds != nil {
		d := int32(*req.DurationSeconds)
		grpcReq.DurationSeconds = &d
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}

	resp, err := client.UpdateTimeEntry(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleDeleteTimeEntry(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	entryID := chi.URLParam(r, "id")

	_, err = client.DeleteTimeEntry(r.Context(), &workv1.DeleteTimeEntryRequest{
		Id:     entryID,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "time entry deleted"})
}

func (w *WorkRoutes) HandleListTimeEntries(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")
	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListTimeEntries(r.Context(), &workv1.ListTimeEntriesRequest{
		TaskId:   taskID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

// projectTimeEntryWire is the exact shape the desktop client reads
// (ProjectTimeEntry in useProjects.ts, consumed via useProjectTimeEntries).
type projectTimeEntryWire struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Task        string  `json:"task"`
	Person      string  `json:"person"`
	Hours       float64 `json:"hours"`
	Description string  `json:"description"`
}

// HandleListProjectTimeEntries serves the project-level "Stunden abrechnen"
// roll-up. GET /api/v1/projects/{id}/time-entries?billed=false
func (w *WorkRoutes) HandleListProjectTimeEntries(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	// lean: same gap as GET /finance/time-entries (route_biz_time_entries.go)
	// -- nothing persists which entries an invoice covered, so every entry is
	// reported unbilled until that linkage exists. billed=true always reports
	// none rather than guessing.
	if r.URL.Query().Get("billed") == "true" {
		response.JSON(wr, http.StatusOK, map[string]any{"entries": []projectTimeEntryWire{}})
		return
	}

	resp, err := client.ListProjectTimeEntries(r.Context(), &workv1.ListProjectTimeEntriesRequest{
		ProjectId: projectID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	entries := make([]projectTimeEntryWire, 0, len(resp.GetEntries()))
	for _, e := range resp.GetEntries() {
		entries = append(entries, toProjectTimeEntryWire(e))
	}

	response.JSON(wr, http.StatusOK, map[string]any{"entries": entries})
}

func toProjectTimeEntryWire(e *workv1.ProjectTimeEntryProto) projectTimeEntryWire {
	return projectTimeEntryWire{
		ID:          e.GetId(),
		Date:        e.GetDate(),
		Task:        e.GetTask(),
		Person:      e.GetPerson(),
		Hours:       e.GetHours(),
		Description: e.GetDescription(),
	}
}

// utilizationPointWire is one labeled period point in a member's hours trend.
type utilizationPointWire struct {
	Label string  `json:"label"`
	Hours float64 `json:"hours"`
}

// memberUtilizationMemberWire is the exact shape the desktop client reads
// for MemberUtilization.member (useProjects.ts, consumed via
// useProjectTeamUtilization). Deliberately has no "rate" field -- this
// route sits behind project-read, not team:salary:view, so no cost/salary
// figure may flow through it. The desktop client stays mock-served for its
// cost card until that permission boundary is wired up separately.
type memberUtilizationMemberWire struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	AvatarInitial string `json:"avatarInitial"`
	WeeklyTarget  int    `json:"weeklyTarget"`
}

type memberUtilizationWire struct {
	Member      memberUtilizationMemberWire `json:"member"`
	WeeklyData  []utilizationPointWire      `json:"weeklyData"`
	MonthlyData []utilizationPointWire      `json:"monthlyData"`
}

// HandleListProjectTeamUtilization serves the project's "Auslastung"
// roll-up. GET /api/v1/projects/{id}/team-utilization
func (w *WorkRoutes) HandleListProjectTeamUtilization(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListProjectTeamUtilization(r.Context(), &workv1.ListProjectTeamUtilizationRequest{
		ProjectId: projectID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	team := make([]memberUtilizationWire, 0, len(resp.GetTeam()))
	for _, m := range resp.GetTeam() {
		team = append(team, toMemberUtilizationWire(m))
	}

	response.JSON(wr, http.StatusOK, map[string]any{"team": team})
}

func toMemberUtilizationWire(m *workv1.ProjectMemberUtilizationProto) memberUtilizationWire {
	return memberUtilizationWire{
		Member: memberUtilizationMemberWire{
			ID:            m.GetUserId(),
			Name:          m.GetName(),
			Role:          m.GetRole(),
			AvatarInitial: avatarInitial(m.GetName()),
			WeeklyTarget:  int(m.GetWeeklyTarget()),
		},
		WeeklyData:  toUtilizationPointsWire(m.GetWeeklyData()),
		MonthlyData: toUtilizationPointsWire(m.GetMonthlyData()),
	}
}

func toUtilizationPointsWire(points []*workv1.UtilizationPointProto) []utilizationPointWire {
	out := make([]utilizationPointWire, 0, len(points))
	for _, p := range points {
		out = append(out, utilizationPointWire{Label: p.GetLabel(), Hours: p.GetHours()})
	}
	return out
}

func avatarInitial(name string) string {
	r := []rune(strings.TrimSpace(name))
	if len(r) == 0 {
		return ""
	}
	return strings.ToUpper(string(r[0]))
}

func (w *WorkRoutes) HandleGetTaskTimeSummary(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")

	resp, err := client.GetTaskTimeSummary(r.Context(), &workv1.GetTaskTimeSummaryRequest{
		TaskId: taskID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}
