package gateway

// Wire shape of the video/meeting routes against the desktop types
// (api/video-client.ts, api/video-types.ts): recordings and action items as
// arrays the hooks can .find()/.filter(), convert with task_ids, presence as a
// userId map with string levels, co-hosts with user_ids and an explicit
// `room: null` in the main room.

import (
	"net/http"
	"strings"
	"time"

	"github.com/kmuhub/kmuhub/internal/server/response"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// presenceLevelName turns the proto enum into the lowercase union the desktop
// declares (PresenceLevel). Unknown or unspecified reads as offline.
func presenceLevelName(l videov1.PresenceLevel) string {
	if l == videov1.PresenceLevel_PRESENCE_UNSPECIFIED {
		return "offline"
	}
	return strings.ToLower(strings.TrimPrefix(l.String(), "PRESENCE_"))
}

// presenceObject is the UserPresence the desktop reads for one user.
func presenceObject(p *videov1.PresenceStatus) map[string]any {
	var lastActivity any
	if p.GetLastActivity() != nil {
		lastActivity = p.GetLastActivity().AsTime().Format(time.RFC3339Nano)
	}
	return map[string]any{
		"user_id":         p.GetUserId(),
		"status":          presenceLevelName(p.GetStatus()),
		"manual_override": p.GetManualOverride(),
		"last_activity":   lastActivity,
		"first_name":      p.GetFirstName(),
		"last_name":       p.GetLastName(),
	}
}

// presenceByUser keys the bulk result by user id: the desktop iterates
// Object.entries() and reads `.status` from every value.
func presenceByUser(statuses []*videov1.PresenceStatus) map[string]any {
	out := make(map[string]any, len(statuses))
	for _, p := range statuses {
		out[p.GetUserId()] = presenceObject(p)
	}
	return out
}

// convertedTaskIDs lists the task ids of the converted action items; never nil.
func convertedTaskIDs(items []*videov1.ActionItem) []string {
	ids := make([]string, 0, len(items))
	for _, it := range items {
		if it.TaskId != nil && *it.TaskId != "" {
			ids = append(ids, *it.TaskId)
		}
	}
	return ids
}

// writeConvertedActionItems answers the convert route: the items plus the count
// (as before) and the `task_ids` the desktop toast counts.
func writeConvertedActionItems(w http.ResponseWriter, resp *videov1.ConvertActionItemsToTasksResponse) {
	response.ProtoListWrapped(w, http.StatusOK, "action_items", resp.GetActionItems(), map[string]any{
		"converted_count": resp.GetConvertedCount(),
		"task_ids":        convertedTaskIDs(resp.GetActionItems()),
	})
}

// coHostUserIDs lists the user ids of the co-hosts; never nil.
func coHostUserIDs(hosts []*videov1.MeetingCoHost) []string {
	ids := make([]string, 0, len(hosts))
	for _, h := range hosts {
		ids = append(ids, h.GetUserId())
	}
	return ids
}

// writeCoHosts keeps `co_hosts` and adds the flat `user_ids` the desktop reads.
func writeCoHosts(w http.ResponseWriter, resp *videov1.ListCoHostsResponse) {
	response.ProtoListWrapped(w, http.StatusOK, "co_hosts", resp.GetCoHosts(), map[string]any{
		"user_ids": coHostUserIDs(resp.GetCoHosts()),
	})
}

// writeBreakoutAssignment answers `{room: null}` in the main room; a missing key
// would read as undefined, which the desktop type (`room: BreakoutRoom | null`)
// does not allow.
func writeBreakoutAssignment(w http.ResponseWriter, resp *videov1.GetBreakoutAssignmentResponse) {
	if resp.Room == nil {
		response.JSON(w, http.StatusOK, map[string]any{"room": nil})
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// writeMeetingRecordings keeps the paginated envelope the desktop type
// (RecordingsByMeetingResponse) declares, with `recordings` always an array.
func writeMeetingRecordings(w http.ResponseWriter, resp *videov1.ListRecordingsByMeetingResponse) {
	response.ProtoListWrapped(w, http.StatusOK, "recordings", resp.Recordings, map[string]any{
		"total": resp.Total,
	})
}
